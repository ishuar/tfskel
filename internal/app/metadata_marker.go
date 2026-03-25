package app

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

	// tagsHashPattern matches both ## tfskel-tags-hash: <hex> (HCL) and # tfskel-tags-hash: <hex> (YAML/other)
	tagsHashPattern = regexp.MustCompile(`#[#]?\s*tfskel-tags-hash:\s*([0-9a-f]+)`)
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

// commentPrefix returns "##" for .tf/.hcl files, "#" for everything else.
func commentPrefix(fileExt string) string {
	if fileExt == ".tf" || fileExt == ".hcl" {
		return "##"
	}
	return "#"
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
	return fmt.Sprintf("%s tfskel-metadata: %s", commentPrefix(fileExt), string(data))
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
	return fmt.Sprintf("%s tfskel-tags-hash: %s", commentPrefix(fileExt), hash)
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
