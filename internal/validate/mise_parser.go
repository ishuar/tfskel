package validate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

// ErrMiseTomlNotFound is returned when .mise.toml does not exist in the given directory.
var ErrMiseTomlNotFound = errors.New(".mise.toml not found")

// MiseToolConfig holds the parsed [tools] section from a .mise.toml file.
type MiseToolConfig struct {
	Tools map[string]string
}

// miseFileRaw mirrors the raw TOML structure. The [tools] section can contain
// either string values ("1.13.0") or array values (["1.13.0", "1.12.0"]).
type miseFileRaw struct {
	Tools map[string]any `toml:"tools"`
}

// ParseMiseToml reads and parses .mise.toml from the given directory.
// Returns ErrMiseTomlNotFound if the file does not exist.
func ParseMiseToml(dir string) (*MiseToolConfig, error) {
	path := filepath.Join(dir, ".mise.toml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMiseTomlNotFound
		}
		return nil, fmt.Errorf("failed to read .mise.toml: %w", err)
	}

	var raw miseFileRaw
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse .mise.toml: %w", err)
	}

	result := &MiseToolConfig{
		Tools: make(map[string]string, len(raw.Tools)),
	}

	for name, val := range raw.Tools {
		switch v := val.(type) {
		case string:
			result.Tools[name] = v
		case []any:
			// mise supports arrays like ["1.13.0", "1.12.0"]; take the first element.
			if len(v) > 0 {
				if s, ok := v[0].(string); ok {
					result.Tools[name] = s
				}
			}
		}
	}

	return result, nil
}
