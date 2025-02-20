package utils

import (
	ws "webshell/websocket"
)

type WebsocketWriter struct {
	Service    string
	Id         string
	Action     string
	Conn       *ws.Conn
	Tranformer func([]byte) []byte
}

func (w *WebsocketWriter) Write(p []byte) (n int, err error) {
	var transformed []byte
	if w.Tranformer != nil {
		transformed = w.Tranformer(p)
	} else {
		transformed = p
	}

	err = w.Conn.WriteJSON(&ws.ServiceMessage{
		Service: w.Service,
		Id:      w.Id,
		Action:  w.Action,
		Data:    transformed,
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}
