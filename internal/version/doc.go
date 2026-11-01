// Package version finds Terraform and provider version constraints in a
// repository and reports where they drift from .tfskel.yaml.
//
// Detector walks a directory tree, skipping hidden directories such as
// .git and .terraform, and parses each .tf file with the HCL syntax parser
// to pull required_version and required_providers constraints into
// VersionInfo records. Parse failures are recorded per file and do not
// stop the scan. Analyzer compares those records with the config and
// produces a DriftReport with per-file DriftRecords and a DriftSummary.
//
// The package reads files with the standard library and never writes.
// internal/validate turns a DriftReport into findings.
package version
