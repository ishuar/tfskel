package fs

import (
	"os"
	"path/filepath"
)

// FileSystem provides an abstraction over filesystem operations
// This allows for easier testing by using mock implementations
type FileSystem interface {
	// MkdirAll creates a directory path, creating parent directories as needed
	MkdirAll(path string, perm os.FileMode) error

	// WriteFile writes data to a file, creating it if it doesn't exist
	WriteFile(path string, data []byte, perm os.FileMode) error

	// ReadFile reads the contents of a file
	ReadFile(path string) ([]byte, error)

	// FileExists checks if a file exists
	FileExists(path string) bool

	// DirExists checks if a directory exists
	DirExists(path string) bool
}

// OSFileSystem implements FileSystem using the real OS filesystem
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// MkdirAll creates a directory path
func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// WriteFile writes data to a file
func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

// ReadFile reads a file
func (fs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// FileExists checks if a file exists
func (fs *OSFileSystem) FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func (fs *OSFileSystem) DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
