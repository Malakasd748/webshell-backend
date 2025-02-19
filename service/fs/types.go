package fs

import "os"

// FileSystemEntry represents common file metadata.
type FileSystemEntry struct {
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime int64       `json:"modTime"`
	IsDir   bool        `json:"isDir"`
}

// FileSystem defines a common interface for file operations.
// Both local and remote implementations should conform to this interface.
type FileSystem interface {
	GetRoot() ([]*FileSystemEntry, error)

	// List returns the directory entries at the given path. The showHidden flag indicates whether hidden files should be included.
	List(path string, showHidden bool) ([]*FileSystemEntry, error)

	// Rename renames the file or directory at oldPath to the newName (keeping the same parent directory).
	Rename(oldPath, newName string) error

	// Create creates a new file or directory (if isDir is true) with the given name under the specified parent path.
	Create(parentPath, name string, isDir bool) error

	// Delete removes the file or directory at the given path.
	Delete(path string) error

	// Copy duplicates the file or directory from src to dest.
	Copy(src, dest string) error

	// Move relocates the file or directory from src to dest.
	Move(src, dest string) error
}
