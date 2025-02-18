package sftp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	ws "webshell/websocket"
)

// FileInfoJSON represents file metadata in JSON format.
type FileInfoJSON struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Mode   string `json:"mode"`
	Modify string `json:"modify"`
	IsDir  bool   `json:"isDir"`
}

// FileInfoToJSON converts os.FileInfo to FileInfoJSON.
func FileInfoToJSON(fileInfo os.FileInfo) FileInfoJSON {
	return FileInfoJSON{
		Name:   fileInfo.Name(),
		Size:   fileInfo.Size(),
		Mode:   fileInfo.Mode().String(),
		Modify: fileInfo.ModTime().String(),
		IsDir:  fileInfo.IsDir(),
	}
}

// SFTPService provides sftp functionalities over websocket.
// It can be initialized with an existing ssh.Client to reuse the connection
// or create a new ssh connection if none is provided.

type SFTPService struct {
	conn       *ws.Conn
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	Logger     *log.Logger
	*sync.RWMutex
	// reuseSSH indicates if the sshClient was reused (true) or created by this service (false).
	reuseSSH bool
}

// Register assigns the websocket connection to the service.
func (s *SFTPService) Register(conn *ws.Conn) {
	s.conn = conn
}

// Name returns the service name.
func (s *SFTPService) Name() string {
	return "sftp"
}

// HandleTextMessage processes incoming text messages over websocket.
// Supported actions: "list", "upload", "download".
func (s *SFTPService) HandleTextMessage(id string, action string, data json.RawMessage) {
	s.RLock()
	defer s.RUnlock()

	s.Logger.Printf("(id: %s) received action: %s", id, action)

	switch action {
	case "list":
	case "upload":
		// For simplicity, this implementation expects the entire file content in the payload.
		// In production, consider streaming large files.
		var payload struct {
			Path   string `json:"path"`
			Offset int64  `json:"offset"`
			Update bool   `json:"update"`
			Data   []byte `json:"data"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			s.Logger.Printf("(id: %s) error unmarshalling upload payload: %v", id, err)
			return
		}
		// Check if file exists
		_, err := s.sftpClient.Lstat(payload.Path)
		if err == nil && !payload.Update {
			s.Logger.Printf("(id: %s) file exists and update flag is false", id)
			return
		}
		var dstFile *sftp.File
		if payload.Offset == 0 {
			dstFile, err = s.sftpClient.OpenFile(payload.Path, os.O_CREATE|os.O_RDWR)
		} else {
			dstFile, err = s.sftpClient.OpenFile(payload.Path, os.O_RDWR)
		}
		if err != nil {
			s.Logger.Printf("(id: %s) error opening file: %v", id, err)
			return
		}
		defer dstFile.Close()
		_, err = dstFile.Seek(payload.Offset, io.SeekStart)
		if err != nil {
			s.Logger.Printf("(id: %s) error seeking file: %v", id, err)
			return
		}
		_, err = dstFile.Write(payload.Data)
		if err != nil {
			s.Logger.Printf("(id: %s) error writing file: %v", id, err)
			return
		}
		if err := s.conn.WriteJSON(&ws.ServiceMessage{Service: s.Name(), Id: id, Action: action, Payload: map[string]interface{}{"success": "yes"}}); err != nil {
			s.Logger.Printf("(id: %s) error sending upload response: %v", id, err)
		}

	case "download":
		var payload struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			s.Logger.Printf("(id: %s) error unmarshalling download payload: %v", id, err)
			return
		}
		fileInfo, err := s.sftpClient.Lstat(payload.Path)
		if err != nil {
			s.Logger.Printf("(id: %s) error getting file info: %v", id, err)
			return
		}
		// For simplicity, if it's a directory, we do not support zipping here.
		if fileInfo.IsDir() {
			s.Logger.Printf("(id: %s) download for directories is not implemented", id)
			return
		}
		file, err := s.sftpClient.Open(payload.Path)
		if err != nil {
			s.Logger.Printf("(id: %s) error opening file for download: %v", id, err)
			return
		}
		defer file.Close()
		dataBytes, err := io.ReadAll(file)
		if err != nil {
			s.Logger.Printf("(id: %s) error reading file: %v", id, err)
			return
		}
		response := map[string]interface{}{
			"name":    fileInfo.Name(),
			"size":    fileInfo.Size(),
			"mode":    fileInfo.Mode().String(),
			"modify":  fileInfo.ModTime().String(),
			"data":    dataBytes,
			"success": "yes",
		}
		if err := s.conn.WriteJSON(&ws.ServiceMessage{Service: s.Name(), Id: id, Action: action, Payload: response}); err != nil {
			s.Logger.Printf("(id: %s) error sending download response: %v", id, err)
		}

	default:
		s.Logger.Printf("(id: %s) unknown action: %s", id, action)
	}
}

func (s *SFTPService) handleList(id string, data json.RawMessage) {
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		s.Logger.Printf("(id: %s) error unmarshalling list payload: %v", id, err)
		return
	}
	files, err := s.sftpClient.ReadDir(payload.Path)
	if err != nil {
		s.Logger.Printf("(id: %s) error reading directory '%s': %v", id, payload.Path, err)
		return
	}
	var listContent []FileInfoJSON
	for _, file := range files {
		listContent = append(listContent, FileInfoToJSON(file))
	}
	response := map[string]interface{}{
		"listContent": listContent,
		"listLength":  len(files),
		"success":     "yes",
	}
	if err := s.conn.WriteJSON(&ws.ServiceMessage{Service: s.Name(), Id: id, Action: action, Payload: response}); err != nil {
		s.Logger.Printf("(id: %s) error sending list response: %v", id, err)
	}

}

// Cleanup closes the sftp client and the underlying ssh client if it was created by the service.
func (s *SFTPService) Cleanup(err error) {
	if s.sftpClient != nil {
		s.sftpClient.Close()
	}
	if !s.reuseSSH && s.sshClient != nil {
		s.sshClient.Close()
	}
}

// NewService creates a new SFTPService instance.
// If an existing ssh.Client is provided, it will be reused; otherwise, a new ssh connection is established.
func NewService(existing *ssh.Client, network, addr string, config *ssh.ClientConfig) (ws.Service, error) {
	var sshClient *ssh.Client
	reuse := false
	if existing != nil {
		sshClient = existing
		reuse = true
	} else {
		var err error
		sshClient, err = ssh.Dial(network, addr, config)
		if err != nil {
			return nil, fmt.Errorf("failed to dial ssh: %v", err)
		}
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		if !reuse {
			sshClient.Close()
		}
		return nil, fmt.Errorf("failed to create sftp client: %v", err)
	}

	service := &SFTPService{
		sshClient:  sshClient,
		sftpClient: sftpClient,
		Logger:     log.New(log.Writer(), "sftp: ", log.LstdFlags),
		RWMutex:    new(sync.RWMutex),
		reuseSSH:   reuse,
	}

	return service, nil
}

// Optional: A method to run background tasks if needed.
func (s *SFTPService) Run(ctx context.Context) {
	// Example: wait for context cancellation to cleanup
	go func() {
		<-ctx.Done()
		s.Cleanup(nil)
	}()
	// ...other background tasks...
}
