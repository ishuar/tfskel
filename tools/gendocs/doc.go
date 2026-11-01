// Command gendocs renders the tfskel command tree to Markdown under
// docs/reference. It is the single source of the command reference: flags,
// defaults, and examples come from the same cobra definitions that back
// --help, so the docs cannot describe a flag the binary does not have.
//
// Run it through `make docs`. `make check` and CI regenerate and diff the
// output, so a change to cmd/ that is not regenerated fails the build.
package main
