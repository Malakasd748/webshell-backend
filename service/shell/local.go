package shell

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type PTYShell struct {
	terminate context.CancelFunc

	*os.File
	*log.Logger
}

func (p *PTYShell) Resize(rows, cols int) error {
	return pty.Setsize(p.File, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (p *PTYShell) Close() error {
	p.terminate()
	return p.File.Close()
}

type LocalShellProvider struct {
	*log.Logger
}

func (l *LocalShellProvider) NewShell(cwd string) (Shell, error) {
	ctx, cancel := context.WithCancel(context.Background())

	command := exec.CommandContext(ctx, "bash", "-l")

	if cwd != "" {
		command.Dir = cwd
	} else {
		command.Dir = ptyCWD
	}

	command.Env = append(os.Environ(), "TERM=xterm-256color")

	f, err := pty.Start(command)
	if err != nil {
		l.Printf("Failed to start pty: %v", err)
		cancel()
		return nil, err
	}

	sh := &PTYShell{
		File:      f,
		terminate: cancel,
		Logger:    l.Logger,
	}

	go func() {
		<-ctx.Done()
		sh.Close()
	}()

	return sh, nil
}
