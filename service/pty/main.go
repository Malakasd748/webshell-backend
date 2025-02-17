package pty

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"

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
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}
type startData struct {
	Cwd string `json:"cwd"`
}

type ptyProcess struct {
	*os.File
	terminate context.CancelFunc
}

type PTYService struct {
	conn  *ws.Conn
	procs map[string]*ptyProcess

	*log.Logger
	*sync.RWMutex
}

func (s *PTYService) Name() string {
	return "pty"
}

func (s *PTYService) Register(conn *ws.Conn) {
	s.conn = conn
}

func (s *PTYService) HandleTextMessage(id string, action string, data json.RawMessage) {
	s.RLock()
	proc, exists := s.procs[id]
	s.RUnlock()

	if action != actionStart && !exists {
		s.Printf("(id: %s) received message before terminal started", id)
		return
	} else if action == actionStart && exists {
		s.Printf("(id: %s) received start message after terminal started", id)
		return
	}
	switch action {
	case actionCommand:
		var command commandData
		if err := json.Unmarshal(data, &command); err != nil {
			s.Printf("(id: %s) error unmarshalling command payload: %v", id, err)
			return
		}
		if _, err := proc.Write([]byte(command)); err != nil {
			s.Printf("(id: %s) error writing to pty: %v", id, err)
			return
		}
	case actionResize:
		var resize resizeData
		if err := json.Unmarshal(data, &resize); err != nil {
			s.Printf("(id: %s) error unmarshalling resize payload: %v", id, err)
			return
		}
		if err := pty.Setsize(proc.File, &pty.Winsize{Cols: resize.Cols, Rows: resize.Rows}); err != nil {
			s.Printf("(id: %s) error resizing pty: %v", id, err)
			return
		}
	case actionStart:
		var start startData
		if err := json.Unmarshal(data, &start); err != nil {
			s.Printf("(id: %s) error unmarshalling start payload: %v", id, err)
			return
		}
		if err := s.startPty(id, start.Cwd); err != nil {
			s.Printf("(id: %s) error starting pty: %v", id, err)
		}
		s.conn.WriteJSON(&ws.ServiceMessage{
			Service: s.Name(),
			Id:      id,
			Action:  actionStart,
		})
	case actionTerminate:
		proc.terminate()
		s.Lock()
		delete(s.procs, id)
		s.Unlock()
	}
}

func (s *PTYService) Cleanup(err error) {
	for _, proc := range s.procs {
		proc.terminate()
	}
	s.procs = nil
}

func (s *PTYService) startPty(id string, cwd string) error {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, "bash", "-l")
	if cwd != "" {
		cmd.Dir = cwd
	} else {
		cmd.Dir = ptyCWD
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "LC_ALL=C.UTF-8")

	f, err := pty.Start(cmd)
	if err != nil {
		s.Printf("Failed to start pty: %v", err)
		cancel()
		return err
	}

	p := &ptyProcess{File: f, terminate: cancel}

	s.Lock()
	s.procs[id] = p
	s.Unlock()

	// 清理资源
	go func() {
		<-ctx.Done()
		cmd.Process.Kill()
		cmd.Process.Wait()
		if p.File != nil {
			p.Close()
		}
	}()

	// 发送 pty 输出
	go func() {
		buf := make([]byte, 1024)
		for {
			read, err := p.Read(buf)
			if err != nil {
				s.Printf("(id: %s) pty read error: %v", id, err)
				break
			}

			commandData, _ := json.Marshal(string(buf[:read]))

			s.conn.WriteJSON(&ws.ServiceMessage{
				Service: s.Name(),
				Id:      id,
				Action:  actionCommand,
				Data:    commandData,
			})
		}
	}()

	return nil
}

func NewService() ws.Service {
	return &PTYService{
		procs:   make(map[string]*ptyProcess),
		Logger:  log.New(os.Stdout, "[pty] ", log.LstdFlags),
		RWMutex: &sync.RWMutex{},
	}
}
