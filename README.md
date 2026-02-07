[![Go Version][go-version-img]][go-version]
[![GitHub Release][release-img]][release]
[![Test][test-img]][test]
[![Go Report Card][go-report-img]][go-report]
[![License: MIT][license-img]][license]

# tfskel

**Opinionated Terraform scaffolding for real-world teams.**

tfskel is a CLI tool that scaffolds Terraform monorepos with an **opinionated**, **scalable** and **consistent** way by using environment-based directory structure across multiple regions. No wrappers, no complexity—just vanilla Terraform with consistent backend configs, version **drift detection**, and sensible defaults. Spend less time on project setup and more time writing infrastructure code.

## What It Does

- Scaffolds complete Terraform monorepo structures organized by environment (dev/stg/prd) and region
- Generates AWS provider & backend configuration and custom terraform configs using re-usable go templates.
- Drift Detection on Terraform and AWS provider versions across your entire repository
- Safe to run multiple times—only creates new files, never overwrites existing ones
- Uses simple YAML configuration with smart defaults
- Works with vanilla Terraform—no custom wrappers or proprietary tooling

## Installation

```bash
# Install via Go
go install github.com/ishuar/tfskel@latest

# Or download from releases
# https://github.com/ishuar/tfskel/releases
```

Make sure `$HOME/go/bin` is in your PATH.

## Quick Start

1. Help and available commands

```bash
tfskel --help
```
2. Initialize a new Terraform monorepo:

```bash
# Create project structure with default config
tfskel init

# Or specify a custom directory
tfskel init --dir /path/to/your/project

```
This creates an opinionated structure with environment directories and configuration files:

```
.
├── .tfskel.yaml             # Project configuration
├── .gitignore               # Terraform-specific ignores
├── .pre-commit-config.yaml  # Pre-commit hooks
├── .tflint.hcl             # Linting config
├── trivy.yaml              # Security scanning config
└── envs/
    ├── dev/
    │   ├── .terraform-version
    │   └── eu-central-1/
    ├── stg/
    │   ├── .terraform-version
    │   └── eu-central-1/
    └── prd/
        ├── .terraform-version
        └── eu-central-1/
```

3. Generate Terraform code for a specific application:

```bash
tfskel generate myapp --env dev --region us-east-1

```

4. Check drift for AWS provider and Terraform versions with respect to .tfskel.yaml config
```bash
# default is table format from current working directory
tfskel drift
```
## Usage Examples

**Deploy the same app across multiple regions:**
```bash
tfskel generate myapp --env prd --region us-east-1
tfskel generate myapp --env prd --region eu-west-1
tfskel generate myapp --env prd --region ap-south-1
```

**Use a custom configuration file:**
```bash
tfskel generate myapp --config ./my-config.yaml --env stg --region us-west-2
```

## Configuration

Create a `.tfskel.yaml` in your project root to customize defaults:

```yaml
terraform_version: ~> 1.13
templates_dir: "/path/to/your/templates-directory" # Custom templates_dir
extra_template_extensions: ["md.tmpl"] # by default .tf.tmpl templates are processed only
backend:
  s3:
    bucket_name: CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME
provider:
  aws:
    version: ~> 6.0
    default_tags:
      - generated_by: tfskel
    regions:
      - eu-central-1
    account_mapping:
      dev: "123456789012"
      prd: "210987654321"
      stg: "109876543210"
```
> [!TIP]
> Use [.tfskel.yaml.example](.tfskel.yaml.example) for reference.
> Configuration precedence: CLI flags → config file → defaults

See [docs/tfskel-book.md](docs/tfskel-book.md) for detailed configuration options.

## What Gets Generated

Running `tfskel generate` creates a complete Terraform module directory with backend and version configuration:

```bash
envs/dev/us-east-1/myapp/
├── backend.tf       # S3 backend with state locking & encryption enabled
└── versions.tf      # Terraform and provider versions
```

You can extend this by creating custom templates for additional files (`main.tf`, `variables.tf`, `outputs.tf`, etc.). Place templates in a directory and tfskel will use them alongside the defaults.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Development quick start:
```bash
git clone https://github.com/ishuar/tfskel.git
cd tfskel
make test          # Run tests
make check         # Run all quality checks
make install       # Install locally
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

[go-version-img]: https://img.shields.io/badge/Go-1.24%2B-blue.svg
[go-version]: https://golang.org
[test]: https://github.com/ishuar/tfskel/actions/workflows/test.yaml
[test-img]: https://github.com/ishuar/tfskel/actions/workflows/test.yaml/badge.svg
[go-report]: https://goreportcard.com/report/github.com/ishuar/tfskel
[go-report-img]: https://goreportcard.com/badge/github.com/ishuar/tfskel
[release]: https://github.com/ishuar/tfskel/releases
[release-img]: https://img.shields.io/github/release/ishuar/tfskel.svg?logo=github
[license]: https://github.com/ishuar/tfskel/blob/main/LICENSE
[license-img]: https://img.shields.io/badge/MIT-blue.svg
