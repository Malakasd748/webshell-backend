package fs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"webshell/service"
	"webshell/websocket"
)

const (
	actionList   = "list"
	actionRoot   = "get_root"
	actionRename = "rename"
	actionCreate = "create"
	actionDelete = "delete"
	actionCopy   = "copy"
	actionMove   = "move"
)

type fsEntry struct {
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	IsDir   bool        `json:"isDir"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime int64       `json:"modTime"`
}

type listData struct {
	// req
	ShowHidden bool `json:"showHidden,omitempty"`
	// res
	Entries []fsEntry `json:"entries"`
}
type rootData fsEntry
type renameData struct {
	NewName string `json:"newName"`
}
type createData struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}
type copyData struct {
	Dest string `json:"dest"`
}
type moveData struct {
	Dest string `json:"dest"`
}

type FSService struct {
	conn *websocket.Conn

	*log.Logger
}

func (s *FSService) Name() string {
	return "fs"
}

func (s *FSService) HandleMessage(id, action string, data json.RawMessage) {
	switch action {
	case actionList:
		go s.handleList(id, data)
	case actionRoot:
		go s.handleGetRoot(id)
	case actionRename:
		go s.handleRename(id, data)
	case actionCreate:
		go s.handleCreate(id, data)
	case actionDelete:
		go s.handleDelete(id, data)
	case actionCopy:
		go s.handleCopy(id, data)
	case actionMove:
		go s.handleMove(id, data)
	}
}

func (s *FSService) Cleanup(err error) {}

func (s *FSService) handleMove(id string, data json.RawMessage) {
	var d moveData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error unmarshalling fs move payload: %v", err)
		return
	}

	oldPath := id

	oldStat, err := os.Stat(oldPath)
	if err != nil {
		s.handleError(id, actionMove, err)
		return
	}

	newPath := path.Join(d.Dest, oldStat.Name())
	if _, err := os.Stat(newPath); err == nil {
		newPath += " copy"
	}

	_, err = os.Stat(newPath)
	if err == nil {
		s.handleError(id, actionMove, fmt.Errorf("目标路径已存在: %s", newPath))
		return
	}

	mv := exec.Command("mv", "-n", oldPath, newPath)
	if output, err := mv.CombinedOutput(); err != nil {
		s.handleError(id, actionMove, fmt.Errorf("移动文件失败: %s", output))
		return
	}

	s.conn.WriteJSON(service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  actionMove,
	})
}

func (s *FSService) handleCopy(id string, data json.RawMessage) {
	var d copyData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error unmarshalling fs copy payload: %v", err)
		return
	}

	oldPath := id

	stat, err := os.Stat(oldPath)
	if err != nil {
		s.handleError(id, actionCopy, err)
		return
	}

	newPath := path.Join(d.Dest, stat.Name())
	if _, err := os.Stat(newPath); err == nil {
		newPath += " copy"
	}

	cp := exec.Command("cp", "-pR", oldPath, newPath)
	if output, err := cp.CombinedOutput(); err != nil {
		errMsg := string(output)
		if strings.Contains(errMsg, "quota exceeded") {
			errMsg = "磁盘空间不足"
			// 应该不会有什么问题吧
			os.RemoveAll(newPath)
		}
		s.handleError(id, actionCopy, fmt.Errorf("复制文件失败: %s", errMsg))
		return
	}

	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  actionCopy,
	})
}

func (s *FSService) handleDelete(id string, data json.RawMessage) {
	if err := os.RemoveAll(id); err != nil {
		s.handleError(id, actionDelete, err)
		return
	}
	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  actionDelete,
	})
}

func (s *FSService) handleCreate(id string, data json.RawMessage) {
	var d createData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error unmarshalling fs create payload: %v", err)
		return
	}

	newPath := path.Join(id, d.Name)

	if _, err := os.Stat(newPath); err == nil {
		s.handleError(id, actionCreate, fmt.Errorf("目标路径已存在: %s", newPath))
		return
	}

	if d.IsDir {
		if err := os.Mkdir(newPath, 0750); err != nil {
			s.handleError(id, actionCreate, err)
			return
		}
	} else {
		f, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			s.handleError(id, actionCreate, err)
			return
		}
		f.Close()
	}

	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  actionCreate,
	})
}

func (s *FSService) handleRename(id string, data json.RawMessage) {
	var d renameData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error unmarshalling fs rename payload: %v", err)
		return
	}

	if strings.Contains(d.NewName, "/") {
		s.handleError(id, actionRename, fmt.Errorf("文件名不合法: %s", d.NewName))
		return
	}

	oldPath := id
	newPath := path.Join(path.Dir(oldPath), d.NewName)

	if _, err := os.Stat(newPath); err == nil {
		s.handleError(id, actionRename, fmt.Errorf("目标路径已存在: %s", newPath))
		return
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		s.handleError(id, actionRename, err)
		return
	}

	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      oldPath,
		Action:  actionRename,
	})
}

func (s *FSService) handleList(id string, data json.RawMessage) {
	var d listData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error unmarshalling fs list payload: %v", err)
		return
	}

	dirEntries, err := os.ReadDir(id)
	if err != nil {
		s.handleError(id, actionList, err)
		return
	}

	entries := make([]fsEntry, 0, len(dirEntries))

	for _, dirEntry := range dirEntries {
		info, err := dirEntry.Info()
		if err != nil {
			s.Printf("error getting file info: %v", err)
			continue
		}
		if !d.ShowHidden && dirEntry.Name()[0] == '.' {
			continue
		}
		entries = append(entries, fsEntry{
			Name:    dirEntry.Name(),
			Path:    path.Join(id, dirEntry.Name()),
			IsDir:   dirEntry.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime().UnixMilli(),
		})
	}

	d.Entries = entries

	r, err := json.Marshal(d)
	if err != nil {
		s.Printf("error marshalling list response: %v", err)
		return
	}

	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  actionList,
		Data:    r,
	})
}

func (s *FSService) handleGetRoot(id string) {
	rootDir := fsRoot

	info, err := os.Stat(rootDir)
	if err != nil {
		s.Printf("error getting root directory info: %v", err)
		s.handleError(id, actionRoot, err)
		return
	}

	d := rootData{
		Name:    "/",
		Path:    rootDir,
		IsDir:   true,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime().UnixMilli(),
	}

	r, err := json.Marshal(d)
	if err != nil {
		s.Printf("error marshalling root response: %v", err)
		return
	}

	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  actionRoot,
		Data:    r,
	})
}

func (s *FSService) handleError(id, action string, err error) {
	s.Println(err)

	s.conn.WriteJSON(&service.Message{
		Service: s.Name(),
		Id:      id,
		Action:  action,
		Error:   err.Error(),
	})
}

func NewFSService(conn *websocket.Conn) *FSService {
	return &FSService{
		conn:   conn,
		Logger: log.New(os.Stdout, "[fs] ", log.LstdFlags),
	}
}
