package downloader

import (
	"io"
	"os"
)

// FileInfo represents metadata about a file
type FileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime int64
	IsDir   bool
}

// Downloader defines the interface for downloading files
type Downloader interface {
	// Download streams a file from the given path
	Download(path string) (io.ReadCloser, *FileInfo, error)

	// DownloadDir streams a directory as a zip archive
	DownloadDir(path string) (io.ReadCloser, *FileInfo, error)

	// Stat returns file information without downloading
	Stat(path string) (*FileInfo, error)
}
