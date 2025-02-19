package websocket

import (
	"net/http"
	"time"
)

func NewServer(w http.ResponseWriter, r *http.Request) (*Server, error) {
	conn, err := NewConn(w, r)
	if err != nil {
		return nil, err
	}

	server := &Server{
		Conn:           conn,
		services:       make(map[string]Service),
		lastActiveTime: time.Now(),
	}

	return server, nil
}
