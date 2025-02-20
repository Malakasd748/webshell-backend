package controller

import (
	"net/http"
	"webshell/service/fs"
	"webshell/service/heartbeat"
	"webshell/service/shell"
	"webshell/websocket"

	"github.com/gin-gonic/gin"
)

type LocalShellRequest struct {
	Cwd string `json:"cwd"`
}

type SSHShellRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

type ShellResponse struct {
	WsURL string `json:"wsUrl"`
}

func HandleLocalShell(c *gin.Context) {
	var req LocalShellRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wsServer := websocket.NewServer()

	// Register services
	shellService := shell.NewLocalService()
	fsService := fs.NewLocalService()
	heartbeatService := heartbeat.NewService()

	wsServer.Register(shellService)
	wsServer.Register(fsService)
	wsServer.Register(heartbeatService)

	wsServer.Start(c.Writer, c.Request)
}
