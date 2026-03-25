package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRunFileSystem(t *testing.T) {
	t.Run("WriteFile is a no-op", func(t *testing.T) {
		dir := t.TempDir()
		osFS := NewOSFileSystem()
		dry := NewDryRunFileSystem(osFS)

		path := filepath.Join(dir, "should-not-exist.txt")
		err := dry.WriteFile(path, []byte("hello"), 0644)
		require.NoError(t, err)

		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr), "file should not have been created")
	})

	t.Run("MkdirAll is a no-op", func(t *testing.T) {
		dir := t.TempDir()
		dry := NewDryRunFileSystem(NewOSFileSystem())

		path := filepath.Join(dir, "a", "b", "c")
		err := dry.MkdirAll(path, 0755)
		require.NoError(t, err)

		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr), "directory should not have been created")
	})

	t.Run("ReadFile delegates to real FS", func(t *testing.T) {
		dir := t.TempDir()
		osFS := NewOSFileSystem()
		dry := NewDryRunFileSystem(osFS)

		path := filepath.Join(dir, "existing.txt")
		require.NoError(t, os.WriteFile(path, []byte("content"), 0644))

		data, err := dry.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "content", string(data))
	})

	t.Run("FileExists delegates to real FS", func(t *testing.T) {
		dir := t.TempDir()
		dry := NewDryRunFileSystem(NewOSFileSystem())

		path := filepath.Join(dir, "existing.txt")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

		assert.True(t, dry.FileExists(path))
		assert.False(t, dry.FileExists(filepath.Join(dir, "nope.txt")))
	})

	t.Run("DirExists delegates to real FS", func(t *testing.T) {
		dir := t.TempDir()
		dry := NewDryRunFileSystem(NewOSFileSystem())

		assert.True(t, dry.DirExists(dir))
		assert.False(t, dry.DirExists(filepath.Join(dir, "nope")))
	})
}
