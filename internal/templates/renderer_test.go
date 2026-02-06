package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)
	assert.NotNil(t, renderer)
}

func TestRenderGitignore(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	data := &Data{
		Region: "us-east-1",
	}

	content, err := renderer.Render("root/.gitignore.tmpl", data)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestRenderVersionsTF(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	data := &Data{
		Env:                "dev",
		Region:             "us-east-1",
		AppDir:             "myapp",
		AccountID:          "123456789012",
		ShortRegion:        "ue1",
		S3BucketName:       "my-bucket",
		TerraformVersion:   "~> 1.13",
		AWSProviderVersion: "~> 6.0",
	}

	content, err := renderer.Render("tf/versions.tf.tmpl", data)
	require.NoError(t, err)
	assert.Contains(t, content, "~> 1.13")
	assert.Contains(t, content, "~> 6.0")
	assert.Contains(t, content, "dev")
	assert.Contains(t, content, "myapp")
}

func TestRenderBackendTF(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	data := &Data{
		Env:          "dev",
		Region:       "us-east-1",
		AppDir:       "myapp",
		AccountID:    "123456789012",
		S3BucketName: "terraform-state-dev-use1",
	}

	content, err := renderer.Render("tf/backend.tf.tmpl", data)
	require.NoError(t, err)
	assert.Contains(t, content, "terraform-state-dev-use1")
	assert.Contains(t, content, "123456789012")
	assert.Contains(t, content, "backend \"s3\"")
}

func TestStripConstraint(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"~> 1.14.3", "1.14.3"},
		{">= 5.0.0", "5.0.0"},
		{"<= 2.0", "2.0"},
		{"> 1.0", "1.0"},
		{"< 3.0", "3.0"},
		{"= 1.5.0", "1.5.0"},
		{"1.2.3", "1.2.3"},
		{"  ~> 4.5.6  ", "4.5.6"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripConstraint(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTemplateNames(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	names := renderer.GetTemplateNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "tf/versions.tf.tmpl")
	assert.Contains(t, names, "tf/backend.tf.tmpl")
	assert.Contains(t, names, "root/.gitignore.tmpl")
}

func TestGetTemplateSource(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	source := renderer.GetTemplateSource("tf/versions.tf.tmpl")
	assert.Contains(t, source, "embedded:tf/versions.tf.tmpl")

	nonExistent := renderer.GetTemplateSource("nonexistent.tmpl")
	assert.Empty(t, nonExistent)
}

func TestRenderWithDefaultTags(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	data := &Data{
		Env:                "prod",
		Region:             "eu-west-1",
		AppDir:             "webapp",
		TerraformVersion:   "~> 1.14",
		AWSProviderVersion: "~> 6.0",
		DefaultTags: map[string]string{
			"team":    "platform",
			"project": "infrastructure",
		},
	}

	content, err := renderer.Render("tf/versions.tf.tmpl", data)
	require.NoError(t, err)
	assert.Contains(t, content, "team")
	assert.Contains(t, content, "platform")
	assert.Contains(t, content, "project")
	assert.Contains(t, content, "infrastructure")
}

func TestRenderNonExistentTemplate(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	_, err = renderer.Render("nonexistent.tmpl", &Data{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template")
	assert.Contains(t, err.Error(), "not found")
}

func TestNewRendererWithCustomTemplates(t *testing.T) {
	t.Run("with empty custom dir uses defaults", func(t *testing.T) {
		renderer, err := NewRendererWithCustomTemplates("", []string{"tf.tmpl"})
		require.NoError(t, err)
		assert.NotNil(t, renderer)

		names := renderer.GetTemplateNames()
		assert.Contains(t, names, "tf/versions.tf.tmpl")
	})

	t.Run("with non-existent custom dir returns error", func(t *testing.T) {
		_, err := NewRendererWithCustomTemplates("/nonexistent/path", []string{"tf.tmpl"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}
