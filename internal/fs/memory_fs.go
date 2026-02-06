package fs

import (
	"os"
	"sync"
)

// MemoryFileSystem implements FileSystem in memory for testing
type MemoryFileSystem struct {
	mu    sync.RWMutex
	files map[string][]byte
	dirs  map[string]bool
}

// NewMemoryFileSystem creates a new in-memory filesystem
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// MkdirAll creates a directory path
func (fs *MemoryFileSystem) MkdirAll(path string, perm os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.dirs[path] = true
	return nil
}

// WriteFile writes data to a file
func (fs *MemoryFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[path] = data
	return nil
}

// ReadFile reads a file
func (fs *MemoryFileSystem) ReadFile(path string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	data, ok := fs.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

// FileExists checks if a file exists
func (fs *MemoryFileSystem) FileExists(path string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	_, ok := fs.files[path]
	return ok
}

// DirExists checks if a directory exists
func (fs *MemoryFileSystem) DirExists(path string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	_, ok := fs.dirs[path]
	return ok
}
