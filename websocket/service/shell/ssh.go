package shell

import (
	"io"
	"log"
	"sync"
	ws "webshell/websocket"

	"golang.org/x/crypto/ssh"
)

type sshShell struct {
	stdinWriter  io.WriteCloser
	stdoutReader io.Reader

	*ssh.Session
	*log.Logger
}

// Close implements Shell.
func (s *sshShell) Close() error {
	if err := s.stdinWriter.Close(); err != nil {
		s.Printf("close stdin writer error: %v", err)
	}
	s.Session.Close()
	return nil
}

// Read implements Shell.
func (s *sshShell) Read(p []byte) (n int, err error) {
	return s.stdoutReader.Read(p)
}

// Resize implements Shell.
func (s *sshShell) Resize(rows int, cols int) error {
	return s.WindowChange(rows, cols)
}

// Write implements Shell.
func (s *sshShell) Write(p []byte) (n int, err error) {
	return s.stdinWriter.Write(p)
}

type SSHShellProvider struct {
	*ssh.Client
	*log.Logger
}

// NewShell implements ShellProvider.
func (s *SSHShellProvider) NewShell(cwd string) (Shell, error) {
	session, err := s.Client.NewSession()
	if err != nil {
		return nil, err
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	// Combine stdout and stderr
	combinedOutput := io.MultiReader(stdout, stderr)

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", 60, 80, modes); err != nil {
		session.Close()
		return nil, err
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		session.Close()
		return nil, err
	}

	sh := &sshShell{
		Session:      session,
		stdinWriter:  stdinPipe,
		stdoutReader: combinedOutput,
		Logger:       s.Logger,
	}

	return sh, nil
}

func NewSSHService(client *ssh.Client) ws.Service {
	logger := log.New(log.Writer(), "[shell] ", log.LstdFlags)

	sp := &SSHShellProvider{
		Client: client,
		Logger: logger,
	}

	return &ShellService{
		ShellProvider: sp,
		shells:        make(map[string]Shell),
		Logger:        logger,
		RWMutex:       &sync.RWMutex{},
	}
}
