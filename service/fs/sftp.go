package fs

import (
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPFileSystem struct {
	*sftp.Client
	sshClient *ssh.Client
	*log.Logger
}

// NewSFTPFileSystem creates a new SFTP filesystem with both SFTP and SSH clients
func NewSFTPFileSystem(sshClient *ssh.Client, sftpClient *sftp.Client, logger *log.Logger) *SFTPFileSystem {
	return &SFTPFileSystem{
		Client:    sftpClient,
		sshClient: sshClient,
		Logger:    logger,
	}
}

// GetRoot implements fileSystem.
func (s *SFTPFileSystem) GetRoot() ([]*FileSystemEntry, error) {
	return s.List("/", true)
}

// List implements fileSystem.
func (s *SFTPFileSystem) List(path string, showHidden bool) ([]*FileSystemEntry, error) {
	files, err := s.Client.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var entries []*FileSystemEntry
	for _, file := range files {
		// Skip hidden files if not showing them
		if !showHidden && file.Name()[0] == '.' {
			continue
		}

		entries = append(entries, &FileSystemEntry{
			Name:    file.Name(),
			Path:    filepath.Join(path, file.Name()),
			Size:    file.Size(),
			Mode:    file.Mode(),
			ModTime: file.ModTime().Unix(),
			IsDir:   file.IsDir(),
		})
	}
	return entries, nil
}

// Create implements fileSystem.
func (s *SFTPFileSystem) Create(parentPath string, name string, isDir bool) error {
	fullPath := filepath.Join(parentPath, name)

	if isDir {
		return s.Client.MkdirAll(fullPath)
	}

	file, err := s.Client.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return file.Close()
}

// Delete implements fileSystem.
func (s *SFTPFileSystem) Delete(path string) error {
	// Check if it's a directory first
	info, err := s.Client.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// Remove directory and its contents
		return s.Client.RemoveAll(path)
	}

	// Remove single file
	return s.Client.Remove(path)
}

// Copy implements fileSystem.
func (s *SFTPFileSystem) Copy(src string, dest string) error {
	srcInfo, err := s.Client.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	destPath := filepath.Join(dest, filepath.Base(src))
	if _, err := s.Client.Stat(destPath); err == nil {
		destPath += " copy"
	}

	// 首先尝试使用 cp 命令
	if s.sshClient != nil {
		session, err := s.sshClient.NewSession()
		if err == nil {
			defer session.Close()
			// 在同一个会话中检查并执行 cp 命令，区分不同的错误情况
			cmd := fmt.Sprintf(`
if ! command -v cp >/dev/null 2>&1; then
    echo "cp command not found" >&2
    exit 126
else
    cp -a %q %q
    exit $?
fi`, src, destPath)
			err = session.Run(cmd)
			if err == nil {
				return nil
			}
			if exitErr, ok := err.(*ssh.ExitError); ok && exitErr.ExitStatus() == 126 {
				s.Logger.Printf("cp command not found, falling back to manual copy")
			} else {
				s.Logger.Printf("cp command failed: %v, falling back to manual copy", err)
			}
		}
	}

	// 回退到手动复制实现
	if srcInfo.IsDir() {
		if err := s.Client.MkdirAll(destPath); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		entries, err := s.Client.ReadDir(src)
		if err != nil {
			return fmt.Errorf("failed to read source directory: %w", err)
		}

		for _, entry := range entries {
			srcPath := filepath.Join(src, entry.Name())
			if err := s.Copy(srcPath, destPath); err != nil {
				// cleanup on error
				s.Client.RemoveAll(destPath)
				return err
			}
		}
		return nil
	}

	// 复制普通文件
	srcFile, err := s.Client.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := s.Client.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		s.Client.Remove(destPath) // cleanup on error
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return s.Client.Chmod(destPath, srcInfo.Mode())
}

// Move implements fileSystem.
func (s *SFTPFileSystem) Move(src string, dest string) error {
	return s.Client.Rename(src, dest)
}

// Rename implements fileSystem.
func (s *SFTPFileSystem) Rename(oldPath string, newName string) error {
	// Get the parent directory of the oldPath
	parentDir := filepath.Dir(oldPath)
	// Construct the new full path
	newPath := filepath.Join(parentDir, newName)

	err := s.Client.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}
	return nil
}
