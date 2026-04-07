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

func TestRenderer_Render(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	t.Run("gitignore template", func(t *testing.T) {
		data := &Data{
			Region: "us-east-1",
		}

		content, err := renderer.Render("root/.gitignore.tmpl", data)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("versions.tf template", func(t *testing.T) {
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
	})

	t.Run("backend.tf template", func(t *testing.T) {
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
	})

	t.Run("versions.tf with default tags", func(t *testing.T) {
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
	})

	t.Run("non-existent template returns error", func(t *testing.T) {
		_, err := renderer.Render("nonexistent.tmpl", &Data{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template")
		assert.Contains(t, err.Error(), "not found")
	})
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
			result := StripConstraint(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderer_GetTemplateNames(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	names := renderer.GetTemplateNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "tf/versions.tf.tmpl")
	assert.Contains(t, names, "tf/backend.tf.tmpl")
	assert.Contains(t, names, "root/.gitignore.tmpl")
}

func TestRenderer_GetTemplateSource(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	t.Run("returns source for known template", func(t *testing.T) {
		source := renderer.GetTemplateSource("tf/versions.tf.tmpl")
		assert.Contains(t, source, "embedded:tf/versions.tf.tmpl")
	})

	t.Run("returns empty for non-existent template", func(t *testing.T) {
		source := renderer.GetTemplateSource("nonexistent.tmpl")
		assert.Empty(t, source)
	})
}

func TestNewRendererWithCustomTemplates(t *testing.T) {
	t.Run("with empty custom dir uses defaults", func(t *testing.T) {
		renderer, err := NewRendererWithCustomTemplates("")
		require.NoError(t, err)
		assert.NotNil(t, renderer)

		names := renderer.GetTemplateNames()
		assert.Contains(t, names, "tf/versions.tf.tmpl")
	})

	t.Run("with non-existent custom dir returns error", func(t *testing.T) {
		_, err := NewRendererWithCustomTemplates("/nonexistent/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRenderer_GetTemplateHash(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	t.Run("returns consistent hash for known template", func(t *testing.T) {
		hash1 := renderer.GetTemplateHash("tf/backend.tf.tmpl")
		hash2 := renderer.GetTemplateHash("tf/backend.tf.tmpl")
		assert.NotEmpty(t, hash1)
		assert.Equal(t, hash1, hash2, "same template should produce same hash")
		assert.Len(t, hash1, 16, "hash should be 16 hex chars")
	})

	t.Run("returns different hashes for different templates", func(t *testing.T) {
		hash1 := renderer.GetTemplateHash("tf/backend.tf.tmpl")
		hash2 := renderer.GetTemplateHash("tf/versions.tf.tmpl")
		assert.NotEqual(t, hash1, hash2, "different templates should have different hashes")
	})

	t.Run("returns empty string for unknown template", func(t *testing.T) {
		hash := renderer.GetTemplateHash("nonexistent/template.tmpl")
		assert.Empty(t, hash)
	})

	t.Run("hashes static content too", func(t *testing.T) {
		hash := renderer.GetTemplateHash("github/lint.yaml")
		assert.NotEmpty(t, hash, "static files should also have hashes")
	})
}

func TestBuildMiseTools(t *testing.T) {
	t.Run("includes all defaults as latest", func(t *testing.T) {
		result := BuildMiseTools()
		names := make([]string, len(result))
		for i, mt := range result {
			names[i] = mt.Name
		}
		assert.Equal(t, []string{"awscli", "pre-commit", "tflint", "trivy"}, names)
		for _, mt := range result {
			assert.Equal(t, "latest", mt.Version)
		}
	})

	t.Run("output is sorted alphabetically", func(t *testing.T) {
		result := BuildMiseTools()
		for i := 1; i < len(result); i++ {
			assert.True(t, result[i-1].Name < result[i].Name)
		}
	})
}

func TestRenderer_Render_MiseToml(t *testing.T) {
	renderer, err := NewRenderer()
	require.NoError(t, err)

	t.Run("renders with all tools as latest", func(t *testing.T) {
		data := &Data{
			TerraformVersion: "1.13.1",
		}

		content, err := renderer.Render("root/.mise.toml.tmpl", data)
		require.NoError(t, err)
		assert.Contains(t, content, `terraform = "1.13.1"`)
		assert.Contains(t, content, `tflint = "latest"`)
		assert.Contains(t, content, `trivy = "latest"`)
	})

	t.Run("strips constraint from terraform version", func(t *testing.T) {
		data := &Data{
			TerraformVersion: ">= 1.10.2",
		}

		content, err := renderer.Render("root/.mise.toml.tmpl", data)
		require.NoError(t, err)
		assert.Contains(t, content, `terraform = "1.10.2"`)
		assert.NotContains(t, content, ">=")
	})
}
