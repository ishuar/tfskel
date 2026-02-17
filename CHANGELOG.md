# Changelog

## [0.1.0](https://github.com/ishuar/tfskel/compare/v0.0.1...v0.1.0) (2026-02-17)


### ⚠ BREAKING CHANGES

* **generate:** Configuration structure has changed. The `templates_dir` and `extra_template_extensions` settings must now be nested under the `generate` configuration block in .tfskel.yaml.

### Features

* **generate:** moved `templates_dir` & `extra_template_extensions` under generate config block ([672d7fc](https://github.com/ishuar/tfskel/commit/672d7fc8f8d9c127a767313787a258453afd9d06))


### Bug Fixes

* add release-please annotation to have correct release ([58fa71a](https://github.com/ishuar/tfskel/commit/58fa71a759f8f78e579f0bdb063a5eeae0a4296b))
* **generate:** `default_tags` metadata based generation and improved logging ([5515511](https://github.com/ishuar/tfskel/commit/5515511f8857aca0101998b65218ce7adf5239d9))

## [0.1.0](https://github.com/ishuar/tfskel/compare/v0.0.1...v0.1.0) (2026-02-17)


### ⚠ BREAKING CHANGES

* **generate:** Configuration structure has changed. The `templates_dir` and `extra_template_extensions` settings must now be nested under the `generate` configuration block in .tfskel.yaml.

### Features

* **generate:** moved `templates_dir` & `extra_template_extensions` under generate config block ([672d7fc](https://github.com/ishuar/tfskel/commit/672d7fc8f8d9c127a767313787a258453afd9d06))


### Bug Fixes

* **generate:** `default_tags` metadata based generation and improved logging ([5515511](https://github.com/ishuar/tfskel/commit/5515511f8857aca0101998b65218ce7adf5239d9))

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
