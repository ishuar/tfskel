// Package generate renders templates into an application directory under
// envs/<env>/<region>/<app>/ and into .github/workflows/. It backs
// `tfskel scaffold` and `tfskel scaffold workflows`; `tfskel init` uses the
// sibling package internal/bootstrap for one-time project setup.
//
// Generator owns the whole flow for one run: build the template Data from
// config and flags (templatedata.go), decide each output path (writer.go),
// render with markers injected (source_marker.go, metadata_marker.go), and
// record every create, skip and upgrade in an OpTracker for the summary line.
//
// Three markers are written as comments into generated files and read back
// on later runs:
//
//   - tfskel-source: template name and a hash of the template text. Drives
//     --upgrade (upgrade.go): a file is rewritten only when its re-rendered
//     content differs; --force extends that to files without a marker.
//   - tfskel-metadata: the config values backend.tf and versions.tf were
//     rendered from. Compared on every run (configupdate.go), so a changed
//     bucket name or version constraint rewrites the file without --upgrade.
//   - tfskel-tags-hash: hash of default_tags, same purpose, versions.tf only.
//
// HCL outputs are passed through hclwrite.Format before writing so generated
// files match terraform fmt. The package never prompts and never overwrites
// a file by default; every write goes through fs.FileSystem.
//
// User-facing behavior is documented in docs/templates.md and
// docs/upgrading.md.
package generate
