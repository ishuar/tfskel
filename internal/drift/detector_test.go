package drift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector("/test/path")
	assert.NotNil(t, detector)
	assert.Equal(t, "/test/path", detector.rootPath)
}

func TestNewDetector_HomeDirectory(t *testing.T) {
	detector := NewDetector("~/test")
	assert.NotNil(t, detector)
	// Should expand ~ to home directory
	assert.NotEqual(t, "~/test", detector.rootPath)
	if homeDir, err := os.UserHomeDir(); err == nil {
		expected := filepath.Join(homeDir, "test")
		assert.Equal(t, expected, detector.rootPath)
	}
}

func TestDetector_ParseWithHCL_ValidTerraformBlock(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "versions.tf")

	content := `terraform {
  required_version = "~> 1.16"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}`

	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	info, err := detector.parseWithHCL(testFile, "versions.tf")

	assert.NoError(t, err)
	assert.Equal(t, "versions.tf", info.FilePath)
	assert.Equal(t, "~> 1.16", info.TerraformVersion)
	assert.Len(t, info.Providers, 1)
	assert.Equal(t, "hashicorp/aws", info.Providers["aws"].Source)
	assert.Equal(t, "~> 6.0", info.Providers["aws"].Version)
}

func TestDetector_ParseWithHCL_MultipleProviders(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "versions.tf")

	content := `terraform {
  required_version = "~> 1.16"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }
}`

	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	info, err := detector.parseWithHCL(testFile, "versions.tf")

	assert.NoError(t, err)
	assert.Equal(t, "~> 1.16", info.TerraformVersion)
	assert.Len(t, info.Providers, 2)
	assert.Equal(t, "~> 6.0", info.Providers["aws"].Version)
	assert.Equal(t, "~> 3.5", info.Providers["random"].Version)
}

func TestDetector_ParseWithHCL_NoTerraformBlock(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.tf")

	content := `resource "aws_instance" "example" {
  ami           = "ami-123456"
  instance_type = "t2.micro"
}`

	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	info, err := detector.parseWithHCL(testFile, "main.tf")

	assert.NoError(t, err)
	assert.Equal(t, "main.tf", info.FilePath)
	assert.Empty(t, info.TerraformVersion)
	assert.Empty(t, info.Providers)
}

func TestDetector_ParseWithHCL_InvalidHCL(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "bad.tf")

	content := `terraform {
  required_version = "~> 1.16
  # Missing closing quote and brace
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	info, err := detector.parseWithHCL(testFile, "bad.tf")

	assert.NoError(t, err) // Parser is lenient
	assert.NotNil(t, info.ParseError)
}

func TestDetector_ScanDirectory_SimpleStructure(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Create versions.tf
	versionsContent := `terraform {
  required_version = "~> 1.16"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "versions.tf"), []byte(versionsContent), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	results, err := detector.ScanDirectory()

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "versions.tf", results[0].FilePath)
	assert.Equal(t, "~> 1.16", results[0].TerraformVersion)
}

func TestDetector_ScanDirectory_NestedStructure(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "envs", "dev")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create versions.tf in root
	rootVersions := `terraform {
  required_version = "~> 1.16"
}`
	err = os.WriteFile(filepath.Join(tmpDir, "versions.tf"), []byte(rootVersions), 0644)
	require.NoError(t, err)

	// Create versions.tf in subdirectory
	subVersions := `terraform {
  required_version = "~> 1.15"
}`
	err = os.WriteFile(filepath.Join(subDir, "versions.tf"), []byte(subVersions), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	results, err := detector.ScanDirectory()

	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Check both files were found
	foundPaths := make(map[string]bool)
	for _, result := range results {
		foundPaths[result.FilePath] = true
	}
	assert.True(t, foundPaths["versions.tf"])
	assert.True(t, foundPaths[filepath.Join("envs", "dev", "versions.tf")])
}

func TestDetector_ScanDirectory_SkipsHiddenDirs(t *testing.T) {
	// Create directory with hidden subdirectory
	tmpDir := t.TempDir()
	hiddenDir := filepath.Join(tmpDir, ".terraform")
	err := os.MkdirAll(hiddenDir, 0755)
	require.NoError(t, err)

	// Create versions.tf in hidden directory (should be skipped)
	hiddenVersions := `terraform {
  required_version = "~> 1.0"
}`
	err = os.WriteFile(filepath.Join(hiddenDir, "versions.tf"), []byte(hiddenVersions), 0644)
	require.NoError(t, err)

	// Create versions.tf in root (should be found)
	rootVersions := `terraform {
  required_version = "~> 1.16"
}`
	err = os.WriteFile(filepath.Join(tmpDir, "versions.tf"), []byte(rootVersions), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	results, err := detector.ScanDirectory()

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "versions.tf", results[0].FilePath)
	assert.Equal(t, "~> 1.16", results[0].TerraformVersion)
}

func TestDetector_ScanDirectory_PrefersVersionsTf(t *testing.T) {
	// Create directory with both versions.tf and main.tf
	tmpDir := t.TempDir()

	versionsContent := `terraform {
  required_version = "~> 1.16"
}`
	err := os.WriteFile(filepath.Join(tmpDir, "versions.tf"), []byte(versionsContent), 0644)
	require.NoError(t, err)

	mainContent := `terraform {
  required_version = "~> 1.15"
}`
	err = os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainContent), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	results, err := detector.ScanDirectory()

	assert.NoError(t, err)
	// Should return both files but prioritize versions.tf when found
	assert.GreaterOrEqual(t, len(results), 1)

	// versions.tf should be in the results
	foundVersionsTf := false
	for _, result := range results {
		if result.FilePath == "versions.tf" {
			foundVersionsTf = true
			assert.Equal(t, "~> 1.16", result.TerraformVersion)
		}
	}
	assert.True(t, foundVersionsTf, "versions.tf should be in results")
}

func TestDetector_ScanDirectory_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	detector := NewDetector(tmpDir)
	results, err := detector.ScanDirectory()

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestDetector_ScanDirectory_NoVersionInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .tf file without version information
	content := `resource "aws_instance" "example" {
  ami = "ami-123"
}`
	err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0644)
	require.NoError(t, err)

	detector := NewDetector(tmpDir)
	results, err := detector.ScanDirectory()

	assert.NoError(t, err)
	// Should not return files without version info
	assert.Empty(t, results)
}
