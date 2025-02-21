package controller

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"webshell/websocket"
	"webshell/websocket/service/fs"
	"webshell/websocket/service/heartbeat"
	"webshell/websocket/service/shell"
	"webshell/websocket/service/upload"
)

func StartLocalShell(c *gin.Context) {
	var req struct {
		Cwd string `json:"cwd"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wsServer, err := websocket.NewServer(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Register services
	shellService := shell.NewLocalService()
	fsService := fs.NewLocalService()
	heartbeatService := heartbeat.NewService()
	uploadService := upload.NewLocalService()

	wsServer.Register(shellService)
	wsServer.Register(fsService)
	wsServer.Register(uploadService)

	wsServer.RegisterPassive(heartbeatService)

	wsServer.Start()
}

type SSHController struct {
	Clients map[string]*ssh.Client
	*sync.RWMutex
}

func NewSSHController() *SSHController {
	return &SSHController{
		Clients: make(map[string]*ssh.Client),
		RWMutex: &sync.RWMutex{},
	}
}

type sshInfo struct {
	Host     string `json:"host" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Port     int    `json:"port"`
}

func (sc *SSHController) LoginSSH(c *gin.Context) {
	var sshInfo sshInfo
	if err := c.ShouldBindJSON(&sshInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if sshInfo.Port == 0 {
		sshInfo.Port = 22
	}

	config := &ssh.ClientConfig{
		User:            sshInfo.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
	}

	if sshInfo.Password != "" {
		config.Auth = append(config.Auth, ssh.Password(sshInfo.Password))
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No authentication method provided"})
		return
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sshInfo.Host, sshInfo.Port), config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id := uuid.NewString()
	sc.Lock()
	sc.Clients[id] = client
	sc.Unlock()

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (sc *SSHController) StartSSHShell(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SSH client ID"})
		return
	}

	sc.RLock()
	sshClient, exists := sc.Clients[id]
	sc.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SSH client ID"})
		return
	}

	// Create websocket server
	wsServer, err := websocket.NewServer(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create SFTP service using the SSH connection
	fsService, err := fs.NewSFTPService(sshClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fsSvc := fsService.(*fs.FSService)
	sftpClient := fsSvc.FS.(*fs.SFTPFileSystem).Client

	uploadService := upload.NewSFTPService(sftpClient)
	shellService := shell.NewSSHService(sshClient)
	heartbeatService := heartbeat.NewService()

	// Register all services
	wsServer.Register(shellService)
	wsServer.Register(fsService)
	wsServer.Register(uploadService)

	wsServer.RegisterPassive(heartbeatService)

	wsServer.Start()
}
