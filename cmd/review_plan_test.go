package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reviewPlanFixture is a minimal valid plan: one replacement (critical
// severity → exit code 2) and one creation.
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

func TestReviewPlanCommand_ChangesExitWithCriticalCode(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture))

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr, "changes must surface as an ExitError")
	assert.Equal(t, 2, exitErr.Code, "replacement present → critical exit code")
}

func TestReviewPlanCommand_NoChangesSucceeds(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixtureNoChanges))
	assert.NoError(t, err)
}

func TestReviewPlanCommand_MachineFormats(t *testing.T) {
	for _, f := range []string{"json", "csv"} {
		t.Run("format="+f, func(t *testing.T) {
			err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--format", f)

			var exitErr *ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, 2, exitErr.Code)
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

func TestReviewPlanCommand_FilterNarrowsReport(t *testing.T) {
	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--min-severity", "critical")

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 2, exitErr.Code, "exit code reflects the full analysis, not the filtered view")
}

// TestReviewPlanCommand_AIMissingKeyIsAdditive verifies the additive-failure
// contract end to end through cobra: --ai with no API key warns and skips,
// and the exit code stays whatever the structured analysis decided.
func TestReviewPlanCommand_AIMissingKeyIsAdditive(t *testing.T) {
	t.Setenv(ai.ProviderEnvVar, "")
	t.Setenv(ai.APIKeyEnvVar, "")

	err := execReviewPlan(t, writeReviewPlanFixture(t, reviewPlanFixture), "--ai", "--ai-model", "claude-test", "--ai-max-tokens", "128")

	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr, "AI failure must not replace the analysis exit code")
	assert.Equal(t, 2, exitErr.Code)
}
