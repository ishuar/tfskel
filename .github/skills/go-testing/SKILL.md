---
name: go-testing
description: Testing best practices for Go projects including table-driven tests, mocking, and test organization. Use this when writing or reviewing test code.
---

# Go Testing Best Practices

## Table-Driven Tests

Use table-driven tests for functions with multiple scenarios:

```go
func TestValidateProjectName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:    "valid project name",
            input:   "my-terraform-project",
            wantErr: false,
        },
        {
            name:    "empty project name",
            input:   "",
            wantErr: true,
        },
        {
            name:    "invalid characters",
            input:   "my_project@123",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateProjectName(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

```
## Mocking with Interfaces
- Define interfaces for dependencies:

```go
// In pkg/structure/filesystem.go
type FileSystem interface {
    MkdirAll(path string, perm os.FileMode) error
    WriteFile(filename string, data []byte, perm os.FileMode) error
}

// Production implementation
type OSFileSystem struct{}

func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
    return os.MkdirAll(path, perm)
}

// Test mock
type MockFileSystem struct {
    mock.Mock
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
    args := m.Called(path, perm)
    return args.Error(0)
}
```

## Test Helpers
- Create helpers in testutil/ package:
```go
package testutil

func CreateTempDir(t *testing.T) string {
    t.Helper()
    dir, err := os.MkdirTemp("", "tfskel-test-*")
    require.NoError(t, err)
    t.Cleanup(func() { os.RemoveAll(dir) })
    return dir
}
```
