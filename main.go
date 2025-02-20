package main

import (
	"webshell/server"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	server.SetupRoutes(r)
	r.Run(":1234")
}
