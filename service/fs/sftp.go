package fs

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

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
	// Get home directory using ssh session
	session, err := s.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh session: %w", err)
	}
	defer session.Close()

	output, err := session.Output("echo $HOME")
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Trim any whitespace/newlines from the output
	homePath := strings.TrimSpace(string(output))
	if homePath == "" {
		homePath = "~" // fallback to ~ if we couldn't get the explicit path
	}

	// Get file info of the home directory
	info, err := s.Client.Stat(homePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory info: %w", err)
	}

	entry := &FileSystemEntry{
		Name:    "/",
		Path:    homePath,
		IsDir:   true,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime().Unix(),
	}

	return []*FileSystemEntry{entry}, nil
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
	// 先检查源路径是否存在
	if _, err := s.Client.Stat(src); err != nil {
		return fmt.Errorf("source path does not exist: %w", err)
	}

	destPath := filepath.Join(dest, filepath.Base(src))
	if _, err := s.Client.Stat(destPath); err == nil {
		destPath += " copy"
	}

	// 尝试使用cp命令
	if s.sshClient != nil {
		// 先检查 cp 命令是否存在
		checkSession, err := s.sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create ssh session: %w", err)
		}

		defer checkSession.Close()
		err = checkSession.Run("which cp")
		if err != nil {
			return fmt.Errorf("cp command not found: %w", err)
		}

		// cp命令存在，执行复制操作
		copySession, err := s.sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create ssh session: %w", err)
		}
		defer copySession.Close()

		if err := copySession.Run(fmt.Sprintf("cp -a %q %q", src, destPath)); err != nil {
			return fmt.Errorf("cp command failed: %w", err)
		}
		return nil
	}
	return fmt.Errorf("ssh client not available")
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
