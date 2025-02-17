package websocket

import (
	"encoding/json"
)

type Service interface {
	HandleTextMessage(id string, action string, data json.RawMessage)
	HandleBinaryMessage(data []byte)
	Name() string
	Cleanup(err error)
	Register(conn *Conn)
}

type ServiceMessage struct {
	Service string          `json:"service"`
	Id      string          `json:"id,omitempty"`
	Action  string          `json:"action,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}
