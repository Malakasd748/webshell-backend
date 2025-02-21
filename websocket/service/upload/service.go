package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	ws "webshell/websocket"
)

const (
	actionStartSession    = "start_session"
	actionCompleteSession = "complete_session"
	actionCancelSession   = "cancel_session"
	actionStartFile       = "start_file"
	actionCompleteFile    = "complete_file"
	actionChunk           = "chunk"
	actionMkdir           = "mkdir"
	// 文件夹写入策略
	policyOverwrite = "overwrite" // 没用到, 默认覆盖
	policySkip      = "skip"
	policyRename    = "rename"
)

type startSessionData struct {
	Policy string `json:"policy,omitempty"`

	NeedConfirm bool `json:"needConfirm,omitempty"`
}
type startFileData struct {
	Path string `json:"path,omitempty"`

	Skip bool `json:"skip,omitempty"`
}
type chunkData struct {
	Progress uint `json:"progress"`
}
type completeFileData struct {
	Digest string `json:"digest,omitempty"`
}

type chunkMeta struct {
	id       string
	progress uint
}

type UploadService struct {
	conn *ws.Conn

	sessions map[string]*uploadSession
	*sync.RWMutex

	backend uploadBackend

	// buffered
	chunkMeta chan *chunkMeta
	chunkData chan []byte

	*log.Logger
}

// Register implements websocket.Service.
func (s *UploadService) Register(conn *ws.Conn) {
	s.conn = conn

	go func() {
		for {
			meta, ok := <-s.chunkMeta
			if !ok {
				break
			}
			data, ok := <-s.chunkData
			if !ok {
				break
			}
			s.writeChunkData(meta, data)
		}
	}()
}

func (s *UploadService) Name() string {
	return "upload"
}

func (s *UploadService) HandleTextMessage(id, action string, data json.RawMessage) {
	switch action {
	case actionStartSession:
		s.handleStartSession(id, data)
	case actionCompleteSession:
		s.handleCompleteSession(id)
	case actionCancelSession:
		s.handleCancelSession(id)
	case actionStartFile:
		s.handleStartFile(id, data)
	case actionCompleteFile:
		s.handleCompleteFile(id, data)
	case actionMkdir:
		s.handleMkdir(id, data)
	case actionChunk:
		s.handleChunk(id, data)
	}
}

func (s *UploadService) handleChunk(id string, data json.RawMessage) {
	var d chunkData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error decoding chunk data: %v", err)
		return
	}

	s.chunkMeta <- &chunkMeta{id: id, progress: d.Progress}

	s.conn.BinaryChan <- s.chunkData
}

// handleCompleteSession 处理上传会话完成的逻辑
func (s *UploadService) handleCompleteSession(id string) {
	s.RLock()
	ss, exists := s.sessions[id]
	s.RUnlock()

	if !exists {
		s.Printf("session not found, cannot complete session: %s", id)
		return
	}

	// 确保文件已关闭
	if ss.file != nil {
		ss.Lock()
		ss.file.Close()
		ss.Unlock()
		ss.file = nil
	}

	// 移除会话
	s.Lock()
	delete(s.sessions, id)
	s.Unlock()

	// 向客户端发送完成消息
	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionCompleteSession,
	})
}

func (s *UploadService) Cleanup(err error) {
	s.Printf("close error: %v", err)

	close(s.chunkData)
	close(s.chunkMeta)

	for _, ss := range s.sessions {
		if ss.file != nil {
			ss.Lock()
			ss.file.Close()
			ss.Unlock()
			// 需要删吗？
			go s.backend.DeletePath(ss.dest)
			ss.file = nil
		}
	}
}

func (s *UploadService) handleMkdir(id string, data json.RawMessage) {
	var d string
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error decoding mkdir data: %v", err)
		return
	}
	if err := s.backend.MkdirAll(d); err != nil {
		s.handleError(id, actionMkdir, fmt.Errorf("上传失败: 创建文件夹失败: %s", err))
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionMkdir,
	})
}

func (s *UploadService) handleStartSession(id string, data json.RawMessage) {
	var d startSessionData
	if len(data) > 0 {
		if err := json.Unmarshal(data, &d); err != nil {
			s.Printf("error decoding start session data: %v", err)
			return
		}
	}

	_, err := s.backend.Stat(id)

	// File exists, request frontend confirmation
	if d.Policy == "" && err == nil {
		s.conn.WriteJSON(&ws.ServiceMessage{
			Service: s.Name(),
			Id:      id,
			Action:  actionStartSession,
			Data:    json.RawMessage(`{"needConfirm":true}`),
		})
		return
	}

	dest := id
	// Rename if policy is keepBoth
	if d.Policy == policyRename {
		dest = getUniqueFilename(id)
	}

	s.Lock()
	s.sessions[id] = &uploadSession{
		dest:   dest,
		policy: d.Policy,
		Mutex:  new(sync.Mutex),
		hasher: sha256.New(),
	}
	s.Unlock()

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionStartSession,
		Data:    json.RawMessage(`{"needConfirm":false}`),
	})
}

func (s *UploadService) handleCancelSession(id string) {
	s.RLock()
	ss, exists := s.sessions[id]
	s.RUnlock()

	if !exists {
		s.Printf("session not found, cannot cancel: %s", id)
		return
	}

	if ss.file != nil {
		ss.Lock()
		ss.file.Close()
		ss.Unlock()
		go s.backend.DeletePath(ss.dest)
	}

	s.Lock()
	delete(s.sessions, id)
	s.Unlock()

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionCancelSession,
	})
}

func (s *UploadService) handleStartFile(id string, data json.RawMessage) {
	s.RLock()
	ss, exists := s.sessions[id]
	s.RUnlock()

	if !exists {
		s.Printf("session not found, cannot start file: %s", id)
		return
	}

	if ss.file != nil {
		s.Printf("didn't finish previous file, cannot start new one: %s", id)
		return
	}

	var d startFileData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error decoding start file data: %v", err)
		return
	}

	// 将 d.Path 转换为相对于 id 的相对路径
	relPath := d.Path
	if strings.HasPrefix(d.Path, id) {
		relPath = path.Clean(strings.TrimPrefix(d.Path, id))
		relPath = strings.TrimPrefix(relPath, "/")
	}

	p := path.Join(ss.dest, relPath)

	stat, statErr := s.backend.Stat(p)
	// Check if we should skip file
	if ss.policy == policySkip && stat == nil {
		s.conn.WriteJSON(&ws.ServiceMessage{
			Service: s.Name(),
			Id:      id,
			Action:  actionStartFile,
			Data:    json.RawMessage(`{"skip":true}`),
		})
		return
	}

	if err := s.backend.MkdirAll(path.Dir(p)); err != nil {
		s.handleError(id, actionStartFile, err)
		return
	}

	// Avoid empty filename or directory path from frontend
	if statErr == nil && stat.IsDir() {
		p = path.Join(p, fmt.Sprint("_", time.Now().Unix()))
	}

	f, err := s.backend.OpenFile(p)
	if err != nil {
		s.handleError(id, actionStartFile, err)
		return
	}

	ss.file = f
	ss.hasher.Reset()

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionStartFile,
		Data:    json.RawMessage(`{"skip":false}`),
	})
}

func (s *UploadService) handleCompleteFile(id string, data json.RawMessage) {
	s.RLock()
	ss, exists := s.sessions[id]
	s.RUnlock()

	if !exists {
		s.Printf("session not found, cannot complete file: %s", id)
		return
	}

	if ss.file == nil {
		return
	}

	ss.Lock()
	ss.file.Close()
	ss.Unlock()

	ss.file = nil

	myHash := hex.EncodeToString(ss.hasher.Sum(nil))

	var d completeFileData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error decoding complete file data: %v", err)
		return
	}

	if myHash != d.Digest {
		s.Printf("hash mismatch, local: %s, peer: %s", myHash, d.Digest)
		s.handleError(id, actionCompleteFile, fmt.Errorf("上传失败: 文件完整性校验失败"))
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionCompleteFile,
	})
}

func getUniqueFilename(filepath string) string {
	var baseName, suffix string
	if strings.HasSuffix(filepath, "/") {
		baseName = filepath[:len(filepath)-1]
		suffix = "/"
	} else {
		idx := strings.LastIndex(filepath, ".")
		if idx != -1 {
			baseName = filepath[:idx]
			suffix = filepath[idx:]
		} else {
			baseName = filepath
		}
	}

	var result string
	for num := 1; ; num++ {
		result = fmt.Sprintf("%s_%d%s", baseName, num, suffix)
		if _, err := os.Stat(result); os.IsNotExist(err) {
			break
		}
	}
	return result
}

func (s *UploadService) writeChunkData(meta *chunkMeta, data []byte) {
	s.RLock()
	ss := s.sessions[meta.id]
	s.RUnlock()

	ss.Lock()
	// Check if file is closed using stat to avoid write errors
	_, err := s.backend.Stat(ss.dest)
	if err != nil {
		s.Printf("error stat-ing file: %v", err)
		ss.Unlock()
		return
	}
	written, err := ss.file.Write(data)
	ss.Unlock()

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "quota exceeded") {
			errMsg = "磁盘空间不足"
		}
		s.handleError(meta.id, actionChunk, fmt.Errorf("上传失败: %s", errMsg))
		return
	}

	meta.progress += uint(written)

	ss.hasher.Write(data)

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      meta.id,
		Action:  actionChunk,
		Data:    json.RawMessage(fmt.Sprintf(`{"progress":%d}`, meta.progress)),
	})
}

func (s *UploadService) handleError(id, action string, err error) {
	s.Println(err)

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  action,
		Error:   err.Error(),
	})

	s.RLock()
	ss := s.sessions[id]
	s.RUnlock()

	if ss.file != nil {
		ss.Lock()
		ss.file.Close()
		ss.Unlock()
		go s.backend.DeletePath(ss.dest)
		ss.file = nil
	}
}

func newServiceBase() *UploadService {
	return &UploadService{
		sessions: make(map[string]*uploadSession),
		RWMutex:  new(sync.RWMutex),

		chunkMeta: make(chan *chunkMeta, 1),
		chunkData: make(chan []byte, 1),

		Logger: log.New(os.Stdout, "[upload] ", log.LstdFlags),
	}
}
