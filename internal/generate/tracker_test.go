package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpTracker(t *testing.T) {
	t.Run("empty tracker returns empty summary", func(t *testing.T) {
		tracker := NewOpTracker()
		assert.Equal(t, "", tracker.Summary(false))
	})

	t.Run("counts operations correctly", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpCreated, "a.tf")
		tracker.Record(OpCreated, "b.tf")
		tracker.Record(OpSkipped, "c.tf")
		tracker.Record(OpUpgraded, "d.tf")
		tracker.Record(OpForced, "e.tf")

		assert.Equal(t, 2, tracker.Count(OpCreated))
		assert.Equal(t, 1, tracker.Count(OpSkipped))
		assert.Equal(t, 1, tracker.Count(OpUpgraded))
		assert.Equal(t, 1, tracker.Count(OpForced))
		assert.Equal(t, 0, tracker.Count(OpDirCreated))
	})

	t.Run("summary formats correctly", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpCreated, "a.tf")
		tracker.Record(OpCreated, "b.tf")
		tracker.Record(OpSkipped, "c.tf")

		assert.Equal(t, "2 files created, 1 file skipped", tracker.Summary(false))
	})

	t.Run("summary with single operation type", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpUpgraded, "a.tf")

		assert.Equal(t, "1 file upgraded", tracker.Summary(false))
	})

	t.Run("summary omits zero counts", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpForced, "a.tf")
		tracker.Record(OpForced, "b.tf")

		assert.Equal(t, "2 files force-upgraded", tracker.Summary(false))
	})

	t.Run("dir created is tracked but not in summary", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpDirCreated, "envs/dev")
		tracker.Record(OpCreated, "a.tf")

		assert.Equal(t, 1, tracker.Count(OpDirCreated))
		assert.Equal(t, "1 file created", tracker.Summary(false))
	})

	t.Run("dry-run summary uses future tense", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpCreated, "a.tf")
		tracker.Record(OpCreated, "b.tf")
		tracker.Record(OpSkipped, "c.tf")

		assert.Equal(t, "2 files would be created, 1 file skipped", tracker.Summary(true))
	})

	t.Run("dry-run summary with upgraded and forced", func(t *testing.T) {
		tracker := NewOpTracker()
		tracker.Record(OpUpgraded, "a.tf")
		tracker.Record(OpForced, "b.tf")

		assert.Equal(t, "1 file would be upgraded, 1 file would be force-upgraded", tracker.Summary(true))
	})
}
