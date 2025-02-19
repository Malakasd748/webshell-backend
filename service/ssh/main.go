package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"golang.org/x/crypto/ssh"

	"webshell/utils"
	ws "webshell/websocket"
)

const (
	actionCommand   = "command"
	actionResize    = "resize"
	actionStart     = "start"
	actionTerminate = "terminate"
)

type commandData string
type resizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}
type startData struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type sshSession struct {
	*ssh.Session
	stdin     io.WriteCloser
	terminate context.CancelFunc
}

type SSHService struct {
	conn     *ws.Conn
	sessions map[string]*sshSession

	*ssh.Client
	*log.Logger
	*sync.RWMutex
}

func (s *SSHService) Register(conn *ws.Conn) {
	s.conn = conn
}

func (s *SSHService) Name() string {
	return "ssh"
}

func (s *SSHService) HandleTextMessage(id string, action string, data json.RawMessage) {
	s.RLock()
	session, exists := s.sessions[id]
	s.RUnlock()

	if action != actionStart && !exists {
		s.Printf("(id: %s) received message before SSH session started", id)
		return
	} else if action == actionStart && exists {
		s.Printf("(id: %s) received start message after SSH session started", id)
		return
	}

	switch action {
	case actionCommand:
		var command commandData
		if err := json.Unmarshal(data, &command); err != nil {
			s.Printf("(id: %s) error unmarshalling command payload: %v", id, err)
			return
		}
		if _, err := session.stdin.Write([]byte(command)); err != nil {
			s.Printf("(id: %s) error writing to ssh session: %v", id, err)
			return
		}
	case actionResize:
		var resize resizeData
		if err := json.Unmarshal(data, &resize); err != nil {
			s.Printf("(id: %s) error unmarshalling resize payload: %v", id, err)
			return
		}
		if err := session.Session.WindowChange(resize.Rows, resize.Cols); err != nil {
			s.Printf("(id: %s) error resizing ssh window: %v", id, err)
			return
		}
	case actionStart:
		var start startData
		if err := json.Unmarshal(data, &start); err != nil {
			s.Printf("(id: %s) error unmarshalling start payload: %v", id, err)
			return
		}
		if err := s.startSession(id); err != nil {
			s.Printf("(id: %s) error starting ssh session: %v", id, err)
			return
		}
		s.conn.WriteJSON(&ws.ServiceMessage{
			Service: s.Name(),
			Id:      id,
			Action:  actionStart,
		})
	case actionTerminate:
		session.terminate()
		s.Lock()
		delete(s.sessions, id)
		s.Unlock()
	}
}

func (s *SSHService) Cleanup(err error) {
	for _, session := range s.sessions {
		session.terminate()
	}
	s.sessions = nil
	s.Close()
}

func (s *SSHService) startSession(id string) error {
	ctx, cancel := context.WithCancel(context.Background())

	ss, err := s.NewSession()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create session: %v", err)
	}

	stdin, err := ss.StdinPipe()
	if err != nil {
		ss.Close()
		cancel()
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := ss.RequestPty("xterm-256color", 60, 80, modes); err != nil {
		stdin.Close()
		ss.Close()
		cancel()
		return fmt.Errorf("failed to request pty: %v", err)
	}

	stdout, err := ss.StdoutPipe()
	if err != nil {
		stdin.Close()
		ss.Close()
		cancel()
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	stderr, err := ss.StderrPipe()
	if err != nil {
		stdin.Close()
		ss.Close()
		cancel()
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	if err := ss.Shell(); err != nil {
		stdin.Close()
		ss.Close()
		cancel()
		return fmt.Errorf("failed to start shell: %v", err)
	}

	session := &sshSession{
		Session:   ss,
		stdin:     stdin,
		terminate: cancel,
	}

	s.Lock()
	s.sessions[id] = session
	s.Unlock()

	// Cleanup when context is done
	go func() {
		<-ctx.Done()
		stdin.Close()
		ss.Close()
	}()

	// Handle output
	writer := &utils.WebsocketWriter{
		Service: s.Name(),
		Id:      id,
		Action:  actionCommand,
		Conn:    s.conn,
		Tranformer: func(b []byte) []byte {
			data, _ := json.Marshal(string(b))
			return data
		},
	}

	io.Copy(writer, stdout)
	io.Copy(writer, stderr)

	return nil
}

func NewService(network, addr string, config *ssh.ClientConfig) (ws.Service, error) {
	client, err := ssh.Dial(network, addr, config)
	if err != nil {
		return nil, err
	}

	service := &SSHService{
		Client:   client,
		Logger:   log.New(log.Writer(), "ssh: ", log.LstdFlags),
		RWMutex:  new(sync.RWMutex),
		sessions: make(map[string]*sshSession),
	}

	return service, nil
}
