package review

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/ai"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAIClient satisfies ai.Client at the Request.NewAIClient seam. It
// captures the payload it was handed so tests can assert the alignment
// invariant, writes text to the report writer, and returns err afterwards.
type stubAIClient struct {
	text    string
	err     error
	payload *ai.Payload
}

func (s *stubAIClient) Explain(_ context.Context, payload *ai.Payload, w io.Writer) error {
	s.payload = payload
	if s.text != "" {
		if _, err := io.WriteString(w, s.text); err != nil {
			return err
		}
	}
	return s.err
}

func (s *stubAIClient) Provider() string { return "stub" }
func (s *stubAIClient) Model() string    { return "stub-model" }

// planWithChanges is a minimal valid plan: one replacement (critical), one
// creation (low), one data source and one no-op (both must be ignored).
const planWithChanges = `{
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
    },
    {
      "address": "data.aws_ami.latest",
      "mode": "data",
      "type": "aws_ami",
      "name": "latest",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {"actions": ["read"]}
    },
    {
      "address": "aws_vpc.main",
      "mode": "managed",
      "type": "aws_vpc",
      "name": "main",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {"actions": ["no-op"]}
    }
  ]
}`

const planNoChanges = `{
  "format_version": "1.2",
  "terraform_version": "1.9.0",
  "resource_changes": []
}`

func writePlanFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tfplan.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func baseRequest(t *testing.T, planContent string) Request {
	t.Helper()
	return Request{
		PlanFile:          writePlanFile(t, planContent),
		Format:            format.FormatTable,
		Filter:            &plan.ResourceFilter{},
		TopResourcesCount: plan.DefaultTopResourcesCount,
		CriticalResources: plan.DefaultCriticalResources(),
	}
}

func testLogger() *logger.Logger {
	return logger.NewWithOptions(false, false)
}

func TestRun_WritesReportAndDerivesExitCode(t *testing.T) {
	var out bytes.Buffer
	result, err := Run(context.Background(), baseRequest(t, planWithChanges), &out, testLogger())

	require.NoError(t, err)
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode, "replacement present → critical exit code")
	assert.Contains(t, out.String(), "aws_db_instance")
	assert.Contains(t, out.String(), "aws_s3_bucket")
	assert.NotContains(t, out.String(), "## AI Analysis", "no AI section without NewAIClient")
	assert.Len(t, result.Analysis.ResourceChanges, 2, "data source and no-op excluded")
}

func TestRun_NoChanges(t *testing.T) {
	var out bytes.Buffer
	result, err := Run(context.Background(), baseRequest(t, planNoChanges), &out, testLogger())

	require.NoError(t, err)
	assert.Equal(t, plan.ExitCodeSuccess, result.ExitCode)
	assert.Empty(t, out.String(), "no report content for a changeless plan")
}

func TestRun_ParseFailure(t *testing.T) {
	req := baseRequest(t, planWithChanges)
	req.PlanFile = filepath.Join(t.TempDir(), "missing.json")

	var out bytes.Buffer
	_, err := Run(context.Background(), req, &out, testLogger())
	require.Error(t, err)
}

func TestRun_AIAppendsNarrative(t *testing.T) {
	stub := &stubAIClient{text: "The replacement of the database is the highest-risk change."}
	req := baseRequest(t, planWithChanges)
	req.NewAIClient = func(context.Context) (ai.Client, error) { return stub, nil }

	var out bytes.Buffer
	result, err := Run(context.Background(), req, &out, testLogger())

	require.NoError(t, err)
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode, "AI never changes the exit code")
	assert.Contains(t, out.String(), "## AI Analysis")
	assert.Contains(t, out.String(), "**Provider:** stub")
	assert.Contains(t, out.String(), "**Model:** stub-model")
	assert.Contains(t, out.String(), stub.text)
}

func TestRun_AIClientConstructionFailureIsAdditive(t *testing.T) {
	req := baseRequest(t, planWithChanges)
	req.NewAIClient = func(context.Context) (ai.Client, error) {
		return nil, errors.New("ANTHROPIC_API_KEY is not set")
	}

	var out bytes.Buffer
	result, err := Run(context.Background(), req, &out, testLogger())

	require.NoError(t, err, "AI construction failure must not surface as a command error")
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode)
	assert.NotContains(t, out.String(), "## AI Analysis")
}

func TestRun_AIExplainFailureIsAdditive(t *testing.T) {
	stub := &stubAIClient{err: errors.New("network blip")}
	req := baseRequest(t, planWithChanges)
	req.NewAIClient = func(context.Context) (ai.Client, error) { return stub, nil }

	var out bytes.Buffer
	result, err := Run(context.Background(), req, &out, testLogger())

	require.NoError(t, err)
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode)
}

func TestRun_AITruncationFinishesMarkdown(t *testing.T) {
	stub := &stubAIClient{text: "Partial analysis", err: ai.ErrResponseTruncated}
	req := baseRequest(t, planWithChanges)
	req.NewAIClient = func(context.Context) (ai.Client, error) { return stub, nil }

	var out bytes.Buffer
	result, err := Run(context.Background(), req, &out, testLogger())

	require.NoError(t, err, "truncation is additive, not fatal")
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode)
	assert.Contains(t, out.String(), "Partial analysis")
	assert.Contains(t, out.String(), "**Output truncated**")
	assert.Contains(t, out.String(), "--ai-max-tokens")
}

// TestRun_FilteredAnalysisIsPayloadSourceOfTruth exercises the module's
// central invariant end to end: with an active filter, the AI payload
// contains exactly the filtered resources, every one carrying a severity.
func TestRun_FilteredAnalysisIsPayloadSourceOfTruth(t *testing.T) {
	stub := &stubAIClient{text: "narrative"}
	req := baseRequest(t, planWithChanges)
	req.Filter = &plan.ResourceFilter{MinSeverity: "critical"}
	req.NewAIClient = func(context.Context) (ai.Client, error) { return stub, nil }

	var out bytes.Buffer
	result, err := Run(context.Background(), req, &out, testLogger())

	require.NoError(t, err)
	require.Len(t, result.Analysis.ResourceChanges, 1, "only the replacement survives the filter")

	require.NotNil(t, stub.payload, "AI client must receive a payload")
	require.Len(t, stub.payload.Resources, 1, "payload mirrors the filtered analysis exactly")
	assert.Equal(t, "aws_db_instance.main", stub.payload.Resources[0].Address)
	for _, r := range stub.payload.Resources {
		assert.NotEmpty(t, r.Severity, "every resource on the wire carries a computed severity")
	}
}
