package main

import (
	"webshell/controller"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	controller.SetupRoutes(r)
	r.Run(":1234")
}
