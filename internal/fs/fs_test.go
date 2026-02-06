package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOSFileSystem_MkdirAll(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "create valid directory",
			path:    filepath.Join(t.TempDir(), "test", "nested", "dir"),
			wantErr: false,
		},
		{
			name:    "create directory with valid permissions",
			path:    filepath.Join(t.TempDir(), "permtest"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewOSFileSystem()
			err := fs.MkdirAll(tt.path, 0755)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				info, statErr := os.Stat(tt.path)
				assert.NoError(t, statErr)
				assert.True(t, info.IsDir())
			}
		})
	}
}

func TestOSFileSystem_WriteFile(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		perm    os.FileMode
		wantErr bool
	}{
		{
			name:    "write text content",
			content: []byte("test content"),
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "write binary content",
			content: []byte{0x00, 0x01, 0x02, 0xFF},
			perm:    0644,
			wantErr: false,
		},
		{
			name:    "write empty file",
			content: []byte{},
			perm:    0644,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "testfile.txt")

			fs := NewOSFileSystem()
			err := fs.WriteFile(filePath, tt.content, tt.perm)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				readContent, readErr := os.ReadFile(filePath)
				assert.NoError(t, readErr)
				assert.Equal(t, tt.content, readContent)
			}
		})
	}
}

func TestOSFileSystem_ReadFile(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		expectError bool
	}{
		{
			name:        "read existing file",
			content:     []byte("test content"),
			expectError: false,
		},
		{
			name:        "read empty file",
			content:     []byte{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "testfile.txt")

			err := os.WriteFile(filePath, tt.content, 0644)
			require.NoError(t, err)

			fs := NewOSFileSystem()
			content, err := fs.ReadFile(filePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.content, content)
			}
		})
	}

	t.Run("read non-existent file", func(t *testing.T) {
		fs := NewOSFileSystem()
		_, err := fs.ReadFile("/non/existent/file.txt")
		assert.Error(t, err)
	})
}

func TestOSFileSystem_FileExists(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "file exists",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "exists.txt")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)
				return filePath
			},
			expected: true,
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				return "/non/existent/file.txt"
			},
			expected: false,
		},
		{
			name: "directory instead of file",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			fs := NewOSFileSystem()
			result := fs.FileExists(path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOSFileSystem_DirExists(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "directory exists",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: true,
		},
		{
			name: "directory does not exist",
			setup: func(t *testing.T) string {
				return "/non/existent/directory"
			},
			expected: false,
		},
		{
			name: "file instead of directory",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "file.txt")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)
				return filePath
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			fs := NewOSFileSystem()
			result := fs.DirExists(path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMemoryFileSystem_Integration(t *testing.T) {
	t.Run("complete workflow", func(t *testing.T) {
		fs := NewMemoryFileSystem()

		err := fs.MkdirAll("/test/dir", 0755)
		assert.NoError(t, err)
		assert.True(t, fs.DirExists("/test/dir"))

		testContent := []byte("test content")
		err = fs.WriteFile("/test/dir/file.txt", testContent, 0644)
		assert.NoError(t, err)
		assert.True(t, fs.FileExists("/test/dir/file.txt"))

		content, err := fs.ReadFile("/test/dir/file.txt")
		assert.NoError(t, err)
		assert.Equal(t, testContent, content)
	})
}
