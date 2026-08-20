# Changelog

## [0.8.5](https://github.com/ishuar/tfskel/compare/v0.8.4...v0.8.5) (2026-08-20)


### ✨ Features

* **review:** AI narrative analysis with internal/review module ([#105](https://github.com/ishuar/tfskel/issues/105)) ([4425cb1](https://github.com/ishuar/tfskel/commit/4425cb15a98e12d665a09f7dc6c95dee25639b72))

## [0.8.4](https://github.com/ishuar/tfskel/compare/v0.8.3...v0.8.4) (2026-05-06)


### ✨ Features

* **scaffold:** add terraform_extra_secrets input to reusable workflow ([#103](https://github.com/ishuar/tfskel/issues/103)) ([1cfcd13](https://github.com/ishuar/tfskel/commit/1cfcd13540290b92189b5726e9c17048c8dd2f0d))


### 📦 Other Changes

* extract version line via grep in install success message ([#101](https://github.com/ishuar/tfskel/issues/101)) ([7465753](https://github.com/ishuar/tfskel/commit/7465753dae11a32a4fcdcc888dc907c9122950da))
* **scaffold:** install tfskel via install.sh in reusable workflow ([#104](https://github.com/ishuar/tfskel/issues/104)) ([3c19128](https://github.com/ishuar/tfskel/commit/3c19128a7688fb6bf3000be9fc489d7c8b6f956f))

## [0.8.3](https://github.com/ishuar/tfskel/compare/v0.8.2...v0.8.3) (2026-04-26)


### 🐞 Bug Fixes

* **scaffold:** prevent false content drift on .tf files after terraform fmt ([#98](https://github.com/ishuar/tfskel/issues/98)) ([7df7881](https://github.com/ishuar/tfskel/commit/7df7881794d17e96b57af2f05f950a25795fe091))
* **scaffold:** remove duplicate directory log that ignored existing dirs ([#100](https://github.com/ishuar/tfskel/issues/100)) ([c14c962](https://github.com/ishuar/tfskel/commit/c14c96200667a2484f6e8cd74b27800722399552))
* **scaffold:** use HTML comments for markdown metadata markers ([#97](https://github.com/ishuar/tfskel/issues/97)) ([39d80d7](https://github.com/ishuar/tfskel/commit/39d80d76a81537533cabd99039ac884dfd15b6df))

## [0.8.2](https://github.com/ishuar/tfskel/compare/v0.8.1...v0.8.2) (2026-04-23)


### ✨ Features

* add rich version command with build-origin detection ([#92](https://github.com/ishuar/tfskel/issues/92)) ([fe24345](https://github.com/ishuar/tfskel/commit/fe24345176b839b8235ab0df5a29bfe15fd4dcef))
* **validate:** add project context header to validate report ([#88](https://github.com/ishuar/tfskel/issues/88)) ([1475409](https://github.com/ishuar/tfskel/commit/1475409a9f1d0a096668576ab48f66deec3b8316))


### 🐞 Bug Fixes

* **init:** only install pre-commit hooks at git repo root ([#96](https://github.com/ishuar/tfskel/issues/96)) ([e560660](https://github.com/ishuar/tfskel/commit/e560660980e8c850375fe3193bb495cb46a72c9e))


### 📦 Other Changes

* add guidance against extracting trivial helpers ([#94](https://github.com/ishuar/tfskel/issues/94)) ([c106e1f](https://github.com/ishuar/tfskel/commit/c106e1fecdc500321fc4e91ae7b3b96da64299f4))
* align commit scopes with user-facing commands ([#91](https://github.com/ishuar/tfskel/issues/91)) ([29bc7ba](https://github.com/ishuar/tfskel/commit/29bc7bad96aadcb1d72142416adce957e8c40544))
* **init:** extract bootstrap logic into internal/bootstrap package ([#93](https://github.com/ishuar/tfskel/issues/93)) ([ea33468](https://github.com/ishuar/tfskel/commit/ea33468c99905310de4946c99f4f5ae3a86305bc))
* migrate cmd/ to factory-function pattern and tighten lint config ([#95](https://github.com/ishuar/tfskel/issues/95)) ([2651143](https://github.com/ishuar/tfskel/commit/26511432007891d636c978a0ef6614584ff2f094))

## [0.8.1](https://github.com/ishuar/tfskel/compare/v0.8.0...v0.8.1) (2026-04-15)


### ✨ Features

* **cmd:** add --upgrade-all and --skip flags to scaffold command ([#81](https://github.com/ishuar/tfskel/issues/81)) ([014f610](https://github.com/ishuar/tfskel/commit/014f6105ec947e5b2380fef1456c4a7fb325ebae))
* **cmd:** add debug log when workflow creation is skipped in init ([#85](https://github.com/ishuar/tfskel/issues/85)) ([4557b1f](https://github.com/ishuar/tfskel/commit/4557b1fd8f481ef6b9f6b6e09f5d7756f4f78ef5))
* **cmd:** make .gitignore user-owned and add --skip flag to init ([#84](https://github.com/ishuar/tfskel/issues/84)) ([a0cc578](https://github.com/ishuar/tfskel/commit/a0cc57828c720fcf566b03240d794c71d6aa05bc))


### 🐞 Bug Fixes

* **cmd:** validate now fails fast on invalid configs instead of proceeding with confusing downstream results ([aba3111](https://github.com/ishuar/tfskel/commit/aba3111e588bd2d6a173f93d452396733e9e1aea))


### 📦 Other Changes

* **cmd:** remove outdated breaking change note from validate help ([#86](https://github.com/ishuar/tfskel/issues/86)) ([8c56242](https://github.com/ishuar/tfskel/commit/8c56242f948bfb98a14f908d2d0b482ab00396cd))

## [0.8.0](https://github.com/ishuar/tfskel/compare/v0.7.2...v0.8.0) (2026-04-07)


### ⚠ BREAKING CHANGES

* **cmd:** `tfskel diff config` has been removed. Use `tfskel validate` instead.

### ✨ Features

* **cmd:** replace diff config with validate command ([#79](https://github.com/ishuar/tfskel/issues/79)) ([682a679](https://github.com/ishuar/tfskel/commit/682a679c7c0907dd8b9dd6b27ba5baf451b5c8eb))
* **toolcheck:** add mise-aware pre-flight checks with actionable shell hints ([8307d09](https://github.com/ishuar/tfskel/commit/8307d09c724b85222b5e8888a15384ffc8e1d317))


### 🐞 Bug Fixes

* **cmd:** improve validate accuracy, unique-resource counting, and error handling ([682a679](https://github.com/ishuar/tfskel/commit/682a679c7c0907dd8b9dd6b27ba5baf451b5c8eb))
* **cmd:** replace inline toolcheck in init with pointer to tfskel validate ([682a679](https://github.com/ishuar/tfskel/commit/682a679c7c0907dd8b9dd6b27ba5baf451b5c8eb))
* **cmd:** use segment-boundary version matching so 1.13 does not falsely match 1.130 ([682a679](https://github.com/ishuar/tfskel/commit/682a679c7c0907dd8b9dd6b27ba5baf451b5c8eb))
* **generator:** upgradeFile now compares full rendered content (matching `internal/generate/upgrade.go` behavior) ([8307d09](https://github.com/ishuar/tfskel/commit/8307d09c724b85222b5e8888a15384ffc8e1d317))


### 📦 Other Changes

* **cmd:** update docs, demo GIF, and CI for validate command ([682a679](https://github.com/ishuar/tfskel/commit/682a679c7c0907dd8b9dd6b27ba5baf451b5c8eb))
* improve installation section readability in README ([#80](https://github.com/ishuar/tfskel/issues/80)) ([be4900f](https://github.com/ishuar/tfskel/commit/be4900feb7fd812c45c05be83b87601f8bd0f720))

## [0.7.2](https://github.com/ishuar/tfskel/compare/v0.7.1...v0.7.2) (2026-03-28)


### ✨ Features

* **plan:** add severity and action filters for plan review ([#71](https://github.com/ishuar/tfskel/issues/71)) ([1a3290e](https://github.com/ishuar/tfskel/commit/1a3290e75995c07bee7073d7ef0920ad808c87af))
* **plan:** show total resource count when top-N truncates results ([#75](https://github.com/ishuar/tfskel/issues/75)) ([b8f5b03](https://github.com/ishuar/tfskel/commit/b8f5b0326730a736b899b93373c0ff1c26633037))


### 📦 Other Changes

* **cmd:** move SilenceUsage to command structs and centralize flag error handling ([#73](https://github.com/ishuar/tfskel/issues/73)) ([04f8a5a](https://github.com/ishuar/tfskel/commit/04f8a5aa3707310486d5f29c46f7f7bacf5714fc))

## [0.7.1](https://github.com/ishuar/tfskel/compare/v0.7.0...v0.7.1) (2026-03-27)


### ✨ Features

* **plan:** include terraform output changes in the plan review ([#65](https://github.com/ishuar/tfskel/issues/65)) ([439cba7](https://github.com/ishuar/tfskel/commit/439cba72f2538ba9d7a5d3745a7b12f49ee4a36d))


### 📦 Other Changes

* added an installation script to avoid go dependency ([#66](https://github.com/ishuar/tfskel/issues/66)) ([ca97d3f](https://github.com/ishuar/tfskel/commit/ca97d3f54450b5ae037ab2ec10066f488ba17f65))
* **review/plan:** bisected plan_formatter into dedicated components ([439cba7](https://github.com/ishuar/tfskel/commit/439cba72f2538ba9d7a5d3745a7b12f49ee4a36d))
* update scopes for contributing ([#69](https://github.com/ishuar/tfskel/issues/69)) ([0bbf033](https://github.com/ishuar/tfskel/commit/0bbf033f7a27fa768fe9e332dc955f4ef50b66e4))
* updated docs & release please config to include other changes section ([#67](https://github.com/ishuar/tfskel/issues/67)) ([f72bd36](https://github.com/ishuar/tfskel/commit/f72bd36fd538d8aba6cd906ccf504d1ba3c8cdb1))

## [0.7.0](https://github.com/ishuar/tfskel/compare/v0.6.0...v0.7.0) (2026-03-26)


### ⚠ BREAKING CHANGES

* **internal:** internal package paths changed — affects import paths only, no public CLI or config behavior changes.

### ✨ Features

* **cmd:** add --dry-run flag and improve logging with config awareness ([#63](https://github.com/ishuar/tfskel/issues/63)) ([ead0640](https://github.com/ishuar/tfskel/commit/ead06402541739c1b78166bfd8bd382268024042))
* **generator:** upgrade tfskel rendered files using source markers using --upgrade flag with init and scaffold cmd ([#60](https://github.com/ishuar/tfskel/issues/60)) ([0dba32c](https://github.com/ishuar/tfskel/commit/0dba32c27139ab5739699189db10fbbf72e7d4e1))


### ♻️ Refactoring

* **internal:** restructure packages following Go naming conventions ([#62](https://github.com/ishuar/tfskel/issues/62)) ([1d2470a](https://github.com/ishuar/tfskel/commit/1d2470a24f1ab5692ca2ace8afd7d7429bb9a577))

## [0.6.0](https://github.com/ishuar/tfskel/compare/v0.5.2...v0.6.0) (2026-03-24)


### ⚠ BREAKING CHANGES

* **cmd/scaffold:** per-app-dir lint and terraform GitHub workflows are removed. Global terraform lint workflow replaces per-app-dir lint. Per-env terraform plan/apply workflow replaces per-app-dir workflow.
* **cmd/scaffold:** `--workflows` flag is removed from scaffold command. Static reusable GitHub workflows are now created via `tfskel init --workflows`. Per-env terraform plan/apply workflow creation moved to `tfskel scaffold workflows --env`.
* **cmd/scaffold:** `workflows.name_template` is removed. Use `workflows.name` instead; Go templating in the value is no longer supported.

### ✨ Features

* **cmd/scaffold:** moved static gh workflows creation to init from scaffold cmd and added workflows subcommand for consolidated per env gh workflow ([#58](https://github.com/ishuar/tfskel/issues/58)) ([7286b5a](https://github.com/ishuar/tfskel/commit/7286b5a98e61577ee8f7dc81dbd279229f7ef522))

## [0.5.2](https://github.com/ishuar/tfskel/compare/v0.5.1...v0.5.2) (2026-03-20)


### 🐞 Bug Fixes

* **template:** added missing permissions to caller terraform github workflow template ([#55](https://github.com/ishuar/tfskel/issues/55)) ([ad2d348](https://github.com/ishuar/tfskel/commit/ad2d3486b64174cb3d4248d9783bc01fc01a493d))

## [0.5.1](https://github.com/ishuar/tfskel/compare/v0.5.0...v0.5.1) (2026-03-19)


### 🐞 Bug Fixes

* **format:** force lipgloss to render colors even in non-TTY environments ([#50](https://github.com/ishuar/tfskel/issues/50)) ([a9746f7](https://github.com/ishuar/tfskel/commit/a9746f7926714df1bde73418e46a370f0a656a0f))
* **template:** added `content.read` permissions to job and removed extras in lint workflow ([59c5559](https://github.com/ishuar/tfskel/commit/59c5559004284004fd80cb401b5714af45c9ea58))
* **template:** fixed cache, sshKey for private modules and tfskel usage in the re-usable tf workflow ([#53](https://github.com/ishuar/tfskel/issues/53)) ([59c5559](https://github.com/ishuar/tfskel/commit/59c5559004284004fd80cb401b5714af45c9ea58))


### 🧪 Tests

* **cmd/diff:** added tests for config subcommand in diff command ([#54](https://github.com/ishuar/tfskel/issues/54)) ([a1adb2e](https://github.com/ishuar/tfskel/commit/a1adb2e1b399b1f3a73c2c2df934d91c4c827b4f))

## [0.5.0](https://github.com/ishuar/tfskel/compare/v0.4.0...v0.5.0) (2026-03-18)


### ⚠ BREAKING CHANGES

* **plan:** tfskel plan cmd is removed
    - `tfskel plan review` is renamed to `tfskel review plan`

### ✨ Features

* **plan:** rename tfskel plan review to tfskel review plan ([#48](https://github.com/ishuar/tfskel/issues/48)) ([ce3bcdb](https://github.com/ishuar/tfskel/commit/ce3bcdbeab8402fa66e87bc78b30e49eceb09b81))

## [0.4.0](https://github.com/ishuar/tfskel/compare/v0.3.0...v0.4.0) (2026-03-18)


### ⚠ BREAKING CHANGES

* **drift:** Split `tfskel drift` command into two separate commands:
    - `tfskel drift version --path | -p` is replaced with `tfskel diff config --dir | -d`
    - `tfskel drift plan --plan-file` is replaced with `tfskel plan review --json-file`
    - Removed `top_n_count` from .tfskel.yaml config
      - use `top_resources_count` or `--top-resources-count` instead with `tfskel plan review --json-file <JSON-PLAN-FILE>`
* **drift:** removed tfskel drift all command. tfskel drift version and tfskel drift plan command can be used individually to have the same results
* **generate:** generate cmd is renamed to scaffold cmd. Additionally generate.* .tfskel.yaml configs are not available anymore.
    - generate.templates_dir is moved to templates.dir
    - generate.github_workflows.* is moved to workflows.*
    - --create-github-workflows is replaced by --workflows

### ✨ Features

* **drift:** removed all subcommand from drift command ([#45](https://github.com/ishuar/tfskel/issues/45)) ([20d16fb](https://github.com/ishuar/tfskel/commit/20d16fbed1aa89240754b73f67007a121500bab3))
* **drift:** split drift command into diff and plan commands ([#47](https://github.com/ishuar/tfskel/issues/47)) ([5c5725e](https://github.com/ishuar/tfskel/commit/5c5725e65e2ca59a8cf4f9f657bb0e595a7c4702))
* **generate:** rename generate cmd to scaffold cmd ([#43](https://github.com/ishuar/tfskel/issues/43)) ([87840db](https://github.com/ishuar/tfskel/commit/87840db882faffa669ef77dc4975b90817ae3bf1))

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
