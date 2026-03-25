package generate

import (
	"fmt"
	"strings"
)

// FileOp represents the type of file operation performed.
type FileOp int

const (
	// OpCreated indicates a new file was created.
	OpCreated FileOp = iota
	// OpSkipped indicates an existing file was left unchanged.
	OpSkipped
	// OpUpgraded indicates an existing file was re-rendered from its template.
	OpUpgraded
	// OpForced indicates a file without a source marker was overwritten via --force.
	OpForced
	// OpDirCreated indicates a new directory was created.
	OpDirCreated
)

// OpRecord captures a single file operation.
type OpRecord struct {
	Op   FileOp
	Path string
}

// OpTracker accumulates file operation records during a generation run.
type OpTracker struct {
	records []OpRecord
}

// NewOpTracker creates a new OpTracker.
func NewOpTracker() *OpTracker {
	return &OpTracker{}
}

// Record adds a file operation to the tracker.
func (t *OpTracker) Record(op FileOp, path string) {
	t.records = append(t.records, OpRecord{Op: op, Path: path})
}

// Count returns the number of operations matching the given type.
func (t *OpTracker) Count(op FileOp) int {
	n := 0
	for _, r := range t.records {
		if r.Op == op {
			n++
		}
	}
	return n
}

// Summary returns a human-readable summary of all operations.
// When dryRun is true, verbs use future tense ("would be created" instead of "created").
// Returns an empty string if no operations were recorded.
func (t *OpTracker) Summary(dryRun bool) string {
	if len(t.records) == 0 {
		return ""
	}

	created := t.Count(OpCreated)
	skipped := t.Count(OpSkipped)
	upgraded := t.Count(OpUpgraded)
	forced := t.Count(OpForced)

	createdVerb := "created"
	upgradedVerb := "upgraded"
	forcedVerb := "force-upgraded"
	if dryRun {
		createdVerb = "would be created"
		upgradedVerb = "would be upgraded"
		forcedVerb = "would be force-upgraded"
	}

	parts := []string{}
	if created > 0 {
		parts = append(parts, fmt.Sprintf("%d %s %s", created, fileWord(created), createdVerb))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d %s skipped", skipped, fileWord(skipped)))
	}
	if upgraded > 0 {
		parts = append(parts, fmt.Sprintf("%d %s %s", upgraded, fileWord(upgraded), upgradedVerb))
	}
	if forced > 0 {
		parts = append(parts, fmt.Sprintf("%d %s %s", forced, fileWord(forced), forcedVerb))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}

func fileWord(n int) string {
	if n == 1 {
		return "file"
	}
	return "files"
}
