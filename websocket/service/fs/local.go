package fs

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	ws "webshell/websocket"
)

type LocalFileSystem struct {
	*log.Logger
}

func (l *LocalFileSystem) GetRoot() ([]*FileSystemEntry, error) {
	info, err := os.Stat(fsRoot)

	if err != nil {
		l.Printf("error getting root directory info: %v", err)
		return nil, err
	}

	entry := &FileSystemEntry{
		Name:    "/",
		Path:    fsRoot,
		IsDir:   true,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime().UnixMilli(),
	}

	return []*FileSystemEntry{entry}, nil
}

// Copy implements fileSystem.
func (l *LocalFileSystem) Copy(src string, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	destPath := path.Join(dest, srcInfo.Name())
	if _, err := os.Stat(destPath); err == nil {
		destPath += " copy"
	}

	// 在类Unix系统上尝试使用cp命令
	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("cp"); err == nil {
			// cp命令存在，使用cp命令
			cmd := exec.Command("cp", "-a", src, destPath)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("cp command failed: %w", err)
			}
			return nil
		}
		l.Printf("cp command not found, falling back to Go implementation")
	}

	// cp命令不存在，继续使用Go实现
	if srcInfo.IsDir() {
		// Create destination directory
		if err := os.MkdirAll(destPath, srcInfo.Mode()); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Read source directory
		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("failed to read source directory: %w", err)
		}

		// Recursively copy contents
		for _, entry := range entries {
			srcPath := path.Join(src, entry.Name())
			if err := l.Copy(srcPath, destPath); err != nil {
				// cleanup on error
				os.RemoveAll(destPath)
				return err
			}
		}
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, srcFile); err != nil {
		os.Remove(destPath) // cleanup on error
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// Create implements fileSystem.
func (l *LocalFileSystem) Create(parentPath string, name string, isDir bool) error {
	newPath := path.Join(parentPath, name)

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("目标路径已存在: %s", newPath)
	}

	if isDir {
		if err := os.Mkdir(newPath, 0750); err != nil {
			return err
		}
	} else {
		f, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return err
		}
		f.Close()
	}

	return nil
}

// Delete implements fileSystem.
func (l *LocalFileSystem) Delete(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}

// List implements fileSystem.
func (l *LocalFileSystem) List(dirPath string, showHidden bool) ([]*FileSystemEntry, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	entries := make([]*FileSystemEntry, 0, len(dirEntries))

	for _, dirEntry := range dirEntries {
		info, err := dirEntry.Info()
		if err != nil {
			l.Printf("error getting file info: %v", err)
			continue
		}
		if !showHidden && dirEntry.Name()[0] == '.' {
			continue
		}
		entries = append(entries, &FileSystemEntry{
			Name:    dirEntry.Name(),
			Path:    path.Join(dirPath, dirEntry.Name()),
			IsDir:   dirEntry.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime().UnixMilli(),
		})
	}

	return entries, nil
}

// Move implements fileSystem.
func (l *LocalFileSystem) Move(src string, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	destPath := path.Join(dest, srcInfo.Name())
	if _, err := os.Stat(destPath); err == nil {
		destPath += " copy"
	}

	return os.Rename(src, destPath)
}

// Rename implements fileSystem.
func (l *LocalFileSystem) Rename(oldPath string, newName string) error {
	if strings.Contains(newName, "/") {
		return fmt.Errorf("文件名不合法: %s", newName)
	}

	newPath := path.Join(path.Dir(oldPath), newName)

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("目标路径已存在: %s", newPath)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	return nil
}

func NewLocalService() ws.Service {
	logger := log.New(log.Writer(), "[fs] ", log.LstdFlags)
	fs := &LocalFileSystem{
		Logger: logger,
	}
	service := &FSService{
		FS:     fs,
		Logger: logger,
	}
	return service
}
