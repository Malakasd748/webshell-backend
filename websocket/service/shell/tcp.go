package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
	ws "webshell/websocket"
)

type tcpShell struct {
	conn   net.Conn
	reader *bufio.Reader
	*log.Logger
}

// Close implements Shell.
func (t *tcpShell) Close() error {
	return t.conn.Close()
}

// Read implements Shell.
func (t *tcpShell) Read(p []byte) (n int, err error) {
	return t.reader.Read(p)
}

// Resize implements Shell.
func (t *tcpShell) Resize(rows int, cols int) error {
	// For raw TCP connections, just log the resize
	// The remote service may not support resize notifications
	t.Printf("Terminal resized to: %dx%d", rows, cols)
	return nil
}

// Write implements Shell.
func (t *tcpShell) Write(p []byte) (n int, err error) {
	// Debug: Log what we're sending
	// t.Printf("Sending %d bytes: %q", len(p), string(p))

	// For debugging, let's try different line ending strategies
	if bytes.Contains(p, []byte{'\r'}) {
		// If we get CR, try sending just LF
		processed := bytes.ReplaceAll(p, []byte{'\r'}, []byte{'\n'})
		t.Printf("Converted CR to LF: %q", string(processed))
		return t.conn.Write(processed)
	}

	// Send as-is for other data
	return t.conn.Write(p)
}

type TCPShellProvider struct {
	Host string
	Port int
	*log.Logger
}

// NewShell implements ShellProvider.
func (t *TCPShellProvider) NewShell(cwd string) (Shell, error) {
	// cwd parameter is ignored for TCP connections
	address := net.JoinHostPort(t.Host, fmt.Sprintf("%d", t.Port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	t.Printf("Connected to TCP host: %s", address)

	shell := &tcpShell{
		conn:   conn,
		reader: bufio.NewReader(conn),
		Logger: t.Logger,
	}

	return shell, nil
}

func NewTCPService(host string, port int) ws.Service {
	logger := log.New(log.Writer(), "[tcp-shell] ", log.LstdFlags)

	if port == 0 {
		port = 23 // Default to telnet port
	}

	sp := &TCPShellProvider{
		Host:   host,
		Port:   port,
		Logger: logger,
	}

	return &ShellService{
		ShellProvider: sp,
		shells:        make(map[string]Shell),
		Logger:        logger,
		RWMutex:       &sync.RWMutex{},
	}
}
