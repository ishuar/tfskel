// Package format holds the output primitives shared by every command that
// prints a report: the OutputFormat names (table, json, csv), the lipgloss
// styles used for tables and status words, terminal-width constants for
// column sizing, and ShouldUseColor, the single place that turns NO_COLOR,
// FORCE_COLOR, CI and --no-color into one yes/no decision.
//
// It imports nothing else from this module so that plan, validate and the
// CLI can all depend on it without cycles. Formatting of a specific report
// (plan tables, validate findings) lives next to the data it formats.
package format
