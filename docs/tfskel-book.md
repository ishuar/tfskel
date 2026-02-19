# tfskel - The Complete Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Core Concepts](#core-concepts)
4. [Configuration](#configuration)
5. [Commands](#commands)
6. [Templates](#templates)
7. [Directory Structure](#directory-structure)
8. [Advanced Usage](#advanced-usage)

---

## Introduction

### What is tfskel?

**tfskel** (Terraform Skeleton) is a CLI tool that scaffolds Terraform monorepos with an **opinionated**, **scalable** and **consistent** way by using environment-based directory structure across multiple regions. No wrappers, no complexity, just vanilla Terraform with consistent backend configs, version **drift detection**, **terraform plan analysis**, and sensible defaults. Spend less time on project setup and more time writing infrastructure code.


### Why tfskel?

When starting a new Terraform project or adding a new application/environment, you typically need to:

- ✅ Create consistent directory structures across environments
- ✅ Set up backend configuration for state management
- ✅ Configure provider versions and constraints
- ✅ Create Terraform variable and output files
- ✅ Set up environment-specific configurations
- ✅ Configure AWS account mappings per environment
- ✅ Set up region-specific resources
- ✅ Configure linting and security scanning
- ✅ Optionally set up CI/CD workflows

**tfskel automates all of this**, ensuring consistency across all your Terraform projects and team members.

### Key Features

- 🚀 **Environment-Based Structure**: Organizes projects by environment (dev, stg, prd) and region
- 🌍 **Multi-Region Support**: Handles multiple AWS regions with proper naming conventions
- 📝 **Smart File Generation**: Only creates new files, preserves existing ones
- 🔄 **Intelligent Updates**: Detects configuration changes and updates only what's needed
- 🎨 **Custom Templates**: Override default templates with your own
- ⚙️ **YAML Configuration**: Flexible configuration with sensible defaults
- 🏷️ **Metadata Tracking**: Embeds metadata in files for intelligent updates such as `default_tags`
- 🔧 **Backend Configuration**: Pre-configured S3 backend with state locking
- 🔍 **Drift Detection**: Detect Terraform and provider version inconsistencies across repos
- 📦 **Zero Runtime Dependencies**: Single binary with embedded templates

### Architecture Highlights

tfskel is designed with clean architecture principles:

- **Interface-based design** for testability
- **Dependency injection** for flexibility
- **Embedded templates** for zero-dependency distribution
- **In-memory filesystem** for fast, isolated tests
- **Structured logging** with color-coded output
- **Idempotent operations** for safe re-runs

---

## Getting Started

### Installation

#### Using Go Install

```bash
go install github.com/ishuar/tfskel@latest
```

#### From Source

```bash
git clone https://github.com/ishuar/tfskel.git
cd tfskel
go build -o tfskel
sudo mv tfskel /usr/local/bin/
```

#### Pre-built Binaries

Download from the [releases page](https://github.com/ishuar/tfskel/releases).

### Quick Start

1. **Initialize a new project**:

```bash
tfskel init
```

2. **Generate application structure**:

```bash
tfskel generate myapp --env dev --region us-east-1
```

3. **Check for version drift**:

```bash
tfskel drift version
```

---

## Core Concepts

### Components Overview

tfskel is built around several key components:

#### 1. Configuration System

The configuration system (`internal/config`) loads and validates YAML configuration files that define:
- Project metadata (name, description)
- Terraform version constraints
- AWS provider configuration with version and regions
- Environment-specific AWS account ID mappings
- S3 backend configuration
- Default tags for AWS resources
- Custom template directory location

#### 2. Template Renderer

The template renderer (`internal/templates`) uses Go's `text/template` package to:
- Parse embedded template files
- Execute templates with configuration data
- Support custom template directories
- Apply custom functions (stripConstraint, string manipulation)
- Validate template syntax

#### 3. File System Abstraction

The file system abstraction (`internal/fs`) provides:
- A `FileSystem` interface for all I/O operations
- `OSFileSystem` implementation for real file system operations
- `MemoryFileSystem` implementation for testing
- This abstraction makes the entire codebase testable without touching disk

#### 4. Generator

The generator (`internal/app`) orchestrates the entire generation process:
1. Validates configuration
2. Creates directory structure
3. Renders templates (embedded and custom)
4. Writes files to disk with metadata
5. Detects and handles configuration changes
6. Reports progress and errors

#### 5. Drift Detection

The drift detection system (`internal/drift`) provides:
- HCL parsing of Terraform files
- Version extraction from terraform and required_providers blocks
- Comparison against .tfskel.yaml configuration
- Multi-format output (table, JSON, CSV)
- Comprehensive reporting with drift categorization

#### 6. Logger

The logger (`internal/logger`) provides structured logging with:
- Multiple log levels (DEBUG, INFO, WARN, SUCCESS, ERROR, FATAL)
- Color-coded console output
- Contextual information
- Test-friendly silent mode

#### 7. Utilities

Utility functions (`internal/util`) provide:
- Region name transformations (eu-central-1 → euc1)
- String transformations
- Version constraint parsing
---

## Configuration

### Configuration File Format

`tfskel` uses YAML configuration files with the following structure:

> [!Tip]
> Complete configuration is available in [.tfskel.example.yaml](../.tfskel.example.yaml)

### Configuration Sections Explained

#### Terraform Version

- `terraform_version`: Version constraint for Terraform (default: ~> 1.13)
- Supports standard Terraform version syntax (~>, >=, <=, etc.)

#### Backend Section

Configures S3 backend for Terraform state:
- `backend.s3.bucket_name`: S3 bucket name for state storage
- Supports template variables like {{.Env}}, {{.Region}}, {{.AppDir}}


#### Provider Section

Defines AWS provider configuration:
- `provider.aws.version`: AWS provider version constraint (default: ~> 6.0)
- `provider.aws.regions`: List of AWS regions for the project
- `provider.aws.account_mapping`: Maps environment names to AWS account IDs
- `provider.aws.default_tags`: Default tags applied to all AWS resources. Tag keys are automatically normalized to lowercase (but NOT converted to snake_case).
    ```yaml
    provider:
      aws:
        default_tags:
          Team: platform          # Becomes: team (lowercase)
          ManagedBy: terraform    # Becomes: managedby (lowercase, no underscore added)
          Cost_Center: eng        # Becomes: cost_center (lowercase, keeps existing underscore)
    ```
#### Custom Templates

- `generate.templates_dir`: Path to custom template directory
- All files ending with `.tmpl` extension are processed as Go templates
- Custom templates override embedded defaults
- Useful for adding main.tf, variables.tf, outputs.tf, etc.

#### GitHub Workflows Generation

Automate creation of GitHub Actions workflows for Terraform CI/CD:


**Configuration Fields**:
- `generate.github_workflows.create`: Enable/disable workflow generation (default: false)
- `generate.github_workflows.name_template`: Custom workflow filename pattern (optional)
  - Available variables: `{{.AppDir}}`, `{{.Env}}`, `{{.Region}}`, `{{.ShortRegion}}`
  - Workflow type (`-lint`, `-terraform`) is automatically appended
  - Default pattern: `{{.AppDir}}-{{.Env}}-{{.ShortRegion}}`
- `generate.github_workflows.aws_role_name`: IAM role name for AWS authentication (optional)
  - Automatically constructs ARN: `arn:aws:iam::<account-id>:role/<role-name>`
- `generate.github_workflows.aws_role_arn`: Explicit IAM role ARN (optional, takes priority)

**Priority Order for AWS Role**:
1. `aws_role_arn` (if specified) - Explicit ARN
2. `aws_role_name` (if specified) - Constructs ARN using account ID from environment mapping
3. Default placeholder - `arn:aws:iam::<account-id>:role/REPLACE_WITH_ROLE_TO_ASSUME`

**Generated Workflows**:
1. `<name>-lint.yaml` - Terraform linting and validation workflow
2. `<name>-terraform.yaml` - Terraform plan and apply workflow
3. `reusable-lint.yaml` - Reusable linting workflow (called by lint workflows)
4. `reusable-terraform-plan-apply.yaml` - Reusable Terraform workflow (called by terraform workflows)

**Workflow Features**:
- Triggered on pull requests and pushes to main branch
- Path-based filtering (only triggers for changes in specific app directory)
- Self-referencing trigger paths (automatically updates when renamed)
- AWS OIDC authentication with configurable IAM roles
- Manual workflow dispatch with input parameters
- Terraform docs validation for modules
- TFLint integration with caching
- Plan artifacts and PR comments

#### Drift Detection Configuration

Configure drift detection behavior for version and plan analysis:

**Configuration Fields**:
- `critical_resources`: Additional AWS resource types to mark as critical (extends defaults)
  - Default critical resources include databases (RDS, DynamoDB), S3 buckets, VPCs, security groups, IAM roles, KMS keys, WAF rules, etc.
  - User-defined resources are merged with defaults without duplicates
  - Critical resource changes are marked with "Critical" severity in plan analysis
- `top_n_count`: Maximum number of items to display in drift summaries (default: 10)
  - Applies to resource type groupings, module groupings, and action counts
  - Set to 0 to show all items without limit

---

## Commands

### `tfskel init`

Initialize a new tfskel project structure with configuration files.

**Usage**:
```bash
tfskel init [flags]
```

**Flags**:
- `--dir, -d`: Output directory (default: current directory)
- `--config, -c`: Path to config file (default: .tfskel.yaml in current directory)
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Initialize in current directory (uses .tfskel.yaml if present)
tfskel init

# Initialize in specific directory
tfskel init --dir /path/to/project

# Initialize with custom config file
tfskel init --config /path/to/config.yaml
```

**What it does**:
1. Reads existing .tfskel.yaml configuration if present (or uses defaults)
2. Creates root-level configuration files:
   - `.gitignore` - Terraform-specific ignore patterns
   - `.pre-commit-config.yaml` - Pre-commit hooks configuration
   - `.tflint.hcl` - TFLint configuration
   - `trivy.yaml` - Trivy security scanner configuration
   - `.tfskel.yaml` - Default tfskel configuration (if not exists)
3. Creates environment directories based on account_mapping in config
4. Creates region subdirectories for each environment
5. Creates `.terraform-version` files for each environment

### `tfskel generate`

Generate Terraform project structure for a specific application.

**Usage**:
```bash
tfskel generate <app-dir> [flags]
```

**Arguments**:
- `app-dir`: Name of the application directory to create (required)

**Flags**:
- `--env, -e`: Target environment (required) - e.g., dev, stg, prd
- `--region, -r`: AWS region (required) - e.g., us-east-1, eu-central-1
- `--config, -c`: Path to config file (default: .tfskel.yaml in current directory)
- `--templates-dir`: Directory containing custom template files
- `--s3-bucket-name`: Override S3 bucket name for Terraform state
- `--extra-template-extensions`: Additional template file extensions to process
- `--create-github-workflows`: Enable GitHub Actions workflow generation
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Generate structure for an app in dev environment
tfskel generate myapp --env dev --region us-east-1

# Generate with GitHub Actions workflows
tfskel generate myapp --env dev --region us-east-1 --create-github-workflows

# Generate with custom configuration file
tfskel generate myapp --config ./my-config.yaml --env dev --region us-east-1

# Generate with custom templates
tfskel generate myapp --env stg --region eu-central-1 --templates-dir ./templates

# Override S3 bucket name
tfskel generate myapp --env prd --region us-west-2 --s3-bucket-name my-custom-bucket

# Generate with workflows using config file settings
# (workflows enabled in .tfskel.yaml with custom naming)
tfskel generate api --env prd --region eu-central-1
```

**What it does**:
1. Loads configuration from .tfskel.yaml
2. Validates required configuration (account_mapping for environment)
3. Creates directory structure: `envs/<env>/<region>/<app-dir>`
4. Renders embedded templates:
   - `backend.tf` - S3 backend with metadata
   - `versions.tf` - Terraform and provider versions with metadata
5. Generates GitHub Actions workflows if enabled:
   - `.github/workflows/<name>-lint.yaml` - Linting workflow
   - `.github/workflows/<name>-terraform.yaml` - Terraform workflow
   - `.github/workflows/reusable-lint.yaml` - Reusable lint workflow
   - `.github/workflows/reusable-terraform-plan-apply.yaml` - Reusable Terraform workflow
6. Renders custom templates if `--templates-dir` is provided or configured in `.tfskel.yaml`
7. Embeds metadata in generated files for change detection
7. Only creates new files, preserves existing ones
8. Updates files if configuration metadata has changed

### `tfskel drift`

Parent command for drift detection capabilities. Use subcommands for specific analyses.

**Subcommands**:
- `version` - Detect Terraform and provider version drift
- `plan` - Analyze Terraform plan JSON for resource changes
- `all` - Run both version drift and plan analysis

---

### `tfskel drift version`

Detect Terraform and provider version drift across your repository.

**Usage**:
```bash
tfskel drift version [flags]
```

**Flags**:
- `--path, -p`: Path to scan for Terraform files (default: current directory)
- `--format, -f`: Output format: table, json, csv (default: table)
- `--no-color`: Disable colored output
- `--config`: Path to config file for expected versions (default: .tfskel.yaml)
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Check version drift in current directory
tfskel drift version

# Check specific subdirectory
tfskel drift version --path ./envs

# Output as JSON for CI/CD
tfskel drift version --format json > drift-report.json

# Output as CSV
tfskel drift version --format csv --no-color > drift.csv
```

**What it does**:
1. Recursively scans directory for `.tf` files
2. Parses HCL to extract terraform and provider version constraints
3. Compares against expected versions in .tfskel.yaml
4. Categorizes drift:
   - **In-Sync**: Version matches configuration
   - **Minor Drift**: Patch or minor version difference
   - **Major Drift**: Major version mismatch
   - **Missing**: Version not specified in file
   - **Not Managed**: Provider not in .tfskel.yaml
5. Generates comprehensive report with:
   - Per-file drift details
   - Aggregated version statistics
   - Drift severity summary
6. Outputs in requested format (table/json/csv)

**Drift Detection Features**:
- HCL parsing for accurate version extraction
- Automatic hidden directory filtering (skips .git, .terraform, etc.)
- Intelligent version comparison
- Terminal-aware table formatting
- Color-coded status indicators
- Parse error handling (continues on errors)

---

### `tfskel drift plan`

Analyze Terraform plan JSON to identify resource changes, impact severity, and potential risks.

**Usage**:
```bash
tfskel drift plan [flags]
```

**Flags**:
- `--plan-file`: Path to Terraform plan JSON file (required)
- `--format, -f`: Output format: table, json, csv (default: table)
- `--top-n, -n`: Show top N highest-impact resources (default: 10, use 0 for all)
- `--no-color`: Disable colored output
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Generate and analyze a plan
terraform plan -out=plan.bin
terraform show -json plan.bin > plan.json
tfskel drift plan --plan-file plan.json

# Show top 5 highest-impact changes
tfskel drift plan --plan-file plan.json --top-n 5

# Export as JSON for automation
tfskel drift plan --plan-file plan.json --format json

# Export as CSV for reporting
tfskel drift plan --plan-file plan.json --format csv > changes.csv
```

**What it does**:
1. Parses Terraform plan JSON file
2. Analyzes resource changes (create, update, delete, replace)
3. Calculates impact severity based on:
   - Action type (delete/replace = high, create/update = medium)
   - Number of attributes changed
   - Whether resource must be replaced
4. Groups resources by module and action
5. Generates comprehensive report with:
   - Overall change summary
   - Resource-level details with before/after values
   - Impact severity rankings
   - Module-based groupings
6. Outputs in requested format (table/json/csv)

**Plan Analysis Features**:
- Supports Terraform JSON plan format (version 1.2+)
- Adaptive table sizing based on terminal width
- Intelligent attribute change detection
- Severity-based prioritization
- Module-aware resource grouping
- Handles complex nested attribute changes

---

### `tfskel drift all`

Run both version drift detection and plan analysis in a single command.

**Usage**:
```bash
tfskel drift all [flags]
```

**Flags**:
- `--plan-file`: Path to Terraform plan JSON file (required)
- `--path, -p`: Path to scan for Terraform files (default: current directory)
- `--format, -f`: Output format: table, json, csv (default: table)
- `--top-n, -n`: Show top N highest-impact resources (default: 10)
- `--skip-version`: Skip version drift detection
- `--skip-plan`: Skip plan analysis
- `--no-color`: Disable colored output
- `--config`: Path to config file (default: .tfskel.yaml)
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Run complete drift analysis
terraform plan -out=plan.bin && terraform show -json plan.bin > plan.json
tfskel drift all --plan-file plan.json

# Skip version check, only analyze plan
tfskel drift all --plan-file plan.json --skip-version

# Focus on specific directory for version drift
tfskel drift all --plan-file plan.json --path ./envs/prd

# Export combined report as JSON
tfskel drift all --plan-file plan.json --format json > full-report.json
```

**What it does**:
1. Runs version drift detection (unless `--skip-version` specified)
2. Runs plan analysis (unless `--skip-plan` specified)
3. Provides comprehensive drift visibility in one command
4. Outputs both analyses in the same format
5. Useful for CI/CD pipelines and pre-deployment checks

**Use Cases**:
- **Pre-deployment validation**: Check both version consistency and planned changes
- **CI/CD integration**: Single command for comprehensive drift reporting
- **Compliance audits**: Verify version standards and change impact together
- **Large repository monitoring**: Detect inconsistencies across monorepos before deployment

### `tfskel --version`

Display version information.

**Usage**:
```bash
tfskel --version
```

**Output**:
```bash
tfskel version 0.0.1
```
---

## Templates

### Template System

tfskel uses Go's `text/template` package for all templates. Templates are embedded in the binary using `go:embed` and are rendered with configuration data.

tfskel includes two embedded templates by default and supports custom templates for additional files.

### Embedded Templates

#### 1. `backend.tf.tmpl`

Generates S3 backend configuration with metadata tracking:

- [`backend.tf.tmpl`](../internal/templates/files/tf/backend.tf.tmpl)

**Template Variables**:
- `S3BucketName`: S3 bucket name from configuration
- `AppDir`: Application directory name
- `Env`: Environment (dev, stg, prd)
- `Region`: AWS region
- `AccountID`: AWS account ID for environment

**Metadata**: Embedded JSON metadata allows tfskel to detect configuration changes

#### 2. `versions.tf.tmpl`

Generates Terraform and provider version constraints with AWS provider configuration:

- [`versions.tf.tmpl`](../internal/templates/files/tf/versions.tf.tmpl)

**Template Variables**:
- `TerraformVersion`: Terraform version constraint
- `AWSProviderVersion`: AWS provider version constraint
- `DefaultTags`: Map of default tags
- `Env`: Environment name
- `AppDir`: Application directory name

**Smart Features**:
- Region is dynamically determined from directory path
- Metadata tracks version changes for automatic updates
- Tags are automatically applied to all AWS resources

#### 3. GitHub Actions Workflow Templates

**Generated when `--create-github-workflows` flag is used or `generate.github_workflows.create: true` in config**

##### Lint Workflow (`lint.yaml.tmpl`)

Generates a workflow for Terraform linting and validation:

- [`lint.yaml.tmpl`](../internal/templates/files/github/lint.yaml.tmpl)

**Template Variables**:
- `Env`: Environment name
- `Region`: AWS region
- `AppDir`: Application directory
- `WorkflowFileName`: Auto-generated workflow filename for self-reference

**Features**:
- Triggers on PR changes to app directory or workflow file itself
- Manual dispatch with configurable terraform path
- Calls reusable lint workflow for consistency
- Configurable terraform-docs check (useful for modules)

##### Terraform Workflow (`terraform.yaml.tmpl`)

Generates a workflow for Terraform plan and apply:
- [`terraform.yaml.tmpl`](../internal/templates/files/github/terraform.yaml.tmpl)

**Template Variables**:
- `Env`: Environment name
- `Region`: AWS region
- `AppDir`: Application directory
- `AWSRoleArn`: AWS IAM role ARN for authentication
- `WorkflowFileName`: Auto-generated workflow filename

**Features**:
- Runs on PR for plan, on push to main for apply
- Self-referencing trigger paths
- AWS OIDC authentication with configurable role
- Manual dispatch with full parameter control
- Safety: apply on PR disabled by default

**Reusable Workflows** (stored as static `.yaml` files):
- `reusable-lint.yaml` - Shared linting logic
- `reusable-terraform-plan-apply.yaml` - Shared Terraform logic
- These are copied as-is, GitHub Actions syntax preserved

### Custom Templates

You can add custom templates for additional files using the `--templates-dir` flag or `generate.templates_dir` config in `.tfskel.yaml`:

```bash
tfskel generate myapp --env dev --region us-east-1 --templates-dir ./custom-templates
```

**Custom Template Structure**:
```
custom-templates/
├── main.tf.tmpl
├── variables.tf.tmpl
├── outputs.tf.tmpl
├── locals.tf.tmpl
├── versions.tf.tmpl    # Should include metadata comments (see below)
└── readme.md.tmpl
```

**Template Processing**:
- All files ending with `.tmpl` extension are processed as Go templates
- Non-`.tmpl` files are copied as static content (e.g., reusable workflows)

### Metadata Comments and Automatic Updates

#### Overview

tfskel uses **metadata comments** embedded in generated files to track configuration values and detect when updates are needed. When you run `tfskel generate` on existing directories, tfskel reads these metadata comments and automatically regenerates files if configuration has changed.

This enables:
- **Automatic version updates** when you change Terraform or provider versions in `.tfskel.yaml`
- **Tag synchronization** when default tags are modified
- **Backend updates** when S3 bucket names change
- **Intelligent file management** - only updates what's changed, preserves custom modifications to untracked sections

#### How It Works

1. **Generation**: When tfskel creates a file, it embeds JSON metadata in comments
2. **Detection**: On subsequent runs, tfskel reads and parses these metadata comments
3. **Comparison**: Compares embedded values with current configuration
4. **Update**: If values differ, regenerates the file with new configuration
5. **Logging**: Reports what changed with clear, actionable messages

#### Metadata Comment Format

Metadata comments are HCL-style comments with JSON payloads:

**Important Rules**:
- Must be valid JSON (use double quotes, proper escaping)
- Should be placed near the top of the file for visibility
- Should not be manually edited (managed automatically by tfskel)
- Tag keys are normalized to lowercase (Terraform convention)

#### Recommended: versions.tf.tmpl with Metadata

**For managing Terraform and provider versions**, the recommended approach is to create a `versions.tf.tmpl` template with metadata comments:

```hcl
## Terraform providers and required versions
## This file is auto-generated by tfskel
## DO NOT REMOVE the tfskel-metadata & tfskel-tags comments - they enable automatic updates
## tfskel-metadata: {"tf_ver": "{{.TerraformVersion}}", "aws_provider_ver": "{{.AWSProviderVersion}}"}
## tfskel-tags: {{.DefaultTagsJSON}}

terraform {
  required_version = "{{.TerraformVersion}}"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "{{.AWSProviderVersion}}"
    }
  }
}

provider "aws" {
  region = basename(dirname(path.cwd))

  default_tags {
    tags = {
{{- if .DefaultTags}}
{{- range $key, $value := .DefaultTags}}
      {{$key}} = "{{$value}}"
{{- end}}
{{- end}}
      env = "{{.Env}}"
      app = "{{.AppDir}}"
    }
  }
}
```

**Why versions.tf?**
- Centralizes all version constraints in one file
- Makes version drift detection easier
- Enables automatic updates when you change `.tfskel.yaml`
- Follows Terraform best practices for file organization

#### Using Metadata in Custom Templates

When creating custom templates, include metadata comments to enable automatic updates:

**Example: Custom backend.tf.tmpl**

```hcl
## Backend configuration for Terraform state
## tfskel-metadata: {"bucket": "{{.S3BucketName}}"}

terraform {
  backend "s3" {
    bucket         = "{{.S3BucketName}}"
    key            = "{{.Env}}/{{.Region}}/{{.AppDir}}/terraform.tfstate"
    region         = "{{.Region}}"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
  }
}
```

#### Available Metadata Keys

**For versions.tf**:
- `tf_ver`: Terraform version constraint
- `aws_provider_ver`: AWS provider version constraint

**For backend.tf**:
- `bucket`: S3 bucket name

**For tags** (separate comment):
- Use `## tfskel-tags: {{.DefaultTagsJSON}}` to track `provider.aws.default_tags`
- Tag keys are automatically normalized to lowercase only (not snake_case)
- Output JSON format: `{"team": "platform", "managedby": "terraform"}`

#### User Experience

When configuration changes are detected, tfskel provides clear feedback:

**Without Verbose Flag**:
```bash
[SUCCESS] Updated versions.tf - initialized configuration tracking (see details with -v flag)
[SUCCESS] Updated versions.tf - tf_ver changed: ~> 1.8 -> ~> 1.13
[SUCCESS] Updated versions.tf - added tag - team: platform
```

**With Verbose Flag (`-v`)**:
```bash
[DEBUG  ] No metadata found in versions.tf, will regenerate with: terraform_version=~> 1.13, aws_provider_version=~> 6.0
[SUCCESS] Updated versions.tf - initialized configuration tracking (see details with -v flag)
```

#### Best Practices

1. **Always include metadata comments** in `versions.tf.tmpl` and `backend.tf.tmpl`
2. **Use descriptive file headers** explaining the file is auto-generated
3. **Don't manually edit metadata** - let tfskel manage it
4. **Use -v flag** for troubleshooting to see detailed change detection
5. **Test templates** with metadata before rolling out to production
6. **Document custom metadata** if you extend the system

#### Files That Should Include Metadata

- `versions.tf.tmpl` - Tracks Terraform/provider versions and tags
- `backend.tf.tmpl` - Tracks S3 bucket configuration

### Template Data Structure

All templates (both embedded and custom) receive a `Data` struct containing all necessary context for rendering. This struct provides information about the environment, region, application, versions, and configuration.

#### Available Template Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `Env` | `string` | Environment name from config | `dev`, `stg`, `prd` |
| `Region` | `string` | Full AWS region name | `eu-central-1`, `us-east-1` |
| `AppDir` | `string` | Application directory name | `myapp`, `web-service` |
| `AccountID` | `string` | AWS account ID from environment mapping | `123456789012` |
| `ShortRegion` | `string` | Abbreviated region name | `euc1`, `use1` |
| `S3BucketName` | `string` | S3 bucket name for Terraform state | `terraform-state-dev` |
| `TerraformVersion` | `string` | Terraform version constraint | `~> 1.13` |
| `AWSProviderVersion` | `string` | AWS provider version constraint | `~> 6.0` |
| `DefaultTags` | `map[string]string` | Map of default AWS tags | `{"team": "platform", "managedby": "terraform"}` |
| `DefaultTagsJSON` | `string` | JSON representation of DefaultTags | `{"team":"platform","managedby":"terraform"}` |
| `AWSRoleArn` | `string` | AWS IAM role ARN for GitHub workflows | `arn:aws:iam::123456789012:role/...` |
| `WorkflowFileName` | `string` | Generated workflow filename (GitHub workflows only) | `myapp-dev-euc1-lint.yaml` |

#### Variable Details and Use Cases

**`Env` (Environment)**
- Source: Command line flag `--env` or inferred from directory path
- Used in: Backend keys, tags, workflow names, resource naming
- Example usage:
  ```hcl
  # In templates
  env = "{{.Env}}"

  # In S3 state key
  key = "{{.Env}}/{{.Region}}/{{.AppDir}}/terraform.tfstate"
  ```

**`Region` (AWS Region)**
- Source: Command line flag `--region` or inferred from directory path
- Format: Standard AWS region format (kebab-case)
- Used in: Backend configuration, provider region, resource naming
- Example usage:
  ```hcl
  # Provider configuration
  region = "{{.Region}}"

  # Dynamic region from directory path
  region = basename(dirname(path.cwd))
  ```

**`AppDir` (Application Directory)**
- Source: First positional argument to `tfskel generate <app-dir>`
- Used in: Backend state keys, tags, resource naming
- Example usage:
  ```hcl
  # In backend key
  key = "{{.Env}}/{{.Region}}/{{.AppDir}}/terraform.tfstate"

  # In tags
  app = "{{.AppDir}}"
  ```

**`AccountID` (AWS Account ID)**
- Source: Mapped from `provider.aws.account_mapping` in `.tfskel.yaml`
- Required: Must be defined for the environment
- Used in: Backend configuration, IAM role ARNs, AWS resource identifiers
- Example usage:
  ```hcl
  # AWS account ID for resources
  account_id = "{{.AccountID}}"
  ```

**`ShortRegion` (Short Region Name)**
- Source: Automatically derived from `Region` using transformation
- Format: Abbreviated format (e.g., `eu-central-1` → `euc1`, `us-east-1` → `use1`)
- Used in: Resource naming, workflow names, short identifiers
- Transformation rules:
  - `eu-central-1` → `euc1`
  - `us-east-1` → `use1`
  - `us-west-2` → `usw2`
  - `ap-southeast-1` → `apse1`
- Example usage:
  ```hcl
  # Short region in resource names
  name_prefix = "{{.AppDir}}-{{.Env}}-{{.ShortRegion}}"
  ```

**`S3BucketName` (S3 Backend Bucket)**
- Source: `backend.s3.bucket_name` in `.tfskel.yaml` or `--s3-bucket-name` flag
- Supports Go template syntax for dynamic values
- Used in: Backend configuration
- **Exception**: Can contain template variables itself (e.g., `terraform-state-{{.Env}}`)
- Example usage:
  ```hcl
  # Backend configuration
  bucket = "{{.S3BucketName}}"
  ```
- Advanced example with dynamic bucket names:
  ```yaml
  # In .tfskel.yaml
  backend:
    s3:
      bucket_name: "terraform-state-{{.Env}}-{{.Region}}"
  ```
  Results in: `terraform-state-dev-eu-central-1`

**`TerraformVersion` (Terraform Version Constraint)**
- Source: `terraform_version` in `.tfskel.yaml`
- Format: Terraform version constraint syntax (~>, >=, <=, =)
- Used in: Required version blocks, version drift detection
- Example usage:
  ```hcl
  terraform {
    required_version = "{{.TerraformVersion}}"
  }
  ```

**`AWSProviderVersion` (AWS Provider Version Constraint)**
- Source: `provider.aws.version` in `.tfskel.yaml`
- Format: Terraform version constraint syntax
- Used in: Required providers blocks, version drift detection
- Example usage:
  ```hcl
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "{{.AWSProviderVersion}}"
    }
  }
  ```

**`DefaultTags` (Default Tags Map)**
- Source: `provider.aws.default_tags` in `.tfskel.yaml`
- Type: `map[string]string` (Go map)
- **Important**: Tag keys are automatically normalized to lowercase
- Used in: Provider default_tags block
- Example usage:
  ```hcl
  default_tags {
    tags = {
  {{- range $key, $value := .DefaultTags}}
      {{$key}} = "{{$value}}"
  {{- end}}
      env = "{{.Env}}"
      app = "{{.AppDir}}"
    }
  }
  ```

**`DefaultTagsJSON` (Default Tags JSON String)**
- Source: JSON representation of `provider.aws.default_tags`
- Type: `string` (JSON formatted)
- **Purpose**: Used exclusively in metadata comments for change detection
- Used in: Metadata comments (`## tfskel-tags: {{.DefaultTagsJSON}}`)
- Example usage:
  ```hcl
  ## tfskel-tags: {{.DefaultTagsJSON}}
  ```
  Results in: `## tfskel-tags: {"team":"platform","managedby":"terraform"}`

**`AWSRoleArn` (GitHub Workflow AWS Role ARN)**
- Source: Derived from `generate.github_workflows.aws_role_arn` or `aws_role_name` in `.tfskel.yaml`
- Format: Full ARN format or constructed from role name
- Used in: GitHub Actions workflow AWS authentication
- Priority order:
  1. Explicit `aws_role_arn` if specified
  2. Constructed from `aws_role_name`: `arn:aws:iam::{{.AccountID}}:role/<role-name>`
  3. Default placeholder: `arn:aws:iam::<account-id>:role/REPLACE_WITH_ROLE_TO_ASSUME`
- Example usage:
  ```yaml
  # In GitHub workflow
  aws-role-to-assume: {{.AWSRoleArn}}
  ```

**`WorkflowFileName` (GitHub Workflow Self-Reference)**
- Source: Auto-generated based on `generate.github_workflows.name_template` or default pattern
- Default pattern: `{{.AppDir}}-{{.Env}}-{{.ShortRegion}}`
- **Purpose**: Enables workflows to self-reference for path-based triggers
- Used in: GitHub workflow trigger paths
- Example usage:
  ```yaml
  # Workflow triggers on changes to itself
  paths:
    - '.github/workflows/{{.WorkflowFileName}}-lint.yaml'
  ```

#### Special Considerations

**Tag Key Normalization**:
- All tag keys in `DefaultTags` are automatically converted to lowercase
- This follows Terraform AWS provider conventions
- **Important**: Only converts to lowercase, does NOT add underscores (not snake_case)
- Examples:
  - `Team` → `team`
  - `CostCenter` → `costcenter` (NOT `cost_center`)
  - `ManagedBy` → `managedby` (NOT `managed_by`)
  - `environment` → `environment` (unchanged)

**Template Rendering in Config Values**:
- `S3BucketName`, `AWSRoleArn`, `AWSRoleName`, and `WorkflowFileName` support Go template expressions
- Allows dynamic values: `bucket_name: "terraform-state-{{.Env}}"`
- **Exception**: Plain strings (no `{{`) are returned unchanged without parsing overhead

**Empty Values**:
- If `DefaultTags` is empty or nil, tag iteration produces no output
- If `AccountID` is missing for environment, generation fails with error
- `AWSRoleArn` defaults to placeholder if not configured

### Template Functions Reference

tfskel provides a comprehensive set of template functions available in all templates (both embedded and custom). These functions are part of Go's `text/template` package with custom additions specific to tfskel.

#### String Manipulation Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `replace` | `replace(old, new, s string) string` | Replace all occurrences of `old` with `new` in string `s` | `{{.AppDir \| replace "-" "_"}}` |
| `toLower` | `toLower(s string) string` | Convert string to lowercase | `{{.Env \| toLower}}` |
| `toUpper` | `toUpper(s string) string` | Convert string to uppercase | `{{.Region \| toUpper}}` |
| `trimSpace` | `trimSpace(s string) string` | Trim leading and trailing whitespace | `{{.AppDir \| trimSpace}}` |
| `trimPrefix` | `trimPrefix(prefix, s string) string` | Remove prefix from string if present | `{{.Region \| trimPrefix "us-"}}` |
| `trimSuffix` | `trimSuffix(suffix, s string) string` | Remove suffix from string if present | `{{.AppDir \| trimSuffix "-api"}}` |

#### String Checking Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `hasPrefix` | `hasPrefix(prefix, s string) bool` | Check if string starts with prefix | `{{if hasPrefix "prod" .Env}}...{{end}}` |
| `hasSuffix` | `hasSuffix(suffix, s string) bool` | Check if string ends with suffix | `{{if hasSuffix "-1" .Region}}...{{end}}` |
| `contains` | `contains(substr, s string) bool` | Check if string contains substring | `{{if contains "central" .Region}}...{{end}}` |

#### Array/Slice Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `join` | `join(sep string, elems []string) string` | Join string array with separator | `{{join "," .Regions}}` |
| `split` | `split(s, sep string) []string` | Split string by separator | `{{range split .Region "-"}}{{.}}{{end}}` |

#### Version Constraint Functions

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `stripConstraint` | `stripConstraint(version string) string` | Remove version operators (~>, >=, <=, >, <, =) and return clean version | `{{.TerraformVersion \| stripConstraint}}` |

**`stripConstraint` Details**:
- **Purpose**: Extract clean version numbers from Terraform version constraints
- **Use case**: Creating `.terraform-version` files, version tags, resource naming
- **Transformation examples**:
  - `~> 1.14.3` → `1.14.3`
  - `>= 1.10.0` → `1.10.0`
  - `<= 2.0.0` → `2.0.0`
  - `1.13` → `1.13` (unchanged if no operator)
- **Example usage**:
  ```plaintext
  # .terraform-version file (needs clean version)
  {{.TerraformVersion | stripConstraint}}
  ```
  Results in: `1.13.0` instead of `~> 1.13.0`

#### Complete Template Example

Here's a comprehensive example using multiple functions:

```hcl
## Custom Terraform configuration
## Environment: {{.Env | toUpper}}
## Region: {{.Region}}
## tfskel-metadata: {"tf_ver": "{{.TerraformVersion}}", "aws_provider_ver": "{{.AWSProviderVersion}}"}

terraform {
  required_version = "{{.TerraformVersion}}"

  backend "s3" {
    bucket = "{{.S3BucketName}}"
    key    = "{{.Env}}/{{.Region}}/{{.AppDir}}/terraform.tfstate"
    {{- if contains "eu" .Region}}
    region = "{{.Region}}"
    {{- else}}
    region = "us-east-1"
    {{- end}}
  }
}

provider "aws" {
  region = "{{.Region}}"

  default_tags {
    tags = {
{{- range $key, $value := .DefaultTags}}
      {{$key}} = "{{$value}}"
{{- end}}
      environment     = "{{.Env | toLower}}"
      application     = "{{.AppDir | replace "-" "_"}}"
      terraform_version = "{{.TerraformVersion | stripConstraint}}"
      {{- if hasPrefix "prd" .Env}}
      criticality     = "high"
      {{- else}}
      criticality     = "low"
      {{- end}}
    }
  }
}

# Resource naming convention
locals {
  name_prefix = "{{.AppDir}}-{{.Env}}-{{.ShortRegion}}"
  name_prefix_uppercase = "{{.AppDir | toUpper}}_{{.Env | toUpper}}_{{.ShortRegion | toUpper}}"

  {{- if eq .Env "prd"}}
  retention_days = 90
  {{- else}}
  retention_days = 7
  {{- end}}
}
```

---

## Directory Structure

### Generated Project Structure

A typical tfskel-generated project has the following structure:

```
project-root/
├── .tfskel.yaml                # Configuration file
├── .gitignore                  # Terraform-specific ignores
├── .pre-commit-config.yaml     # Pre-commit hooks
├── .tflint.hcl                 # TFLint configuration
├── trivy.yaml                  # Trivy security scanner
│
└── envs/                       # Environment-based structure
    ├── dev/
    │   ├── .terraform-version  # Terraform version for dev
    │   ├── eu-central-1/       # Region subdirectory
    │   │   └── myapp/          # Application directory
    │   │       ├── backend.tf  # S3 backend config
    │   │       └── versions.tf # TF & provider versions
    │   └── us-east-1/
    │       └── myapp/
    │           ├── backend.tf
    │           └── versions.tf
    ├── stg/
    │   ├── .terraform-version
    │   └── eu-central-1/
    │       └── myapp/
    │           ├── backend.tf
    │           └── versions.tf
    └── prd/
        ├── .terraform-version
        └── eu-central-1/
            └── myapp/
                ├── backend.tf
                └── versions.tf
```

**With Custom Templates**:

```
project-root/
└── envs/
    └── dev/
        └── eu-central-1/
            └── myapp/
                ├── backend.tf       # Embedded template
                ├── versions.tf      # Embedded template
                ├── main.tf          # Custom template
                ├── variables.tf     # Custom template
                ├── outputs.tf       # Custom template
                ├── locals.tf        # Custom template
                └── readme.md        # Custom template
```

### Directory Purposes

#### Root Directory

Contains project-level configuration files:
- `.tfskel.yaml`: Project configuration
- `.gitignore`: Git ignore patterns for Terraform files
- `.pre-commit-config.yaml`: Pre-commit hooks setup
- `.tflint.hcl`: TFLint linter configuration
- `trivy.yaml`: Trivy security scanner configuration

#### `envs/`

Top-level directory for all environment-specific code:
- Organizes infrastructure by environment first
- Prevents accidental cross-environment deployments
- Enables environment-specific Terraform versions

#### `envs/<environment>/`

Environment-specific directories (dev, stg, prd):
- Contains `.terraform-version` file specifying Terraform version
- Has subdirectories for each AWS region
- Allows different Terraform versions per environment

#### `envs/<environment>/<region>/`

Region-specific directories (eu-central-1, us-east-1, etc.):
- Contains application subdirectories
- Groups resources by AWS region
- Enables multi-region deployments

#### `envs/<environment>/<region>/<app>/`

Application-specific directories:
- Contains Terraform configuration for specific application
- Includes `backend.tf` and `versions.tf` at minimum
- Can include custom templates (main.tf, variables.tf, etc.)

---

## Advanced Usage

### Custom Templates

You can extend tfskel's default templates by providing custom templates:

```bash
# Create custom templates directory
mkdir -p custom-templates

# Create custom main.tf template
cat > custom-templates/main.tf.tmpl <<'EOF'
# Main infrastructure for {{.AppDir}} in {{.Env}}

module "vpc" {
  source = "../../modules/vpc"

  environment = "{{.Env}}"
  region      = "{{.Region}}"
}
EOF

# Generate with custom templates
tfskel generate myapp --env dev --region us-east-1 --templates-dir custom-templates
```

**Supported Custom Templates**:
- Any file ending in `.tf.tmpl` (processed by default)
- Additional extensions via `--extra-template-extensions` flag

**Example: Add variables.tf template**:
```hcl
# custom-templates/variables.tf.tmpl
variable "environment" {
  description = "Environment name"
  type        = string
  default     = "{{.Env}}"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "{{.Region}}"
}
```

> [!IMPORTANT]
> For templates that should be automatically updated when configuration changes (like `versions.tf` or `backend.tf`), include metadata comments. See [Metadata Comments and Automatic Updates](#metadata-comments-and-automatic-updates) for details and examples.

### Multi-Region Deployment

Deploy the same application across multiple regions:

```bash
# Deploy to US regions
tfskel generate webapp --env prd --region us-east-1
tfskel generate webapp --env prd --region us-west-2

# Deploy to EU regions
tfskel generate webapp --env prd --region eu-central-1
tfskel generate webapp --env prd --region eu-west-1

# Deploy to Asia-Pacific
tfskel generate webapp --env prd --region ap-south-1
tfskel generate webapp --env prd --region ap-southeast-1
```

### Version Drift Management

Detect and fix version inconsistencies:

```bash
# Check for drift
tfskel drift version

# Export drift report to CSV
tfskel drift version --format csv --no-color > drift-report.csv

# Check specific environment
tfskel drift version --path ./envs/prd

# Output as JSON for automated processing
tfskel drift version --format json > drift.json
```

**Common Drift Scenarios**:

1. **Major version drift**: Different major versions across environments
   ```bash
   # Found: dev uses ~> 5.0, prd uses ~> 6.0
   # Action: Update .tfskel.yaml and regenerate
   ```
> [!Note]
> Integrate with CI as a linter

2. **Easier terraform plan changes with custom priorities**: Instead of reading large plans, human readable summary of the changes and severity assignation as per custom rules as per the stack.

> [!Note]
> Integrate with CI after the terraform plan step for easier reviews.

### Dynamic S3 Bucket Names

Use template variables in S3 bucket names:

```yaml
# .tfskel.yaml
backend:
  s3:
    bucket_name: terraform-state-{{.Env}}-{{.ShortRegion}}-12345
```

**Available Variables**:
- `{{.Env}}`: Environment (dev, stg, prd)
- `{{.Region}}`: Full region (eu-central-1)
- `{{.ShortRegion}}`: Short region (euc1)
- `{{.AppDir}}`: Application name
- `{{.AccountID}}`: AWS account ID

**Result**:
- dev + eu-central-1 → `terraform-state-dev-euc1-12345`
- prd + us-east-1 → `terraform-state-prd-use1-12345`
