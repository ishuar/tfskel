// Package toolcheck detects whether the tools a tfskel project needs
// (terraform, tflint, trivy, pre-commit, awscli) are installed, which
// version is on PATH, and how that relates to mise.
//
// Checker runs each ToolDef's version command through a CommandRunner
// (OSCommandRunner in production, a stub in tests) and reports one of five
// states per tool: installed, missing, installable by mise, installed by
// mise but not activated in the shell, or found on the global PATH outside
// mise. Report and FormatReport give the aggregate view that
// internal/validate turns into findings.
package toolcheck
