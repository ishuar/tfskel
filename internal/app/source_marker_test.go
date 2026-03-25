package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSourceComment(t *testing.T) {
	t.Run("uses ## prefix for .tf files", func(t *testing.T) {
		comment := BuildSourceComment("tf/backend.tf.tmpl", "abc123", ".tf")
		assert.Contains(t, comment, "## tfskel-source:")
		assert.Contains(t, comment, `"template":"tf/backend.tf.tmpl"`)
		assert.Contains(t, comment, `"hash":"abc123"`)
	})

	t.Run("uses ## prefix for .hcl files", func(t *testing.T) {
		comment := BuildSourceComment("root/.tflint.hcl.tmpl", "def456", ".hcl")
		assert.Contains(t, comment, "## tfskel-source:")
	})

	t.Run("uses # prefix for .yaml files", func(t *testing.T) {
		comment := BuildSourceComment("github/lint.yaml", "ghi789", ".yaml")
		assert.True(t, strings.HasPrefix(comment, "# tfskel-source:"), "should start with # tfskel-source:")
		assert.NotContains(t, comment, "## tfskel-source:")
	})

	t.Run("uses # prefix for other file types", func(t *testing.T) {
		comment := BuildSourceComment("root/.gitignore.tmpl", "jkl012", "")
		assert.True(t, strings.HasPrefix(comment, "# tfskel-source:"), "should start with # tfskel-source:")
	})
}

func TestExtractSourceMarker(t *testing.T) {
	t.Run("extracts marker from HCL comment", func(t *testing.T) {
		content := `## tfskel-source: {"template":"tf/backend.tf.tmpl","hash":"abc123"}
## tfskel-metadata: {"bucket":"my-bucket"}
terraform {
  backend "s3" {}
}`
		marker, err := ExtractSourceMarker(content)
		require.NoError(t, err)
		assert.Equal(t, "tf/backend.tf.tmpl", marker.Template)
		assert.Equal(t, "abc123", marker.Hash)
	})

	t.Run("extracts marker from YAML comment", func(t *testing.T) {
		content := `# tfskel-source: {"template":"github/lint.yaml","hash":"def456"}
name: lint
on: push`
		marker, err := ExtractSourceMarker(content)
		require.NoError(t, err)
		assert.Equal(t, "github/lint.yaml", marker.Template)
		assert.Equal(t, "def456", marker.Hash)
	})

	t.Run("returns error when no marker present", func(t *testing.T) {
		content := `terraform {
  backend "s3" {}
}`
		_, err := ExtractSourceMarker(content)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSourceMarkerNotFound)
	})

	t.Run("returns error on malformed JSON", func(t *testing.T) {
		content := `## tfskel-source: {not valid json}`
		_, err := ExtractSourceMarker(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse source marker JSON")
	})

	t.Run("handles marker with extra whitespace", func(t *testing.T) {
		content := `##  tfskel-source:  {"template":"tf/main.tf.tmpl","hash":"xyz"}
resource "aws_s3_bucket" "main" {}`
		marker, err := ExtractSourceMarker(content)
		require.NoError(t, err)
		assert.Equal(t, "tf/main.tf.tmpl", marker.Template)
		assert.Equal(t, "xyz", marker.Hash)
	})
}

func TestInjectSourceMarker(t *testing.T) {
	t.Run("prepends marker as first line", func(t *testing.T) {
		content := "terraform {\n  backend \"s3\" {}\n}"
		comment := "## tfskel-source: {\"template\":\"tf/backend.tf.tmpl\",\"hash\":\"abc\"}"

		result := InjectSourceMarker(content, comment)
		assert.True(t, strings.HasPrefix(result, comment+"\n"), "should start with marker comment")
		assert.Contains(t, result, "terraform {")
	})

	t.Run("works with empty content", func(t *testing.T) {
		result := InjectSourceMarker("", "# tfskel-source: {}")
		assert.Equal(t, "# tfskel-source: {}\n", result)
	})
}

func TestSourceCommentForFile(t *testing.T) {
	t.Run("detects .tf extension", func(t *testing.T) {
		comment := SourceCommentForFile("tf/backend.tf.tmpl", "abc", "envs/dev/us-east-1/myapp/backend.tf")
		assert.Contains(t, comment, "## tfskel-source:")
	})

	t.Run("detects .yaml extension", func(t *testing.T) {
		comment := SourceCommentForFile("github/lint.yaml", "abc", ".github/workflows/lint.yaml")
		assert.True(t, strings.HasPrefix(comment, "# tfskel-source:"), "should start with # tfskel-source:")
		assert.NotContains(t, comment, "## tfskel-source:")
	})

	t.Run("detects .hcl extension", func(t *testing.T) {
		comment := SourceCommentForFile("root/.tflint.hcl.tmpl", "abc", ".tflint.hcl")
		assert.Contains(t, comment, "## tfskel-source:")
	})

	t.Run("returns empty for .terraform-version", func(t *testing.T) {
		comment := SourceCommentForFile("root/.terraform-version.tmpl", "abc", "envs/dev/.terraform-version")
		assert.Empty(t, comment, ".terraform-version does not support comment markers")
	})
}
