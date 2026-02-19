# Changelog

## [0.1.1](https://github.com/ishuar/tfskel/compare/v0.2.0...v0.1.1) (2026-02-19)


### ⚠ BREAKING CHANGES

* **generate:** explicit allow all files as templates with .tmpl extension ([#20](https://github.com/ishuar/tfskel/issues/20))
* **generate:** removed extra_template_extensions config ; added warning log message regarding removal of extra_template_extensions
* **generate:** Configuration structure has changed. The templates_dir and extra_template_extensions settings must now be nested under the generate configuration block in .tfskel.yaml.

### ✨ Features

* Added drift subcommands for comprehensive infrastructure analysis ([d9648a3](https://github.com/ishuar/tfskel/commit/d9648a336c0a4278eee1671e151ab6b603901543))
* **drift:** add shared infrastructure for drift commands ([469d5fd](https://github.com/ishuar/tfskel/commit/469d5fdd5c7d683797c022ec45d585dfddb76196))
* **drift:** added critical resources default list and over-ride via viper config ([f272220](https://github.com/ishuar/tfskel/commit/f27222058236864137d6e7796520bac847651550))
* **drift:** implement 'drift all' subcommand ([6044a19](https://github.com/ishuar/tfskel/commit/6044a1935d28a3a8d2a5ca36a6439145432daf58))
* **drift:** implement 'drift plan' subcommand ([ab6cf2d](https://github.com/ishuar/tfskel/commit/ab6cf2df77ae4c81b51c636ed924bb59534da88b))
* **drift:** implement 'drift version' subcommand ([6f07bdb](https://github.com/ishuar/tfskel/commit/6f07bdb42a40cfdb2787328e43d92847171f5645))
* **generate:** accept templated strings for aws role name & arn values ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** added github workflows generation ([fed1620](https://github.com/ishuar/tfskel/commit/fed162069db3708bad62b46b0aa09054c5842a8e))
* **generate:** added optional github workflow generation with scaffolding tf stack ([f147a8e](https://github.com/ishuar/tfskel/commit/f147a8e6fcd7fb6d41506735235b17b9831b5ec0))
* **generate:** explicit allow all files as templates with .tmpl extension ([#20](https://github.com/ishuar/tfskel/issues/20)) ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** moved templates_dir & extra_template_extensions under generate config block ([9e33472](https://github.com/ishuar/tfskel/commit/9e334721308429b9f65fd2f85dde58fcd0f06404))
* **generate:** removed extra_template_extensions config ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** the renderer accepts .tmpl exclusively only for templated files ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* initial commit (added tfskel) ([3dd38ec](https://github.com/ishuar/tfskel/commit/3dd38ec2267e9cb89c8df11d35a43fb25d459b9a))


### 🐞 Bug Fixes

* add release-please annotation to have correct release ([07efcf9](https://github.com/ishuar/tfskel/commit/07efcf97502ecb06f8d4b95c68a40ddc17aa584f))
* added short `-c` version for --config flag ([a306b1e](https://github.com/ishuar/tfskel/commit/a306b1ea239da22699c00a89e9880a7bdac27819))
* address Copilot review feedback ([f5dc283](https://github.com/ishuar/tfskel/commit/f5dc283faea8053c2f36af24085a68e8c2c48d01))
* disable draft release in release please ([07d6fbf](https://github.com/ishuar/tfskel/commit/07d6fbf3d07a4563a31cdc7f7dad14211f941bd9))
* disable draft release in release please ([#27](https://github.com/ishuar/tfskel/issues/27)) ([07d6fbf](https://github.com/ishuar/tfskel/commit/07d6fbf3d07a4563a31cdc7f7dad14211f941bd9))
* **drift:** configure `top_n_count` via config file for over-riding ([5a8c0d8](https://github.com/ishuar/tfskel/commit/5a8c0d84ee849a52e2ebb3c4a8b70a5f2536660b))
* **drift:** version table alignment ([8fe101d](https://github.com/ishuar/tfskel/commit/8fe101d5092638716e1b3b208f83bac891bd655d))
* **generate:** correct upload-artifact action version in re-usable tf workflow ([d88a915](https://github.com/ishuar/tfskel/commit/d88a9151251871e6415afac89284967b8d922adf))
* **generate:** default_tags metadata based generation and improved logging ([7fc09de](https://github.com/ishuar/tfskel/commit/7fc09de446434a7691552fca1f5f9185671b47e1))
* **generate:** re-usable workflow,  refactored sanitize function and more tests to generator ([626afee](https://github.com/ishuar/tfskel/commit/626afeeca647833caada79bb196e744dc9a9b8fd))
* make golanci-lint happier ([c1e5527](https://github.com/ishuar/tfskel/commit/c1e552760588e71bbf3409a78db06bc6d303a5b4))
* resolve golangci-lint errors ([6964e81](https://github.com/ishuar/tfskel/commit/6964e81fe859683b0d81b8b51a4cc6c8e752a00c))
* version output  ([#16](https://github.com/ishuar/tfskel/issues/16)) ([d1a484d](https://github.com/ishuar/tfskel/commit/d1a484dab1ee1cd073d105b4f7ae7f9cf43d00a3))


### 🔧 Miscellaneous

* clean goreleaser and removed version 0.1.0 changelogs as it is not available ([cf504f6](https://github.com/ishuar/tfskel/commit/cf504f665b64614e58641e2abe7770bcc82346dc))


### 🧪 Tests

* added more tests ([c03e319](https://github.com/ishuar/tfskel/commit/c03e3199282d35eed6d5f243da59b2ad1b1156f3))
* **config:** added tests for validating latest .tfskel viper config ([d2b14d4](https://github.com/ishuar/tfskel/commit/d2b14d40480cdfba3334b1d07eaf4c0248c7a44a))
* removed unwanted comma ([22f4e0f](https://github.com/ishuar/tfskel/commit/22f4e0f6ee7e85c02a0fe103da684289fb6761ef))
* update formatter tests for consistent table width implementation ([20dafdd](https://github.com/ishuar/tfskel/commit/20dafddf0e9d508c5303ff486dad7b6156c33fb4))
* use ElementsMatch instead of Equal to avoid ordering ([7e44125](https://github.com/ishuar/tfskel/commit/7e44125936c52d43a8f90100e0c451f86d0a716f))

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
