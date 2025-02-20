package server

import (
	"webshell/controller"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	shell := r.Group("/shell")
	{
		shell.GET("/local", controller.HandleLocalShell)
	}
}
