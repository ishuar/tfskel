package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReport_ExitCode(t *testing.T) {
	tests := []struct {
		name   string
		checks []CheckResult
		want   int
	}{
		{
			name: "all pass",
			checks: []CheckResult{
				{Check: CheckConfig, Status: StatusPass},
				{Check: CheckTools, Status: StatusPass},
			},
			want: 0,
		},
		{
			name: "one fail",
			checks: []CheckResult{
				{Check: CheckConfig, Status: StatusFail, Issues: 2},
				{Check: CheckTools, Status: StatusPass},
			},
			want: 1,
		},
		{
			name: "one error",
			checks: []CheckResult{
				{Check: CheckConfig, Status: StatusError},
				{Check: CheckTools, Status: StatusPass},
			},
			want: 2,
		},
		{
			name: "error takes precedence over fail",
			checks: []CheckResult{
				{Check: CheckConfig, Status: StatusError},
				{Check: CheckTools, Status: StatusFail, Issues: 1},
			},
			want: 2,
		},
		{
			name: "skipped checks ignored",
			checks: []CheckResult{
				{Check: CheckConfig, Status: StatusSkipped},
				{Check: CheckTools, Status: StatusPass},
			},
			want: 0,
		},
		{
			name:   "empty report",
			checks: []CheckResult{},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Report{Checks: tt.checks}
			assert.Equal(t, tt.want, r.ExitCode())
		})
	}
}

func TestReport_IssueCount(t *testing.T) {
	r := &Report{
		Checks: []CheckResult{
			{Check: CheckConfig, Issues: 3},
			{Check: CheckTools, Issues: 0},
		},
	}
	assert.Equal(t, 3, r.IssueCount())
}

func TestAllChecks(t *testing.T) {
	checks := AllChecks()
	assert.Equal(t, []CheckName{CheckConfig, CheckTools}, checks)
}

func TestValidCheckName(t *testing.T) {
	assert.True(t, ValidCheckName("config"))
	assert.True(t, ValidCheckName("tools"))
	assert.False(t, ValidCheckName("mise"))
	assert.False(t, ValidCheckName("files"))
	assert.False(t, ValidCheckName("invalid"))
	assert.False(t, ValidCheckName(""))
}
