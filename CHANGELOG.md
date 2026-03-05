# Changelog

## [0.3.0](https://github.com/ishuar/tfskel/compare/v0.2.2...v0.3.0) (2026-03-05)


### ⚠ BREAKING CHANGES

* **generate:** provider.aws.account_mapping values must now be valid 12-digit AWS account IDs and backend.s3.bucket_name can no longer be empty or use the placeholder.
    #### Details
    - provider.aws.account_mapping[key].value is now mandatory and must be a valid 12-digit numeric AWS account ID.
    - backend.s3.bucket_name can no longer be empty or use the default placeholder value.
* **template:** rename enable_apply_on_pr input to auto_apply and fixed reusable terraform plan-apply workflow ([#37](https://github.com/ishuar/tfskel/issues/37))
The input `enable_apply_on_pr` has been renamed to `auto_apply` in the Terraform plan-apply reusable workflow. All caller workflows must replace `enable_apply_on_pr` with `auto_apply`. The GitHub environment input dependency has been removed from the Terraform auto-apply logic. Auto-apply now depends only on `inputs.auto_apply` and will always run on push to `main`.

### ✨ Features

* **template:** rename enable_apply_on_pr input to auto_apply and fixed reusable terraform plan-apply workflow ([#37](https://github.com/ishuar/tfskel/issues/37)) ([2b34394](https://github.com/ishuar/tfskel/commit/2b343948e426acb99f30d4d55d237e2e379e239d))
* **template:** use tfskel for easier plan review in terraform plan-apply re-usable workflow ([2b34394](https://github.com/ishuar/tfskel/commit/2b343948e426acb99f30d4d55d237e2e379e239d))


### 🐞 Bug Fixes

* **generate:** require 12-digit AWS account ID and explicit S3 bucket name ([#39](https://github.com/ishuar/tfskel/issues/39)) ([fff6760](https://github.com/ishuar/tfskel/commit/fff67606e1062d530801339ddeea25ac057d0a51))
* **template:** added workflow permissions when not enabled on repo and inherit secrets ([2b34394](https://github.com/ishuar/tfskel/commit/2b343948e426acb99f30d4d55d237e2e379e239d))
* **template:** respects workflow_dispatch input on auto_apply ([2b34394](https://github.com/ishuar/tfskel/commit/2b343948e426acb99f30d4d55d237e2e379e239d))

## [0.2.2](https://github.com/ishuar/tfskel/compare/v0.2.1...v0.2.2) (2026-02-27)


### 🐞 Bug Fixes

* **help:** improve help outputs for all commands ([#35](https://github.com/ishuar/tfskel/issues/35)) ([da8c500](https://github.com/ishuar/tfskel/commit/da8c500011b414c96e48767c434a48aad3fb37db))

## [0.2.1](https://github.com/ishuar/tfskel/compare/v0.2.0...v0.2.1) (2026-02-25)


### 🐞 Bug Fixes

* disable draft release in release please ([07d6fbf](https://github.com/ishuar/tfskel/commit/07d6fbf3d07a4563a31cdc7f7dad14211f941bd9))
* disable draft release in release please ([#27](https://github.com/ishuar/tfskel/issues/27)) ([07d6fbf](https://github.com/ishuar/tfskel/commit/07d6fbf3d07a4563a31cdc7f7dad14211f941bd9))
* **generate:** accept nested value for app-dir ([#32](https://github.com/ishuar/tfskel/issues/32)) ([e3befff](https://github.com/ishuar/tfskel/commit/e3befff2f4700aea3ca5fe44f79aa2af2b8b5746))


### 🧪 Tests

* added tests for nested app-dir value ([e3befff](https://github.com/ishuar/tfskel/commit/e3befff2f4700aea3ca5fe44f79aa2af2b8b5746))

## [0.2.0](https://github.com/ishuar/tfskel/compare/v0.1.1...v0.2.0) (2026-02-19)


### ⚠ BREAKING CHANGES

* **generate:** explicit allow all files as templates with .tmpl extension ([#20](https://github.com/ishuar/tfskel/issues/20))
* **generate:** removed extra_template_extensions config ; added warning log message regarding removal of extra_template_extensions

### ✨ Features

* **generate:** accept templated strings for aws role name & arn values ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** explicit allow all files as templates with .tmpl extension ([#20](https://github.com/ishuar/tfskel/issues/20)) ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** removed extra_template_extensions config ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** the renderer accepts .tmpl exclusively only for templated files ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))


### 🐞 Bug Fixes

* **generate:** correct upload-artifact action version in re-usable tf workflow ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))

## 0.1.1 (2026-02-18)

## Breaking Change
* **generate**: namespaced configs for generate block & fixed metadata based default_tags by @ishuar in https://github.com/ishuar/tfskel/pull/12
  * moved `templates_dir` & `extra_template_extensions` under generate config block
## What's Changed
* docs: added docs , updated example config and changelog showing unreleased changes before 0.0.1 by @ishuar in https://github.com/ishuar/tfskel/pull/11
* chore: Create review instructions for Copilot role by @ishuar in https://github.com/ishuar/tfskel/pull/13
* fix: add release-please annotation to have correct release in [07efcf9](https://github.com/ishuar/tfskel/commit/07efcf97502ecb06f8d4b95c68a40ddc17aa584f)

**Full Changelog**: https://github.com/ishuar/tfskel/compare/v0.0.1...v0.1.1

## [0.0.1](https://github.com/ishuar/tfskel/releases/tag/v0.0.1) (2026-02-15)


### Features

* Added drift subcommands for comprehensive infrastructure analysis ([d9648a3](https://github.com/ishuar/tfskel/commit/d9648a336c0a4278eee1671e151ab6b603901543))
* **drift:** add shared infrastructure for drift commands ([469d5fd](https://github.com/ishuar/tfskel/commit/469d5fdd5c7d683797c022ec45d585dfddb76196))
* **drift:** added critical resources default list and over-ride via viper config ([f272220](https://github.com/ishuar/tfskel/commit/f27222058236864137d6e7796520bac847651550))
* **drift:** implement 'drift all' subcommand ([6044a19](https://github.com/ishuar/tfskel/commit/6044a1935d28a3a8d2a5ca36a6439145432daf58))
* **drift:** implement 'drift plan' subcommand ([ab6cf2d](https://github.com/ishuar/tfskel/commit/ab6cf2df77ae4c81b51c636ed924bb59534da88b))
* **drift:** implement 'drift version' subcommand ([6f07bdb](https://github.com/ishuar/tfskel/commit/6f07bdb42a40cfdb2787328e43d92847171f5645))
* **generate:** added github workflows generation ([fed1620](https://github.com/ishuar/tfskel/commit/fed162069db3708bad62b46b0aa09054c5842a8e))
* **generate:** added optional github workflow generation with scaffolding tf stack ([f147a8e](https://github.com/ishuar/tfskel/commit/f147a8e6fcd7fb6d41506735235b17b9831b5ec0))
* initial commit (added tfskel) ([3dd38ec](https://github.com/ishuar/tfskel/commit/3dd38ec2267e9cb89c8df11d35a43fb25d459b9a))


### Bug Fixes

* added short `-c` version for --config flag ([a306b1e](https://github.com/ishuar/tfskel/commit/a306b1ea239da22699c00a89e9880a7bdac27819))
* address Copilot review feedback ([f5dc283](https://github.com/ishuar/tfskel/commit/f5dc283faea8053c2f36af24085a68e8c2c48d01))
* **drift:** configure `top_n_count` via config file for over-riding ([5a8c0d8](https://github.com/ishuar/tfskel/commit/5a8c0d84ee849a52e2ebb3c4a8b70a5f2536660b))
* **drift:** version table alignment ([8fe101d](https://github.com/ishuar/tfskel/commit/8fe101d5092638716e1b3b208f83bac891bd655d))
* **generate:** re-usable workflow,  refactored sanitize function and more tests to generator ([626afee](https://github.com/ishuar/tfskel/commit/626afeeca647833caada79bb196e744dc9a9b8fd))
* make golanci-lint happier ([c1e5527](https://github.com/ishuar/tfskel/commit/c1e552760588e71bbf3409a78db06bc6d303a5b4))
* resolve golangci-lint errors ([6964e81](https://github.com/ishuar/tfskel/commit/6964e81fe859683b0d81b8b51a4cc6c8e752a00c))

[Unreleased]: https://github.com/ishuar/tfskel/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/ishuar/tfskel/releases/tag/v0.0.1



## [Unreleased]

### Features

* CLI framework with init, generate, and drift commands for Terraform project management
  * **init**: Project initialization with pre-commit hooks, TFLint, Trivy, and terraform .gitignore configuration
  * **drift**: Drift detection for Terraform and provider version inconsistencies
  * **drift**: Multiple output formats (table, JSON, CSV) with color-coded status and terminal-aware formatting
  * **generate**: Rich template function library for string manipulation and version utilities
  * **generate**: Template-driven file generation with embedded templates and custom template override support
  * **generate**: AWS provider `default_tags` management with metadata-based change detection and idempotent operations
  * **generate**: Multi-region AWS support with environment-specific account mapping
* YAML-based configuration (.tfskel.yaml) with CLI flags, config file, and interactive prompt precedence
* Structured logging with multiple levels and verbose mode support
