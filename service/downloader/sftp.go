package downloader

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPDownloader struct {
	client    *sftp.Client
	sshClient *ssh.Client
}

func NewSFTPDownloader(sshClient *ssh.Client) (*SFTPDownloader, error) {
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	return &SFTPDownloader{
		client:    sftpClient,
		sshClient: sshClient,
	}, nil
}

func (s *SFTPDownloader) Download(path string) (io.ReadCloser, *FileInfo, error) {
	file, err := s.client.Open(path)
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

func (s *SFTPDownloader) DownloadDir(path string) (io.ReadCloser, *FileInfo, error) {
	info, err := s.client.Stat(path)
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

		walker := s.client.Walk(path)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				pw.CloseWithError(err)
				return
			}

			filePath := walker.Path()
			info := walker.Stat()

			// Create zip header
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create zip header: %w", err))
				return
			}

			// Update header name with relative path
			relPath, err := filepath.Rel(path, filePath)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to get relative path: %w", err))
				return
			}
			header.Name = relPath

			if info.IsDir() {
				header.Name += "/"
			}

			// Create file in zip
			writer, err := zw.CreateHeader(header)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create file in zip: %w", err))
				return
			}

			if info.IsDir() {
				continue
			}

			// Copy file content
			file, err := s.client.Open(filePath)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to open file: %w", err))
				return
			}

			_, err = io.Copy(writer, file)
			file.Close()
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to copy file content: %w", err))
				return
			}
		}
	}()

	return pr, toFileInfo(info), nil
}

func (s *SFTPDownloader) Stat(path string) (*FileInfo, error) {
	info, err := s.client.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	return toFileInfo(info), nil
}
