// Package fs abstracts the filesystem writes tfskel performs so the
// generators can be tested without touching disk and so --dry-run is
// implemented in one place.
//
// FileSystem is the interface. OSFileSystem is the real implementation,
// MemoryFileSystem backs tests, and DryRunFileSystem wraps any FileSystem:
// it delegates reads, so upgrade decisions still see the real files, and
// turns writes and directory creation into no-ops.
//
// Code that only inspects the repository (plan JSON, HCL scanning) uses the
// standard library directly; this package exists for the write path.
package fs
