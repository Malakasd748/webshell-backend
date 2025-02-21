package fs

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalFileSystem(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()
	fs := &LocalFileSystem{}

	t.Run("GetRoot", func(t *testing.T) {
		entries, err := fs.GetRoot()
		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.True(t, entries[0].IsDir)
		assert.Equal(t, "/", entries[0].Name)
	})

	t.Run("Create and List", func(t *testing.T) {
		// 创建文件夹
		err := fs.Create(tmpDir, "testdir", true)
		assert.NoError(t, err)

		// 创建文件
		err = fs.Create(tmpDir, "testfile.txt", false)
		assert.NoError(t, err)

		// 列出目录内容
		entries, err := fs.List(tmpDir, true)
		assert.NoError(t, err)
		assert.Len(t, entries, 2)

		// 验证文件夹
		var foundDir, foundFile bool
		for _, entry := range entries {
			if entry.Name == "testdir" {
				assert.True(t, entry.IsDir)
				foundDir = true
			}
			if entry.Name == "testfile.txt" {
				assert.False(t, entry.IsDir)
				foundFile = true
			}
		}
		assert.True(t, foundDir, "Directory not found")
		assert.True(t, foundFile, "File not found")
	})

	t.Run("Copy", func(t *testing.T) {
		// 创建源文件
		srcPath := path.Join(tmpDir, "source.txt")
		content := []byte("test content")
		err := os.WriteFile(srcPath, content, 0644)
		assert.NoError(t, err)

		// 复制文件
		err = fs.Copy(srcPath, tmpDir)
		assert.NoError(t, err)

		// 验证复制的文件
		copiedPath := path.Join(tmpDir, "source.txt copy")
		copiedContent, err := os.ReadFile(copiedPath)
		assert.NoError(t, err)
		assert.Equal(t, content, copiedContent)
	})

	t.Run("Move", func(t *testing.T) {
		// 创建源文件
		srcPath := path.Join(tmpDir, "tomove.txt")
		content := []byte("move test")
		err := os.WriteFile(srcPath, content, 0644)
		assert.NoError(t, err)

		// 移动文件
		destDir := path.Join(tmpDir, "testdir")
		err = fs.Move(srcPath, destDir)
		assert.NoError(t, err)

		// 验证文件已移动
		movedPath := path.Join(destDir, "tomove.txt")
		_, err = os.Stat(srcPath)
		assert.True(t, os.IsNotExist(err))
		
		movedContent, err := os.ReadFile(movedPath)
		assert.NoError(t, err)
		assert.Equal(t, content, movedContent)
	})

	t.Run("Delete", func(t *testing.T) {
		filePath := path.Join(tmpDir, "todelete.txt")
		err := os.WriteFile(filePath, []byte("to be deleted"), 0644)
		assert.NoError(t, err)

		err = fs.Delete(filePath)
		assert.NoError(t, err)

		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Rename", func(t *testing.T) {
		// 创建文件
		oldPath := path.Join(tmpDir, "oldname.txt")
		content := []byte("rename test")
		err := os.WriteFile(oldPath, content, 0644)
		assert.NoError(t, err)

		// 重命名
		err = fs.Rename(oldPath, "newname.txt")
		assert.NoError(t, err)

		// 验证旧文件不存在
		_, err = os.Stat(oldPath)
		assert.True(t, os.IsNotExist(err))

		// 验证新文件存在且内容正确
		newPath := path.Join(tmpDir, "newname.txt")
		newContent, err := os.ReadFile(newPath)
		assert.NoError(t, err)
		assert.Equal(t, content, newContent)
	})
}