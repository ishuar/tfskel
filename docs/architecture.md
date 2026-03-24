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
│  ┌──────────┐  ┌──────────┐   ┌──────────┐                  │
│  │   init   │  │ scaffold │   │  drift   │                  │
│  └──────────┘  └──────────┘   └────┬─────┘                  │
│                                    │                        │
│                    ┌───────────────┼───────────────┐        │
│                    │                               │        │
│             ┌──────▼──────┐                 ┌───── ▼────┐   │
│             │   version   │                 │    plan   │   │
│             └─────────────┘                 └───────────┘   │
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
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐     │
│  │   Logger     │   │   Drift      │   │   Utilities  │     │
│  │ (internal/   │   │ (internal/   │   │ (internal/   │     │
│  │  logger)     │   │   drift)     │   │   util)      │     │
│  │              │   │              │   │              │     │
│  │ - Structured │   │ - Version    │   │ - Transform  │     │
│  │   logging    │   │   detection  │   │ - Validation │     │
│  │              │   │ - Plan       │   │              │     │
│  │              │   │   analysis   │   │              │     │
│  └──────────────┘   └──────────────┘   └──────────────┘     │
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
    if err := viper.BindPFlag(key, scaffoldCmd.Flags().Lookup(flagName)); err != nil {
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
- `scaffold.go`: Scaffold Terraform project structure (generates env/region/app directories)
- `version.go`: Version constant
- `drift.go`: Parent drift command
- `drift_version.go`: Version drift detection command
- `drift_plan.go`: Plan analysis command
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

**Flag Binding Pattern** (in `scaffold.go`):
```go
// Fail-fast helper for flag binding - panics on developer errors
mustBindPFlag := func(key string, flagName string) {
    if err := viper.BindPFlag(key, scaffoldCmd.Flags().Lookup(flagName)); err != nil {
        panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
    }
}

// Bind flags to viper with strict validation
mustBindPFlag("templates.dir", "templates-dir")
mustBindPFlag("backend.s3.bucket_name", "s3-bucket-name")
mustBindPFlag("workflows.create", "workflows")
```
```

**Sentinel Errors**:
```go
// scaffold.go
// Input validation now uses util.TrimAndValidateInput and returns wrapped errors.
// No exported sentinel errors are defined for this command.

// init.go
var (
    ErrUnsupportedDataType   = errors.New("unsupported data type for template rendering")
    ErrMissingAccountMapping = errors.New("provider.aws.account_mapping is missing or empty")
)

// drift_version.go
var (
    ErrPathDoesNotExist = errors.New("path does not exist")
    ErrPathNotDirectory = errors.New("path is not a directory")
)

// drift_plan.go
var (
    ErrPlanFileRequired = errors.New("plan file is required")
    ErrPlanFileNotFound = errors.New("plan file not found")
)
```

**Key Functions**:
```go
// version.go
const Version = "0.3.0"  // x-release-please-version (Updated by release-please)

// root.go
var Commit, Date, BuildTime string  // Build metadata
func Execute() error
func initConfig()

// init.go
func runInit(cmd *cobra.Command, _ []string) error

// scaffold.go
func runScaffold(cmd *cobra.Command, args []string) error

// drift_version.go
func runDriftVersions(cmd *cobra.Command, _ []string) error

// drift_plan.go
func runDriftPlan(cmd *cobra.Command, _ []string) error

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
1. Load configuration from .tfskel.yaml (via config.Load)
2. Validate configuration structure (via cfg.Validate):
   - AWS provider exists with account_mapping
   - All account IDs are exactly 12-digit numbers (validateAccountIDs)
   - Backend S3 bucket_name is set and not placeholder
3. Create Generator with validated config, filesystem, and logger
4. Prepare template data for environment:
   - Call GetAccountID(env) to retrieve and validate account ID exists for environment
   - Returns descriptive error with available environments if not found
5. Create directory structure (env/region/appDir hierarchy)
6. Render templates with validated configuration data (guaranteed valid account ID)
7. Write rendered files to file system
8. Generate optional GitHub workflow files if requested
9. Log progress and results

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
    ErrInvalidAccountID = errors.New("AWS account ID must be a 12-digit number")

    // ErrInvalidBucketName indicates the S3 bucket name is not properly configured
    ErrInvalidBucketName = errors.New("backend.s3.bucket_name is invalid")
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

    // Validate backend configuration exists
    if c.Backend == nil || c.Backend.S3 == nil {
        return fmt.Errorf("%w: must not be empty", ErrInvalidBucketName)
    }

    // Check bucket name is not empty or whitespace
    bucketName := strings.TrimSpace(c.Backend.S3.BucketName)
    if bucketName == "" {
        return fmt.Errorf("%w: must not be empty", ErrInvalidBucketName)
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
            return fmt.Errorf("%w: Update the account mapping %q: %q",
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
    Templates        *Templates `mapstructure:"templates"`
    Workflows        *Workflows `mapstructure:"workflows"`
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

// Templates holds scaffold command specific template configuration
type Templates struct {
    Dir string `mapstructure:"dir"`
}

// Workflows holds GitHub Actions workflow configuration
type Workflows struct {
    Create       bool   `mapstructure:"create"`
    Name string `mapstructure:"name"`
    AWSRoleName  string `mapstructure:"aws_role_name"`
    AWSRoleArn   string `mapstructure:"aws_role_arn"`
}
```

#### Templates Package (internal/templates)

**Responsibility**: Render templates with configuration data.

**API**:
```go
// NewRenderer creates a template renderer
func NewRenderer() (*Renderer, error)

// NewRendererWithCustomTemplates creates a renderer with custom template directory
func NewRendererWithCustomTemplates(customTemplateDir string) (*Renderer, error)

// Render renders a template with data
func (r *Renderer) Render(templateName string, data *Data) (string, error)

// RenderConfigValue renders a config string that may contain Go template syntax
// Plain strings without "{{" are returned unchanged
func (r *Renderer) RenderConfigValue(value, name string, data *Data) (string, error)

// GetTemplateNames returns all loaded template names
func (r *Renderer) GetTemplateNames() []string

// GetTemplateSource returns the source path of a template
func (r *Renderer) GetTemplateSource(templateName string) string
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

// stripConstraint removes version constraint operators and returns just the version number
// Example: "~> 1.14.3" -> "1.14.3"
func stripConstraint(version string) string
```

**Template Data Structure**:
```go
// Data holds all the data needed for template rendering
type Data struct {
    Env                string
    Region             string
    AppDir             string
    AccountID          string
    ShortRegion        string            // Compact region name (e.g., euc1)
    S3BucketName       string
    TerraformVersion   string
    AWSProviderVersion string
    DefaultTags        map[string]string
    DefaultTagsJSON    string            // JSON string for metadata comments
    AWSRoleArn         string            // AWS role ARN for terraform workflows
    WorkflowFileName   string            // Generated workflow filename for self-reference
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
- `version_models.go`: Data models for version analysis
- `plan_parser.go`: Parses Terraform plan JSON files
- `plan_analyzer.go`: Analyzes plan changes and categorizes severity
- `plan_formatter.go`: Formats plan analysis results
- `plan_models.go`: Data models for plan analysis
- `config.go`: Drift-specific configuration
- `critical_resources.go`: Defines critical AWS resources for severity analysis
- `styles.go`: Terminal styling for formatted output

**API**:
```go
// Version Detection
func NewDetector(rootPath string) *Detector
func (d *Detector) ScanDirectory() ([]VersionInfo, error)

// Version Analysis
func NewVersionAnalyzer(cfg *config.Config) *VersionAnalyzer
func (a *VersionAnalyzer) Analyze(versionInfo []VersionInfo) *VersionsAnalysis

// Version Formatting
func FormatVersionsTable(analysis *VersionsAnalysis, noColor bool) (string, error)
func FormatVersionsJSON(analysis *VersionsAnalysis) (string, error)
func FormatVersionsCSV(analysis *VersionsAnalysis) (string, error)

// Plan Parsing
func ParsePlanFile(filename string) (*TerraformPlan, error)

// Plan Analysis
func NewPlanAnalyzer() *PlanAnalyzer
func NewPlanAnalyzerWithConfig(v *viper.Viper) *PlanAnalyzer
func (a *PlanAnalyzer) Analyze(plan *TerraformPlan) *PlanAnalysis

// Plan Formatting
func FormatPlanAnalysis(analysis *PlanAnalysis, format OutputFormat, noColor bool) (string, error)

// Critical Resources
func DefaultCriticalResources() []string
func MergeCriticalResources(defaults, userDefined []string) []string

// Configuration
func LoadDriftConfig(v *viper.Viper) *DriftConfig
```

**Configuration Structure**:
```go
type DriftConfig struct {
    CriticalResources []string `mapstructure:"critical_resources"`
    TopResourcesCount int      `mapstructure:"top_resources_count"`  // Default: 10
}
```

**Output Format Constants**:
```go
type OutputFormat string

const (
    FormatTable OutputFormat = "table"
    FormatJSON  OutputFormat = "json"
    FormatCSV   OutputFormat = "csv"

    // Terminal width constants
    defaultTerminalWidth = 120
    minDriftTableWidth   = 113
    minPlanTableWidth    = 80
    maxPlanTableWidth    = 150

    // Summary display constants
    defaultTopResourcesCount  = 10  // Default number of resources in top-N summaries
    severityTopResourcesCount = 0   // Show all severity items (no limit)
)
```

**Features**:
- HCL parsing for accurate version extraction
- Multiple output formats (table, JSON, CSV)
- Critical resource detection for risk assessment
- Binary plan file detection and helpful error messages
- Configurable critical resources via .tfskel.yaml
- Configurable top-N resource count for summary displays
- Auto-detection of terminal width for table formatting
- Color-coded severity indicators (can be disabled with --no-color)
- Combined analysis support (version + plan in single command)

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

### Scaffold Command Flow

```
User runs: tfskel scaffold myapp --env dev --region us-east-1

1. CLI Layer (cmd/scaffold.go)
   ├─ Parse flags (env, region, templates-dir, etc.)
   ├─ Extract app directory from args
   └─ Validate required flags (env, region)
       │
       ▼
2. Config Layer (internal/config)
   ├─ Load .tfskel.yaml via config.Load()
   ├─ Apply flag overrides
   ├─ Set defaults
   └─ Validate configuration (cfg.Validate())
       │
       ▼
3. Application Layer (internal/app)
   ├─ Create Generator(cfg, fs, logger)
   ├─ Prepare template data (env, region, appDir, accountID)
   ├─ Validate account mapping exists for environment
   └─ Call generator.Run(env, region, appDir)
       │
       ▼
4. Template Layer (internal/templates)
   ├─ Load templates (embedded or custom)
   ├─ Render with template data
   └─ Process metadata (output paths, categories)
       │
       ▼
5. File System Layer (internal/fs)
   ├─ Create directory structure (envs/env/region/appDir)
   ├─ Write Terraform files (backend.tf, versions.tf)
   └─ Write optional GitHub workflows
       │
       ▼
6. Result
   └─ Generated project structure:
       envs/dev/us-east-1/myapp/
       ├── backend.tf
       ├── versions.tf
       └── ...
```

### Init Command Flow

```
User runs: tfskel init

1. CLI Layer (cmd/init.go)
   ├─ Parse flags (dir, config)
   └─ Determine target directory
       │
       ▼
2. Configuration Creation
   ├─ Check if .tfskel.yaml exists
   ├─ Load or create default configuration
   └─ Render .tfskel.yaml template
       │
       ▼
3. Template Rendering
   ├─ Render root-level templates (trivy.yaml)
   ├─ Render GitHub workflow templates
   └─ Write .tfskel.yaml config file
       │
       ▼
4. Result
   └─ Initialized project:
       ├── .tfskel.yaml
       ├── trivy.yaml
       └── .github/workflows/
```

### Drift Command Flows

#### Drift Version Flow
```
User runs: tfskel diff config --dir ./envs

1. CLI Layer (cmd/drift_version.go)
   ├─ Parse flags (path, format, no-color)
   └─ Validate path exists and is directory
       │
       ▼
2. Config Layer (internal/config)
   └─ Load expected versions from .tfskel.yaml
       │
       ▼
3. Drift Detection (internal/diff)
   ├─ Scan directory tree recursively
   ├─ Parse .tf files with HCL parser
   ├─ Extract version constraints
   └─ Compare against expected versions
       │
       ▼
4. Analysis and Formatting
   ├─ Categorize drift (matches, mismatches, missing)
   └─ Format output (table/JSON/CSV)
       │
       ▼
5. Result
   └─ Version drift report with exit code
```

#### Drift Plan Flow
```
User runs: tfskel review plan --json-file tfplan.json

1. CLI Layer (cmd/drift_plan.go)
   ├─ Parse flags (plan-file, format, no-color)
   └─ Validate plan file exists
       │
       ▼
2. Plan Parsing (internal/drift)
   ├─ Read JSON plan file
   ├─ Detect binary plans (error with helpful message)
   └─ Parse resource changes
       │
       ▼
3. Plan Analysis (internal/drift)
   ├─ Categorize changes (add, change, delete, replace)
   ├─ Calculate severity based on critical resources
   └─ Generate summary statistics
       │
       ▼
4. Formatting and Output
   ├─ Format based on output type
   └─ Display top-N resources by change type
       │
       ▼
5. Result
   └─ Plan analysis report with severity and exit code
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
    WriteFile(path string, data []byte, perm os.FileMode) error
    ReadFile(path string) ([]byte, error)
    MkdirAll(path string, perm os.FileMode) error
    FileExists(path string) bool
    DirExists(path string) bool
}

// Production
fs := fs.NewOSFileSystem()

// Testing
fs := fs.NewMemoryFileSystem()

// Both work with same interface
generator := app.NewGenerator(config, fs, logger)
```

### 2. Template Method Pattern (Generator)

Generator orchestrates a fixed workflow, with customizable steps:

```go
func (g *Generator) Run(env, region, appDir string) error {
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
func (g *Generator) Run(env, region, appDir string) error {
    // ... existing steps ...

    if g.config.CustomWorkflow {
        if err := g.runCustomWorkflow(env, region, appDir); err != nil {
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
func TestAddCommand(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()

    // Create config file
    configPath := filepath.Join(tmpDir, ".tfskel.yaml")
    writeConfig(configPath, testConfig)

    // Run command
    cmd := exec.Command("tfskel", "add", "myapp",
        "--env", "dev",
        "--region", "us-east-1",
        "--config", configPath)
    output, err := cmd.CombinedOutput()

    // Assert
    assert.NoError(t, err)
    assert.DirExists(t, filepath.Join(tmpDir, "envs/dev/us-east-1/myapp"))
    assert.FileExists(t, filepath.Join(tmpDir, "envs/dev/us-east-1/myapp/backend.tf"))
    assert.FileExists(t, filepath.Join(tmpDir, "envs/dev/us-east-1/myapp/versions.tf"))
}
```
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
func (g *Generator) Run(env, region, appDir string) error {
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
