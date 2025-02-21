package shell

import (
	"log"
	"os"
)

var (
	ptyCWD = getEnvCWD()
)

const (
	envName = "WEBSHELL_PTY_CWD"
)

func getEnvCWD() string {
	if cwd := os.Getenv(envName); cwd == "" {
		log.Printf("$%s not set, using home directory", envName)
	} else {
		_, err := os.Stat(cwd)
		if err == nil {
			return cwd
		}
		log.Printf("$%s (%s) is not a valid path, using home directory", envName, cwd)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("failed to get user home directory: %v", err)
		log.Printf("using cwd as fallback")
		return "."
	}

	return homeDir
}
