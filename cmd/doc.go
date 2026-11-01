// Package cmd is the CLI boundary of tfskel. It defines the cobra command
// tree (init, scaffold, scaffold workflows, validate, review plan, version),
// parses flags, resolves the flag > config > default precedence once, and
// hands fully resolved inputs to the internal packages.
//
// Rules that hold across every command:
//
//   - NewRootCmd builds a fresh tree on each call. There is no package-level
//     command state, so tests construct their own trees.
//   - Validation happens here, at the boundary: flag combinations, file
//     existence, format names. Internal packages trust what they receive.
//   - Exit codes are the contract with CI. Exit 1 always means an error.
//     Non-error, non-zero codes (validate 2; review plan 2 and 3 with
//     --detailed-exitcode) are returned as *ExitError so main can exit with
//     the code without printing a usage block.
//   - Usage is printed only when the user does not know what commands exist.
//     Flag and argument mistakes return the error alone.
//
// The Long and Example strings on each command are the user documentation:
// tools/gendocs renders them into docs/reference, so write them for readers.
package cmd
