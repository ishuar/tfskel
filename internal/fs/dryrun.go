package fs

import "os"

// DryRunFileSystem wraps an inner FileSystem, delegating reads but
// making all write operations no-ops. This lets the generator
// run its full logic (including upgrade checks that read files)
// without modifying anything on disk.
type DryRunFileSystem struct {
	inner FileSystem
}

// NewDryRunFileSystem creates a DryRunFileSystem that delegates reads
// to the given inner filesystem and silently skips all writes.
func NewDryRunFileSystem(inner FileSystem) *DryRunFileSystem {
	return &DryRunFileSystem{inner: inner}
}

// MkdirAll is a no-op in dry-run mode.
func (d *DryRunFileSystem) MkdirAll(_ string, _ os.FileMode) error {
	return nil
}

// WriteFile is a no-op in dry-run mode.
func (d *DryRunFileSystem) WriteFile(_ string, _ []byte, _ os.FileMode) error {
	return nil
}

// ReadFile delegates to the inner filesystem.
func (d *DryRunFileSystem) ReadFile(path string) ([]byte, error) {
	return d.inner.ReadFile(path)
}

// ReadDir delegates to the inner filesystem.
func (d *DryRunFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return d.inner.ReadDir(path)
}

// FileExists delegates to the inner filesystem.
func (d *DryRunFileSystem) FileExists(path string) bool {
	return d.inner.FileExists(path)
}

// DirExists delegates to the inner filesystem.
func (d *DryRunFileSystem) DirExists(path string) bool {
	return d.inner.DirExists(path)
}
