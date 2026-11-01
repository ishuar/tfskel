// Package bootstrap creates a new tfskel project from zero: root config files,
// environment/region skeleton, optional GitHub workflow files, and the default
// .tfskel.yaml. It backs the `tfskel init` command.
//
// Paired mental model with internal/generate:
//   - bootstrap: project-level, one-time, from-zero scaffolding
//   - generate:  per-app, repeatable file generation under envs/<env>/<region>/<app>/
package bootstrap
