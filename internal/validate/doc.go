// Package validate implements `tfskel validate`: a report of everything in
// the repository that disagrees with .tfskel.yaml and .mise.toml.
//
// Two checks exist, each selectable by name:
//
//   - config: walks the repository for .tf files, uses internal/version to
//     extract Terraform and provider version constraints, compares them to
//     the config, and checks every envs/<env>/.terraform-version file.
//   - tools: uses internal/toolcheck to confirm required tools are installed
//     and match the versions pinned in .mise.toml.
//
// Runner executes the selected checks and returns one Report of Findings.
// Findings carry no severity. The report's exit code is 2 whenever any
// finding exists and 1 when a check could not run; 0 otherwise. Formatter
// renders the report as table, JSON or CSV.
package validate
