package fs

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockFileInfo 实现 os.FileInfo 接口
type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

// mockFileSystem 实现 FileSystem 接口
type mockFileSystem struct {
	mock.Mock
}

func (m *mockFileSystem) GetRoot() ([]*FileSystemEntry, error) {
	args := m.Called()
	if entries := args.Get(0); entries != nil {
		return entries.([]*FileSystemEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockFileSystem) List(path string, showHidden bool) ([]*FileSystemEntry, error) {
	args := m.Called(path, showHidden)
	if entries := args.Get(0); entries != nil {
		return entries.([]*FileSystemEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockFileSystem) Create(parentPath string, name string, isDir bool) error {
	args := m.Called(parentPath, name, isDir)
	return args.Error(0)
}

func (m *mockFileSystem) Delete(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *mockFileSystem) Copy(src string, dest string) error {
	args := m.Called(src, dest)
	return args.Error(0)
}

func (m *mockFileSystem) Move(src string, dest string) error {
	args := m.Called(src, dest)
	return args.Error(0)
}

func (m *mockFileSystem) Rename(oldPath string, newName string) error {
	args := m.Called(oldPath, newName)
	return args.Error(0)
}

func TestFileSystem(t *testing.T) {
	mockFS := new(mockFileSystem)

	t.Run("GetRoot", func(t *testing.T) {
		expected := []*FileSystemEntry{{
			Name:    "/",
			Path:    "/home/user",
			IsDir:   true,
			Size:    4096,
			Mode:    os.ModeDir | 0755,
			ModTime: time.Now().Unix(),
		}}

		mockFS.On("GetRoot").Return(expected, nil)

		entries, err := mockFS.GetRoot()
		assert.NoError(t, err)
		assert.Equal(t, expected, entries)

		mockFS.AssertExpectations(t)
	})

	t.Run("List", func(t *testing.T) {
		path := "/home/user"
		expected := []*FileSystemEntry{
			{
				Name:    "file1.txt",
				Path:    "/home/user/file1.txt",
				IsDir:   false,
				Size:    100,
				Mode:    0644,
				ModTime: time.Now().Unix(),
			},
			{
				Name:    "dir1",
				Path:    "/home/user/dir1",
				IsDir:   true,
				Size:    4096,
				Mode:    os.ModeDir | 0755,
				ModTime: time.Now().Unix(),
			},
		}

		mockFS.On("List", path, true).Return(expected, nil)

		entries, err := mockFS.List(path, true)
		assert.NoError(t, err)
		assert.Equal(t, expected, entries)

		mockFS.AssertExpectations(t)
	})

	t.Run("Create", func(t *testing.T) {
		parentPath := "/home/user"
		name := "newfile.txt"
		isDir := false

		mockFS.On("Create", parentPath, name, isDir).Return(nil)

		err := mockFS.Create(parentPath, name, isDir)
		assert.NoError(t, err)

		mockFS.AssertExpectations(t)
	})

	t.Run("Delete", func(t *testing.T) {
		path := "/home/user/file.txt"

		mockFS.On("Delete", path).Return(nil)

		err := mockFS.Delete(path)
		assert.NoError(t, err)

		mockFS.AssertExpectations(t)
	})

	t.Run("Copy", func(t *testing.T) {
		src := "/home/user/source.txt"
		dest := "/home/user/dest"

		mockFS.On("Copy", src, dest).Return(nil)

		err := mockFS.Copy(src, dest)
		assert.NoError(t, err)

		mockFS.AssertExpectations(t)
	})

	t.Run("Move", func(t *testing.T) {
		src := "/home/user/source.txt"
		dest := "/home/user/dest"

		mockFS.On("Move", src, dest).Return(nil)

		err := mockFS.Move(src, dest)
		assert.NoError(t, err)

		mockFS.AssertExpectations(t)
	})

	t.Run("Rename", func(t *testing.T) {
		oldPath := "/home/user/oldfile.txt"
		newName := "newfile.txt"

		mockFS.On("Rename", oldPath, newName).Return(nil)

		err := mockFS.Rename(oldPath, newName)
		assert.NoError(t, err)

		mockFS.AssertExpectations(t)
	})
}
