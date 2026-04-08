package fs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
func (fs *MemoryFileSystem) MkdirAll(path string, _ os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.dirs[path] = true
	return nil
}

// WriteFile writes data to a file
func (fs *MemoryFileSystem) WriteFile(path string, data []byte, _ os.FileMode) error {
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

// ReadDir reads a directory and returns its direct child directories.
// Only directories registered via MkdirAll are returned; files are not
// tracked by MemoryFileSystem and will not appear in the results.
func (fs *MemoryFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if !fs.dirs[path] {
		return nil, os.ErrNotExist
	}

	prefix := filepath.Clean(path) + string(filepath.Separator)
	seen := make(map[string]bool)
	var entries []os.DirEntry

	for dir := range fs.dirs {
		if !strings.HasPrefix(dir, prefix) {
			continue
		}
		// Get the relative path and take only the first segment (direct child)
		rel := strings.TrimPrefix(dir, prefix)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		name := parts[0]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		entries = append(entries, &memDirEntry{name: name})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

// errInfoNotAvailable is returned by memDirEntry.Info as file info is not tracked in memory.
var errInfoNotAvailable = errors.New("file info not available for in-memory directory entry")

// memDirEntry implements os.DirEntry for in-memory directories.
type memDirEntry struct {
	name string
}

// Name returns the directory entry name.
func (e *memDirEntry) Name() string { return e.name }

// IsDir reports whether the entry is a directory (always true for memDirEntry).
func (e *memDirEntry) IsDir() bool { return true }

// Type returns the type bits for the entry.
func (e *memDirEntry) Type() fs.FileMode { return fs.ModeDir }

// Info returns the FileInfo for the entry (not available for in-memory entries).
func (e *memDirEntry) Info() (fs.FileInfo, error) { return nil, errInfoNotAvailable }
