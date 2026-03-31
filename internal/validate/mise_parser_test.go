package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMiseToml(t *testing.T) {
	t.Run("parses simple string versions", func(t *testing.T) {
		dir := t.TempDir()
		writeMiseToml(t, dir, `
[tools]
terraform = "1.13.0"
tflint = "0.50.0"
trivy = "0.58.2"
`)
		cfg, err := ParseMiseToml(dir)
		require.NoError(t, err)
		assert.Equal(t, "1.13.0", cfg.Tools["terraform"])
		assert.Equal(t, "0.50.0", cfg.Tools["tflint"])
		assert.Equal(t, "0.58.2", cfg.Tools["trivy"])
	})

	t.Run("parses array versions taking first element", func(t *testing.T) {
		dir := t.TempDir()
		writeMiseToml(t, dir, `
[tools]
terraform = ["1.13.0", "1.12.0"]
`)
		cfg, err := ParseMiseToml(dir)
		require.NoError(t, err)
		assert.Equal(t, "1.13.0", cfg.Tools["terraform"])
	})

	t.Run("returns error when file not found", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ParseMiseToml(dir)
		assert.ErrorIs(t, err, ErrMiseTomlNotFound)
	})

	t.Run("returns error for invalid TOML", func(t *testing.T) {
		dir := t.TempDir()
		writeMiseToml(t, dir, `not valid toml [[[`)
		_, err := ParseMiseToml(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse .mise.toml")
	})

	t.Run("handles empty tools section", func(t *testing.T) {
		dir := t.TempDir()
		writeMiseToml(t, dir, `
min_version = "2024.9.0"

[tools]
`)
		cfg, err := ParseMiseToml(dir)
		require.NoError(t, err)
		assert.Empty(t, cfg.Tools)
	})

	t.Run("handles file with no tools section", func(t *testing.T) {
		dir := t.TempDir()
		writeMiseToml(t, dir, `
min_version = "2024.9.0"
`)
		cfg, err := ParseMiseToml(dir)
		require.NoError(t, err)
		assert.Empty(t, cfg.Tools)
	})

	t.Run("handles latest as version", func(t *testing.T) {
		dir := t.TempDir()
		writeMiseToml(t, dir, `
[tools]
terraform = "latest"
`)
		cfg, err := ParseMiseToml(dir)
		require.NoError(t, err)
		assert.Equal(t, "latest", cfg.Tools["terraform"])
	})
}

func writeMiseToml(t *testing.T, dir, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, ".mise.toml"), []byte(content), 0o644)
	require.NoError(t, err)
}
