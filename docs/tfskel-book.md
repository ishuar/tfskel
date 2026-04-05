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
   - [Dry Run Mode](#dry-run-mode)
   - [Operation Summary](#operation-summary)
   - [Config Source Debugging](#config-source-debugging)
   - [Color Detection](#color-detection)
   - [Upgrading Generated Files](#upgrading-generated-files)

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
- ⬆️ **Upgrade Support**: Re-render previously generated files with `--upgrade` when templates or config change
- 🔍 **Dry Run Mode**: Preview all file operations with `--dry-run` before writing anything to disk
- 📊 **Operation Summary**: After each run, see a count of files created, skipped, upgraded, or force-upgraded
- 🎨 **Custom Templates**: Override default templates with your own
- ⚙️ **YAML Configuration**: Flexible configuration with sensible defaults
- 🏷️ **Metadata Tracking**: Embeds metadata in files for intelligent updates such as `default_tags`
- 🔧 **Backend Configuration**: Pre-configured S3 backend with state locking
- 🔍 **Drift Detection**: Detect Terraform and provider version inconsistencies across repos
- 📦 **Zero Runtime Dependencies**: Single binary with embedded templates

### Architecture Highlights

tfskel is designed with clean architecture principles:

- **Interface-based design** for testability — consumer-defined `Logger` interfaces in each package
- **Dependency injection** for flexibility
- **Embedded templates** for zero-dependency distribution
- **In-memory filesystem** for fast, isolated tests
- **Dry-run filesystem** decorator that delegates reads but silently skips all writes
- **Structured logging** with color-coded output and CI-aware color detection
- **Operation tracking** — `OpTracker` records every file operation for end-of-run summaries
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

1. **Initialize a new project** (add `--workflows` to also generate shared GitHub Actions files):

```bash
tfskel init
# or with shared workflow files:
tfskel init --workflows
```

2. **Scaffold application structure**:

```bash
tfskel scaffold myapp --env dev --region us-east-1
```

3. **Generate a per-environment Terraform workflow** (requires shared workflow files from step 1):

```bash
tfskel scaffold workflows --env dev
```

4. **Validate project health**:

```bash
tfskel validate
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
- `DryRunFileSystem` decorator that wraps any `FileSystem`, delegating reads (so upgrade checks still work) but making all writes (`WriteFile`, `MkdirAll`) no-ops
- This abstraction makes the entire codebase testable without touching disk and enables `--dry-run` mode

#### 4. Generator

The generator (`internal/generate`) orchestrates the entire generation process:
1. Validates configuration
2. Creates directory structure
3. Renders templates (embedded and custom) — only `tf/` category templates for `scaffold`; `root/` and `github/` are handled by `init` and `scaffold workflows` respectively
4. Writes files to disk with metadata
5. Detects and handles configuration changes
6. Tracks every file operation via `OpTracker` and reports a summary (e.g. `3 files created, 1 file skipped`)
7. In dry-run mode, logs intended actions with `[dry-run]` prefix without writing

#### 5. Version Drift & Plan Analysis

The version drift detection (`internal/diff`) provides:
- HCL parsing of Terraform files
- Version extraction from terraform and required_providers blocks
- Comparison against .tfskel.yaml configuration
- Multi-format output (table, JSON, CSV)
- Comprehensive reporting with drift categorization

The plan analysis (`internal/plan`) provides:
- JSON plan file parsing
- Resource change categorization
- Critical resource detection
- Severity assessment
- Multi-format output (table, JSON, CSV)

#### 6. Logger

The logger (`internal/logger`) provides structured logging with:
- Multiple log levels (DEBUG, INFO, WARN, SUCCESS, ERROR, FATAL)
- Color-coded console output controlled via `NewWithOptions(verbose, useColor)`
- CI-aware color detection — automatically disables colors when `CI=true` (GitHub Actions, GitLab CI, etc.), respects `NO_COLOR` and `FORCE_COLOR` env vars
- `SetMachineOutput()` method that redirects output to stderr and disables colors for machine-readable formats (JSON/CSV)
- Consumer-defined `Logger` interfaces in `internal/generate` and `internal/config` packages (Go best practice: interfaces belong in the consumer)
- Test-friendly constructors: `NewWithWriters(verbose, out, errOut)` for capturing output in tests

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
  - **Required**: Cannot be empty or left as placeholder value
  - **Invalid**: Placeholder "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME" will be rejected
  - **Must be replaced**: With actual S3 bucket name before running scaffold command
- Supports template variables like {{.Env}}, {{.Region}}, {{.AppDir}}


#### Provider Section

Defines AWS provider configuration:
- `provider.aws.version`: AWS provider version constraint (default: ~> 6.0)
- `provider.aws.regions`: List of AWS regions for the project
- `provider.aws.account_mapping`: Maps environment names to AWS account IDs
  - **Required**: At least one environment mapping must be defined
  - **Account ID format**: Must be exactly 12 numeric digits (e.g., "123456789012")
  - **No placeholders**: Account IDs like "REPLACE_WITH_YOUR_DEV_ACCOUNT_ID" will be rejected during validation
  - **Environment matching**: If you use `tfskel scaffold --env <env>`, the account ID for that environment must exist in the mapping
  - **Example**: If mapping has `dev: "123456789012"`, you can use `--env dev`, but `--env prod` will fail if `prod` is not in the mapping
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

- `templates.dir`: Path to custom template directory
- All files ending with `.tmpl` extension are processed as Go templates
- Custom templates override embedded defaults
- Useful for adding main.tf, variables.tf, outputs.tf, etc.

#### GitHub Workflows Generation

Automate creation of GitHub Actions workflows for Terraform CI/CD. Workflows are split into two groups: **shared static files** (generated by `tfskel init --workflows`) and **per-environment caller workflows** (generated by `tfskel scaffold workflows --env <env>`).

**Configuration Fields**:
- `workflows.create`: When `true`, `tfskel init` also generates the shared workflow files (same as `--workflows` flag)
- `workflows.name`: Plain string stem for the per-env workflow filename (optional)
  - **Go template syntax is NOT supported**
  - The environment prefix and `.yaml` extension are added automatically
  - Example: `name: "terraform"` + `--env dev` → `dev-terraform.yaml`
  - Default: `"terraform"`
- `workflows.aws_role_name`: IAM role name for AWS authentication (optional, accepts Go template syntax)
  - Automatically constructs ARN: `arn:aws:iam::<account-id>:role/<role-name>`
- `workflows.aws_role_arn`: Explicit IAM role ARN (optional, takes priority, accepts Go template syntax)

**Priority Order for AWS Role**:
1. `aws_role_arn` (if specified) - Explicit ARN
2. `aws_role_name` (if specified) - Constructs ARN using account ID from environment mapping
3. Default placeholder - `arn:aws:iam::<account-id>:role/REPLACE_WITH_ROLE_TO_ASSUME`

**Shared Workflow Files** (generated by `tfskel init --workflows`):
1. `lint.yaml` - Global Terraform linting caller (detects changed app directories)
2. `reusable-detect-changes.yaml` - Detects which Terraform app directories have changed
3. `reusable-lint.yaml` - Reusable linting workflow (called by `lint.yaml`)
4. `reusable-terraform-plan-apply.yaml` - Reusable Terraform plan/apply workflow

**Per-Environment Workflow** (generated by `tfskel scaffold workflows --env <env>`):
- `<env>-<name>.yaml` - Terraform plan/apply caller for a specific environment

**Workflow Features**:
- Triggered on pull requests and pushes to main branch
- Change detection — only triggers for app directories with actual file changes
- AWS OIDC authentication with configurable IAM roles
- Manual workflow dispatch with input parameters
- TFLint integration with caching
- Plan artifacts and PR comments

**⚠️ Auto-Apply Safety**:

> [!IMPORTANT]
> The `auto_apply` parameter controls automatic terraform apply execution.
> However **On push to main**: always apply irrespective of `auto_apply` value

**Default Behavior** (when `auto_apply` is not explicitly set):
- ✅ **On PR**: Plan only, no auto-apply (safe for all environments)
- ⚠️ **On manual dispatch**: Requires explicit `auto_apply` input

**Production Safety Recommendations**:
1. **Always set `auto_apply: false`** in workflow inputs for production environments
2. Use GitHub environment protection rules with required reviewers
3. Enable branch protection on `main` with required PR reviews
4. Consider manual approval gates for production applies
5. The workflow respects any environment naming (not just 'prd') - safety is **your responsibility**

**Example Production Workflow**:
```yaml
uses: ./.github/workflows/reusable-terraform-plan-apply.yaml
with:
  environment: production  # Or prd, prod, live - any name works
  auto_apply: ${{ inputs.auto_apply || false }}
  # ... other inputs
```

#### Drift Detection Configuration

Configure drift detection behavior for version and plan analysis:

**Configuration Fields**:
- `critical_resources`: Additional AWS resource types to mark as critical (extends defaults)
  - Default critical resources include databases (RDS, DynamoDB), S3 buckets, VPCs, security groups, IAM roles, KMS keys, WAF rules, etc.
  - User-defined resources are merged with defaults without duplicates
  - Critical resource changes are marked with "Critical" severity in plan analysis
- `top_resources_count`: Maximum number of resources to display in plan analysis summaries (default: 10)
  - Applies to resource type groupings, module groupings, and action counts
  - Set to 0 to show all items without limit
  - Can also be set via `--top-resources-count` flag for `review plan` command

---

## Validation and Error Handling

### Configuration Validation

When running `tfskel scaffold`, the configuration undergoes strict validation to ensure all required values are properly set.

#### Required Configuration Validations

1. **AWS Provider Configuration**
   - Provider AWS section must exist
   - Account mapping must be defined and not empty

2. **Account ID Format Validation**
   - All account IDs must be exactly 12 numeric digits
   - Format: `^\d{12}$` (e.g., "123456789012")
   - Invalid examples:
     - `REPLACE_WITH_YOUR_DEV_ACCOUNT_ID` (placeholder text)
     - `12345678901A` (contains letters)
     - `12345` (too short)
     - `1234567890123` (too long)

3. **Environment-Specific Account Mapping**
   - The environment specified with `--env` flag must have a corresponding account ID in `account_mapping`
   - If mapping not found, error shows available environments to help you fix it

4. **Backend S3 Bucket Name Validation**
   - `backend.s3.bucket_name` must be set to a non-empty value
   - Cannot be left as placeholder: `CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME`
   - Must be replaced with actual S3 bucket name

### Common Validation Errors

Here are common errors you may encounter and how to fix them:

#### "AWS account ID must be a 12-digit number"

**Cause**: Account ID in `account_mapping` is not exactly 12 numeric digits.

**Example Error**:
```
AWS account ID must be a 12-digit number: Update the account mapping "dev": "REPLACE_WITH_YOUR_DEV_ACCOUNT_ID"
```

**Fix**: Replace with valid 12-digit AWS account ID:
```yaml
provider:
  aws:
    account_mapping:
      dev: "123456789012"  # Valid format
```

#### "no account mapping found for environment"

**Cause**: The environment specified with `--env` flag doesn't exist in your `account_mapping`.

**Example Error**:
```
no account mapping found for environment "prod", available: [dev, prd, stg]
```

**Fix**: Either add the missing environment or use an available one:
```yaml
provider:
  aws:
    account_mapping:
      dev: "123456789012"
      stg: "234567890123"
      prd: "345678901234"
      prod: "456789012345"  # Add the missing environment
```

Or use an existing environment:
```bash
tfskel scaffold myapp --env prd --region us-east-1  # Use 'prd' instead of 'prod'
```

#### "backend.s3.bucket_name is invalid"

**Cause**: S3 bucket name is empty or still set to placeholder value.

**Example Errors**:
```
backend.s3.bucket_name is invalid: must not be empty
```
or
```
backend.s3.bucket_name is invalid: placeholder value must be replaced with actual bucket name
```

**Fix**: Set a valid S3 bucket name:
```yaml
backend:
  s3:
    bucket_name: "my-terraform-state-bucket"  # Replace with your bucket
```

#### "AWS provider configuration is required"

**Cause**: AWS provider section is missing or incomplete in `.tfskel.yaml`.

**Fix**: Ensure your configuration includes:
```yaml
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
    regions:
      - us-east-1
```

### Validation Workflow

The validation process follows this order:

1. **Load Configuration** - Parse `.tfskel.yaml` and apply flag overrides
2. **Validate Structure** - Check AWS provider and account_mapping exist
3. **Validate Account IDs** - Verify all account IDs are 12-digit numbers
4. **Validate Backend** - Ensure bucket_name is set and not placeholder
5. **Validate Environment Mapping** - Check specified `--env` has account ID
6. **Prepare Template Data** - Retrieve account ID for the environment
7. **Generate Files** - Create directory structure and render templates

Validation errors are caught early and provide actionable error messages with suggestions to help you fix the issue quickly.

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
- `--workflows`: Generate shared GitHub Actions reusable workflow files (default: false)
- `--upgrade`: Re-render init-managed files with latest embedded templates (default: false)
- `--force`: With `--upgrade`, overwrite files even without source markers (default: false)
- `--dry-run`: Show what would happen without writing files (global flag, works with all commands)
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

# Also generate shared GitHub Actions workflow files
tfskel init --workflows

# Re-render init files from latest templates
tfskel init --upgrade

# Force overwrite all init files, even without source markers
tfskel init --upgrade --force

# Preview what init would do without writing files
tfskel init --dry-run
```

**What it does**:
1. Reads existing .tfskel.yaml configuration if present (or uses defaults)
2. Creates root-level configuration files:
   - `.gitignore` - Terraform-specific ignore patterns
   - `.pre-commit-config.yaml` - Pre-commit hooks configuration
   - `.tflint.hcl` - TFLint configuration
   - `trivy.yaml` - Trivy security scanner configuration
   - `.tfskel.yaml` - Default tfskel configuration with:
     - Default `account_mapping` for [`dev`,`stg`,`prd`] envs with placeholder values
     - Empty `critical_resources` list
     - Placeholder S3 bucket name that must be replaced
     - Terraform version constraint `~> 1.13` (instead of specific version)
3. Creates environment directories based on account_mapping in config
4. Creates region subdirectories for each environment
5. Creates `.terraform-version` files for each environment
6. If `--workflows` flag is set or `workflows.create: true` in config, generates shared reusable workflow files under `.github/workflows/`:
   - `lint.yaml` - Global Terraform linting caller workflow
   - `reusable-detect-changes.yaml` - Detects changed Terraform app directories
   - `reusable-terraform-plan-apply.yaml` - Reusable Terraform plan/apply workflow
   - `reusable-lint.yaml` - Reusable linting workflow

> [!IMPORTANT]
> After running `tfskel init`, you **must** update `.tfskel.yaml` with:
> - Your AWS account IDs in `provider.aws.account_mapping` (12-digit format)
> - Your S3 bucket name in `backend.s3.bucket_name`
> - Before running `tfskel scaffold`, these values must be properly configured

### `tfskel scaffold`

Scaffold Terraform project structure for a specific application.

**Usage**:
```bash
tfskel scaffold <app-dir> [flags]
```

**Aliases**: `sc`

**Arguments**:
- `app-dir`: Name of the application directory to create (required)

**Subcommands**:
- `workflows` - Generate a per-environment GitHub Actions Terraform plan/apply workflow

**Flags**:
- `--env, -e`: Target environment (required) - e.g., dev, stg, prd
- `--region, -r`: AWS region (required) - e.g., us-east-1, eu-central-1
- `--config, -c`: Path to config file (default: .tfskel.yaml in current directory)
- `--templates-dir`: Directory containing custom template files
- `--s3-bucket-name`: Override S3 bucket name for Terraform state
- `--upgrade`: Re-render existing files from updated templates (default: false)
- `--force`: With `--upgrade`, overwrite files even without source markers (default: false)
- `--dry-run`: Show what would happen without writing files (global flag)
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Scaffold structure for an app in dev environment
tfskel scaffold myapp --env dev --region us-east-1

# Scaffold with custom configuration file
tfskel scaffold myapp --config ./my-config.yaml --env dev --region us-east-1

# Scaffold with custom templates
tfskel scaffold myapp --env stg --region eu-central-1 --templates-dir ./templates

# Override S3 bucket name
tfskel scaffold myapp --env prd --region us-west-2 --s3-bucket-name my-custom-bucket

# Using the short alias
tfskel sc myapp --env dev --region us-east-1

# Re-render files after updating templates or config
tfskel scaffold myapp --env dev --region us-east-1 --upgrade

# Force overwrite all files, even without source markers
tfskel scaffold myapp --env dev --region us-east-1 --upgrade --force

# Preview what scaffold would do (no files written)
tfskel scaffold myapp --env dev --region us-east-1 --dry-run

# Generate per-env Terraform plan/apply workflow
tfskel scaffold workflows --env dev
```

**What it does**:
1. Loads configuration from .tfskel.yaml
2. Validates required configuration:
   - Checks that `provider.aws.account_mapping` exists and is not empty
   - Validates that the specified `--env` has a corresponding account ID in the mapping
   - Returns a helpful error showing available environments if mapping is missing
3. Creates directory structure: `envs/<env>/<region>/<app-dir>`
4. Renders embedded templates:
   - `backend.tf` - S3 backend with metadata
   - `versions.tf` - Terraform and provider versions with metadata
5. Renders custom templates if `--templates-dir` is provided or configured in `.tfskel.yaml`
6. Embeds source markers and metadata in generated files for change detection
7. Only creates new files, preserves existing ones
8. Updates files if configuration metadata has changed
9. With `--upgrade`, re-renders existing files when templates or config have drifted

### `tfskel scaffold workflows`

Generate a per-environment GitHub Actions Terraform plan/apply caller workflow. Run this once per environment. Shared reusable workflow files must already exist (created by `tfskel init --workflows`).

**Usage**:
```bash
tfskel scaffold workflows --env <env>
```

**Flags**:
- `--env, -e`: Target environment (required) - e.g., dev, stg, prd
- `--upgrade`: Re-render existing workflow files from updated templates (default: false)
- `--force`: With `--upgrade`, overwrite workflow files even without source markers (default: false)
- `--dry-run`: Show what would happen without writing files (global flag)
- `--config, -c`: Path to config file (default: .tfskel.yaml in current directory)
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Generate workflow for dev environment
tfskel scaffold workflows --env dev

# Generate with a custom config file
tfskel scaffold workflows --env prd --config ./my-config.yaml

# Re-render an existing workflow after config changes
tfskel scaffold workflows --env dev --upgrade

# Preview what would be generated
tfskel scaffold workflows --env dev --dry-run
```

**What it does**:
1. Loads configuration from .tfskel.yaml
2. Renders the per-env Terraform plan/apply caller workflow template
3. Writes `.github/workflows/<env>-<name>.yaml`

> [!NOTE]
> `workflows.name` is a **plain string** — Go template syntax is not supported. The env prefix and `.yaml` extension are added automatically. Example: `name: "terraform"` + `--env dev` → `dev-terraform.yaml`.

### `tfskel validate`

Checks whether your project is in sync with `.tfskel.yaml` by running two validation checks: config drift detection and tool installation status.

**Usage**:
```bash
tfskel validate [flags]
```

**Flags**:
- `--skip`: Comma-separated checks to skip (`config`, `tools`)
- `--format, -f`: Output format: table, json, csv (default: table)
- `--no-color`: Disable colored output
- `--config`: Path to config file (default: .tfskel.yaml)
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Run all checks
tfskel validate

# Skip tool checks
tfskel validate --skip tools

# Output as JSON for CI/CD
tfskel validate --format json > validate-report.json

# Output as CSV
tfskel validate --format csv --no-color > validate.csv
```

**What it does**:
1. **Config check**: Recursively scans for `.tf` files, parses HCL to extract version constraints, compares against `.tfskel.yaml`, and checks `.terraform-version` files for mismatches
2. **Tool check**: Detects required tools (terraform, tflint, trivy, etc.), checks installation status via mise, and compares installed versions against `.mise.toml` pins
3. Generates a unified report with findings and exit codes for CI/CD
6. Outputs in requested format (table/json/csv)

**Drift Detection Features**:
- HCL parsing for accurate version extraction
- Automatic hidden directory filtering (skips .git, .terraform, etc.)
- Intelligent version comparison
- Terminal-aware table formatting
- Color-coded status indicators
- Parse error handling (continues on errors)

---

### `tfskel review plan`

Analyze Terraform plan JSON to identify resource changes, impact severity, and potential risks.

**Usage**:
```bash
tfskel review plan [flags]
```

**Flags**:
- `--json-file`: Path to Terraform plan JSON file (required)
- `--format, -f`: Output format: table, json, csv (default: table)
- `--top-resources-count`: Show top N highest-impact resources (default: 10, use 0 for all)
- `--no-color`: Disable colored output
- `--verbose, -v`: Enable verbose output

**Examples**:

```bash
# Generate and analyze a plan
terraform plan -out=plan.bin
terraform show -json plan.bin > plan.json
tfskel review plan --json-file plan.json

# Show top 5 highest-impact changes
tfskel review plan --json-file plan.json --top-resources-count 5

# Export as JSON for automation
tfskel review plan --json-file plan.json --format json

# Export as CSV for reporting
tfskel review plan --json-file plan.json --format csv > changes.csv
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

Workflow generation is split into **static files** (copied as-is by `tfskel init --workflows`) and a **rendered per-env template** (rendered by `tfskel scaffold workflows --env <env>`).

##### Static Shared Workflows (generated by `tfskel init --workflows`)

These files are embedded in the binary and copied without rendering:

- [`lint.yaml`](../internal/templates/files/github/lint.yaml) - Global lint caller; uses `reusable-detect-changes.yaml` to find changed app directories, then fans out to `reusable-lint.yaml`
- [`reusable-detect-changes.yaml`](../internal/templates/files/github/reusable-detect-changes.yaml) - Detects which Terraform app directories have changed files on a PR
- [`reusable-lint.yaml`](../internal/templates/files/github/reusable-lint.yaml) - Shared linting logic (TFLint, validate, terraform-docs)
- [`reusable-terraform-plan-apply.yaml`](../internal/templates/files/github/reusable-terraform-plan-apply.yaml) - Shared Terraform plan/apply logic with AWS OIDC auth

##### Per-Environment Terraform Workflow (`terraform-plan-apply.yaml.tmpl`)

Generates a per-environment Terraform plan/apply caller:

- [`terraform-plan-apply.yaml.tmpl`](../internal/templates/files/github/terraform-plan-apply.yaml.tmpl)

**Template Variables**:
- `Env`: Environment name
- `AWSRoleArn`: AWS IAM role ARN for authentication
- `WorkflowFileName`: Auto-generated workflow filename (e.g. `dev-terraform.yaml`)

**Features**:
- Runs on PR for plan, on push to main for apply
- Self-referencing trigger path (workflow re-runs when itself changes)
- AWS OIDC authentication with configurable role
- Manual dispatch with full parameter control
- `auto_apply` parameter controls PR auto-apply (default: false for safety)
- Calls `reusable-terraform-plan-apply.yaml` for shared Terraform logic

### Custom Templates

You can add custom templates for additional files using the `--templates-dir` flag or `templates.dir` config in `.tfskel.yaml`:

```bash
tfskel scaffold myapp --env dev --region us-east-1 --templates-dir ./custom-templates
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

tfskel uses **metadata comments** embedded in generated files to track configuration values and detect when updates are needed. When you run `tfskel scaffold` on existing directories, tfskel reads these metadata comments and automatically regenerates files if configuration has changed.

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
| `WorkflowFileName` | `string` | Generated per-env workflow filename (GitHub workflows only) | `dev-terraform.yaml` |

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
- Source: First positional argument to `tfskel scaffold <app-dir>`
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
- Source: Derived from `workflows.aws_role_arn` or `aws_role_name` in `.tfskel.yaml`
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
- Source: Auto-generated based on `workflows.name` or default pattern
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
- If `AccountID` is missing or invalid for environment, generation fails with descriptive error message showing available environments
- `AWSRoleArn` defaults to placeholder if not configured

**Validation Guarantees**:
- `AccountID` is always a valid 12-digit number when templates are rendered
- `S3BucketName` is never empty or placeholder value in generated files
- All template variables are validated before rendering begins

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

### Dry Run Mode

The `--dry-run` global flag lets you preview what tfskel would do without writing any files to disk. This is useful for auditing changes before committing, testing upgrade paths, or understanding what a command will produce.

```bash
# Preview init
tfskel init --dry-run

# Preview scaffold
tfskel scaffold myapp --env dev --region us-east-1 --dry-run

# Preview upgrade
tfskel scaffold myapp --env dev --region us-east-1 --upgrade --dry-run

# Combine with --verbose for maximum detail
tfskel scaffold myapp --env dev --region us-east-1 --dry-run --verbose
```

**How it works**:

- For `scaffold` and `scaffold workflows`, tfskel wraps the real filesystem with a `DryRunFileSystem` decorator. This decorator delegates all **read** operations (so upgrade checks, source marker extraction, and hash comparisons still work correctly) but makes all **write** operations (`WriteFile`, `MkdirAll`) no-ops.
- For `init`, each file-writing function checks the dry-run flag and logs the intended action with a `[dry-run]` prefix instead of writing.
- All log messages use future tense in dry-run mode (e.g. `[dry-run] Would create backend.tf from tf/backend.tf.tmpl`).
- The operation summary also reflects dry-run mode (e.g. `2 files would be created, 1 file skipped`).

### Operation Summary

After each `scaffold` or `scaffold workflows` run, tfskel prints a one-line summary of all file operations:

```
Summary: 3 files created, 2 files skipped
```

The summary tracks five operation types:

| Operation | Meaning |
|---|---|
| **created** | New file written from a template |
| **skipped** | Existing file left unchanged (already exists or up to date) |
| **upgraded** | Existing file re-rendered because the template or config changed |
| **force-upgraded** | Existing file overwritten via `--force` (no source marker or invalid marker) |
| **dir created** | New directory created (tracked internally but not shown in summary) |

In `--dry-run` mode, verbs switch to future tense: `would be created`, `would be upgraded`, `would be force-upgraded`.

### Config Source Debugging

When using `--verbose`, tfskel logs where each configuration value came from at debug level. This helps troubleshoot viper's merge behavior when flags, environment variables, and config files interact:

```bash
tfskel scaffold myapp --env dev --region us-east-1 --verbose
# [DEBUG] Config terraform_version = ~> 1.13 (from config file .tfskel.yaml)
# [DEBUG] Config backend.s3.bucket_name = my-bucket (from flag --s3-bucket-name)
# [DEBUG] Config provider.aws.version = ~> 6.0 (from env TFSKEL_PROVIDER_AWS_VERSION)
```

**Source detection priority** (highest to lowest):
1. CLI flag (`--s3-bucket-name`, `--templates-dir`, `--workflows`)
2. Environment variable (`TFSKEL_TERRAFORM_VERSION`, `TFSKEL_BACKEND_S3_BUCKET_NAME`, etc.)
3. Config file (`.tfskel.yaml`)
4. Default value

### Color Detection

tfskel automatically determines whether to use colored output based on the following precedence:

1. `NO_COLOR` env var — always disables colors if set (per [no-color.org](https://no-color.org/))
2. `FORCE_COLOR` env var — enables colors if set to a non-zero/non-false value
3. `CI` env var — disables colors when set to `true` or `1` (set by GitHub Actions, GitLab CI, Jenkins, etc.)
4. `--no-color` flag — used as fallback if no environment variables are set

### Upgrading Generated Files

tfskel supports re-rendering previously generated files when templates or configuration change, using the `--upgrade` flag available on `init`, `scaffold`, and `scaffold workflows`.

#### Source Markers

Every file generated by tfskel (except `.terraform-version`) includes a **source marker** as its first line:

```hcl
## tfskel-source: {"template":"tf/backend.tf.tmpl","hash":"a1b2c3d4e5f6..."}
```

For YAML files the comment prefix is `#` instead of `##`:

```yaml
# tfskel-source: {"template":"github/lint.yaml","hash":"f6e5d4c3b2a1..."}
```

The marker records:
- **template**: Which template produced the file (full template key, e.g. `tf/backend.tf.tmpl`)
- **hash**: First 16 hex characters of the SHA-256 hash of the raw template/static file content at generation time

#### How `--upgrade` Works

1. **Read existing file** and extract its source marker
2. **Skip** if no source marker is found (unless `--force` is used)
3. **Verify** the template name matches what is expected
4. **Compare template hash** — the marker's hash is compared against the current template hash to determine whether the template source itself changed
5. **Re-render** the template with current configuration
6. **Compare** the newly rendered content against the existing file on disk to detect drift
7. **Overwrite** the file only if the rendered content differs from what is currently on disk
8. **Log the reason** — messages now distinguish between template changes (`template: a1b2c3 -> d4e5f6`) and content drift (`content drift detected`)

This means `--upgrade` is safe to run repeatedly — files whose contents already match the current templates are left untouched. In `--dry-run` mode, all upgrade operations are logged with `[dry-run] Would upgrade` prefixes without modifying files.

#### Using `--force`

`--force` must always be combined with `--upgrade`. It overrides the source-marker check and re-renders files unconditionally. This is useful for:

- Files generated before source markers were introduced
- Files where the source marker was accidentally removed
- Bulk re-rendering of all managed files

```bash
# Error: --force cannot be used alone
tfskel scaffold myapp --env dev --region us-east-1 --force  # ✗

# Correct: combine with --upgrade
tfskel scaffold myapp --env dev --region us-east-1 --upgrade --force  # ✓
```

#### Selective Upgrade with `templates.upgrade`

By default, all templates with source markers are eligible for `--upgrade`. You can restrict this to a whitelist in `.tfskel.yaml`:

```yaml
templates:
  dir: ./custom-templates
  upgrade:
    - backend.tf.tmpl
    - versions.tf.tmpl
```

When `templates.upgrade` is specified, only the listed templates will be re-rendered. All other files are skipped even if they have source markers. When empty or absent, all templates are eligible.

#### Upgrade Across Commands

| Command | What gets upgraded |
|---|---|
| `tfskel init --upgrade` | Init-managed files: `.gitignore`, `.pre-commit-config.yaml`, `.tflint.hcl`, `trivy.yaml`, shared workflow files |
| `tfskel scaffold --upgrade` | Scaffolded app files: `backend.tf`, `versions.tf`, custom template outputs |
| `tfskel scaffold workflows --upgrade` | Per-environment GitHub Actions workflow files |

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

# Scaffold with custom templates
tfskel scaffold myapp --env dev --region us-east-1 --templates-dir custom-templates
```

**Supported Custom Templates**:
- Any file ending in `.tmpl` (all template files are processed automatically)

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
tfskel scaffold webapp --env prd --region us-east-1
tfskel scaffold webapp --env prd --region us-west-2

# Deploy to EU regions
tfskel scaffold webapp --env prd --region eu-central-1
tfskel scaffold webapp --env prd --region eu-west-1

# Deploy to Asia-Pacific
tfskel scaffold webapp --env prd --region ap-south-1
tfskel scaffold webapp --env prd --region ap-southeast-1
```

### Version Drift Management

Detect and fix version inconsistencies:

```bash
# Run all validation checks
tfskel validate

# Export validation report to CSV
tfskel validate --format csv --no-color > validate-report.csv

# Skip tool checks, only check config drift
tfskel validate --skip tools

# Output as JSON for automated processing
tfskel validate --format json > validate.json
```

**Common Drift Scenarios**:

1. **Version drift**: Different versions across environments
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
