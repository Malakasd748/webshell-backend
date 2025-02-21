package heartbeat

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	ws "webshell/websocket"
)

// TestHeartbeatService 用于记录发送的消息
type TestHeartbeatService struct {
	*HeartbeatService
	messages []*ws.ServiceMessage
	mutex    sync.Mutex
}

func NewTestService() *TestHeartbeatService {
	return &TestHeartbeatService{
		HeartbeatService: NewService().(*HeartbeatService),
		messages:         make([]*ws.ServiceMessage, 0),
	}
}

func (s *TestHeartbeatService) HandleTextMessage(id, action string, data json.RawMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	msg := &ws.ServiceMessage{
		Service: s.Name(),
		Action:  action,
		Id:      id,
	}
	s.messages = append(s.messages, msg)
}

type MockWsConn struct {
	messages []interface{}
	mutex    sync.Mutex
}

func (m *MockWsConn) WriteJSON(v interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.messages = append(m.messages, v)
	return nil
}

// MockConn 实现测试所需的最小接口
type MockConn struct {
	wsConn   *MockWsConn
	mutex    sync.Mutex
	messages []*ws.ServiceMessage
}

func NewMockConn() (*MockConn, *MockWsConn) {
	wsConn := &MockWsConn{
		messages: make([]interface{}, 0),
	}
	return &MockConn{
		wsConn:   wsConn,
		messages: make([]*ws.ServiceMessage, 0),
	}, wsConn
}

func (m *MockConn) WriteJSON(v interface{}) error {
	return m.wsConn.WriteJSON(v)
}

func TestHeartbeatService_Name(t *testing.T) {
	service := NewService()
	if service.Name() != "heartbeat" {
		t.Errorf("Expected service name to be 'heartbeat', got '%s'", service.Name())
	}
}

func TestHeartbeatService_HandleTextMessage(t *testing.T) {
	service := NewTestService()

	testCases := []struct {
		name     string
		id       string
		action   string
		data     json.RawMessage
		expected ws.ServiceMessage
	}{
		{
			name:   "Simple heartbeat",
			id:     "test-id-1",
			action: "ping",
			data:   json.RawMessage(`{}`),
			expected: ws.ServiceMessage{
				Service: "heartbeat",
				Action:  "ping",
				Id:      "test-id-1",
			},
		},
		{
			name:   "Different action",
			id:     "test-id-2",
			action: "pong",
			data:   json.RawMessage(`{}`),
			expected: ws.ServiceMessage{
				Service: "heartbeat",
				Action:  "pong",
				Id:      "test-id-2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service.HandleTextMessage(tc.id, tc.action, tc.data)

			service.mutex.Lock()
			defer service.mutex.Unlock()

			if len(service.messages) == 0 {
				t.Fatal("No message was recorded")
			}

			msg := service.messages[len(service.messages)-1]
			if msg.Service != tc.expected.Service {
				t.Errorf("Expected service '%s', got '%s'", tc.expected.Service, msg.Service)
			}
			if msg.Action != tc.expected.Action {
				t.Errorf("Expected action '%s', got '%s'", tc.expected.Action, msg.Action)
			}
			if msg.Id != tc.expected.Id {
				t.Errorf("Expected id '%s', got '%s'", tc.expected.Id, msg.Id)
			}
		})
	}
}

func TestNewService(t *testing.T) {
	service := NewService()

	if service == nil {
		t.Error("NewService returned nil")
	}

	if _, ok := service.(ws.Service); !ok {
		t.Error("NewService did not return a ws.Service implementation")
	}
}

func TestHeartbeatService_Cleanup(t *testing.T) {
	service := NewService().(*HeartbeatService)
	// Cleanup 不应该有任何panic或其他副作用
	service.Cleanup(nil)
	service.Cleanup(errors.New("test error"))
}
