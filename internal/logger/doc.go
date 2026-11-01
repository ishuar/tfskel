// Package logger is the console logger used by every command. It writes
// human-readable, optionally colored lines at Debug, Info, Warn and Error
// levels; Debug lines carry a timestamp and the caller's file:line.
//
// Two rules keep report output machine-safe:
//
//   - Report content (tables, JSON, CSV) is never written through the
//     logger. Commands write it to stdout directly.
//   - SetMachineOutput moves every log line to stderr and disables color.
//     Commands call it when --format json or csv is selected so stdout
//     holds only the document.
//
// Color is decided by the caller (internal/format.ShouldUseColor) and
// passed to NewWithOptions. NewWithWriters exists for tests that capture
// output. Packages that need only a few methods declare their own small
// Logger interface (internal/config, internal/generate) instead of
// importing this type.
package logger
