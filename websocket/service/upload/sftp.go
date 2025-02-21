package upload

import (
	"io"
	"os"
	ws "webshell/websocket"

	"github.com/pkg/sftp"
)

type sftpBackend struct {
	client *sftp.Client
}

// Stat implements uploadBackend
func (s *sftpBackend) Stat(path string) (os.FileInfo, error) {
	return s.client.Stat(path)
}

// DeletePath implements uploadBackend
func (s *sftpBackend) DeletePath(path string) error {
	return s.client.Remove(path)
}

// MkdirAll implements uploadBackend
func (s *sftpBackend) MkdirAll(path string) error {
	return s.client.MkdirAll(path)
}

// OpenFile implements uploadBackend
func (s *sftpBackend) OpenFile(path string) (io.WriteCloser, error) {
	return s.client.OpenFile(path, os.O_CREATE|os.O_WRONLY)
}

func NewSFTPBackend(client *sftp.Client) uploadBackend {
	return &sftpBackend{client: client}
}

func NewSFTPService(client *sftp.Client) ws.Service {
	s := newServiceBase()
	s.backend = NewSFTPBackend(client)
	return s
}
