package controller

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	shell := r.Group("/shell")
	{
		shell.GET("/local", StartLocalShell)
		shell.GET("/tcp", StartTCPShell)

		sshController := NewSSHController()
		shell.POST("/ssh", sshController.LoginSSH)
		shell.GET("/ssh/:id", sshController.StartSSHShell)
		// 添加文件下载路由
		shell.GET("/ssh/:id/download", sshController.Download)
	}
}
