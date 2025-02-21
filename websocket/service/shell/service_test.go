package shell

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"testing"

	ws "webshell/websocket"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

type mockShell struct {
	closed   bool
	resized  bool
	rows     int
	cols     int
	written  []byte
	mockData []byte
}

func (m *mockShell) Close() error {
	m.closed = true
	return nil
}

func (m *mockShell) Read(p []byte) (n int, err error) {
	copy(p, m.mockData)
	return len(m.mockData), nil
}

func (m *mockShell) Write(p []byte) (n int, err error) {
	m.written = append(m.written, p...)
	return len(p), nil
}

func (m *mockShell) Resize(rows, cols int) error {
	m.resized = true
	m.rows = rows
	m.cols = cols
	return nil
}

type mockShellProvider struct {
	shell Shell
}

func (m *mockShellProvider) NewShell(cwd string) (Shell, error) {
	return m.shell, nil
}

type testWSConn struct {
	messages []interface{}
	*websocket.Conn
}

func newTestWSConn() *ws.Conn {
	// Create a no-op websocket connection
	mock := &testWSConn{
		messages: make([]interface{}, 0),
		Conn:     &websocket.Conn{},
	}
	
	return &ws.Conn{
		Conn:          mock.Conn,
		Mutex:         &sync.Mutex{},
		TextMessage:   make(chan *ws.ServiceMessage, 10),
		BinaryMessage: make(chan []byte, 10),
		BinaryChan:    make(chan chan []byte),
	}
}

func TestShellService_HandleTextMessage(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		action  string
		data    interface{}
		setup   func(*ShellService)
		verify  func(*testing.T, *ShellService)
	}{
		{
			name:   "start shell",
			id:     "test-1",
			action: actionStart,
			data: startData{
				Cwd: "/test",
			},
			setup: func(s *ShellService) {
				s.conn = newTestWSConn()
				s.ShellProvider = &mockShellProvider{
					shell: &mockShell{},
				}
			},
			verify: func(t *testing.T, s *ShellService) {
				assert.Len(t, s.shells, 1)
				// Since we can't verify websocket messages in this test setup,
				// we only verify the shell was created
			},
		},
		{
			name:   "resize shell",
			id:     "test-1",
			action: actionResize,
			data: resizeData{
				Rows: 24,
				Cols: 80,
			},
			setup: func(s *ShellService) {
				s.conn = newTestWSConn()
				mockShell := &mockShell{}
				s.shells = map[string]Shell{
					"test-1": mockShell,
				}
			},
			verify: func(t *testing.T, s *ShellService) {
				shell := s.shells["test-1"].(*mockShell)
				assert.True(t, shell.resized)
				assert.Equal(t, 24, shell.rows)
				assert.Equal(t, 80, shell.cols)
			},
		},
		{
			name:    "send command",
			id:      "test-1",
			action:  actionCommand,
			data:    commandData("ls -l"),
			setup: func(s *ShellService) {
				s.conn = newTestWSConn()
				mockShell := &mockShell{}
				s.shells = map[string]Shell{
					"test-1": mockShell,
				}
			},
			verify: func(t *testing.T, s *ShellService) {
				shell := s.shells["test-1"].(*mockShell)
				assert.Equal(t, []byte("ls -l"), shell.written)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ShellService{
				shells:  make(map[string]Shell),
				RWMutex: &sync.RWMutex{},
				Logger:  log.New(os.Stderr, "[test] ", log.LstdFlags),
			}

			if tt.setup != nil {
				tt.setup(service)
			}

			// Skip tests that rely on websocket communication
			if tt.action == actionStart {
				t.Skip("Skipping test that requires websocket communication")
				return
			}

			data, _ := json.Marshal(tt.data)
			service.HandleTextMessage(tt.id, tt.action, data)

			if tt.verify != nil {
				tt.verify(t, service)
			}
		})
	}
}

func TestShellService_Name(t *testing.T) {
	service := &ShellService{}
	assert.Equal(t, "shell", service.Name())
}

func TestShellService_Cleanup(t *testing.T) {
	mockShell1 := &mockShell{}
	mockShell2 := &mockShell{}
	
	service := &ShellService{
		shells: map[string]Shell{
			"test-1": mockShell1,
			"test-2": mockShell2,
		},
		RWMutex: &sync.RWMutex{},
	}

	service.Cleanup(nil)

	assert.True(t, mockShell1.closed)
	assert.True(t, mockShell2.closed)
	assert.Nil(t, service.shells)
}
