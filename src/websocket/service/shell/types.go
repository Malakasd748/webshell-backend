package shell

import "io"

type ShellProvider interface {
	NewShell(cwd string) (Shell, error)
}

type Shell interface {
	io.ReadWriteCloser
	Resize(rows, cols int) error
}
