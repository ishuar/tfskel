package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShouldUseColor verifies color detection logic with flags and environment variables
func TestShouldUseColor(t *testing.T) {
	tests := []struct {
		name           string
		noColorFlag    bool
		noColorEnv     string
		forceColorEnv  string
		expectedResult bool
	}{
		{
			name:           "no env vars, flag false - colors enabled",
			noColorFlag:    false,
			noColorEnv:     "",
			forceColorEnv:  "",
			expectedResult: true,
		},
		{
			name:           "no env vars, flag true - colors disabled",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "",
			expectedResult: false,
		},
		{
			name:           "NO_COLOR set - colors disabled (overrides flag)",
			noColorFlag:    false,
			noColorEnv:     "1",
			forceColorEnv:  "",
			expectedResult: false,
		},
		{
			name:           "NO_COLOR set - colors disabled (even with FORCE_COLOR)",
			noColorFlag:    false,
			noColorEnv:     "1",
			forceColorEnv:  "1",
			expectedResult: false,
		},
		{
			name:           "FORCE_COLOR=1 - colors enabled (overrides flag)",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "1",
			expectedResult: true,
		},
		{
			name:           "FORCE_COLOR=true - colors enabled",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "true",
			expectedResult: true,
		},
		{
			name:           "FORCE_COLOR=yes - colors enabled",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "yes",
			expectedResult: true,
		},
		{
			name:           "FORCE_COLOR=0 - colors disabled (respects flag)",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "0",
			expectedResult: false,
		},
		{
			name:           "FORCE_COLOR=false - colors disabled (respects flag)",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "false",
			expectedResult: false,
		},
		{
			name:           "FORCE_COLOR empty string - colors disabled (respects flag)",
			noColorFlag:    true,
			noColorEnv:     "",
			forceColorEnv:  "",
			expectedResult: false,
		},
		{
			name:           "precedence: NO_COLOR over FORCE_COLOR over flag",
			noColorFlag:    false,
			noColorEnv:     "any-value",
			forceColorEnv:  "1",
			expectedResult: false,
		},
		{
			name:           "FORCE_COLOR with no-color flag false - colors enabled",
			noColorFlag:    false,
			noColorEnv:     "",
			forceColorEnv:  "1",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use t.Setenv for automatic, race-safe cleanup (Go 1.17+)
			// It automatically restores the original value after the test
			if tt.noColorEnv != "" {
				t.Setenv("NO_COLOR", tt.noColorEnv)
			}
			if tt.forceColorEnv != "" {
				t.Setenv("FORCE_COLOR", tt.forceColorEnv)
			}

			// Test the function
			result := ShouldUseColor(tt.noColorFlag)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
