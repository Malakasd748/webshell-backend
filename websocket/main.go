package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	ws "github.com/gorilla/websocket"
)

type Conn struct {
	*ws.Conn
	*sync.Mutex
	// Exposed channels for text and binary messages
	TextChan   chan *ServiceMessage
	BinaryChan chan []byte
}

var (
	upgrader = ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func (c *Conn) WriteJSON(v any) error {
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

// NewConn initializes the connection and its channels
func NewConn(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Websocket upgrade error: %v", err)
		return nil, err
	}

	result := &Conn{
		Conn:       conn,
		Mutex:      new(sync.Mutex),
		TextChan:   make(chan *ServiceMessage, 10),
		BinaryChan: make(chan []byte, 10),
	}

	return result, nil
}

// StartDispatch reads messages and dispatches them into appropriate channels
func (c *Conn) StartDispatch() error {
	for {
		msgType, data, err := c.ReadMessage()
		if err != nil {
			close(c.TextChan)
			close(c.BinaryChan)
			return err
		}

		if msgType == ws.BinaryMessage {
			c.BinaryChan <- data
		} else {
			var msg ServiceMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("error unmarshalling message: %v", err)
				continue
			}
			c.TextChan <- &msg
		}
	}
}
