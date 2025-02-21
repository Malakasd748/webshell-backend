package upload

import (
	"io"
	"os"
	ws "webshell/websocket"
)

type localBackend struct{}

// CheckPathExists implements uploadBackend.
func (l *localBackend) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// DeletePath implements uploadBackend.
func (l *localBackend) DeletePath(path string) error {
	return os.RemoveAll(path)
}

// MkdirAll implements uploadBackend.
func (l *localBackend) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (l *localBackend) OpenFile(path string) (io.WriteCloser, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
}

func NewLocalService() ws.Service {
	s := newServiceBase()
	s.backend = &localBackend{}
	return s
}
