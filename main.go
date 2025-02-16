package main

import (
	"flag"
	"webshell/server"
)

func main() {
	var port uint

	flag.UintVar(&port, "port", 1234, "The port to listen on")
	flag.Parse()

	<-server.Start(port)
}
