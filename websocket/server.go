package websocket

import (
	"log"
	"net/http"
	"slices"
	"time"
)

type Server struct {
	*Conn
	// 暂时不需要锁
	services map[string]Service

	lastActiveTime time.Time
	activeServices []string
}

func (s *Server) checkTimeout() {
	ticker := time.NewTicker(time.Second * 10)
	for range ticker.C {
		if time.Since(s.lastActiveTime) > connectionTimeout {
			s.Close()
		}
	}
}

func (s *Server) Register(service Service) {
	s.RegisterPassive(service)
	s.activeServices = append(s.activeServices, service.Name())
}

func (s *Server) RegisterPassive(service Service) {
	if _, exists := s.services[service.Name()]; exists {
		log.Printf("service %s already registered", service.Name())
		return
	}

	service.Register(s.Conn)
	s.services[service.Name()] = service

}

func (s *Server) Start() {
	go s.checkTimeout()

	err := s.StartDispatch()
	for _, s := range s.services {
		s.Cleanup(err)
	}

	// Start goroutine to handle messages
	go func() {
		for msg := range s.TextChan {
			if slices.Contains(s.activeServices, msg.Service) {
				s.lastActiveTime = time.Now()
			}
			if s, exists := s.services[msg.Service]; exists {
				s.HandleTextMessage(msg.Id, msg.Action, msg.Data)
			}
		}
	}()
}

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
