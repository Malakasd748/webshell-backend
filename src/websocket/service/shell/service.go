package shell

import (
	"encoding/json"
	"io"
	"log"
	"sync"

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
	Cwd string `json:"cwd"`
}

type ShellService struct {
	conn   *ws.Conn
	shells map[string]Shell

	ShellProvider

	*log.Logger
	*sync.RWMutex
}

func (s *ShellService) Name() string {
	return "shell"
}

func (s *ShellService) Register(conn *ws.Conn) {
	s.conn = conn
}

func (s *ShellService) HandleTextMessage(id string, action string, data json.RawMessage) {
	s.RLock()
	sh, exists := s.shells[id]
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
		if _, err := sh.Write([]byte(command)); err != nil {
			s.Printf("(id: %s) error writing to shell: %v", id, err)
			// 在前端 shell 里输 `exit` 后，shell 已关闭，但 websocket 连接还没断
			// 暂时在此处这样处理，可以在用户再次操作时触发重连
			s.conn.Close()
			return
		}
	case actionResize:
		var resize resizeData
		if err := json.Unmarshal(data, &resize); err != nil {
			s.Printf("(id: %s) error unmarshalling resize payload: %v", id, err)
			return
		}
		if err := sh.Resize(resize.Rows, resize.Cols); err != nil {
			s.Printf("(id: %s) error resizing shell: %v", id, err)
			return
		}
	case actionStart:
		var start startData
		if err := json.Unmarshal(data, &start); err != nil {
			s.Printf("(id: %s) error unmarshalling start payload: %v", id, err)
			return
		}
		if err := s.startShell(id, start.Cwd); err != nil {
			s.Printf("(id: %s) error starting shell: %v", id, err)
			return
		}
		s.conn.WriteJSON(&ws.ServiceMessage{
			Service: s.Name(),
			Id:      id,
			Action:  actionStart,
		})
	case actionTerminate:
		sh.Close()

		s.Lock()
		delete(s.shells, id)
		s.Unlock()
	}
}

func (s *ShellService) Cleanup(err error) {
	for _, sh := range s.shells {
		sh.Close()
	}
	s.shells = nil
}

func (s *ShellService) startShell(id string, cwd string) error {
	sh, err := s.ShellProvider.NewShell(cwd)
	if err != nil {
		return err
	}

	s.Lock()
	s.shells[id] = sh
	s.Unlock()

	// 发送 shell 输出
	writer := &utils.WebsocketWriter{
		Service: s.Name(),
		Id:      id,
		Action:  actionCommand,
		Conn:    s.conn,
		Tranformer: func(p []byte) []byte {
			d, _ := json.Marshal(string(p))
			return d
		},
	}
	go io.Copy(writer, sh)

	return nil
}
