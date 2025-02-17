package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	libws "github.com/gorilla/websocket"

	"webshell/service/fs"
	"webshell/service/heartbeat"
	"webshell/service/pty"
	"webshell/service/upload"
	ws "webshell/websocket"
)

type Server struct {
	*ws.Conn
	// 暂时不需要锁
	services map[string]ws.Service

	lastActiveTime time.Time

	textChan   chan ws.ServiceMessage
	binaryChan chan []byte
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

func (s *Server) register(service ws.Service) {
	if _, exists := s.services[service.Name()]; exists {
		log.Printf("service %s already registered", service.Name())
		return
	}

	service.Register(s.Conn)
	s.services[service.Name()] = service
}

func NewServer(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.NewConn(w, r)
	if err != nil {
		return
	}

	hearbeat := heartbeat.NewService()
	ptyService := pty.NewService()
	fsService := fs.NewService()
	uploadService := upload.NewService()

	server := &Server{
		Conn:           conn,
		services:       make(map[string]ws.Service, 4),
		lastActiveTime: time.Now(),
	}

	// Initialize channels for text and binary messages
	server.textChan = make(chan ws.ServiceMessage, 10)
	server.binaryChan = make(chan []byte, 10)

	server.checkTimeout()

	server.register(hearbeat)
	server.register(ptyService)
	server.register(fsService)
	server.register(uploadService)

	// Start goroutine to handle binary messages
	go func() {
		for binData := range server.binaryChan {
			uploadService.WriteChunkData(binData)
		}
	}()

	// Start goroutine to handle text messages
	go func() {
		for msg := range server.textChan {
			if msg.Service != hearbeat.Name() {
				server.lastActiveTime = time.Now()
			}
			if s, exists := server.services[msg.Service]; exists {
				s.HandleMessage(msg.Id, msg.Action, msg.Data)
			}
		}
	}()

	for {
		msgType, data, err := server.ReadMessage()
		if err != nil {
			for _, s := range server.services {
				s.Cleanup(err)
			}
			break
		}

		if msgType == libws.BinaryMessage {
			server.binaryChan <- data
			continue
		}

		var msg ws.ServiceMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("error unmarshalling message: %v", err)
			continue
		}

		server.textChan <- msg
	}

	// Close channels after loop ends
	close(server.binaryChan)
	close(server.textChan)
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
