package fs

import (
	"encoding/json"
	"log"

	ws "webshell/websocket"
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

type listData struct {
	// req
	ShowHidden bool `json:"showHidden,omitempty"`
	// res
	Entries []*FileSystemEntry `json:"entries"`
}
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
	conn *ws.Conn

	FS FileSystem
	*log.Logger
}

// Register implements service.Service.
func (s *FSService) Register(conn *ws.Conn) {
	s.conn = conn
}

func (s *FSService) Name() string {
	return "fs"
}

func (s *FSService) HandleTextMessage(id, action string, data json.RawMessage) {
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

	err := s.FS.Move(id, d.Dest)
	if err != nil {
		s.handleError(id, actionMove, err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
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

	err := s.FS.Copy(id, d.Dest)
	if err != nil {
		s.handleError(id, actionCopy, err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionCopy,
	})
}

func (s *FSService) handleDelete(id string, _ json.RawMessage) {
	err := s.FS.Delete(id)
	if err != nil {
		s.handleError(id, actionDelete, err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
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

	err := s.FS.Create(id, d.Name, d.IsDir)
	if err != nil {
		s.handleError(id, actionCreate, err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
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

	err := s.FS.Rename(id, d.NewName)
	if err != nil {
		s.handleError(id, actionRename, err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionRename,
	})
}

func (s *FSService) handleList(id string, data json.RawMessage) {
	var d listData
	if err := json.Unmarshal(data, &d); err != nil {
		s.Printf("error unmarshalling fs list payload: %v", err)
		return
	}

	entries, err := s.FS.List(id, d.ShowHidden)
	if err != nil {
		s.handleError(id, actionList, err)
		return
	}

	d.Entries = entries

	r, err := json.Marshal(d)
	if err != nil {
		s.Printf("error marshalling list response: %v", err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionList,
		Data:    r,
	})
}

func (s *FSService) handleGetRoot(id string) {
	root, err := s.FS.GetRoot()
	if err != nil {
		s.handleError(id, actionRoot, err)
		return
	}

	r, err := json.Marshal(root)
	if err != nil {
		s.Printf("error marshalling root response: %v", err)
		return
	}

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  actionRoot,
		Data:    r,
	})
}

func (s *FSService) handleError(id, action string, err error) {
	s.Println(err)

	s.conn.WriteJSON(&ws.ServiceMessage{
		Service: s.Name(),
		Id:      id,
		Action:  action,
		Error:   err.Error(),
	})
}
