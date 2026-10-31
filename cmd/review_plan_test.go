package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/ai"
	"github.com/ishuar/tfskel/internal/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reviewPlanFixture is a minimal valid plan: one replacement (critical
// severity → detailed exit code 3) and one creation.
const reviewPlanFixture = `{
  "format_version": "1.2",
  "terraform_version": "1.9.0",
  "resource_changes": [
    {
      "address": "aws_db_instance.main",
      "mode": "managed",
      "type": "aws_db_instance",
      "name": "main",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete", "create"],
        "before": {"instance_class": "db.t3.micro"},
        "after": {"instance_class": "db.t3.small"}
      }
    },
    {
      "address": "aws_s3_bucket.logs",
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "logs",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "after": {"bucket": "logs"}
      }
    }
  ]
}`

// reviewPlanFixtureCreateOnly has a single addition (non-critical → detailed
// exit code 2).
const reviewPlanFixtureCreateOnly = `{
  "format_version": "1.2",
  "terraform_version": "1.9.0",
  "resource_changes": [
    {
      "address": "aws_s3_bucket.logs",
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "logs",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "after": {"bucket": "logs"}
      }
    }
  ]
}`

const reviewPlanFixtureNoChanges = `{
  "format_version": "1.2",
  "terraform_version": "1.9.0",
  "resource_changes": []
}`

func writeReviewPlanFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tfplan.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// execReviewPlan runs `tfskel review plan` through the real cobra tree with
// the given extra args, returning the execution error.
func execReviewPlan(t *testing.T, planFile string, extraArgs ...string) error {
	t.Helper()
	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	args := append([]string{"review", "plan", "--json-file", planFile, "--no-color"}, extraArgs...)
	root.SetArgs(args)
	return root.Execute()
}

// TestReviewPlanCommand_ChangesExitZeroByDefault pins the terraform-like
// default: a plan with changes is a successful review, not a failure.
func TestReviewPlanCommand_ChangesExitZeroByDefault(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture))
	assert.NoError(t, err, "changes must not fail the command without --detailed-exitcode")
}

func TestReviewPlanCommand_NoChangesSucceeds(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixtureNoChanges))
	assert.NoError(t, err)
}

func TestReviewPlanCommand_DetailedExitCode(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantCode int
	}{
		{name: "no changes", fixture: reviewPlanFixtureNoChanges, wantCode: plan.ExitCodeSuccess},
		{name: "create only", fixture: reviewPlanFixtureCreateOnly, wantCode: plan.ExitCodeChanges},
		{name: "replacement", fixture: reviewPlanFixture, wantCode: plan.ExitCodeCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execReviewPlan(t, writeReviewPlanFixture(t, tt.fixture), "--detailed-exitcode")

			if tt.wantCode == plan.ExitCodeSuccess {
				assert.NoError(t, err)
				return
			}
			var exitErr *ExitError
			require.ErrorAs(t, err, &exitErr, "changes must surface as an ExitError")
			assert.Equal(t, tt.wantCode, exitErr.Code)
		})
	}
}

func TestReviewPlanCommand_MachineFormats(t *testing.T) {
	for _, f := range []string{"json", "csv"} {
		t.Run("format="+f, func(t *testing.T) {
			err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--format", f, "--detailed-exitcode")

			var exitErr *ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, plan.ExitCodeCritical, exitErr.Code)
		})
	}
}

func TestReviewPlanCommand_InvalidFormat(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--format", "yaml")
	assert.True(t, errors.Is(err, ErrInvalidFormat), "expected ErrInvalidFormat, got %v", err)
}

func TestReviewPlanCommand_InvalidFilter(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--filter-severity", "catastrophic")
	assert.True(t, errors.Is(err, ErrInvalidFilter), "expected ErrInvalidFilter, got %v", err)
}

func TestReviewPlanCommand_FileNotFound(t *testing.T) {
	err := execReviewPlan(t, filepath.Join(t.TempDir(), "missing.json"))
	assert.True(t, errors.Is(err, ErrFileNotFound), "expected ErrFileNotFound, got %v", err)
}

// TestReviewPlanCommand_FileNotFoundWithDetailedExitCode pins the contract that
// errors exit 1 even with --detailed-exitcode: the command must return a plain
// error, not an ExitError, so main.go falls through to os.Exit(1).
func TestReviewPlanCommand_FileNotFoundWithDetailedExitCode(t *testing.T) {
	err := execReviewPlan(t, filepath.Join(t.TempDir(), "missing.json"), "--detailed-exitcode")

	assert.True(t, errors.Is(err, ErrFileNotFound), "expected ErrFileNotFound, got %v", err)
	var exitErr *ExitError
	assert.False(t, errors.As(err, &exitErr), "errors must not carry a detailed exit code")
}

func TestReviewPlanCommand_FilterNarrowsReport(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--min-severity", "critical", "--detailed-exitcode")

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, plan.ExitCodeCritical, exitErr.Code, "exit code reflects the full analysis, not the filtered view")
}

// TestReviewPlanCommand_AIMissingKeyIsAdditive verifies the additive-failure
// contract end to end through cobra: --ai with no API key warns and skips,
// and the exit code stays whatever the structured analysis decided.
func TestReviewPlanCommand_AIMissingKeyIsAdditive(t *testing.T) {
	t.Setenv(ai.ProviderEnvVar, "")
	t.Setenv(ai.APIKeyEnvVar, "")

	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--ai", "--ai-model", "claude-test", "--ai-max-tokens", "128", "--detailed-exitcode")

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr, "AI failure must not replace the analysis exit code")
	assert.Equal(t, plan.ExitCodeCritical, exitErr.Code)
}
