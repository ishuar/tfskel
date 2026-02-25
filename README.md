<div align="center">
<!-- <img src="assets/tfskel-logo-more-width.png" alt="tfskel logo" width="500" /> -->

[![Test][test-img]][test]
[![GitHub Release][release-img]][release]
[![License: MIT][license-img]][license]
[![Stargazers][stars-shield]][stars-url]
[![Go Report Card][go-report-img]][go-report]
[![codecov](https://codecov.io/gh/ishuar/tfskel/graph/badge.svg?token=66VT000UYO)](https://codecov.io/gh/ishuar/tfskel)
[![Go Version][go-version-img]][go-version]

<div align="center">
<img src="assets/tfskel-logo.svg?raw=true" alt="tfskel logo" height="125" />

<em>Opinionated Terraform scaffolding. No vendor lock-in, just better project structure</em>
</div>

</div>

# tfskel

[`tfskel`](https://github.com/ishuar/tfskel) is a CLI tool that scaffolds Terraform monorepos with an **opinionated**, **scalable** and **consistent** way by using environment-based directory structure. No wrappers, no complexity, just vanilla Terraform with consistent terraform root modules, version **drift detection**,and **terraform plan analysis**. Spend less time on project setup and more time writing infrastructure code.

## Why tfskel

Tired of spending hours setting up the same Terraform folder structure, pinning provider versions, and keeping everything in sync as your infrastructure grows? You're not alone. Most teams waste valuable time reinventing the wheel for every new environment or region.

[`tfskel`](https://github.com/ishuar/tfskel) eliminates that pain. It gives you a proven, scalable monorepo layout—ready to go in seconds. No more copy-pasting, no more "did we forget that file?" moments. Just run the CLI and get a clean, consistent foundation for your Terraform code, every time.

### Features
1. Consistent Structure, Every Time
2. Terraform Code Generation via Go Templates
3. Version Drift Detection Across the Entire Repo
4. Terraform Plan Analysis
5. No Wrappers, No Lock-in Just terraform.

> [!NOTE]
> *⭐️ For Latest updates Don't forget to star the repo! ⭐️*

## Installation

```bash
# Install via Go
go install github.com/ishuar/tfskel@latest

# Or download from releases
# https://github.com/ishuar/tfskel/releases
```
Make sure `$HOME/go/bin` is in your PATH.

> [!CAUTION]
> This project is developed with AI assistance. Please review the code carefully and perform your own due diligence. Thank you 🙏

## Quick Start

### Global Flags

These flags are available on every `tfskel` command:

| Flag | Short | Default | Description |
|---|---|---|---|
| `--config` | `-c` | `.tfskel.yaml` (current dir) | Path to config file; takes precedence over auto-discovery |
| `--verbose` | `-v` | `false` | Enable verbose/debug output |
| `--version` | | | Print the current tfskel version and exit |

```bash
# Print the installed tfskel version
tfskel --version

# Show all available commands and flags
tfskel --help

# Use a custom config file for any command
tfskel --config /path/to/my-config.yaml <command>

# Enable verbose logging
tfskel --verbose <command>
```

### `tfskel init`

Initializes a new Terraform monorepo with an environment-and-region-based directory layout (`dev`/`stg`/`prd`) with sensible defaults already in place — `.gitignore`, `.pre-commit-config.yaml`, `.tflint.hcl`, `trivy.yaml`, `.tfskel.yaml`, and per-environment `.terraform-version` files. The same structure, every time, across every project.

If a `.tfskel.yaml` already exists in the target directory, `init` reads environments from `provider.aws.account_mapping` and regions from `provider.aws.regions`, so your existing configuration drives the scaffold.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--dir` | `-d` | current directory | Directory to initialize |

```bash
# Initialize in the current directory
tfskel init

# Initialize in a specific directory
tfskel init --dir /path/to/your/project

# Initialize with an explicit config file
tfskel init --config /path/to/config.yaml
```

<p align="left">
  <img src="assets/tfskel-init.gif" alt="tfskel init demo" width="600" />
</p>

### `tfskel generate`

Creates per-application root module directories with a pre-configured `backend.tf` (S3 with state locking and encryption) and a `versions.tf` with pinned Terraform and AWS provider versions, plus optional GitHub Actions workflows. You can extend this with your own `.tmpl` files — any custom template you place in your templates directory is processed alongside the built-in defaults; if the same filename is provided, the custom template takes precedence.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--env` | `-e` | *(required)* | Target environment (e.g. `dev`, `stg`, `prd`) |
| `--region` | `-r` | *(required)* | AWS region (e.g. `us-east-1`, `eu-central-1`) |
| `--templates-dir` | | `""` | Directory containing custom `.tmpl` template files |
| `--s3-bucket-name` | | `""` | S3 bucket name for Terraform state (overrides config) |
| `--create-github-workflows` | | `false` | Generate GitHub Actions workflow files from default templates |

```bash
# Generate structure for an app in dev/us-east-1
tfskel generate myapp --env dev --region us-east-1

# Generate with a custom templates directory
tfskel generate myapp --env stg --region eu-central-1 --templates-dir ./templates

# Override S3 bucket name at runtime
tfskel generate myapp --env prd --region us-east-1 --s3-bucket-name my-tf-state-bucket

# Also create GitHub Actions workflow files
tfskel generate myapp --env dev --region us-east-1 --create-github-workflows

# Generate with a custom config file
tfskel generate myapp --config ./my-config.yaml --env dev --region us-east-1
```

<p align="left">
<img src="assets/tfskel-generate.gif" alt="tfskel generate demo" width="600" />
</p>

> [!Note]
> `/` will be replaced with `-` in `<app-dir>` value for GitHub workflow file naming.

### `tfskel drift`

`tfskel drift` has three subcommands: `version`, `plan`, and `all`.

#### `tfskel drift version`

Scans all environments in the repository and reports Terraform and AWS provider version inconsistencies in one pass from the current directory. Results can be output as JSON, table, or CSV for use in CI/CD pipelines or automated checks.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--path` | `-p` | `.` (current dir) | Path to scan for Terraform files (recursive) |
| `--format` | `-f` | `table` | Output format: `table`, `json`, `csv` |
| `--no-color` | | `false` | Disable colored output |

**Exit codes:**

| Code | Meaning |
|---|---|
| `0` | No drift — all files in sync |
| `1` | Drift detected (minor or major version differences) |
| `2` | Parse errors encountered while scanning `.tf` files |

```bash
# Check for drift in current directory
tfskel drift version

# Check a specific subdirectory
tfskel drift version --path ./envs

# JSON output for CI/CD pipelines
tfskel drift version --path ./envs --format json

# Generate a CSV report without ANSI colors
tfskel drift version --format csv --no-color > drift-report.csv
```

<p align="left">
<img src="assets/tfskel-drift-version.gif" alt="tfskel drift version demo" width="600" />
</p>

#### `tfskel drift plan`

Reads a `plan.json` file, summarises resource changes, flags high-severity updates based on a configurable list of critical resources, and produces a structured report. Output can be exported as CSV, table, or JSON for reporting.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--plan-file` | | *(required)* | Path to the Terraform plan JSON file |
| `--format` | `-f` | `table` | Output format: `table`, `json`, `csv` |
| `--no-color` | | `false` | Disable colored output |

**Exit codes:**

| Code | Meaning |
|---|---|
| `0` | No changes — infrastructure is up to date |
| `1` | Non-critical changes detected (additions or modifications) |
| `2` | Critical changes detected (deletions or replacements) |

```bash
# Generate and analyze a Terraform plan
terraform plan -out plan.bin
terraform show -json plan.bin > plan.json
tfskel drift plan --plan-file plan.json

# Export as CSV for reporting
tfskel drift plan --plan-file plan.json --format csv

# JSON output without colors (suitable for log files)
tfskel drift plan --plan-file plan.json --format json --no-color
```

<p align="left">
<img src="assets/tfskel-drift-plan.gif" alt="tfskel drift plan demo" width="600" />
</p>

#### `tfskel drift all`

Run both version drift and plan analysis together and provide a unified summarised output. Ideal for pre-commit hooks, CI/CD pipelines, and code reviews.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--path` | `-p` | `.` (current dir) | Path to scan for Terraform files |
| `--plan-file` | | `""` | Path to Terraform plan JSON file (optional) |
| `--format` | `-f` | `table` | Output format: `table`, `json`, `csv` |
| `--no-color` | | `false` | Disable colored output |
| `--skip-plan` | | `false` | Skip plan analysis (run version drift only) |
| `--skip-versions` | | `false` | Skip version drift analysis (run plan analysis only) |

**Exit codes:**

| Code | Meaning |
|---|---|
| `0` | No issues found |
| `1` | Version drift or non-critical plan changes detected |
| `2` | Critical changes (deletions/replacements) or major version drift |

```bash
# Run both version drift and plan analysis
tfskel drift all --plan-file plan.json --path /path/to/your/terraform-directories

# Skip plan analysis — version drift only
tfskel drift all --skip-plan

# Skip version drift — plan only
tfskel drift all --plan-file plan.json --skip-versions

# CI/CD usage with JSON output and no colors
tfskel drift all --plan-file plan.json --format json --no-color
```

## Configuration

Create a `.tfskel.yaml` in your project root to customise defaults:

> [!TIP]
> Use [.tfskel.example.yaml](.tfskel.example.yaml) for a full annotated reference.
> Configuration precedence: **CLI flags → config file → defaults**

### Key configuration fields

| Field | Required | Default | Description |
|---|---|---|---|
| `terraform_version` | Yes (for default templates) | — | Terraform version constraint, e.g. `~> 1.13` |
| `provider.aws.version` | Yes (for default templates) | — | AWS provider version constraint, e.g. `~> 6.0` |
| `provider.aws.account_mapping` | Yes | — | Map of environment names → AWS account IDs |
| `provider.aws.regions` | Yes | — | List of AWS regions to scaffold |
| `provider.aws.default_tags` | No | `{}` | Default tags applied to all AWS resources |
| `backend.s3.bucket_name` | Yes | — | Globally unique S3 bucket name for Terraform state |
| `generate.templates_dir` | No | `""` | Path to custom templates directory |
| `generate.github_workflows.create` | No | `false` | Enable GitHub Actions workflow generation |
| `generate.github_workflows.name_template` | No | — | Workflow name template (supports Go template placeholders) |
| `generate.github_workflows.aws_role_arn` | No | — | Full IAM role ARN for GitHub Actions (takes priority) |
| `generate.github_workflows.aws_role_name` | No | — | IAM role name for GitHub Actions |
| `critical_resources` | No | `[]` | Additional resource types flagged as HIGH severity in drift plan analysis |
| `top_n_count` | No | `10` | Max rows shown in "Changes by Resource Type" / "Changes by Module" tables; `0` = show all |

## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are greatly appreciated.

If you have any suggestion that would make this project better, feel free to fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement" with your suggestion.

> [!Tip]
> See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1) Fork the repository on GitHub
2) Clone your fork and create a new branch
3) Make your changes
4) Run tests and checks
5) Push your branch and open a pull request

```bash
git clone https://github.com/<your-username>/tfskel.git
cd tfskel
git checkout -b my-feature-branch
make test   # Run tests
make check  # Run all quality checks
```

> [!Important]
> Please keep your pull requests small and focused. This will make it easier to review and merge.

## License
Released under [MIT LICENSE](/LICENSE)

<p align="right"><a href="#top">Back To Top ⬆️</a></p>

<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->

[go-version-img]: https://img.shields.io/badge/Go-1.24%2B-blue.svg
[go-version]: https://golang.org
[test]: https://github.com/ishuar/tfskel/actions/workflows/test.yaml
[test-img]: https://github.com/ishuar/tfskel/actions/workflows/test.yaml/badge.svg
[go-report]: https://goreportcard.com/report/github.com/ishuar/tfskel
[go-report-img]: https://goreportcard.com/badge/github.com/ishuar/tfskel
[release]: https://github.com/ishuar/tfskel/releases
[release-img]: https://img.shields.io/github/release/ishuar/tfskel.svg?logo=github
[license]: https://github.com/ishuar/tfskel/blob/main/LICENSE
[license-img]: https://img.shields.io/github/license/ishuar/tfskel?color=blue
[stars-url]: https://github.com/ishuar/tfskel/stargazers
[stars-shield]: https://img.shields.io/github/stars/ishuar/tfskel?style=flat&logo=github
