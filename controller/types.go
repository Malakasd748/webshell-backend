package controller

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sshInfo struct {
	Cluster    string `json:"cluster"`
	Name       string `json:"name"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Host       string `json:"host"`
	Port       string `json:"port"`
}

type jumpServerClient struct {
	sftpClient *sftp.Client
	sshClient  *ssh.Client
	sshInfo    *sshInfo
}

type jumpServerController struct {
	clients     map[string]*jumpServerClient
	record      bool
	recordPath  string
	log         bool
	logFilePath string
}
