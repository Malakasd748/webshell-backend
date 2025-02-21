package controller

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	shell := r.Group("/shell")
	{
		shell.GET("/local", StartLocalShell)

		sshController := NewSSHController()
		shell.POST("/ssh", sshController.LoginSSH)
		shell.GET("/ssh/:id", sshController.StartSSHShell)
	}
}
