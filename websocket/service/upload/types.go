package upload

import (
	"hash"
	"io"
	"os"
	"sync"
)

type uploadSession struct {
	dest   string
	policy string
	hasher hash.Hash

	file io.WriteCloser
	*sync.Mutex
}

type uploadBackend interface {
	Stat(path string) (os.FileInfo, error)

	DeletePath(path string) error

	MkdirAll(path string) error

	OpenFile(path string) (io.WriteCloser, error)
}
