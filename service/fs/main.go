package fs

import (
	"log"
	ws "webshell/websocket"
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

func NewSFTPService() ws.Service {
	logger := log.New(log.Writer(), "[fs] ", log.LstdFlags)
	fs := &SFTPFileSystem{
		Logger: logger,
	}
	service := &FSService{
		fs:     fs,
		Logger: logger,
	}
	return service
}
