# tfskel Architecture

## Table of Contents

1. [Overview](#overview)
2. [Design Principles](#design-principles)
3. [Architecture Layers](#architecture-layers)
4. [Component Details](#component-details)
5. [Data Flow](#data-flow)
6. [Design Patterns](#design-patterns)
7. [Extension Points](#extension-points)
8. [Testing Strategy](#testing-strategy)

---

## Overview

tfskel follows a layered, modular architecture that separates concerns and promotes testability. The application is structured into distinct layers, each with specific responsibilities.

### Architecture Diagram

```bash
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
│                    (cmd/ package)                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │   init   │  │ generate │  │  version │                   │
│  └──────────┘  └──────────┘  └──────────┘                   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Application Layer                          │
│                  (internal/app)                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Generator                               │   │
│  │  - Orchestrates generation workflow                  │   │
│  │  - Validates configuration                           │   │
│  │  - Coordinates components                            │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  Config      │   │  Templates   │   │  File System │
│  (internal/  │   │  (internal/  │   │  (internal/  │
│   config)    │   │   templates) │   │   fs)        │
│              │   │              │   │              │
│ - Load YAML  │   │ - Render     │   │ - Abstraction│
│ - Validate   │   │   templates  │   │ - Os/Memory  │
│ - Defaults   │   │ - Functions  │   │   impl       │
└──────────────┘   └──────────────┘   └──────────────┘
        │                   │                   │
        └───────────────────┴───────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Support Layer                             │
│  ┌──────────────┐   ┌──────────────┐                        │
│  │   Logger     │   │   Utilities  │                        │
│  │ (internal/   │   │ (internal/   │                        │
│  │  logger)     │   │  util)       │                        │
│  │              │   │              │                        │
│  │ - Structured │   │ - Transform  │                        │
│  │   logging    │   │ - Validation │                        │
│  └──────────────┘   └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Design Principles

### 1. Separation of Concerns

Each package has a single, well-defined responsibility:
- `cmd/`: CLI interface and command handling
- `internal/app`: Business logic orchestration
- `internal/config`: Configuration management
- `internal/templates`: Template rendering
- `internal/fs`: File system operations
- `internal/logger`: Logging infrastructure
- `internal/util`: Shared utilities

### 2. Dependency Inversion

High-level modules don't depend on low-level modules. Both depend on abstractions:

```go
// High-level module (Generator) depends on interface
type Generator struct {
    fs fs.FileSystem  // Interface, not concrete type
}

// Low-level module implements interface
type OsFS struct {}
func (o *OsFS) WriteFile(path, content string) error { ... }

type MemoryFS struct {}
func (m *MemoryFS) WriteFile(path, content string) error { ... }
```

### 3. Testability First

Every component is designed to be testable in isolation:
- Interfaces allow for mocking
- Pure functions where possible
- Dependency injection for external dependencies

### 4. Explicit Error Handling

Errors are never ignored. They're wrapped with context and propagated:

```go
if err := fs.WriteFile(path, content); err != nil {
    return fmt.Errorf("failed to write file %s: %w", path, err)
}
```

**Init-time vs Runtime Errors**: Developer errors during initialization (like flag binding mismatches) use panic to fail fast, while runtime errors are handled gracefully:

```go
// Developer error - panic during init
mustBindPFlag := func(key string, flagName string) {
    if err := viper.BindPFlag(key, generateCmd.Flags().Lookup(flagName)); err != nil {
        panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
    }
}

// Runtime error - return with context
if err := generator.Run(env, region, appDir); err != nil {
    return fmt.Errorf("generation failed: %w", err)
}
```

### 5. Configuration Over Code

Behavior is driven by configuration, not hardcoded:
- Project structure from config
- Templates from config
- All options configurable

---

## Architecture Layers

### Layer 1: CLI Layer (cmd/)

**Responsibility**: Handle user interaction and command-line interface.

**Components**:
- `root.go`: Root command and global flags
- `init.go`: Initialize new project command
- `generate.go`: Generate project command
- `version.go`: Version constant
- `drift.go`: Parent drift command
- `drift_version.go`: Version drift detection command
- `drift_plan.go`: Plan analysis command
- `drift_all.go`: Combined drift analysis command
- `errors.go`: Custom exit error handling

**Dependencies**:
- Cobra framework for CLI
- Viper for configuration management
- Application layer (internal/app)
- Drift layer (internal/drift)

**Error Handling**:
- **Init-time errors**: Flag binding failures panic (developer errors)
- **Runtime errors**: Returned with context wrapping
- **Consistent approach**: All commands use same error handling pattern

**Flag Binding Pattern** (in `generate.go`):
```go
// Fail-fast helper for flag binding - panics on developer errors
mustBindPFlag := func(key string, flagName string) {
    if err := viper.BindPFlag(key, generateCmd.Flags().Lookup(flagName)); err != nil {
        panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
    }
}

// Bind flags to viper with strict validation
mustBindPFlag("generate.templates_dir", "templates-dir")
mustBindPFlag("backend.s3.bucket_name", "s3-bucket-name")
mustBindPFlag("generate.github_workflows.create", "create-github-workflows")
```

**Key Functions**:
```go
// root.go
const Version = "0.0.1"  // Updated by release-please
var Commit, Date, BuildTime string  // Build metadata
func Execute() error
func initConfig()

// init.go
func runInit(cmd *cobra.Command, _ []string) error

// generate.go
func runGenerate(cmd *cobra.Command, args []string) error

// drift_version.go
func runDriftVersions(cmd *cobra.Command, _ []string) error

// drift_plan.go
func runDriftPlan(cmd *cobra.Command, _ []string) error

// drift_all.go
func runDriftAll(cmd *cobra.Command, _ []string) error

// errors.go
type ExitError struct {
    Code    int
    Message string
}
func NewExitError(code int, message string) *ExitError
```

### Layer 2: Application Layer (internal/app)

**Responsibility**: Orchestrate the generation workflow.

**Components**:
- `generator.go`: Main orchestrator

**API**:
```go
// NewGenerator creates a configured generator
func NewGenerator(
    cfg *config.Config,
    filesystem fs.FileSystem,
    log *logger.Logger,
) *Generator

// Run executes the complete generation workflow with generation parameters
func (g *Generator) Run(env, region, appDir string) error

// Private methods for workflow steps
func (g *Generator) generateFiles(appPath, env, region, appDir string) error
func (g *Generator) determineOutputPath(tmplPath, appPath string, data *templates.Data) (string, bool)
func (g *Generator) shouldRegenerateFile(filePath string, data map[string]string) (bool, []string, error)
```

**Workflow**:
1. Load configuration from .tfskel.yaml
2. Validate configuration structure:
   - AWS provider exists with account_mapping
   - All account IDs are exactly 12-digit numbers (validateAccountIDs)
   - Backend S3 bucket_name is set and not placeholder
3. Prepare template data for environment:
   - Call GetAccountID(env) to retrieve and validate account ID exists for environment
   - Returns descriptive error with available environments if not found
4. Create directory structure
5. Render templates with validated configuration data (guaranteed valid account ID)
6. Write rendered files to file system
7. Log progress and results

**Error Handling Flow**:
- Configuration validation errors stop execution before any file operations
- GetAccountID errors provide helpful messages showing available environments
- Early validation prevents partial scaffolding with invalid data

### Layer 3: Domain Layer (internal/config, internal/templates, internal/fs)

#### Config Package (internal/config)

**Responsibility**: Load, validate, and provide configuration.

**Sentinel Errors**:
```go
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
)
```

**API**:
```go
// Load reads configuration from viper and command line flags
// Handles deprecation warnings, flag overrides, and defaults
func Load(cmd *cobra.Command, v *viper.Viper) (*Config, error)

// Validate checks configuration correctness
func (c *Config) Validate() error

// validateAccountIDs checks that all AWS account IDs are valid 12-digit numbers
func (c *Config) validateAccountIDs() error

// GetBackendConfig returns backend-specific config
func (c *Config) GetBackendConfig() map[string]string

// GetAccountID retrieves the AWS account ID for a given environment
// Returns error if no mapping exists for that environment
func (c *Config) GetAccountID(env string) (string, error)
```

**Configuration Loading Process**:
1. **Unmarshal**: Viper config unmarshaled into Config struct
2. **Deprecation Check**: Warns about old root-level `templates_dir`
3. **Flag Overrides**: Command-line flags override config file values
4. **Defaults**: Apply sensible defaults for optional fields

**Configuration Validation**:
- `Validate()` performs comprehensive checks:
  - AWS provider configuration exists and account_mapping is not empty
  - All account IDs are exactly 12 numeric digits (via `validateAccountIDs()`)
  - Backend S3 bucket_name is set and not a placeholder value
  - Rejects placeholder values like "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME"
- `GetAccountID(env)` validates that the specific environment has an account mapping
- Returns descriptive errors with available environments when mapping is missing
- Uses sentinel errors for precise error handling

**Validation Implementation**:

The `Validate()` method performs multi-stage validation:

```go
func (c *Config) Validate() error {
    // Check AWS provider exists
    if c.Provider == nil || c.Provider.AWS == nil {
        return ErrAWSProviderRequired
    }

    // Check account mapping exists and not empty
    if len(c.Provider.AWS.AccountMapping) == 0 {
        return ErrAccountMappingRequired
    }

    // Validate all account IDs are 12-digit numbers
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

// validateAccountIDs validates that all account IDs are exactly 12 digits
func (c *Config) validateAccountIDs() error {
    accountIDPattern := regexp.MustCompile(`^\d{12}$`)

    for env, accountID := range c.Provider.AWS.AccountMapping {
        if !accountIDPattern.MatchString(accountID) {
            return fmt.Errorf("%w for environment %q: %q",
                ErrInvalidAccountID, env, accountID)
        }
    }

    return nil
}
```

The `GetAccountID()` method provides environment-specific validation with helpful error messages:

```go
func (c *Config) GetAccountID(env string) (string, error) {
    if c.Provider != nil && c.Provider.AWS != nil &&
        c.Provider.AWS.AccountMapping != nil {
        if id, ok := c.Provider.AWS.AccountMapping[env]; ok {
            return id, nil
        }
        // Show available environments to help user fix it
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

**Data Structures**:
```go
type Config struct {
    TerraformVersion string    `mapstructure:"terraform_version"`
    Provider         *Provider `mapstructure:"provider"`
    Backend          *Backend  `mapstructure:"backend"`
    Generate         *Generate `mapstructure:"generate"`
}

type Provider struct {
    AWS *AWSProvider `mapstructure:"aws"`
}

type AWSProvider struct {
    Version        string            `mapstructure:"version"`
    AccountMapping map[string]string `mapstructure:"account_mapping"`
    DefaultTags    map[string]string `mapstructure:"default_tags"`
    Regions        []string          `mapstructure:"regions"`
}

type Backend struct {
    S3 *S3Backend `mapstructure:"s3"`
}

type S3Backend struct {
    BucketName string `mapstructure:"bucket_name"`
}

// Generate holds generate command specific configuration
type Generate struct {
    GithubWorkflows *GithubWorkflows `mapstructure:"github_workflows"`
    TemplatesDir    string           `mapstructure:"templates_dir"`
}

type GithubWorkflows struct {
    Create       bool   `mapstructure:"create"`
    NameTemplate string `mapstructure:"name_template"`
    AWSRoleName  string `mapstructure:"aws_role_name"`
    AWSRoleArn   string `mapstructure:"aws_role_arn"`
}
```

#### Templates Package (internal/templates)

**Responsibility**: Render templates with configuration data.

**API**:
```go
// NewRenderer creates a template renderer
func NewRenderer() *Renderer

// Render renders a template with data
func (r *Renderer) Render(
    templateName string,
    data interface{},
) (string, error)

// RenderToFile renders template directly to file
func (r *Renderer) RenderToFile(
    templateName string,
    data interface{},
    outputPath string,
    fs fs.FileSystem,
) error
```

**Template Functions**:
```go
// Custom functions available in templates
var funcMap = template.FuncMap{
    "replace":         strings.ReplaceAll,
    "toLower":         strings.ToLower,
    "toUpper":         strings.ToUpper,
    "trimSpace":       strings.TrimSpace,
    "trimPrefix":      strings.TrimPrefix,
    "trimSuffix":      strings.TrimSuffix,
    "hasPrefix":       strings.HasPrefix,
    "hasSuffix":       strings.HasSuffix,
    "contains":        strings.Contains,
    "join":            strings.Join,
    "split":           strings.Split,
    "stripConstraint": stripConstraint,  // Strips version constraint operators like ~>, >=, etc.
}
```

#### File System Package (internal/fs)

**Responsibility**: Abstract file system operations for testability.

**Interface**:
```go
type FileSystem interface {
    // WriteFile writes data to a file
    WriteFile(path string, data []byte, perm os.FileMode) error

    // ReadFile reads the contents of a file
    ReadFile(path string) ([]byte, error)

    // MkdirAll creates a directory path, creating parent directories as needed
    MkdirAll(path string, perm os.FileMode) error

    // FileExists checks if a file exists
    FileExists(path string) bool

    // DirExists checks if a directory exists
    DirExists(path string) bool
}
```

**Implementations**:

1. **OSFileSystem**: Real file system operations
```go
type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
    return &OSFileSystem{}
}

func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
    // Ensure directory exists
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    return os.WriteFile(path, data, perm)
}

func (fs *OSFileSystem) DirExists(path string) bool {
    info, err := os.Stat(path)
    if os.IsNotExist(err) {
        return false
    }
    return info.IsDir()
}
```

2. **MemoryFileSystem**: In-memory file system for testing
```go
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
```

### Layer 4: Support Layer (internal/logger, internal/util)

#### Logger Package (internal/logger)

**Responsibility**: Provide structured logging with color output.

**API**:
```go
// New creates a new logger instance
// verbose flag enables DEBUG level and timestamps on all logs
func New(verbose bool) *Logger

// NewWithWriters creates a logger with custom writers (useful for testing)
func NewWithWriters(verbose bool, out, errOut io.Writer) *Logger

// Logging methods
func (l *Logger) Debug(msg string)
func (l *Logger) Debugf(format string, args ...interface{})
func (l *Logger) Info(msg string)
func (l *Logger) Infof(format string, args ...interface{})
func (l *Logger) Warn(msg string)
func (l *Logger) Warnf(format string, args ...interface{})
func (l *Logger) Success(msg string)
func (l *Logger) Successf(format string, args ...interface{})
func (l *Logger) Error(msg string)
func (l *Logger) Errorf(format string, args ...interface{})
```

**Features**:
- Multiple log levels (DEBUG, INFO, WARN, SUCCESS, ERROR)
- Color-coded terminal output
- Environment variable support (TFSKEL_LOG_LEVEL)
- Separate stdout and stderr writers
- Test-friendly with custom writers

#### Drift Package (internal/drift)

**Responsibility**: Detect and analyze Terraform version drift and plan changes.

**Components**:
- `version_detector.go`: Scans directories for Terraform files and extracts version info
- `version_analyzer.go`: Compares versions against expected configuration
- `version_formatter.go`: Formats version drift results (table, JSON, CSV)
- `plan_parser.go`: Parses Terraform plan JSON files
- `plan_analyzer.go`: Analyzes plan changes and categorizes severity
- `plan_formatter.go`: Formats plan analysis results
- `config.go`: Drift-specific configuration
- `critical_resources.go`: Defines critical AWS resources for severity analysis


**API**:
```go
// Version Detection
func NewDetector(rootPath string) *Detector
func (d *Detector) ScanDirectory() ([]VersionInfo, error)

// Version Analysis
func NewVersionAnalyzer(cfg *config.Config) *VersionAnalyzer
func (a *VersionAnalyzer) Analyze(versionInfo []VersionInfo) *VersionsAnalysis

// Plan Parsing
func ParsePlanFile(filename string) (*TerraformPlan, error)

// Plan Analysis
func NewPlanAnalyzer() *PlanAnalyzer
func NewPlanAnalyzerWithConfig(v *viper.Viper) *PlanAnalyzer
func (a *PlanAnalyzer) Analyze(plan *TerraformPlan) *PlanAnalysis

// Critical Resources
func DefaultCriticalResources() []string
func MergeCriticalResources(defaults, userDefined []string) []string

// Configuration
func LoadDriftConfig(v *viper.Viper) *DriftConfig
```

**Features**:
- HCL parsing for accurate version extraction
- Multiple output formats (table, JSON, CSV)
- Critical resource detection for risk assessment
- Binary plan file detection and helpful error messages
- Configurable critical resources via .tfskel.yaml
- Auto-detection of terminal width for table formatting

#### Utilities Package (internal/util)

**Responsibility**: Provide shared utility functions for region transformations.

**API**:
```go
// TransformRegionName converts AWS region names to shorter format
// Examples: eu-central-1 -> euc1, us-west-2 -> usw2, eu-west-1 -> euw1
func TransformRegionName(region string) string
```

**Features**:
- Converts AWS region names to compact alphanumeric format
- Preserves numbers (availability zones)
- Takes first letter of direction parts (west -> w, north -> n)
- Keeps short parts as-is (eu, us, ap)

---

## Data Flow

### Generate Command Flow

```
User runs: tfskel generate --config tfskel.yaml

1. CLI Layer (cmd/generate.go)
   ├─ Parse flags
   ├─ Create logger
   └─ Load config file
       │
       ▼
2. Config Layer (internal/config)
   ├─ Parse YAML
   ├─ Validate structure
   ├─ Apply defaults
   └─ Return Config object
       │
       ▼
3. Application Layer (internal/app)
   ├─ Create Generator
   ├─ Validate config
   ├─ Create directories
   │   └─> FileSystem: MkdirAll()
   ├─ Render templates
   │   └─> Templates: Render()
   └─ Write files
       └─> FileSystem: WriteFile()
           │
           ▼
4. File System Layer (internal/fs)
   ├─ Create directories on disk
   └─ Write files to disk
       │
       ▼
5. Result
   └─ Generated project structure
```

### Configuration Loading Flow

```
Command Line + .tfskel.yaml
    │
    ▼
config.Load(cmd, viper)
    │
    ├─ 1. Viper Unmarshal (YAML → Struct)
    │   └─ Uses mapstructure tags
    │
    ├─ 2. Check Deprecated Config
    │   ├─ Detect old root-level templates_dir
    │   ├─ Detect extra_template_extensions
    │   └─ Log warnings with migration guidance
    │
    ├─ 3. Apply Flag Overrides
    │   ├─ --templates-dir → Generate.TemplatesDir
    │   ├─ --s3-bucket-name → Backend.S3.BucketName
    │   └─ --create-github-workflows → Generate.GithubWorkflows.Create
    │
    ├─ 4. Set Defaults
    │   ├─ Initialize nil pointers (Provider, Backend, Generate)
    │   ├─ Set default template extensions
    │   └─ Set default GitHub workflow patterns
    │
    └─ 5. Normalize Extensions
        ├─ Remove duplicates via map
        ├─ Always include "tf.tmpl"
        └─ Sort for deterministic order
            │
            ▼
        Config object ready for validation
```

**Key Features**:
- **Backward Compatibility**: Warns about deprecated config structure
- **Flag Priority**: Command-line flags override config file
- **Fail-Fast Init**: Flag binding errors panic (developer errors)
- **Deterministic**: Sorted extensions prevent flaky behavior

### Template Rendering Flow

```
Template Name + Data
    │
    ▼
Renderer.Render()
    │
    ├─ Load embedded template
    ├─ Parse template
    ├─ Add custom functions
    └─ Execute template
        │
        ├─ Access data fields
        ├─ Apply functions
        └─ Iterate collections
            │
            ▼
        Rendered content
```

---

## Design Patterns

### 1. Strategy Pattern (File System)

Different file system implementations (OS vs Memory) for different contexts:

```go
type FileSystem interface {
    WriteFile(path, content string) error
}

// Production
fs := &fs.OsFS{}

// Testing
fs := &fs.MemoryFS{}

// Both work with same interface
generator := app.NewGenerator(config, fs, logger)
```

### 2. Template Method Pattern (Generator)

Generator orchestrates a fixed workflow, with customizable steps:

```go
func (g *Generator) Run() error {
    // Fixed workflow
    if err := g.validateConfig(); err != nil {
        return err
    }

    if err := g.createDirectories(); err != nil {
        return err
    }

    files, err := g.renderTemplates()
    if err != nil {
        return err
    }

    return g.writeFiles(files)
}
```

### 3. Builder Pattern (Configuration)

Configuration is built step by step with validation:

```go
config := &Config{}
config.SetDefaults()
config.Validate()
config.ApplyOverrides(flags)
```

### 4. Dependency Injection

Dependencies are injected, not created internally:

```go
// Bad: Creates dependencies internally
func NewGenerator(cfg *Config) *Generator {
    fs := &fs.OsFS{}  // Hardcoded dependency
    log := logger.New("INFO")
    return &Generator{config: cfg, fs: fs, log: log}
}

// Good: Accepts dependencies
func NewGenerator(
    cfg *Config,
    filesystem fs.FileSystem,  // Injected
    log *logger.Logger,        // Injected
) *Generator {
    return &Generator{
        config: cfg,
        fs:     filesystem,
        log:    log,
    }
}
```

### 5. Facade Pattern (Generator)

Generator provides a simple interface to a complex subsystem:

```go
// Simple interface
generator := app.NewGenerator(config, fs, logger)
err := generator.Run()

// Hides complexity of:
// - Config validation
// - Directory creation
// - Template rendering
// - File writing
// - Error handling
```

---

## Extension Points

### 1. Custom Templates

Users can override default templates by:
1. Creating `templates/` directory
2. Adding `.tmpl` files with same names
3. Templates are loaded with priority: user templates > embedded templates

### 2. Custom Backends

Add new backend support by:
1. Adding backend type to config validation
2. Creating backend-specific template
3. Updating backend config validation

```go
// In config.go
func (b *BackendConfig) Validate() error {
    switch b.Type {
    case "s3":
        return b.validateS3()
    case "azurerm":
        return b.validateAzure()
    case "custom":  // New backend
        return b.validateCustom()
    default:
        return fmt.Errorf("unsupported backend: %s", b.Type)
    }
}
```

### 3. Custom Template Functions

Add new template functions:

```go
// In renderer.go
var templateFuncs = template.FuncMap{
    "snakeCase":  util.ToSnakeCase,
    "customFunc": myCustomFunction,  // New function
}
```

### 4. Custom Workflows

Extend generator workflow:

```go
// In generator.go
func (g *Generator) Run() error {
    // ... existing steps ...

    if g.config.CustomWorkflow {
        if err := g.runCustomWorkflow(); err != nil {
            return err
        }
    }

    return nil
}
```

---

## Testing Strategy

### Unit Tests

Each package has unit tests:

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── templates/
│   ├── renderer.go
│   └── renderer_test.go
└── util/
    ├── transform.go
    └── transform_test.go
```

**Test Pattern**:
```go
func TestToSnakeCase(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"PascalCase", "MyProject", "my_project"},
        {"camelCase", "myProject", "my_project"},
        {"with spaces", "My Project", "my_project"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ToSnakeCase(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

Test component integration using MemoryFS:

```go
func TestGeneratorIntegration(t *testing.T) {
    // Arrange
    cfg := &config.Config{
        Project: config.ProjectConfig{Name: "test"},
    }
    fs := &fs.MemoryFS{}
    log := logger.NewLogger("ERROR")

    generator := app.NewGenerator(cfg, fs, log)

    // Act
    err := generator.Run()

    // Assert
    assert.NoError(t, err)
    assert.True(t, fs.FileExists("main.tf"))
}
```

### End-to-End Tests

Test complete CLI commands:

```go
func TestGenerateCommand(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()

    // Create config file
    configPath := filepath.Join(tmpDir, "tfskel.yaml")
    writeConfig(configPath, testConfig)

    // Run command
    cmd := exec.Command("tfskel", "generate", "--config", configPath)
    output, err := cmd.CombinedOutput()

    // Assert
    assert.NoError(t, err)
    assert.FileExists(t, filepath.Join(tmpDir, "main.tf"))
}
```

### Test Coverage

Maintain minimum 80% test coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Performance Considerations

### 1. Template Caching

Templates are parsed once and cached:

```go
type Renderer struct {
    templates map[string]*template.Template
}

func (r *Renderer) Render(name string, data interface{}) (string, error) {
    // Check cache first
    if tmpl, ok := r.templates[name]; ok {
        return r.execute(tmpl, data)
    }

    // Parse and cache
    tmpl, err := r.parseTemplate(name)
    if err != nil {
        return "", err
    }
    r.templates[name] = tmpl

    return r.execute(tmpl, data)
}
```

### 2. Concurrent File Writing

When writing many files, use goroutines:

```go
func (g *Generator) writeFiles(files map[string]string) error {
    errCh := make(chan error, len(files))
    var wg sync.WaitGroup

    for path, content := range files {
        wg.Add(1)
        go func(p, c string) {
            defer wg.Done()
            if err := g.fs.WriteFile(p, c); err != nil {
                errCh <- err
            }
        }(path, content)
    }

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

### 3. Memory Efficiency

Use streams for large files:

```go
func (r *Renderer) RenderToFile(
    name string,
    data interface{},
    output string,
    fs fs.FileSystem,
) error {
    var buf bytes.Buffer

    if err := r.execute(tmpl, data, &buf); err != nil {
        return err
    }

    return fs.WriteFile(output, buf.String())
}
```

---

## Security Considerations

### 1. Path Validation

Always validate and clean file paths:

```go
func (g *Generator) WriteFile(path, content string) error {
    // Clean path
    cleanPath := filepath.Clean(path)

    // Ensure within project directory
    if !strings.HasPrefix(cleanPath, g.config.BaseDir) {
        return fmt.Errorf("path outside project: %s", path)
    }

    return g.fs.WriteFile(cleanPath, content)
}
```

### 2. Input Sanitization

Sanitize user input before using in templates:

```go
func (c *Config) Validate() error {
    // Validate project name
    if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(c.Project.Name) {
        return fmt.Errorf("invalid project name: %s", c.Project.Name)
    }

    return nil
}
```

### 3. Secret Management

Never log or display sensitive data:

```go
func (g *Generator) Run() error {
    // Mask sensitive backend config
    maskedConfig := g.config.Backend.Mask()
    g.log.Info("Using backend", "type", g.config.Backend.Type, "config", maskedConfig)
}
```

---

## Future Enhancements

### 1. Plugin System

Allow users to create custom plugins:

```go
type Plugin interface {
    Name() string
    Execute(ctx context.Context, cfg *Config) error
}

func (g *Generator) RunPlugins() error {
    for _, plugin := range g.plugins {
        if err := plugin.Execute(g.ctx, g.config); err != nil {
            return err
        }
    }
    return nil
}
```

### 2. Remote Templates

Support fetching templates from URLs:

```go
type TemplateSource interface {
    Fetch() ([]byte, error)
}

type HTTPSource struct {
    URL string
}

type GitSource struct {
    Repo   string
    Branch string
}
```

### 3. Interactive Mode

Add TUI for configuration:

```go
func (c *InitCommand) RunInteractive() error {
    // Use bubbletea or similar for TUI
    model := NewModel()
    program := tea.NewProgram(model)
    return program.Start()
}
```

---

## Conclusion

tfskel's architecture prioritizes:
- **Modularity**: Clear separation of concerns
- **Testability**: Interfaces and dependency injection
- **Extensibility**: Multiple extension points
- **Maintainability**: Clear patterns and documentation

This design allows the project to grow while maintaining code quality and developer productivity.
