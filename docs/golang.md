# Go Programming Concepts in tfskel

## Table of Contents

1. [Introduction](#introduction)
2. [Go Basics](#go-basics)
3. [Packages and Modules](#packages-and-modules)
4. [Interfaces](#interfaces)
5. [Error Handling](#error-handling)
6. [Concurrency](#concurrency)
7. [Testing](#testing)
8. [Advanced Patterns](#advanced-patterns)
9. [Embedding and Template System](#embedding-and-template-system)
10. [Best Practices](#best-practices)
11. [tfskel Architecture Deep Dive](#tfskel-architecture-deep-dive)
12. [Learning Resources](#learning-resources)

---

## Introduction

This document explains the Go programming concepts used in tfskel, making it a learning resource for developers new to Go. Each concept is explained with **real, working examples from the tfskel codebase**, demonstrating practical patterns you can apply in your own projects.

### What You'll Learn

- **Go fundamentals** through actual production code
- **Interface-based design** for testability and flexibility
- **Error handling patterns** used in CLI applications
- **Template rendering** with Go's `text/template`
- **Testing strategies** including mocks and table-driven tests
- **File system abstraction** for testable I/O operations
- **CLI development** with Cobra and Viper

---

## Go Basics

### Package Declaration

Every Go file starts with a package declaration:

```go
// internal/config/config.go
package config
```

**Naming Convention**:
- `main` package: Contains the entry point (`func main()`)
- Other packages: Named after their directory (e.g., `config`, `templates`)

### Imports

Import other packages to use their functions:

```go
import (
    "fmt"           // Standard library
    "os"            // Standard library

    // External packages
    "github.com/spf13/cobra"

    // Internal packages (relative to module root)
    "github.com/ishuar/tfskel/internal/config"
)
```

**Import Types**:
- Standard library: Built-in packages
- External: Third-party packages (fetched via `go get`)
- Internal: Your own packages

### Variables and Constants

```go
// Variable declarations
var projectName string
var count int = 10

// Short declaration (type inferred)
name := "tfskel"

// Constants (immutable)
const Version = "1.0.0"
const MaxRetries = 3
```

### Functions

```go
// Basic function
func add(a, b int) int {
    return a + b
}

// Multiple return values
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, fmt.Errorf("division by zero")
    }
    return a / b, nil
}

// Named return values
func split(sum int) (x, y int) {
    x = sum * 4 / 9
    y = sum - x
    return  // Naked return
}
```

---

## Packages and Modules

### Go Modules

A Go module is a collection of packages with a `go.mod` file:

```go
// go.mod
module github.com/ishuar/tfskel

go 1.22

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
    go.yaml.in/yaml/v4 v4.0.0-rc.4
)
```

**Module Commands**:
```bash
# Initialize a module
go mod init github.com/ishuar/tfskel

# Download dependencies
go mod download

# Add missing dependencies and remove unused ones
go mod tidy

# Verify dependencies
go mod verify
```

### Internal Packages

The `internal/` directory is special in Go:
- Packages inside `internal/` are only importable by packages in the same module
- This enforces encapsulation

```
github.com/ishuar/tfskel/
├── internal/           # Private packages
│   ├── config/        # Only tfskel can import
│   └── templates/     # Only tfskel can import
└── pkg/               # Public packages (if any)
```

### Package Organization

```go
// internal/config/config.go
package config

// Exported (starts with capital letter)
type Config struct {
    Project ProjectConfig
}

// Exported function
func Load(path string) (*Config, error) {
    // ...
}

// Unexported (starts with lowercase letter)
func parseYAML(data []byte) (*Config, error) {
    // Only accessible within config package
}
```

---

## Interfaces

### What is an Interface?

An interface defines a contract - a set of method signatures:

```go
// internal/fs/fs.go
type FileSystem interface {
    WriteFile(path, content string) error
    ReadFile(path string) (string, error)
    MkdirAll(path string) error
}
```

**Key Point**: Any type that implements these methods automatically satisfies the interface (implicit implementation).

### Implementing Interfaces

**Real Example from tfskel** - The FileSystem Interface:

```go
// internal/fs/fs.go - Interface definition
type FileSystem interface {
    // MkdirAll creates a directory path, creating parent directories as needed
    MkdirAll(path string, perm os.FileMode) error

    // WriteFile writes data to a file, creating it if it doesn't exist
    WriteFile(path string, data []byte, perm os.FileMode) error

    // ReadFile reads the contents of a file
    ReadFile(path string) ([]byte, error)

    // Stat returns file info
    Stat(path string) (os.FileInfo, error)

    // FileExists checks if a file exists
    FileExists(path string) bool

    // DirExists checks if a directory exists
    DirExists(path string) bool

    // Getwd returns the current working directory
    Getwd() (string, error)

    // Chdir changes the current working directory
    Chdir(dir string) error

    // Open opens a file for reading
    Open(name string) (io.ReadCloser, error)
}

// OSFileSystem implements FileSystem using the real OS filesystem
type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
    return &OSFileSystem{}
}

// Implements MkdirAll method
func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
    return os.MkdirAll(path, perm)
}

// Implements WriteFile method
func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
    // Ensure directory exists
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    return os.WriteFile(path, data, perm)
}

// Implements FileExists method
func (fs *OSFileSystem) FileExists(path string) bool {
    info, err := os.Stat(path)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

// OSFileSystem now satisfies FileSystem interface!
```

**Key Benefits in tfskel**:
- Production code uses `OSFileSystem` for real file operations
- Tests use `MemoryFileSystem` (in-memory, no disk I/O)
- Both satisfy the same interface, making tests fast and isolated

### Why Use Interfaces?

**Real tfskel Example** - Generator with Interface Dependency:

```go
// internal/app/generator.go
type Generator struct {
    config   *config.Config
    fs       fs.FileSystem  // Interface, not concrete type!
    log      *logger.Logger
    renderer *templates.Renderer
}

// NewGenerator accepts any FileSystem implementation
func NewGenerator(cfg *config.Config, filesystem fs.FileSystem, log *logger.Logger) *Generator {
    return &Generator{
        config: cfg,
        fs:     filesystem,
        log:    log,
    }
}

// Run executes generation with environment, region, and app directory parameters
func (g *Generator) Run(env, region, appDir string) error {
    appPath := filepath.Join("envs", env, region, appDir)

    // Works with ANY FileSystem implementation
    if err := g.fs.MkdirAll(appPath, 0755); err != nil {
        return fmt.Errorf("failed to create directory structure: %w", err)
    }

    return g.generateFiles(appPath, env, region, appDir)
}
```

**In Production (cmd/generate.go)**:
```go
// Real filesystem for production
filesystem := fs.NewOSFileSystem()
generator := app.NewGenerator(cfg, filesystem, log)

// Run with actual parameters from command flags
if err := generator.Run(env, region, appDir); err != nil {
    return fmt.Errorf("generation failed: %w", err)
}
```

**In Tests (internal/app/generator_test.go)**:
```go
// In-memory filesystem for tests - no disk I/O!
filesystem := fs.NewMemoryFileSystem()
generator := app.NewGenerator(cfg, filesystem, log)

// Test runs fast, no cleanup needed
err := generator.Run("dev", "us-east-1", "myapp")
assert.NoError(t, err)

// Verify files were created in memory
assert.True(t, filesystem.FileExists("envs/dev/us-east-1/myapp/backend.tf"))
assert.True(t, filesystem.FileExists("envs/dev/us-east-1/myapp/versions.tf"))
```

**Why This Matters**:
1. **Testability**: Tests run in milliseconds without touching disk
2. **Flexibility**: Easy to add cloud storage (S3FileSystem) later
3. **Safety**: Tests never corrupt real files
4. **Isolation**: Multiple tests run in parallel safely

### Empty Interface

`interface{}` (or `any` in Go 1.18+) accepts any type:

```go
// Can hold any value
var data interface{}
data = 42
data = "hello"
data = []int{1, 2, 3}

// Type assertion to get the value back
value := data.(string)  // Panics if data is not string

// Safe type assertion
value, ok := data.(string)
if ok {
    fmt.Println("It's a string:", value)
}
```

---

## Error Handling

### The Error Interface

Go doesn't have exceptions. Instead, functions return errors:

```go
// Error is a built-in interface
type error interface {
    Error() string
}
```

### Creating Errors

```go
import "fmt"

// Simple error
err := fmt.Errorf("file not found: %s", filename)

// Custom error type
type ConfigError struct {
    Field   string
    Message string
}

func (e *ConfigError) Error() string {
    return fmt.Sprintf("config error in %s: %s", e.Field, e.Message)
}
```

### Sentinel Errors

Sentinel errors are predefined error values that can be compared using `errors.Is()`:

```go
var (
    ErrNotFound = errors.New("not found")
    ErrInvalid  = errors.New("invalid")
)

// Usage
if errors.Is(err, ErrNotFound) {
    // Handle not found case
}
```

**tfskel Config Package Sentinel Errors**:

```go
// internal/config/config.go
var (
    // ErrAWSProviderRequired indicates AWS provider configuration is missing
    ErrAWSProviderRequired = errors.New("AWS provider configuration is required")

    // ErrAccountMappingRequired indicates AWS account mapping is missing
    ErrAccountMappingRequired = errors.New("AWS account mapping is required in provider configuration")

    // ErrAccountMappingNotFound indicates the specified environment has no account mapping
    ErrAccountMappingNotFound = errors.New("no account mapping found for environment")

    // ErrInvalidAccountID indicates an AWS account ID is not properly formatted
    ErrInvalidAccountID = errors.New("AWS account ID must be a 12-digit number (not a placeholder or invalid format)")

    // ErrInvalidBucketName indicates the S3 bucket name is not properly configured
    ErrInvalidBucketName = errors.New("backend.s3.bucket_name must be set to a valid value (not empty or placeholder)")
}
```

These sentinel errors allow precise error handling and testing:

```go
// In tests or error handling
if errors.Is(err, config.ErrAccountMappingNotFound) {
    // Show helpful message about available environments
}
```

### Error Handling Pattern

```go
// Always check errors
result, err := someFunction()
if err != nil {
    // Handle error
    return fmt.Errorf("failed to do something: %w", err)
}
// Use result
```

### Error Wrapping

Go 1.13+ introduced error wrapping:

```go
// Wrap error with context
if err := loadConfig(path); err != nil {
    return fmt.Errorf("failed to load config from %s: %w", path, err)
}

// Unwrap and check specific error
if errors.Is(err, os.ErrNotExist) {
    // Handle file not found
}

// Check error type
var configErr *ConfigError
if errors.As(err, &configErr) {
    // Handle config error
    fmt.Println("Config field:", configErr.Field)
}
```

### tfskel Error Handling Example

**Real Example from internal/app/generator.go**:

```go
func (g *Generator) Run() error {
    // Step 1: Initialize template renderer
    var renderer *templates.Renderer
    var err error
    if g.config.CustomTemplateDir != "" {
        g.log.Infof("Using custom templates from: %s", g.config.CustomTemplateDir)
        renderer, err = templates.NewRendererWithCustomTemplates(g.config.CustomTemplateDir)
    } else {
        g.log.Debug("Using default embedded templates")
        renderer, err = templates.NewRenderer()
    }
    if err != nil {
        // Wrap error with context
        return fmt.Errorf("failed to initialize template renderer: %w", err)
    }
    g.renderer = renderer

    // Step 2: Create directory structure
    appPath := filepath.Join("envs", g.config.Env, g.config.Region, g.config.AppDir)
    if err := g.fs.MkdirAll(appPath, 0755); err != nil {
        return fmt.Errorf("failed to create directory structure: %w", err)
    }

    // Step 3: Generate files from templates
    if err := g.generateFiles(appPath); err != nil {
        return err  // Already has context from generateFiles
    }

    return nil
}
```

**Error Handling in Config Loading (internal/config/config.go)**:

```go
func Load(cmd *cobra.Command, v *viper.Viper) (*Config, error) {
    cfg := &Config{}

    // Unmarshal with error handling
    if err := v.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // Apply flag overrides
    applyFlagOverrides(cmd, cfg)

    // Set defaults
    setDefaults(cfg)

    // Normalize template extensions
    normalizeTemplateExtensions(cfg)

    return cfg, nil
}

func (c *Config) Validate() error {
    // Validate that required configuration sections exist
    if c.Provider == nil || c.Provider.AWS == nil {
        return ErrAWSProviderRequired
    }
    if len(c.Provider.AWS.AccountMapping) == 0 {
        return ErrAccountMappingRequired
    }
    // Validate AWS account IDs are 12-digit numbers
    if err := c.validateAccountIDs(); err != nil {
        return err
    }
    // Validate backend configuration
    if c.Backend == nil || c.Backend.S3 == nil || c.Backend.S3.BucketName == "" {
        return ErrInvalidBucketName
    }
    // Check if user left the example placeholder value
    if c.Backend.S3.BucketName == "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME" {
        return fmt.Errorf("%w: placeholder value must be replaced with actual bucket name", ErrInvalidBucketName)
    }
    return nil
}

// validateAccountIDs checks that all AWS account IDs are valid 12-digit numbers
func (c *Config) validateAccountIDs() error {
    // AWS account IDs are exactly 12 digits
    accountIDPattern := regexp.MustCompile(`^\d{12}$`)

    for env, accountID := range c.Provider.AWS.AccountMapping {
        // Validate format: must be exactly 12 digits
        if !accountIDPattern.MatchString(accountID) {
            return fmt.Errorf("%w for environment %q: %q",
                ErrInvalidAccountID, env, accountID)
        }
    }

    return nil
}

// GetAccountID returns the AWS account ID for the specified environment,
// or an error if no mapping exists for that environment.
func (c *Config) GetAccountID(env string) (string, error) {
    if c.Provider != nil && c.Provider.AWS != nil &&
        c.Provider.AWS.AccountMapping != nil {
        if id, ok := c.Provider.AWS.AccountMapping[env]; ok {
            return id, nil
        }
        // Show available keys to help the user fix it immediately
        available := make([]string, 0, len(c.Provider.AWS.AccountMapping))
        for k := range c.Provider.AWS.AccountMapping {
            available = append(available, k)
        }
        sort.Strings(available)
        return "", fmt.Errorf(
            "%w %q, available: [%s]",
            ErrAccountMappingNotFound, env, strings.Join(available, ", "),
        )
    }
    return "", ErrAWSProviderRequired
}
```

**Error Propagation in CLI (cmd/generate.go)**:

```go
func runGenerate(cmd *cobra.Command, args []string) error {
    log := logger.New(viper.GetBool("verbose"))

    log.Debug("Starting generate command")
    log.Info("Starting Terraform directory scaffolding...")

    // Get app directory from positional argument
    appDir := args[0]

    // Validate generation parameters
    if err := validateGenerateParams(env, region, appDir); err != nil {
        cmd.SilenceUsage = true
        return fmt.Errorf("invalid parameters: %w", err)
    }

    // Load configuration
    cfg, err := config.Load(cmd, viper.GetViper())
    if err != nil {
        cmd.SilenceUsage = true
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        cmd.SilenceUsage = true
        return fmt.Errorf("configuration validation failed: %w", err)
    }

    // Create filesystem abstraction
    filesystem := fs.NewOSFileSystem()

    // Create and run the generator with generation parameters
    // GetAccountID is called internally during template preparation
    generator := app.NewGenerator(cfg, filesystem, log)
    if err := generator.Run(env, region, appDir); err != nil {
        cmd.SilenceUsage = true
        return fmt.Errorf("generation failed: %w", err)
    }

    log.Success("Terraform directory scaffolding completed!")
    return nil
}
```

**Error Handling in Template Preparation (internal/app/generator.go)**:

```go
func (g *Generator) prepareTemplateData(env, region, appDir string) (*templates.Data, error) {
    // ... other preparation code ...

    // Get account ID for the environment - returns error if not found
    accountID, err := g.config.GetAccountID(env)
    if err != nil {
        return nil, err  // Error includes helpful message with available envs
    }

    // Create template data with validated account ID
    data := &templates.Data{
        Env:                env,
        Region:             region,
        AppDir:             appDir,
        AccountID:          accountID,  // Guaranteed to be non-empty
        // ... other fields ...
    }

    return data, nil
}
```

**Benefits of This Approach**:
- Clear error context at every level
- Easy to trace where errors originate
- Wrapped errors preserve the original error
- User-friendly error messages with actionable information
- Early validation prevents issues downstream
- GetAccountID provides helpful hints showing available environments

---

## Concurrency

### Goroutines

Goroutines are lightweight threads managed by the Go runtime:

```go
// Sequential execution
func processFiles(files []string) {
    for _, file := range files {
        process(file)  // Blocks until done
    }
}

// Concurrent execution
func processFilesConcurrent(files []string) {
    for _, file := range files {
        go process(file)  // Runs in parallel
    }
}
```

### Channels

Channels allow goroutines to communicate:

```go
// Create a channel
ch := make(chan string)

// Send to channel (blocks until received)
go func() {
    ch <- "hello"
}()

// Receive from channel (blocks until sent)
msg := <-ch
fmt.Println(msg)  // "hello"
```

### WaitGroups

Wait for multiple goroutines to complete:

```go
import "sync"

func processFilesConcurrent(files []string) {
    var wg sync.WaitGroup

    for _, file := range files {
        wg.Add(1)  // Increment counter

        go func(f string) {
            defer wg.Done()  // Decrement when done
            process(f)
        }(file)
    }

    wg.Wait()  // Block until counter is 0
}
```

### tfskel Concurrency Example

```go
// internal/app/generator.go
func (g *Generator) writeFiles(files map[string]string) error {
    // Channel for errors
    errCh := make(chan error, len(files))
    var wg sync.WaitGroup

    // Write files concurrently
    for path, content := range files {
        wg.Add(1)
        go func(p, c string) {
            defer wg.Done()

            if err := g.fs.WriteFile(p, c); err != nil {
                errCh <- fmt.Errorf("failed to write %s: %w", p, err)
            }
        }(path, content)
    }

    // Wait for all writes to complete
    wg.Wait()
    close(errCh)

    // Check for errors
    for err := range errCh {
        if err != nil {
            return err
        }
    }

    return nil
}
```

### Select Statement

Choose between multiple channel operations:

```go
select {
case msg := <-ch1:
    fmt.Println("Received from ch1:", msg)
case msg := <-ch2:
    fmt.Println("Received from ch2:", msg)
case <-time.After(1 * time.Second):
    fmt.Println("Timeout")
}
```

---

## Testing

### Test Files

Test files end with `_test.go`:

```
config.go       # Implementation
config_test.go  # Tests
```

### Basic Test

```go
// internal/util/transform_test.go
package util

import "testing"

func TestToSnakeCase(t *testing.T) {
    result := ToSnakeCase("MyProject")
    expected := "my_project"

    if result != expected {
        t.Errorf("ToSnakeCase() = %s; want %s", result, expected)
    }
}
```

### Table-Driven Tests

Test multiple cases efficiently:

```go
func TestToSnakeCase(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"PascalCase", "MyProject", "my_project"},
        {"camelCase", "myProject", "my_project"},
        {"spaces", "My Project", "my_project"},
        {"mixed", "My-Project_Name", "my_project_name"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ToSnakeCase(tt.input)
            if result != tt.expected {
                t.Errorf("got %s, want %s", result, tt.expected)
            }
        })
    }
}
```

### Testify Package

Popular assertion library:

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
    cfg := &Config{Name: "test"}

    // Assertions
    assert.NotNil(t, cfg)
    assert.Equal(t, "test", cfg.Name)
    assert.True(t, len(cfg.Name) > 0)
}
```

### Test Helpers

```go
// Create test helper functions
func createTestConfig(t *testing.T) *Config {
    t.Helper()  // Marks this as a helper function

    cfg := &Config{
        Project: ProjectConfig{Name: "test"},
    }
    return cfg
}

func TestGenerator(t *testing.T) {
    cfg := createTestConfig(t)
    // Use cfg...
}
```

### Testing with Interfaces

**Real tfskel Test Example (internal/app/generator_test.go)**:

```go
func TestGenerator_generateFiles(t *testing.T) {
    t.Run("generate all files from templates", func(t *testing.T) {
        cfg := &config.Config{
            Env:              "dev",
            Region:           "us-east-1",
            AppDir:           "myapp",
            TerraformVersion: "~> 1.13",
            Provider: &config.Provider{
                AWS: &config.AWSProvider{
                    Version: "~> 6.0",
                    AccountMapping: map[string]string{
                        "dev": "123456789012",
                    },
                },
            },
            Backend: &config.Backend{
                S3: &config.S3Backend{
                    BucketName: "my-terraform-state-bucket",
                },
            },
        }

        // Use in-memory filesystem - fast, no cleanup needed
        filesystem := fs.NewMemoryFileSystem()
        log := logger.New(false)

        appPath := "envs/dev/us-east-1/myapp"
        _ = filesystem.MkdirAll(appPath, 0755)

        gen := NewGenerator(cfg, filesystem, log)

        // Initialize renderer
        renderer, err := templates.NewRenderer()
        require.NoError(t, err)
        gen.renderer = renderer

        // Test file generation
        err = gen.generateFiles(appPath)
        assert.NoError(t, err)

        // Verify files were created
        expectedFiles := []string{
            "versions.tf",
            "backend.tf",
        }

        for _, file := range expectedFiles {
            filePath := filepath.Join(appPath, file)
            assert.True(t, filesystem.FileExists(filePath),
                "expected file %s to exist", file)

            content, readErr := filesystem.ReadFile(filePath)
            assert.NoError(t, readErr)
            assert.NotEmpty(t, content,
                "expected file %s to have content", file)
        }
    })

    t.Run("skip existing files", func(t *testing.T) {
        // Setup config and filesystem
        cfg := &config.Config{ /* ... */ }
        filesystem := fs.NewMemoryFileSystem()
        log := logger.New(false)

        appPath := "envs/dev/us-east-1/myapp"
        _ = filesystem.MkdirAll(appPath, 0755)

        // Pre-create a file
        existingFilePath := filepath.Join(appPath, "backend.tf")
        existingContent := []byte("existing content")
        _ = filesystem.WriteFile(existingFilePath, existingContent, 0644)

        gen := NewGenerator(cfg, filesystem, log)
        renderer, _ := templates.NewRenderer()
        gen.renderer = renderer

        // Generate files
        err := gen.generateFiles(appPath)
        assert.NoError(t, err)

        // Verify existing file wasn't overwritten
        content, _ := filesystem.ReadFile(existingFilePath)
        assert.Equal(t, existingContent, content,
            "existing file should not be overwritten")
    })
}
```

**In-Memory FileSystem for Testing (internal/fs/memory_fs.go)**:

```go
// MemoryFileSystem implements FileSystem in memory for testing
type MemoryFileSystem struct {
    mu    sync.RWMutex
    files map[string][]byte
    dirs  map[string]bool
    cwd   string
}

func NewMemoryFileSystem() *MemoryFileSystem {
    return &MemoryFileSystem{
        files: make(map[string][]byte),
        dirs:  make(map[string]bool),
        cwd:   "/",
    }
}

func (fs *MemoryFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
    fs.mu.Lock()
    defer fs.mu.Unlock()
    fs.files[path] = data
    return nil
}

func (fs *MemoryFileSystem) ReadFile(path string) ([]byte, error) {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    data, ok := fs.files[path]
    if !ok {
        return nil, os.ErrNotExist
    }
    return data, nil
}

func (fs *MemoryFileSystem) FileExists(path string) bool {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    _, ok := fs.files[path]
    return ok
}
```

**Why This Is Powerful**:
- Tests run in milliseconds (no disk I/O)
- No cleanup required (everything in memory)
- Tests can run in parallel safely
- Easy to set up complex scenarios
- Deterministic behavior

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./internal/config

# Run specific test
go test -run TestToSnakeCase

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose output
go test -v ./...
```

---

## Advanced Patterns

### Struct Embedding

Go supports composition via embedding:

```go
// Base logger
type Logger struct {
    level string
}

func (l *Logger) Log(msg string) {
    fmt.Println(msg)
}

// Enhanced logger embeds Logger
type FileLogger struct {
    Logger  // Embedded field
    file    *os.File
}

func (f *FileLogger) LogToFile(msg string) {
    f.Log(msg)         // Can call embedded methods
    f.file.WriteString(msg)
}
```

### Method Receivers

Methods can have value or pointer receivers:

```go
// Value receiver (receives a copy)
func (c Config) GetName() string {
    return c.Name
}

// Pointer receiver (receives a reference)
func (c *Config) SetName(name string) {
    c.Name = name  // Modifies original
}

// Rule of thumb:
// - Use pointer receiver if method modifies the receiver
// - Use pointer receiver for large structs (avoid copying)
// - Be consistent: if one method uses pointer, all should
```

### Functional Options Pattern

Flexible function configuration:

```go
// Option is a function that modifies Config
type Option func(*Config)

// Option constructors
func WithBackend(backend string) Option {
    return func(c *Config) {
        c.Backend = backend
    }
}

func WithProviders(providers []string) Option {
    return func(c *Config) {
        c.Providers = providers
    }
}

// Constructor accepts options
func NewConfig(opts ...Option) *Config {
    cfg := &Config{
        // Defaults
        Backend: "local",
    }

    // Apply options
    for _, opt := range opts {
        opt(cfg)
    }

    return cfg
}

// Usage
cfg := NewConfig(
    WithBackend("s3"),
    WithProviders([]string{"aws", "kubernetes"}),
)
```

### Context

Context carries deadlines, cancellation, and request-scoped values:

```go
import "context"

func doWork(ctx context.Context) error {
    select {
    case <-time.After(5 * time.Second):
        return nil  // Work completed
    case <-ctx.Done():
        return ctx.Err()  // Cancelled or timeout
    }
}

// Usage with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := doWork(ctx); err != nil {
    // Handle error (timeout or cancellation)
}
```

### Defer, Panic, Recover

```go
// Defer: Execute at function exit (LIFO order)
func processFile() error {
    f, err := os.Open("file.txt")
    if err != nil {
        return err
    }
    defer f.Close()  // Will close when function returns

    // Process file...
    return nil
}

// Panic: Use for developer errors (not user errors)
// Example from cmd/generate.go - flag binding errors are developer mistakes
mustBindPFlag := func(key string, flagName string) {
    if err := viper.BindPFlag(key, generateCmd.Flags().Lookup(flagName)); err != nil {
        // This should never happen unless code has a bug
        panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
    }
}

// Recover: Catch panics (use sparingly)
func safeExecute(fn func()) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic recovered: %v", r)
        }
    }()

    fn()
    return nil
}
```

---

## Best Practices

### 1. Error Handling

```go
// ✅ Good: Check every error
result, err := doSomething()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// ❌ Bad: Ignore errors with blank identifier
result, _ := doSomething()

// ❌ Bad: Silent error swallowing with nolint
//nolint:errcheck
_ = viper.BindPFlag("key", flag)

// ✅ Good: Panic for developer errors during initialization
mustBindPFlag := func(key string, flagName string) {
    if err := viper.BindPFlag(key, generateCmd.Flags().Lookup(flagName)); err != nil {
        panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
    }
}
```

### 2. Variable Naming

```go
// ✅ Good: Descriptive names
func LoadConfig(filePath string) (*Config, error)

// ❌ Bad: Unclear names
func Load(fp string) (*Config, error)

// Exception: Short names OK in small scopes
for i, item := range items {
    // i and item are clear in context
}
```

### 3. Package Structure

```go
// ✅ Good: Flat structure
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── templates/
│   ├── renderer.go
│   └── renderer_test.go

// ❌ Bad: Deep nesting
internal/
└── pkg/
    └── core/
        └── config/
            └── loader/
                └── config.go
```

### 4. Interfaces

```go
// ✅ Good: Small, focused interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

// ❌ Bad: Large, unfocused interfaces
type FileManager interface {
    Read(p []byte) (n int, err error)
    Write(p []byte) (n int, err error)
    Close() error
    Seek(offset int64, whence int) (int64, error)
    Stat() (os.FileInfo, error)
    // ... many more methods
}
```

### 5. Error Messages

```go
// ✅ Good: Descriptive, lowercase, no punctuation
return fmt.Errorf("failed to load config from %s: %w", path, err)

// ❌ Bad: Vague, capitalized, punctuation
return fmt.Errorf("Error!")
```

### 6. Comments

```go
// ✅ Good: Explain why, not what
// Buffer size of 1024 prevents memory issues with large files
const BufferSize = 1024

// ❌ Bad: States the obvious
// BufferSize is 1024
const BufferSize = 1024

// ✅ Good: Document public APIs
// LoadConfig reads and parses a YAML configuration file.
// It returns an error if the file doesn't exist or is invalid.
func LoadConfig(path string) (*Config, error)
```

### 7. Zero Values

Use meaningful zero values:

```go
// ✅ Good: Zero value is useful
type Config struct {
    Timeout time.Duration  // Zero value (0) is valid
    Retries int           // Zero value (0) is valid
}

// Can use without initialization
cfg := Config{}  // timeout=0, retries=0

// ❌ Bad: Zero value is invalid
type Config struct {
    Timeout *time.Duration  // nil is not useful
}
```

### 8. Make vs New

```go
// make: Creates slices, maps, channels
slice := make([]int, 0, 10)      // Slice with capacity 10
m := make(map[string]int)         // Map
ch := make(chan string, 5)        // Buffered channel

// new: Allocates memory, returns pointer
config := new(Config)             // Equivalent to &Config{}
```

---

## Go Tools

### go fmt

Format code automatically:

```bash
# Format all files
go fmt ./...

# Check formatting without changing
gofmt -l .
```

### go vet

Static analysis:

```bash
# Check for common mistakes
go vet ./...
```

### golangci-lint

Comprehensive linter:

```bash
# Install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run
golangci-lint run
```

### go mod

Module management:

```bash
# Initialize module
go mod init github.com/ishuar/tfskel

# Add dependencies
go get github.com/spf13/cobra@latest

# Update dependencies
go get -u ./...

# Remove unused dependencies
go mod tidy

# Vendor dependencies
go mod vendor
```

### Delve (debugger)

Debug Go programs:

```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug
```

---

## Common Patterns in tfskel

### 1. Factory Pattern

```go
// internal/logger/logger.go
func NewLogger(level string) *Logger {
    return &Logger{
        level:  level,
        output: os.Stdout,
    }
}
```

### 2. Builder Pattern

```go
// internal/config/config.go
func (c *Config) SetDefaults() *Config {
    if c.Structure.BaseDir == "" {
        c.Structure.BaseDir = "."
    }
    return c
}

// Usage
cfg := &Config{}
cfg.SetDefaults().Validate()
```

### 3. Strategy Pattern

```go
// internal/fs/fs.go
type FileSystem interface {
    WriteFile(path, content string) error
}

// Different strategies
type OSFileSystem struct{}      // Real filesystem
type MemoryFileSystem struct{}  // In-memory (testing)
// Future: S3FileSystem, GCSFileSystem, etc.
```

### 4. Template Method Pattern

```go
// internal/app/generator.go
func (g *Generator) Run() error {
    // Template method defines the algorithm structure
    if err := g.initializeRenderer(); err != nil {
        return err
    }
    if err := g.createDirectories(); err != nil {
        return err
    }
    return g.generateFiles()
}
```

---

## Embedding and Template System

### Go's embed Directive

**Real Example from tfskel (internal/templates/renderer.go)**:

```go
package templates

import (
    "embed"
    "io/fs"
)

//go:embed files/*.tmpl
var embeddedTemplates embed.FS

// templateFS is the sub-filesystem without the "files/" prefix
var defaultTemplateFS fs.FS

func init() {
    var err error
    defaultTemplateFS, err = fs.Sub(embeddedTemplates, "files")
    if err != nil {
        panic(fmt.Sprintf("failed to create template sub-filesystem: %v", err))
    }
}
```

**What This Does**:
- Embeds all `.tmpl` files from `internal/templates/files/` into the binary at compile time
- No need to distribute template files separately
- Templates are always available, even in single-binary distributions
- Zero runtime dependencies on filesystem for templates

### Text Template System

**Template Data Structure (internal/templates/renderer.go)**:

```go
// Data holds all the data needed for template rendering
type Data struct {
    Env                string
    Region             string
    AppDir             string
    AccountID          string
    ShortRegion        string
    S3BucketName       string
    TerraformVersion   string
    AWSProviderVersion string
    DefaultTags        map[string]string
}
```

**Template Rendering Example (backend.tf.tmpl)**:

```gotmpl
## This file is auto generated by tfskel
## Verify the bucket name & make sure it exists in your AWS account.
## tfskel-metadata: {"bucket": "{{.S3BucketName}}"}

terraform {
  backend "s3" {
    bucket              = "{{.S3BucketName}}"
    key                 = "{{.AppDir}}-{{.Region}}-{{.Env}}/terraform.tfstate"
    region              = "{{.Region}}"
    encrypt             = true
    use_lockfile        = true
    allowed_account_ids = ["{{.AccountID}}"]
  }
}
```

**Renderer Implementation**:

```go
// Renderer handles template rendering
type Renderer struct {
    templates map[string]*template.Template
}

func NewRenderer() (*Renderer, error) {
    r := &Renderer{
        templates: make(map[string]*template.Template),
    }

    // Load default embedded templates
    if err := r.loadTemplatesFromFS(defaultTemplateFS, "default"); err != nil {
        return nil, err
    }

    return r, nil
}

func (r *Renderer) Render(templateName string, data *Data) (string, error) {
    tmpl, ok := r.templates[templateName]
    if !ok {
        return "", fmt.Errorf("template %s not found", templateName)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }

    return buf.String(), nil
}
```

**Custom Template Support**:

```go
// NewRendererWithCustomTemplates allows overriding default templates
func NewRendererWithCustomTemplates(customTemplateDir string) (*Renderer, error) {
    r := &Renderer{
        templates: make(map[string]*template.Template),
    }

    // Load default embedded templates first
    if err := r.loadTemplatesFromFS(defaultTemplateFS, "default"); err != nil {
        return nil, err
    }

    // Load custom templates (override defaults if same name)
    if customTemplateDir != "" {
        if err := r.loadCustomTemplates(customTemplateDir); err != nil {
            return nil, fmt.Errorf("failed to load custom templates: %w", err)
        }
    }

    return r, nil
}
```

### Advanced Template Patterns

#### Static Content Handling

**Challenge**: Some files need to be embedded but cannot be processed as Go templates because they contain conflicting syntax (e.g., GitHub Actions workflows with `${{ }}` expressions).

**Solution Pattern from internal/templates/renderer.go**:

```go
// Separate storage for static content (no template processing)
type Renderer struct {
    templates     map[string]*template.Template  // Templated files (.tmpl)
    staticContent map[string]string              // Static files (.yaml, .yml)
}

// During loading, differentiate by file extension
func (r *Renderer) loadTemplatesFromFS(fsys fs.FS, source string) error {
    return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
        if d.IsDir() {
            return nil
        }

        content, _ := fs.ReadFile(fsys, path)

        // Static files: store as-is
        if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
            r.staticContent[name] = string(content)
            return nil
        }

        // Template files: parse and compile
        if strings.HasSuffix(path, ".tmpl") {
            tmpl, _ := template.New(name).Funcs(funcMap).Parse(string(content))
            r.templates[name] = tmpl
        }

        return nil
    })
}

// Render handles both types
func (r *Renderer) Render(templateName string, data *Data) (string, error) {
    // Check static content first
    if content, exists := r.staticContent[templateName]; exists {
        return content, nil  // Return as-is
    }

    // Fall back to template rendering
    tmpl := r.templates[templateName]
    // ... execute template with data
}
```

**Why This Matters**:
- GitHub Actions syntax `${{ inputs.value }}` conflicts with Go template `{{ .Value }}`
- Preserves original file syntax without escaping
- Allows embedding any file type, not just templates
- Clean separation: templates are processed, static files are copied

**Use Case**: Reusable GitHub Actions workflows with GitHub-specific syntax:

```yaml
# reusable-lint.yaml (stored as static content)
inputs:
  terraform_files_path:
    required: true
    type: string
  enable_terraform_docs_check:
    required: false
    type: boolean
    default: false

steps:
  - name: Checkout
    uses: actions/checkout@v4

  - name: Setup Terraform
    uses: hashicorp/setup-terraform@v3
    with:
      terraform_version: ${{ inputs.terraform_version }}  # GitHub Actions syntax preserved
```

#### Dynamic Field Injection

**Challenge**: Some template values need to be computed differently for each file being generated, not from the global config.

**Solution Pattern from internal/app/generator.go**:

```go
// Add computed field to template data
type Data struct {
    // ... existing fields from config ...
    WorkflowFileName string  // Dynamically computed per-template
}

// Function to compute dynamic values
func (g *Generator) generateWorkflowFileName(appDir, env, shortRegion, workflowType string) string {
    // Use custom template pattern from config
    nameTemplate := g.config.Generate.GithubWorkflows.NameTemplate
    if nameTemplate == "" {
        nameTemplate = "{{.AppDir}}-{{.Env}}-{{.ShortRegion}}"
    }

    // Process the template
    tmpl := template.Must(template.New("name").Parse(nameTemplate))
    var buf bytes.Buffer
    tmpl.Execute(&buf, Data{
        AppDir:      appDir,
        Env:         env,
        ShortRegion: shortRegion,
    })

    // Automatically append workflow type
    return fmt.Sprintf("%s-%s.yaml", buf.String(), workflowType)
}

// Inject computed value during rendering
func (g *Generator) processTemplate(templateData Data, templateName, workflowType string) (string, error) {
    // Compute dynamic filename
    workflowFileName := g.generateWorkflowFileName(
        templateData.AppDir,
        templateData.Env,
        templateData.ShortRegion,
        workflowType,
    )

    // Inject into template data
    templateData.WorkflowFileName = workflowFileName

    // Render with enhanced data
    return g.renderer.Render(templateName, &templateData)
}
```

**Why This Matters**:
- Enables self-referencing templates (workflows that know their own filename)
- Keeps config simple (users don't need to specify computed values)
- Centralizes naming logic (consistent across all workflows)
- Template has access to its context-specific data

**Use Case**: GitHub Actions workflows that reference themselves in trigger paths:

```yaml
# lint.yaml.tmpl
on:
  pull_request:
    branches:
      - main
    paths:
      - 'envs/{{.Env}}/{{.Region}}/{{.AppDir}}/**'
      - '.github/workflows/{{.WorkflowFileName}}'  # Self-reference using computed field
```

#### Automatic Suffix Appending

**Pattern**: Hide implementation details from users while maintaining internal flexibility.

**Real Implementation**:

```go
// User specifies name pattern WITHOUT workflow type
// Config: name_template: "{{.AppDir}}-{{.Env}}-{{.ShortRegion}}"

// System automatically appends type suffix
func (g *Generator) generateWorkflowFileName(appDir, env, shortRegion, workflowType string) string {
    baseName := processTemplate(nameTemplate, data)  // "myapp-dev-euc1"
    return fmt.Sprintf("%s-%s.yaml", baseName, workflowType)  // "myapp-dev-euc1-lint.yaml"
}

// Generate both workflow types with consistent naming
workflowTypes := []string{"lint", "terraform"}
for _, wfType := range workflowTypes {
    fileName := g.generateWorkflowFileName(appDir, env, region, wfType)
    // Generate: myapp-dev-euc1-lint.yaml
    // Generate: myapp-dev-euc1-terraform.yaml
}
```

**Benefits**:
- Users specify one pattern, get multiple consistent files
- Prevents naming conflicts (guaranteed unique per type)
- Simplifies configuration (no need for type-specific templates)
- Maintainable (adding new workflow types doesn't change config schema)

**Anti-Pattern (avoided)**:

```yaml
# DON'T do this - exposes too much complexity to users
name_template_lint: "{{.AppDir}}-{{.Env}}-{{.ShortRegion}}-lint"
name_template_terraform: "{{.AppDir}}-{{.Env}}-{{.ShortRegion}}-terraform"
```

#### Combining All Patterns

**Complete GitHub Workflows Generation Flow**:

```go
// 1. Determine which files need which treatment
workflows := []struct {
    templateName string
    outputName   string
    workflowType string
    isStatic     bool
}{
    {"github/lint.yaml.tmpl", "", "lint", false},           // Template
    {"github/terraform.yaml.tmpl", "", "terraform", false}, // Template
    {"github/reusable-lint.yaml", "reusable-lint.yaml", "", true},    // Static
    {"github/reusable-terraform-plan-apply.yaml", "reusable-terraform-plan-apply.yaml", "", true}, // Static
}

// 2. Process each workflow
for _, wf := range workflows {
    if wf.isStatic {
        // Static files: retrieve and write as-is
        content, _ := g.renderer.Render(wf.templateName, nil)
        g.fs.WriteFile(outputPath, content)
    } else {
        // Templated files: compute dynamic fields and render
        workflowFileName := g.generateWorkflowFileName(appDir, env, region, wf.workflowType)
        templateData.WorkflowFileName = workflowFileName  // Inject dynamic field

        content, _ := g.renderer.Render(wf.templateName, &templateData)
        outputPath := filepath.Join(".github/workflows", workflowFileName)
        g.fs.WriteFile(outputPath, content)
    }
}
```

**This approach provides**:
- **Flexibility**: Mix templated and static files in same generation run
- **Context-awareness**: Each template has access to computed values specific to it
- **User simplicity**: Config hides complexity behind sensible defaults
- **Maintainability**: Clear separation makes code easy to reason about

---

## tfskel Architecture Deep Dive

### Component Overview

tfskel is organized into several key components, each with a specific responsibility:

```
tfskel/
├── main.go                    # Entry point
├── cmd/                       # CLI commands (Cobra)
│   ├── root.go               # Root command & global config
│   ├── init.go               # Initialize project structure
│   └── scaffold.go           # Scaffold Terraform files
└── internal/                  # Private packages
    ├── app/                  # Core application logic
    │   └── generator.go      # Orchestrates file generation
    ├── config/               # Configuration management
    │   └── config.go         # Load, validate, merge configs
    ├── fs/                   # Filesystem abstraction
    │   ├── fs.go            # Interface & OS implementation
    │   └── memory_fs.go     # In-memory implementation
    ├── logger/              # Structured logging
    │   └── logger.go        # Color-coded CLI output
    ├── templates/           # Template system
    │   ├── renderer.go      # Template rendering engine
    │   └── files/           # Embedded template files
    └── util/               # Utilities
        └── transform.go    # String transformations
```

### Architectural Principles

#### 1. Interface-Based Design

Every external dependency is abstracted behind an interface:

```go
// Filesystem operations
type FileSystem interface { /* ... */ }

// Allows testing without touching disk
filesystem := fs.NewMemoryFileSystem()

// Production uses real filesystem
filesystem := fs.NewOSFileSystem()
```

#### 2. Dependency Injection

Components receive their dependencies through constructors:

```go
// Generator doesn't create its own dependencies
func NewGenerator(cfg *config.Config, fs FileSystem, log *logger.Logger) *Generator {
    return &Generator{
        config: cfg,
        fs:     fs,
        log:    log,
    }
}

// Caller controls what implementations to use
generator := app.NewGenerator(cfg, filesystem, logger)
```

#### 3. Separation of Concerns

Each package has a single, well-defined responsibility:

- **cmd**: CLI interface, flag parsing, user interaction
- **config**: Configuration loading, validation, defaults
- **app**: Business logic, orchestration
- **templates**: Template rendering, file generation
- **fs**: Filesystem operations
- **logger**: Logging, output formatting
- **util**: Reusable utilities

### Data Flow

```
User Command
    ↓
cmd/root.go (Parse flags, load config)
    ↓
cmd/scaffold.go (Validate, create dependencies)
    ↓
app/generator.go (Orchestrate generation)
    ↓
┌───────────────┬──────────────┬────────────────┐
↓               ↓              ↓                ↓
config/         templates/     fs/              logger/
(Read config)   (Render)       (Write files)    (Log progress)
```

### Key Design Patterns in tfskel

#### 1. Factory Pattern

```go
// logger/logger.go
func New(verbose bool) *Logger {
    level := InfoLevel
    if verbose {
        level = DebugLevel
    }

    return &Logger{
        level:  level,
        out:    os.Stdout,
        errOut: os.Stderr,
    }
}
```

#### 2. Strategy Pattern

```go
// Different filesystem strategies
type FileSystem interface { /* ... */ }

type OSFileSystem struct{}      // Production
type MemoryFileSystem struct{}  // Testing
```

#### 3. Template Method Pattern

```go
// generator.go defines the algorithm structure
func (g *Generator) Run() error {
    // Step 1: Initialize
    if err := g.initializeRenderer(); err != nil {
        return err
    }

    // Step 2: Create structure
    if err := g.createDirectories(); err != nil {
        return err
    }

    // Step 3: Generate files
    return g.generateFiles()
}
```

#### 4. Builder Pattern

```go
// Config builder with method chaining
type Config struct { /* ... */ }

func (c *Config) SetDefaults() *Config {
    if c.TerraformVersion == "" {
        c.TerraformVersion = "~> 1.13"
    }
    return c
}

// Usage
cfg := &Config{}.SetDefaults().Validate()
```

### Configuration Management

**Configuration Precedence** (highest to lowest):

1. Command-line flags
2. Configuration file values
3. Interactive prompts
4. Default values

**Implementation (config/config.go)**:

```go
func Load(cmd *cobra.Command, v *viper.Viper) (*Config, error) {
    cfg := &Config{}

    // Load from file
    if err := v.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // Override with command flags
    if cmd.Flags().Changed("env") {
        cfg.Env, _ = cmd.Flags().GetString("env")
    }

    // Set defaults
    if cfg.TerraformVersion == "" {
        cfg.TerraformVersion = "~> 1.13"
    }

    // Prompt for missing values
    if err := cfg.promptForMissingValues(); err != nil {
        return nil, err
    }

    return cfg, nil
}
```

### Logging System

**Structured Logging with Levels (internal/logger/logger.go)**:

```go
type LogLevel int

const (
    DebugLevel LogLevel = iota
    InfoLevel
    WarnLevel
    SuccessLevel
    ErrorLevel
    FatalLevel
)

type Logger struct {
    level  LogLevel
    out    io.Writer
    errOut io.Writer
}

// Methods for different log levels
func (l *Logger) Debug(msg string) {
    if l.level <= DebugLevel {
        l.logWithColor(colorCyan, "DEBUG", msg)
    }
}

func (l *Logger) Info(msg string) {
    if l.level <= InfoLevel {
        l.logWithColor(colorBlue, "INFO", msg)
    }
}

func (l *Logger) Success(msg string) {
    if l.level <= SuccessLevel {
        l.logWithColor(colorGreen, "✓", msg)
    }
}

func (l *Logger) Error(msg string) {
    if l.level <= ErrorLevel {
        l.logToErrOut(colorRed, "ERROR", msg)
    }
}
```

**Usage in Application**:

```go
log := logger.New(verbose)

log.Debug("Starting generator...")
log.Info("Creating directory structure...")
log.Success("Successfully created backend.tf")
log.Warn("File already exists, skipping")
log.Error("Failed to write file")
```

### Template System Details

**Metadata for Idempotency**:

tfskel embeds metadata in generated files to track configuration changes:

```go
// Extract metadata from generated file
func extractMetadata(content, metadataKey string) (map[string]string, error) {
    // Look for pattern: ## tfskel-metadata: {JSON}
    pattern := fmt.Sprintf(`##\s*tfskel-%s:\s*({[^}]*})`, regexp.QuoteMeta(metadataKey))
    re := regexp.MustCompile(pattern)

    matches := re.FindStringSubmatch(content)
    if len(matches) < 2 {
        return nil, fmt.Errorf("metadata key %s not found", metadataKey)
    }

    var metadata map[string]string
    if err := json.Unmarshal([]byte(matches[1]), &metadata); err != nil {
        return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
    }

    return metadata, nil
}

// Compare metadata to detect configuration changes
func compareMetadata(fileMetadata, configMetadata map[string]string) (bool, []string) {
    var changes []string

    for key, configValue := range configMetadata {
        if fileValue, exists := fileMetadata[key]; !exists {
            changes = append(changes, fmt.Sprintf("%s added: %s", key, configValue))
        } else if fileValue != configValue {
            changes = append(changes, fmt.Sprintf("%s changed: %s -> %s",
                key, fileValue, configValue))
        }
    }

    return len(changes) > 0, changes
}
```

**Smart File Updates**:

```go
// Check if backend.tf needs updating
if g.fs.FileExists(backendPath) {
    needsUpdate, err := g.shouldUpdateBackend(backendPath, data.S3BucketName)
    if err != nil {
        g.log.Warnf("Failed to check backend.tf: %v", err)
    } else if needsUpdate {
        // Regenerate only if config changed
        if err := g.updateBackendFile(backendPath, data); err != nil {
            return fmt.Errorf("failed to update backend.tf: %w", err)
        }
        g.log.Successf("Updated backend.tf with new bucket_name: %s", data.S3BucketName)
    }
}
```

### Testing Strategy

#### Unit Tests

Each package has comprehensive unit tests:

```go
// internal/util/transform_test.go
func TestTransformRegionName(t *testing.T) {
    tests := []struct {
        name     string
        region   string
        expected string
    }{
        {"eu-central-1", "eu-central-1", "euc1"},
        {"us-east-1", "us-east-1", "use1"},
        {"ap-south-1", "ap-south-1", "aps1"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := TransformRegionName(tt.region)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### Integration Tests

Test components working together:

```go
// internal/app/generator_test.go
func TestGenerator_Run(t *testing.T) {
    // Setup full configuration
    cfg := &config.Config{ /* complete config */ }

    // Use in-memory filesystem
    filesystem := fs.NewMemoryFileSystem()
    log := logger.New(false)

    // Create and run generator
    gen := NewGenerator(cfg, filesystem, log)
    err := gen.Run()

    // Verify results
    assert.NoError(t, err)
    assert.True(t, filesystem.FileExists("envs/dev/us-east-1/myapp/backend.tf"))
    assert.True(t, filesystem.FileExists("envs/dev/us-east-1/myapp/versions.tf"))
}
```

#### Benefits of tfskel's Testing Approach

1. **Fast**: In-memory filesystem, no disk I/O
2. **Isolated**: Tests don't interfere with each other
3. **Deterministic**: Same input always produces same output
4. **Easy Setup**: No complex mocking frameworks needed
5. **Real Behavior**: Tests use actual code paths, not stubs

---

## Learning Resources

### Official Resources

- [A Tour of Go](https://go.dev/tour/) - Interactive tutorial
- [Effective Go](https://go.dev/doc/effective_go) - Best practices
- [Go by Example](https://gobyexample.com/) - Annotated examples
- [Go Standard Library](https://pkg.go.dev/std) - Complete reference

### Books

- "The Go Programming Language" by Donovan & Kernighan
- "Go in Action" by William Kennedy
- "Concurrency in Go" by Katherine Cox-Buday

### Online Courses

- [Go Programming on Udemy](https://www.udemy.com/topic/go-programming-language/)
- [Go on Exercism](https://exercism.org/tracks/go)
- [Go on Codecademy](https://www.codecademy.com/learn/learn-go)

### Community

- [Go Forum](https://forum.golangbridge.org/)
- [Go subreddit](https://reddit.com/r/golang)
- [Gophers Slack](https://gophers.slack.com/)

---

## Next Steps

1. **Read the tfskel code**: Start with `main.go` and follow the execution flow
2. **Modify and experiment**: Change small pieces and see what happens
3. **Write tests**: Practice writing tests for your changes
4. **Build something**: Create a small Go project using these concepts
5. **Join the community**: Ask questions, share learnings

---

**Happy Learning! 🚀**
