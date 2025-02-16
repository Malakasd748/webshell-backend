package service

import (
	"encoding/json"
)

type Service interface {
	HandleMessage(id string, action string, data json.RawMessage)
	Name() string
	Cleanup(err error)
}

type Message struct {
	Service string          `json:"service"`
	Id      string          `json:"id,omitempty"`
	Action  string          `json:"action,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}
