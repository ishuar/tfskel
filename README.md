<div align="center">
<!-- <img src="assets/tfskel-logo-more-width.png" alt="tfskel logo" width="500" /> -->

[![Test][test-img]][test]
[![GitHub Release][release-img]][release]
[![License: MIT][license-img]][license]
[![Stargazers][stars-shield]][stars-url]
[![Go Report Card][go-report-img]][go-report]
[![codecov](https://codecov.io/gh/ishuar/tfskel/graph/badge.svg?token=66VT000UYO)](https://codecov.io/gh/ishuar/tfskel)
[![Go Version][go-version-img]][go-version]

<p align="center">
<img align="center" src="assets/tfskel-logo.svg?raw=true" alt="tfskel logo"  style="height: 125px" />
</p>

</div>

# tfskel
_Opinionated Terraform scaffolding. No vendor lock-in, just better project structure_

`tfskel` is a CLI tool that scaffolds Terraform monorepos with an **opinionated**, **scalable** and **consistent** way by using environment-based directory structure across multiple regions. No wrappers, no complexity, just vanilla Terraform with consistent terraform root modules, version **drift detection**,and **terraform plan analysis**. Spend less time on project setup and more time writing infrastructure code.

**⭐️ For Latest updates Don't forget to star the repo! ⭐️**

## Why tfskel

Setting up a Terraform monorepo the right way takes time — defining consistent directory layouts, pinning provider versions, and keeping all of it in sync as the project grows. `tfskel` handles that scaffolding work so you don't have to repeat it for every project or environment while using only native terraform.

### Features
1. Consistent Structure, Every Time
2. Terraform Code Generation via Go Templates
3. Version Drift Detection Across the Entire Repo
4. Terraform Plan Analysis
5. No Wrappers, No Lock-in Just terraform.


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
### `tfskel --help`

Help and available commands

```bash
tfskel --help
```

### `tfskel init`

This creates initializes a new Terraform monorepo with an environment-and-region-based directory layout (`dev`/`stg`/`prd`) with sensible defaults already in place — `.gitignore`, `.pre-commit-config.yaml`, `.tflint.hcl`, `trivy.yaml`, and per-environment `.terraform-version` files. The same structure, every time, across every project.

```bash
# Or specify a custom directory
tfskel init --dir /path/to/your/project
```

<p align="left">
  <img src="assets/tfskel-init.gif" alt="tfskel init demo" width="600" />
</p>

### `tfskel generate`

This creates per-application root module directories with a pre-configured `backend.tf` (S3 with state locking and encryption) and a `versions.tf` with pinned Terraform, AWS provider versions and optional github terraform workflows. You can extend this with your own `.tmpl` files — any custom template you place in your templates directory is processed alongside the built-in defaults, where if same name provided custom template will take precedence.

```bash
tfskel generate myapp --env dev --region us-east-1
## custom templates directory via cmd arguments, otherwise use .tfskel.yaml config else default templates
tfskel generate myapp --env dev --region us-east-1 --templates-dir <path-to-templates-dir>
```

<p align="left">
<img src="assets/tfskel-generate.gif" alt="tfskel generate demo" width="600" />
</p>

### `tfskel drift`

tfskel drift has three subcommands `version`, `plan` & `all`

####  `tfskel drift version`
This scans all environments in the repository and reports Terraform and AWS provider version inconsistencies in one pass from the current directory. Results can be output as JSON,table or csv for use in CI/CD pipelines or automated checks.

```bash
tfskel drift version
## with target path
tfskel drift version --path /path/to/your/terraform-directories
```
<p align="left">
<img src="assets/tfskel-drift-version.gif" alt="tfskel drift version demo" width="600" />
</p>

#### `tfskel drift plan`

This reads a `plan.json` file and summaries resource changes, flags high-severity updates based on a configurable list of critical resources, and produces a structured report. Output can be exported as CSV, table and json for reporting.

```bash
# Analyze plan after terraform plan -out=plan.bin
terraform plan -out plan.bin
terraform show -json plan.bin > plan.json
tfskel drift plan --plan-file plan.json

# Export as CSV for reporting
tfskel drift plan --plan-file plan.json --format csv
```

<p align="left">
<img src="assets/tfskel-drift-plan.gif" alt="tfskel drift plan demo" width="600" />
</p>

#### `tfskel drift all`
Run both checks together with `--path` and `--plan-file` argument and provide summarized output.

```bash
# Run both version drift and plan analysis
tfskel drift all --plan-file plan.json --path /path/to/your/terraform-directories
```

## Configuration

Create a `.tfskel.yaml` in your project root to customize defaults:

> [!TIP]
> Use [.tfskel.example.yaml.example](.tfskel.example.yaml) for reference.
> Configuration precedence: CLI flags → config file → defaults

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
