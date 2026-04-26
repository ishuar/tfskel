package generate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrTagsHashNotFound indicates no tfskel-tags-hash marker was found in file content
	ErrTagsHashNotFound = errors.New("tfskel-tags-hash marker not found")

	// tagsHashPattern matches the tfskel-tags-hash payload regardless of comment syntax
	// (#, ##, or <!-- ... -->) — the key name is the anchor.
	tagsHashPattern = regexp.MustCompile(`tfskel-tags-hash:\s*([0-9a-f]+)`)
)

// marshalRaw marshals v to JSON without escaping HTML characters (<, >, &).
// Go's json.Marshal escapes these by default, producing \u003e instead of >,
// which is undesirable in HCL/YAML comment metadata.
func marshalRaw(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encode appends a trailing newline; trim it
	b := buf.Bytes()
	return bytes.TrimRight(b, "\n"), nil
}

// commentFormat maps a file extension to the fmt.Sprintf template used to wrap
// a metadata body in that file's native comment syntax. Unlisted extensions
// fall back to "# %s" (line-comment with single hash).
var commentFormat = map[string]string{
	".tf":       "## %s",
	".hcl":      "## %s",
	".md":       "<!-- %s -->",
	".markdown": "<!-- %s -->",
}

// formatComment wraps body in the comment syntax appropriate for fileExt.
func formatComment(fileExt, body string) string {
	f, ok := commentFormat[fileExt]
	if !ok {
		f = "# %s"
	}
	return fmt.Sprintf(f, body)
}

// BuildMetadataComment builds a tfskel-metadata comment line for the given metadata map.
// Uses json.Marshal which sorts map keys alphabetically for deterministic output.
func BuildMetadataComment(metadata map[string]string, fileExt string) string {
	if len(metadata) == 0 {
		return ""
	}
	// map[string]string always marshals successfully; err is unreachable
	data, err := marshalRaw(metadata)
	if err != nil {
		return ""
	}
	return formatComment(fileExt, "tfskel-metadata: "+string(data))
}

// ComputeTagsHash produces a deterministic SHA-256 hash (first 16 hex chars) of a tags map.
// It marshals the map to JSON (Go's json.Marshal sorts keys alphabetically) and hashes the result.
func ComputeTagsHash(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	// map[string]string always marshals successfully; err is unreachable
	data, err := marshalRaw(tags)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8]) // 16 hex chars, matching GetTemplateHash convention
}

// BuildTagsHashComment builds a tfskel-tags-hash comment line for the given tags map.
// Returns an empty string if the tags map is empty.
func BuildTagsHashComment(tags map[string]string, fileExt string) string {
	hash := ComputeTagsHash(tags)
	if hash == "" {
		return ""
	}
	return formatComment(fileExt, "tfskel-tags-hash: "+hash)
}

// ExtractTagsHash extracts the tfskel-tags-hash value from file content.
// Returns ErrTagsHashNotFound if no marker is present.
func ExtractTagsHash(content string) (string, error) {
	matches := tagsHashPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", ErrTagsHashNotFound
	}
	return matches[1], nil
}

// InjectMetadataMarkers inserts non-empty comment lines after the first line of content
// (which is expected to be the tfskel-source marker). Empty strings are skipped.
func InjectMetadataMarkers(content string, comments ...string) string {
	// Collect non-empty comments
	var lines []string
	for _, c := range comments {
		if c != "" {
			lines = append(lines, c)
		}
	}
	if len(lines) == 0 {
		return content
	}

	// Split after first line (source marker) and insert metadata lines
	firstLine, rest, found := strings.Cut(content, "\n")
	if !found {
		// No newline — append after content
		return content + "\n" + strings.Join(lines, "\n")
	}

	return firstLine + "\n" + strings.Join(lines, "\n") + "\n" + rest
}
