package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCheckSelection(t *testing.T) {
	tests := []struct {
		name    string
		only    string
		skip    string
		want    map[CheckName]bool
		wantErr error
		errMsg  string
	}{
		{
			name: "both empty returns nil (all checks)",
			only: "",
			skip: "",
			want: nil,
		},
		{
			name: "only config",
			only: "config",
			skip: "",
			want: map[CheckName]bool{CheckConfig: true},
		},
		{
			name: "only both checks",
			only: "config,tools",
			skip: "",
			want: map[CheckName]bool{CheckConfig: true, CheckTools: true},
		},
		{
			name: "only with spaces",
			only: " config , tools ",
			skip: "",
			want: map[CheckName]bool{CheckConfig: true, CheckTools: true},
		},
		{
			name: "skip tools",
			only: "",
			skip: "tools",
			want: map[CheckName]bool{CheckConfig: true},
		},
		{
			name: "skip config",
			only: "",
			skip: "config",
			want: map[CheckName]bool{CheckTools: true},
		},
		{
			name: "skip all",
			only: "",
			skip: "config,tools",
			want: map[CheckName]bool{},
		},
		{
			name:    "mutually exclusive",
			only:    "config",
			skip:    "tools",
			wantErr: ErrMutuallyExclusive,
		},
		{
			name:   "invalid only check",
			only:   "bogus",
			skip:   "",
			errMsg: `unknown check "bogus"`,
		},
		{
			name:   "invalid skip check",
			only:   "",
			skip:   "nope",
			errMsg: `unknown check "nope"`,
		},
		{
			name:   "files is no longer valid",
			only:   "files",
			skip:   "",
			errMsg: `unknown check "files"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCheckSelection(tt.only, tt.skip)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
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
