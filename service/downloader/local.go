package downloader

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalDownloader struct {
	rootDir string
}

func NewLocalDownloader(rootDir string) *LocalDownloader {
	return &LocalDownloader{rootDir: rootDir}
}

func (l *LocalDownloader) Download(path string) (io.ReadCloser, *FileInfo, error) {
	fullPath := filepath.Join(l.rootDir, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if info.IsDir() {
		file.Close()
		return nil, nil, fmt.Errorf("path is a directory, use DownloadDir instead")
	}

	return file, toFileInfo(info), nil
}

func (l *LocalDownloader) DownloadDir(path string) (io.ReadCloser, *FileInfo, error) {
	fullPath := filepath.Join(l.rootDir, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get directory info: %w", err)
	}

	if !info.IsDir() {
		return nil, nil, fmt.Errorf("path is not a directory")
	}

	// Create pipes for streaming zip content
	pr, pw := io.Pipe()

	// Create zip writer in a goroutine
	go func() {
		zw := zip.NewWriter(pw)
		defer func() {
			zw.Close()
			pw.Close()
		}()

		err := filepath.Walk(fullPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Create zip header
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return fmt.Errorf("failed to create zip header: %w", err)
			}

			// Update header name with relative path
			relPath, err := filepath.Rel(fullPath, filePath)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
			header.Name = relPath

			if info.IsDir() {
				header.Name += "/"
			}

			// Create file in zip
			writer, err := zw.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("failed to create file in zip: %w", err)
			}

			if info.IsDir() {
				return nil
			}

			// Copy file content
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return fmt.Errorf("failed to copy file content: %w", err)
			}

			return nil
		})

		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, toFileInfo(info), nil
}

func (l *LocalDownloader) Stat(path string) (*FileInfo, error) {
	fullPath := filepath.Join(l.rootDir, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	return toFileInfo(info), nil
}

func toFileInfo(info os.FileInfo) *FileInfo {
	return &FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime().Unix(),
		IsDir:   info.IsDir(),
	}
}
