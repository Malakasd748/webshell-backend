package fs

import (
	"fmt"
	"log"
	ws "webshell/websocket"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func NewLocalService() ws.Service {
	logger := log.New(log.Writer(), "[fs] ", log.LstdFlags)
	fs := &LocalFileSystem{
		Logger: logger,
	}
	service := &FSService{
		fs:     fs,
		Logger: logger,
	}
	return service
}

// NewSFTPService creates a new SFTP filesystem with both SFTP and SSH clients
func NewSFTPService(sshClient *ssh.Client) (ws.Service, error) {
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	logger := log.New(log.Writer(), "[fs] ", log.LstdFlags)

	fs := &SFTPFileSystem{
		Client:    sftpClient,
		sshClient: sshClient,
		Logger:    logger,
	}

	service := &FSService{
		fs:     fs,
		Logger: logger,
	}

	return service, nil
}
