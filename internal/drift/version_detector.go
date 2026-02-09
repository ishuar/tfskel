package drift

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

var (
	ErrHCLParsingWarnings = errors.New("HCL parsing warnings")
)

// Detector scans directories and extracts version information
type Detector struct {
	rootPath    string
	absRootPath string // Absolute path for comparison
}

// NewDetector creates a new version detector
func NewDetector(rootPath string) *Detector {
	// Expand home directory if present
	if strings.HasPrefix(rootPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			rootPath = filepath.Join(homeDir, rootPath[2:])
		}
	}

	// Get absolute path for proper comparison
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		absPath = rootPath
	}

	return &Detector{
		rootPath:    rootPath,
		absRootPath: absPath,
	}
}

// ScanDirectory walks the directory tree and extracts version information
func (d *Detector) ScanDirectory() ([]VersionInfo, error) {
	var results []VersionInfo
	seenFiles := make(map[string]bool) // Track files already processed

	err := filepath.WalkDir(d.rootPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden subdirectories (but not the root directory itself)
		if entry.IsDir() {
			name := entry.Name()
			// Get absolute path to compare with root
			absPath, err := filepath.Abs(path)
			if err == nil && absPath != d.absRootPath {
				// Skip hidden directories, but only for subdirectories
				if strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Only process .tf files
		if !strings.HasSuffix(entry.Name(), ".tf") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(d.rootPath, path)
		if err != nil {
			return err
		}

		// Check if we've already processed a versions.tf in this directory
		dirPath := filepath.Dir(relPath)
		if seenFiles[dirPath] && entry.Name() != "versions.tf" {
			return nil // Skip non-versions.tf if we already have one
		}

		// Extract version information
		versionInfo, err := d.extractVersionInfo(path, relPath)
		if err != nil {
			// Store the error and continue scanning - we record the error in results
			// but don't stop the walk since other files might be valid
			results = append(results, VersionInfo{
				FilePath:   relPath,
				ParseError: err,
			})
			// Continue walking, error is already recorded in results
			return nil //nolint:nilerr // error is captured in results slice
		}

		// Only add if we found version information
		if versionInfo.TerraformVersion != "" || len(versionInfo.Providers) > 0 {
			results = append(results, versionInfo)
			seenFiles[dirPath] = true
		}

		return nil
	})

	return results, err
}

// extractVersionInfo parses a Terraform file and extracts version information using HCL parser
func (d *Detector) extractVersionInfo(path, relPath string) (VersionInfo, error) {
	return d.parseWithHCL(path, relPath)
}

// parseWithHCL uses the official HCL parser for accurate extraction
func (d *Detector) parseWithHCL(path, relPath string) (VersionInfo, error) {
	info := VersionInfo{
		FilePath:  relPath,
		Providers: make(map[string]ProviderVer),
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return VersionInfo{}, err
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(content, path)

	if diags.HasErrors() {
		// Just note parsing issues, don't fail - focus on version drift detection
		info.ParseError = fmt.Errorf("%w: %s", ErrHCLParsingWarnings, diags.Error())
	}

	if file == nil || file.Body == nil {
		return info, nil
	}

	// Extract from HCL body
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return info, nil
	}

	// Look for terraform block
	for _, block := range body.Blocks {
		if block.Type == "terraform" {
			d.extractTerraformBlock(block, &info)
		}
	}

	return info, nil
}

// extractTerraformBlock extracts version information from a terraform{} block
func (d *Detector) extractTerraformBlock(block *hclsyntax.Block, info *VersionInfo) {
	body := block.Body

	// Extract required_version
	if attr, exists := body.Attributes["required_version"]; exists {
		val, diags := attr.Expr.Value(nil)
		if !diags.HasErrors() && val.Type().FriendlyName() == "string" {
			info.TerraformVersion = val.AsString()
		}
	}

	// Extract required_providers block
	for _, innerBlock := range body.Blocks {
		if innerBlock.Type == "required_providers" {
			d.extractProvidersBlock(innerBlock, info)
		}
	}
}

// extractProvidersBlock extracts provider information from required_providers{}
func (d *Detector) extractProvidersBlock(block *hclsyntax.Block, info *VersionInfo) {
	body := block.Body

	// Iterate through provider attributes
	for name, attr := range body.Attributes {
		// Each provider is defined as: provider_name = { source = "...", version = "..." }
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			continue
		}

		if !val.Type().IsObjectType() && !val.Type().IsMapType() {
			continue
		}

		provider := ProviderVer{}

		// Extract source
		if sourceVal := val.GetAttr("source"); sourceVal.Type().FriendlyName() == "string" {
			provider.Source = sourceVal.AsString()
		}

		// Extract version
		if versionVal := val.GetAttr("version"); versionVal.Type().FriendlyName() == "string" {
			provider.Version = versionVal.AsString()
		}

		if provider.Source != "" || provider.Version != "" {
			info.Providers[name] = provider
		}
	}
}
