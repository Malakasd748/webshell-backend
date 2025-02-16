package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"

	"webshell/service"
	"webshell/service/fs"
	"webshell/service/heartbeat"
	"webshell/service/pty"
	"webshell/service/upload"
	"webshell/websocket"
)

type Server struct {
	*websocket.Conn
	// 暂时不需要锁
	services map[string]service.Service

	lastActiveTime time.Time
}

func (s *Server) checkTimeout() {
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for range ticker.C {
			if time.Since(s.lastActiveTime) > connectionTimeout {
				s.Close()
			}
		}
	}()
}

func NewServer(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.NewConn(w, r)
	if err != nil {
		return
	}

	hearbeat := heartbeat.NewHearbeatService(conn)
	pty := pty.NewPTYService(conn)
	fs := fs.NewFSService(conn)
	upload := upload.NewUploadService(conn)

	server := &Server{
		Conn:           conn,
		services:       make(map[string]service.Service, 4),
		lastActiveTime: time.Now(),
	}

	server.checkTimeout()

	server.services[hearbeat.Name()] = hearbeat
	server.services[pty.Name()] = pty
	server.services[fs.Name()] = fs
	server.services[upload.Name()] = upload

	for {
		msgType, data, err := server.ReadMessage()
		if err != nil {
			for _, s := range server.services {
				s.Cleanup(err)
			}
			break
		}

		if msgType == ws.BinaryMessage {
			upload.WriteChunkData(data)
			continue
		}

		var msg service.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("error unmarshalling message: %v", err)
			continue
		}

		if msg.Service != hearbeat.Name() {
			server.lastActiveTime = time.Now()
		}

		if s, exists := server.services[msg.Service]; exists {
			s.HandleMessage(msg.Id, msg.Action, msg.Data)
		}
	}

}

func Start(port uint) chan int {
	http.HandleFunc("/", NewServer)

	finishChan := make(chan int)
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		finishChan <- 1
	}()
	log.Printf("started on port %d", port)
	return finishChan
}
