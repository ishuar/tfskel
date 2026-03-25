package generate

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ishuar/tfskel/internal/templates"
)

// extractMetadata extracts JSON metadata from a comment line in format: ## tfskel-metadata: {...}
func extractMetadata(content, metadataKey string) (map[string]string, error) {
	// Look for pattern: ## tfskel-<metadataKey>: {JSON}
	pattern := fmt.Sprintf(`##\s*tfskel-%s:\s*({[^}]*})`, regexp.QuoteMeta(metadataKey))
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("%w: %s", ErrMetadataKeyNotFound, metadataKey)
	}

	var metadata map[string]string
	if err := json.Unmarshal([]byte(matches[1]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	// Normalize keys to lowercase for tags metadata (Terraform convention)
	// This is needed when READING from files to ensure consistent comparison
	// with tags from config (which are also normalized in prepareTemplateData)
	if metadataKey == "tags" {
		normalized := make(map[string]string, len(metadata))
		for k, v := range metadata {
			normalized[strings.ToLower(k)] = v
		}
		return normalized, nil
	}

	return metadata, nil
}

// buildBackendMetadata creates metadata map for backend.tf
func buildBackendMetadata(bucketName string) map[string]string {
	return map[string]string{
		"bucket": bucketName,
	}
}

// buildVersionsMetadata creates metadata map for versions.tf (terraform version and provider version)
func buildVersionsMetadata(tfVersion, awsProviderVersion string) map[string]string {
	return map[string]string{
		"tf_ver":           tfVersion,
		"aws_provider_ver": awsProviderVersion,
	}
}

// compareMetadata returns true if metadata maps differ, along with list of changes
func compareMetadata(fileMetadata, configMetadata map[string]string) (bool, []string) {
	var changes []string

	// Check for added or changed keys
	for key, configValue := range configMetadata {
		if fileValue, exists := fileMetadata[key]; !exists {
			changes = append(changes, fmt.Sprintf("%s added: %s", key, configValue))
		} else if fileValue != configValue {
			changes = append(changes, fmt.Sprintf("%s changed: %s -> %s", key, fileValue, configValue))
		}
	}

	// Check for removed keys
	for key, fileValue := range fileMetadata {
		if _, exists := configMetadata[key]; !exists {
			changes = append(changes, fmt.Sprintf("%s removed (was: %s)", key, fileValue))
		}
	}

	return len(changes) > 0, changes
}

// metadataCommentsForTemplate returns the tfskel-metadata and tfskel-tags-hash comment lines
// for known template types. Returns empty strings for templates that don't need metadata injection.
func metadataCommentsForTemplate(tmplName string, data *templates.Data, fileExt string) (metadataComment, tagsHashComment string) {
	switch tmplName {
	case tmplBackendTF:
		meta := buildBackendMetadata(data.S3BucketName)
		return BuildMetadataComment(meta, fileExt), ""
	case tmplVersionsTF:
		meta := buildVersionsMetadata(data.TerraformVersion, data.AWSProviderVersion)
		return BuildMetadataComment(meta, fileExt), BuildTagsHashComment(data.DefaultTags, fileExt)
	default:
		return "", ""
	}
}

// updateBackendIfNeeded checks and updates backend.tf if bucket_name changed
func (g *Generator) updateBackendIfNeeded(appPath string, data *templates.Data) error {
	backendPath := filepath.Join(appPath, "backend.tf")
	if !g.fs.FileExists(backendPath) {
		return nil
	}

	needsUpdate, err := g.shouldUpdateBackend(backendPath, data.S3BucketName)
	if err != nil {
		return fmt.Errorf("failed to check backend.tf for updates: %w", err)
	}

	if needsUpdate {
		if err := g.updateBackendFile(backendPath, data); err != nil {
			return fmt.Errorf("failed to update backend.tf: %w", err)
		}
		g.log.Successf("Updated backend.tf with new bucket_name: %s", data.S3BucketName)
	}

	return nil
}

// updateVersionsIfNeeded checks and updates versions.tf if configuration changed
func (g *Generator) updateVersionsIfNeeded(appPath string, data *templates.Data) error {
	versionsPath := filepath.Join(appPath, "versions.tf")
	if !g.fs.FileExists(versionsPath) {
		return nil
	}

	needsUpdate, changes, err := g.shouldUpdateVersions(versionsPath, data)
	if err != nil {
		return fmt.Errorf("failed to check versions.tf for updates: %w", err)
	}

	if needsUpdate {
		if err := g.updateVersionsFile(versionsPath, data); err != nil {
			return fmt.Errorf("failed to update versions.tf: %w", err)
		}
		for _, change := range changes {
			g.log.Successf("Updated versions.tf - %s", change)
		}
	}

	return nil
}

// shouldUpdateBackend checks if the backend.tf needs updating due to bucket_name changes
func (g *Generator) shouldUpdateBackend(backendPath string, expectedBucketName string) (bool, error) {
	content, err := g.fs.ReadFile(backendPath)
	if err != nil {
		return false, fmt.Errorf("failed to read backend.tf: %w", err)
	}

	// Extract metadata from file
	fileMetadata, err := extractMetadata(string(content), "metadata")
	if err != nil {
		// If no metadata found, file was not generated by tfskel with metadata support
		// This is expected for old files, so return true (needs update) but no error
		g.log.Debug("No metadata found in backend.tf, will regenerate to add metadata")
		return true, nil //nolint:nilerr // missing metadata is expected for old files, not an error
	}

	// Compare metadata with expected values
	configMetadata := buildBackendMetadata(expectedBucketName)
	needsUpdate, _ := compareMetadata(fileMetadata, configMetadata)

	return needsUpdate, nil
}

// shouldUpdateVersions checks if the versions.tf needs updating due to terraform_version,
// provider version, or default_tags changes
func (g *Generator) shouldUpdateVersions(versionsPath string, data *templates.Data) (bool, []string, error) {
	content, err := g.fs.ReadFile(versionsPath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read versions.tf: %w", err)
	}

	contentStr := string(content)
	var allChanges []string

	// Extract and compare metadata (terraform version and provider version)
	fileMetadata, err := extractMetadata(contentStr, "metadata")
	if err != nil {
		// If no metadata found, file was not generated by tfskel with metadata support
		// This is expected for old files, so return true (needs update) but no error
		g.log.Debugf("No metadata found in versions.tf, will regenerate with: terraform_version=%s, aws_provider_version=%s",
			data.TerraformVersion, data.AWSProviderVersion)
		return true, []string{"initialized configuration tracking (see details with -v flag)"}, nil //nolint:nilerr // missing metadata is expected for old files, not an error
	}

	configMetadata := buildVersionsMetadata(data.TerraformVersion, data.AWSProviderVersion)
	needsUpdate, changes := compareMetadata(fileMetadata, configMetadata)
	if needsUpdate {
		allChanges = append(allChanges, changes...)
	}

	// Extract and compare tags hash
	fileTagsHash, err := ExtractTagsHash(contentStr)
	configTagsHash := ComputeTagsHash(data.DefaultTags)
	if err != nil {
		// No tags hash found — could be old file with JSON-based tfskel-tags or no tags at all
		if len(data.DefaultTags) > 0 {
			allChanges = append(allChanges, "initialized tags tracking")
		}
	} else if fileTagsHash != configTagsHash {
		// Hash mismatch — tags have changed
		allChanges = append(allChanges, "default_tags changed")
	}

	return len(allChanges) > 0, allChanges, nil
}

// updateBackendFile regenerates the backend.tf file with updated configuration
func (g *Generator) updateBackendFile(backendPath string, data *templates.Data) error {
	return g.renderAndWriteFile(tmplBackendTF, backendPath, data)
}

// updateVersionsFile regenerates the versions.tf file with updated configuration
func (g *Generator) updateVersionsFile(versionsPath string, data *templates.Data) error {
	return g.renderAndWriteFile(tmplVersionsTF, versionsPath, data)
}
