package controller

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	shell := r.Group("/shell")
	{
		sshController := NewSSHController()
		shell.GET("/local", StartLocalShell)
		shell.POST("/ssh", sshController.LoginSSHShell)
		shell.GET("/ssh/:id", sshController.GetSSHShell)
	}
}
