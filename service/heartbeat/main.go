package heartbeat

import (
	"encoding/json"

	ws "webshell/websocket"
)

type HeartbeatService struct {
	conn *ws.Conn
}

func (s *HeartbeatService) Name() string {
	return "heartbeat"
}

func (s *HeartbeatService) Register(conn *ws.Conn) {
	s.conn = conn
}

func (s *HeartbeatService) HandleTextMessage(id, action string, data json.RawMessage) {
	s.conn.WriteJSON(&ws.ServiceMessage{Service: s.Name(), Action: action, Id: id})
}

func (s *HeartbeatService) Cleanup(err error) {}

func NewService() ws.Service {
	return &HeartbeatService{}
}
