package shell

import (
	"log"
	"sync"

	ws "webshell/websocket"

	"golang.org/x/crypto/ssh"
)

func NewLocalService() ws.Service {
	logger := log.New(log.Writer(), "[shell] ", log.LstdFlags)

	sp := &LocalShellProvider{
		Logger: logger,
	}

	return &ShellService{
		ShellProvider: sp,
		shells:        make(map[string]Shell),
		Logger:        logger,
		RWMutex:       &sync.RWMutex{},
	}
}

func NewSSHService(network, addr string, config *ssh.ClientConfig) (ws.Service, error) {
	logger := log.New(log.Writer(), "[shell] ", log.LstdFlags)

	sp, err := NewSSHShellProvider(network, addr, config)
	if err != nil {
		return nil, err
	}

	return &ShellService{
		ShellProvider: sp,
		shells:        make(map[string]Shell),
		Logger:        logger,
		RWMutex:       &sync.RWMutex{},
	}, nil
}
