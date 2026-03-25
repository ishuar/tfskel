package generate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMetadataComment(t *testing.T) {
	t.Run("uses ## prefix for .tf files", func(t *testing.T) {
		comment := BuildMetadataComment(map[string]string{"bucket": "my-bucket"}, ".tf")
		assert.Contains(t, comment, "## tfskel-metadata:")
		assert.Contains(t, comment, `"bucket":"my-bucket"`)
	})

	t.Run("uses # prefix for .yaml files", func(t *testing.T) {
		comment := BuildMetadataComment(map[string]string{"key": "val"}, ".yaml")
		assert.True(t, strings.HasPrefix(comment, "# tfskel-metadata:"))
		assert.NotContains(t, comment, "## tfskel-metadata:")
	})

	t.Run("returns empty for empty map", func(t *testing.T) {
		assert.Empty(t, BuildMetadataComment(map[string]string{}, ".tf"))
	})

	t.Run("returns empty for nil map", func(t *testing.T) {
		assert.Empty(t, BuildMetadataComment(nil, ".tf"))
	})

	t.Run("produces deterministic sorted keys", func(t *testing.T) {
		meta := map[string]string{"z_key": "z", "a_key": "a", "m_key": "m"}
		comment := BuildMetadataComment(meta, ".tf")
		// json.Marshal sorts map keys alphabetically
		assert.Contains(t, comment, `"a_key":"a","m_key":"m","z_key":"z"`)
	})
}

func TestComputeTagsHash(t *testing.T) {
	t.Run("returns empty for empty map", func(t *testing.T) {
		assert.Empty(t, ComputeTagsHash(map[string]string{}))
	})

	t.Run("returns empty for nil map", func(t *testing.T) {
		assert.Empty(t, ComputeTagsHash(nil))
	})

	t.Run("returns 16 hex chars", func(t *testing.T) {
		hash := ComputeTagsHash(map[string]string{"managed_by": "terraform"})
		assert.Len(t, hash, 16)
		assert.Regexp(t, `^[0-9a-f]{16}$`, hash)
	})

	t.Run("is deterministic", func(t *testing.T) {
		tags := map[string]string{"managed_by": "terraform", "team": "platform"}
		h1 := ComputeTagsHash(tags)
		h2 := ComputeTagsHash(tags)
		assert.Equal(t, h1, h2)
	})

	t.Run("same keys produce same hash regardless of insertion order", func(t *testing.T) {
		tags1 := map[string]string{"a": "1", "b": "2", "c": "3"}
		tags2 := map[string]string{"c": "3", "a": "1", "b": "2"}
		assert.Equal(t, ComputeTagsHash(tags1), ComputeTagsHash(tags2))
	})

	t.Run("different tags produce different hash", func(t *testing.T) {
		h1 := ComputeTagsHash(map[string]string{"managed_by": "terraform"})
		h2 := ComputeTagsHash(map[string]string{"managed_by": "tfskel"})
		assert.NotEqual(t, h1, h2)
	})
}

func TestBuildTagsHashComment(t *testing.T) {
	t.Run("uses ## prefix for .tf files", func(t *testing.T) {
		comment := BuildTagsHashComment(map[string]string{"managed_by": "terraform"}, ".tf")
		assert.True(t, strings.HasPrefix(comment, "## tfskel-tags-hash:"))
	})

	t.Run("uses # prefix for .yaml files", func(t *testing.T) {
		comment := BuildTagsHashComment(map[string]string{"managed_by": "terraform"}, ".yaml")
		assert.True(t, strings.HasPrefix(comment, "# tfskel-tags-hash:"))
		assert.NotContains(t, comment, "## tfskel-tags-hash:")
	})

	t.Run("returns empty for empty map", func(t *testing.T) {
		assert.Empty(t, BuildTagsHashComment(map[string]string{}, ".tf"))
	})

	t.Run("contains hash value", func(t *testing.T) {
		tags := map[string]string{"managed_by": "terraform"}
		comment := BuildTagsHashComment(tags, ".tf")
		expectedHash := ComputeTagsHash(tags)
		assert.Contains(t, comment, expectedHash)
	})
}

func TestExtractTagsHash(t *testing.T) {
	t.Run("extracts hash from HCL comment", func(t *testing.T) {
		content := `## tfskel-source: {"template":"tf/versions.tf.tmpl","hash":"abc123"}
## tfskel-metadata: {"tf_ver":"~> 1.14"}
## tfskel-tags-hash: a1b2c3d4e5f67890
terraform {}`
		hash, err := ExtractTagsHash(content)
		require.NoError(t, err)
		assert.Equal(t, "a1b2c3d4e5f67890", hash)
	})

	t.Run("extracts hash from YAML comment", func(t *testing.T) {
		content := `# tfskel-tags-hash: abcdef0123456789
name: test`
		hash, err := ExtractTagsHash(content)
		require.NoError(t, err)
		assert.Equal(t, "abcdef0123456789", hash)
	})

	t.Run("returns error when no marker present", func(t *testing.T) {
		content := `## tfskel-source: {"template":"tf/versions.tf.tmpl","hash":"abc123"}
terraform {}`
		_, err := ExtractTagsHash(content)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTagsHashNotFound)
	})

	t.Run("handles extra whitespace", func(t *testing.T) {
		content := `##  tfskel-tags-hash:  a1b2c3d4e5f67890
terraform {}`
		hash, err := ExtractTagsHash(content)
		require.NoError(t, err)
		assert.Equal(t, "a1b2c3d4e5f67890", hash)
	})
}

func TestInjectMetadataMarkers(t *testing.T) {
	t.Run("inserts after first line", func(t *testing.T) {
		content := "## tfskel-source: {}\nterraform {\n}"
		result := InjectMetadataMarkers(content, "## tfskel-metadata: {\"bucket\":\"b\"}")
		lines := strings.Split(result, "\n")
		assert.Equal(t, "## tfskel-source: {}", lines[0])
		assert.Equal(t, "## tfskel-metadata: {\"bucket\":\"b\"}", lines[1])
		assert.Equal(t, "terraform {", lines[2])
	})

	t.Run("inserts multiple comments in order", func(t *testing.T) {
		content := "## tfskel-source: {}\nterraform {}"
		result := InjectMetadataMarkers(content,
			"## tfskel-metadata: {\"tf_ver\":\"1.14\"}",
			"## tfskel-tags-hash: abc123",
		)
		lines := strings.Split(result, "\n")
		assert.Equal(t, "## tfskel-source: {}", lines[0])
		assert.Equal(t, "## tfskel-metadata: {\"tf_ver\":\"1.14\"}", lines[1])
		assert.Equal(t, "## tfskel-tags-hash: abc123", lines[2])
		assert.Equal(t, "terraform {}", lines[3])
	})

	t.Run("skips empty comments", func(t *testing.T) {
		content := "## tfskel-source: {}\nterraform {}"
		result := InjectMetadataMarkers(content, "## tfskel-metadata: {}", "", "")
		lines := strings.Split(result, "\n")
		assert.Len(t, lines, 3)
		assert.Equal(t, "## tfskel-metadata: {}", lines[1])
	})

	t.Run("returns unchanged content when all comments empty", func(t *testing.T) {
		content := "## tfskel-source: {}\nterraform {}"
		result := InjectMetadataMarkers(content, "", "")
		assert.Equal(t, content, result)
	})

	t.Run("handles content without newline", func(t *testing.T) {
		content := "## tfskel-source: {}"
		result := InjectMetadataMarkers(content, "## tfskel-metadata: {\"k\":\"v\"}")
		assert.Equal(t, "## tfskel-source: {}\n## tfskel-metadata: {\"k\":\"v\"}", result)
	})
}
