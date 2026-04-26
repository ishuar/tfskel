package generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrSourceMarkerNotFound indicates no tfskel-source marker was found in file content
	ErrSourceMarkerNotFound = errors.New("tfskel-source marker not found")

	// sourceMarkerPattern matches the tfskel-source payload regardless of comment syntax
	// (#, ##, or <!-- ... -->) — the key name is the anchor, surrounding delimiters are noise.
	sourceMarkerPattern = regexp.MustCompile(`tfskel-source:\s*(\{[^}]*\})`)
)

// SourceMarker holds the parsed tfskel-source metadata embedded in generated files
type SourceMarker struct {
	Template string `json:"template"`
	Hash     string `json:"hash"`
}

// BuildSourceComment builds the source marker comment line for a given file extension.
// Comment syntax is dispatched via commentFormat (## for .tf/.hcl, <!-- ... --> for .md/.markdown,
// # for everything else).
func BuildSourceComment(templateName, hash, fileExt string) string {
	marker := SourceMarker{
		Template: templateName,
		Hash:     hash,
	}
	data, err := json.Marshal(marker)
	if err != nil {
		// SourceMarker contains only simple strings; this should never happen
		return "# tfskel-source: {}"
	}

	return formatComment(fileExt, "tfskel-source: "+string(data))
}

// ExtractSourceMarker extracts the tfskel-source metadata from file content.
// Returns ErrSourceMarkerNotFound if no marker is present.
func ExtractSourceMarker(content string) (*SourceMarker, error) {
	matches := sourceMarkerPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, ErrSourceMarkerNotFound
	}

	var marker SourceMarker
	if err := json.Unmarshal([]byte(matches[1]), &marker); err != nil {
		return nil, fmt.Errorf("failed to parse source marker JSON: %w", err)
	}

	return &marker, nil
}

// InjectSourceMarker prepends the source marker comment as the first line of content.
func InjectSourceMarker(content, comment string) string {
	return comment + "\n" + content
}

// supportsSourceMarker returns true if the output file format supports comment-based markers.
// Files like .terraform-version contain only a bare version string and would break if a
// comment line were prepended.
func supportsSourceMarker(outputPath string) bool {
	return filepath.Base(outputPath) != ".terraform-version"
}

// SourceCommentForFile is a convenience that builds a source comment using the output file's extension.
// Returns an empty string for files that do not support comment-based markers.
func SourceCommentForFile(templateName, hash, outputPath string) string {
	if !supportsSourceMarker(outputPath) {
		return ""
	}
	ext := filepath.Ext(strings.TrimSuffix(outputPath, ".tmpl"))
	if ext == "" {
		// Files like .gitignore have no extension after the dot
		ext = filepath.Ext(filepath.Base(outputPath))
	}
	return BuildSourceComment(templateName, hash, ext)
}
