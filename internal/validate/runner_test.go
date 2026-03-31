package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCheckSelection(t *testing.T) {
	tests := []struct {
		name   string
		skip   string
		want   map[CheckName]bool
		errMsg string
	}{
		{
			name: "empty returns nil (all checks)",
			skip: "",
			want: nil,
		},
		{
			name: "skip tools",
			skip: "tools",
			want: map[CheckName]bool{CheckConfig: true},
		},
		{
			name: "skip config",
			skip: "config",
			want: map[CheckName]bool{CheckTools: true},
		},
		{
			name: "skip all",
			skip: "config,tools",
			want: map[CheckName]bool{},
		},
		{
			name: "skip with spaces",
			skip: " config , tools ",
			want: map[CheckName]bool{},
		},
		{
			name:   "invalid skip check",
			skip:   "nope",
			errMsg: `unknown check "nope"`,
		},
		{
			name:   "files is no longer valid",
			skip:   "files",
			errMsg: `unknown check "files"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCheckSelection(tt.skip)

			if tt.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewRunner(t *testing.T) {
	t.Run("nil checks enables all", func(t *testing.T) {
		r := NewRunner(nil, "/tmp", nil)

		assert.True(t, r.checks[CheckConfig])
		assert.True(t, r.checks[CheckTools])
	})

	t.Run("explicit check selection", func(t *testing.T) {
		checks := map[CheckName]bool{CheckConfig: true}
		r := NewRunner(nil, "/tmp", checks)

		assert.True(t, r.checks[CheckConfig])
		assert.False(t, r.checks[CheckTools])
	})
}

func TestRunner_Run_SkippedChecks(t *testing.T) {
	// Run with only config check enabled; tools should be skipped.
	checks := map[CheckName]bool{CheckConfig: true}
	r := NewRunner(nil, t.TempDir(), checks)

	report := r.Run()

	require.Len(t, report.Checks, 2, "all checks should appear in report")

	statusByCheck := make(map[CheckName]CheckStatus)
	for _, c := range report.Checks {
		statusByCheck[c.Check] = c.Status
	}

	assert.Equal(t, StatusSkipped, statusByCheck[CheckTools])
	assert.NotEqual(t, StatusSkipped, statusByCheck[CheckConfig])
}

func TestRunner_Run_AllSkipped(t *testing.T) {
	// Empty check map — nothing runs.
	checks := map[CheckName]bool{}
	r := NewRunner(nil, t.TempDir(), checks)

	report := r.Run()

	for _, c := range report.Checks {
		assert.Equal(t, StatusSkipped, c.Status, "check %s should be skipped", c.Check)
	}
	assert.Equal(t, 0, report.ExitCode())
	assert.Equal(t, 0, report.IssueCount())
}
