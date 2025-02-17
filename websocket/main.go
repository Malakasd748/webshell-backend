package websocket

import (
	"log"
	"net/http"
	"sync"

	ws "github.com/gorilla/websocket"
)

type Conn struct {
	*ws.Conn
	*sync.Mutex
}

var (
	upgrader = ws.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool { return true }}
)

func (c *Conn) WriteJSON(v interface{}) error {
	c.Lock()
	err := c.Conn.WriteJSON(v)
	c.Unlock()

	if err != nil {
		log.Printf("Websocket::WriteJson error: %v", err)
	}
	return err
}

func (c *Conn) WriteBinary(data []byte) error {
	c.Lock()
	err := c.Conn.WriteMessage(ws.BinaryMessage, data)
	c.Unlock()

	if err != nil {
		log.Printf("Websocket::WriteBinary error: %v", err)
	}
	return err
}

func NewConn(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Websocket upgrade error: %v", err)
		return nil, err
	}

	result := &Conn{
		Conn:  conn,
		Mutex: new(sync.Mutex),
	}

	return result, nil
}
