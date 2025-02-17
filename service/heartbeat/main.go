package heartbeat

import (
	"encoding/json"

	"webshell/service"
	"webshell/websocket"
)

type HeartbeatService struct {
	conn *websocket.Conn
}

func (s *HeartbeatService) Name() string {
	return "heartbeat"
}

func (s *HeartbeatService) HandleMessage(id, action string, data json.RawMessage) {
	s.conn.WriteJSON(&service.Message{Service: s.Name(), Action: action, Id: id})
}

func (s *HeartbeatService) Cleanup(err error) {}

func NewHearbeatService(conn *websocket.Conn) *HeartbeatService {
	return &HeartbeatService{conn: conn}
}
