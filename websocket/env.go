package websocket

import (
	"log"
	"os"
	"strconv"
	"time"
)

const (
	timeoutName = "WEBSHELL_CONNECTION_TIMEOUT"
)

var (
	connectionTimeout = time.Duration(getEnvTimeout()) * time.Minute
)

func getEnvTimeout() int {
	if timeout := os.Getenv(timeoutName); timeout == "" {
		log.Printf("$%s not set, default to 1 minute", timeoutName)
	} else {
		timeout, err := strconv.Atoi(timeout)
		if err == nil {
			return timeout
		}
		log.Printf("$%s (%v) is not a valid integer, default to 1 minute", timeoutName, timeout)
	}

	return 1
}
