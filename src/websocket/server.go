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
		log.Printf("Service %s already registered.", service.Name())
		return
	}

	service.Register(s.Conn)
	s.services[service.Name()] = service
}

func (s *Server) Start() error {
	go s.checkTimeout()

	// 处理文本信息
	go func() {
		for msg := range s.TextMessage {
			if slices.Contains(s.activeServices, msg.Service) {
				s.lastActiveTime = time.Now()
			}
			if s, exists := s.services[msg.Service]; exists {
				s.HandleTextMessage(msg.Id, msg.Action, msg.Data)
			}
		}
	}()

	// 处理二进制信息
	go func() {
		for data := range s.BinaryMessage {
			// 每次取数据的时候先取 BinaryChan ，取后只传入一条数据。
			// BinaryChan 由服务在准备接受二进制数据时传入。这样就能有多个服务并发处理二进制数据。
			ch := <-s.BinaryChan
			ch <- data
		}
	}()

	err := s.StartDispatch()
	for _, s := range s.services {
		s.Cleanup(err)
	}
	return err
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
		activeServices: make([]string, 0, 3),
	}

	return server, nil
}
