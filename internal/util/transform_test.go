package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimAndValidateInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		fieldName    string
		wantResult   string
		wantError    bool
		errorMessage string
	}{
		{
			name:       "valid input with no whitespace",
			input:      "myapp",
			fieldName:  "app directory",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "valid input with leading whitespace",
			input:      "  myapp",
			fieldName:  "app directory",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "valid input with trailing whitespace",
			input:      "myapp  ",
			fieldName:  "app directory",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "valid input with leading and trailing whitespace",
			input:      "  myapp  ",
			fieldName:  "app directory",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "valid input with tabs",
			input:      "\tmyapp\t",
			fieldName:  "app directory",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "valid input with mixed whitespace",
			input:      " \t myapp \n ",
			fieldName:  "app directory",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "preserves internal whitespace",
			input:      "  my app  ",
			fieldName:  "app directory",
			wantResult: "my app",
			wantError:  false,
		},
		{
			name:         "empty string",
			input:        "",
			fieldName:    "app directory",
			wantResult:   "",
			wantError:    true,
			errorMessage: "app directory cannot be empty or contain only whitespace",
		},
		{
			name:         "only spaces",
			input:        "   ",
			fieldName:    "app directory",
			wantResult:   "",
			wantError:    true,
			errorMessage: "app directory cannot be empty or contain only whitespace",
		},
		{
			name:         "only tabs",
			input:        "\t\t",
			fieldName:    "environment",
			wantResult:   "",
			wantError:    true,
			errorMessage: "environment cannot be empty or contain only whitespace",
		},
		{
			name:         "only newlines",
			input:        "\n\n",
			fieldName:    "region",
			wantResult:   "",
			wantError:    true,
			errorMessage: "region cannot be empty or contain only whitespace",
		},
		{
			name:         "mixed whitespace only",
			input:        " \t\n ",
			fieldName:    "app directory",
			wantResult:   "",
			wantError:    true,
			errorMessage: "app directory cannot be empty or contain only whitespace",
		},
		{
			name:       "empty fieldName with valid input",
			input:      "  myapp  ",
			fieldName:  "",
			wantResult: "myapp",
			wantError:  false,
		},
		{
			name:       "empty fieldName with invalid input",
			input:      "   ",
			fieldName:  "",
			wantResult: "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TrimAndValidateInput(tt.input, tt.fieldName)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.EqualError(t, err, tt.errorMessage)
				}
				assert.Equal(t, tt.wantResult, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func TestTrimAndValidateInput_ErrEmptyInput(t *testing.T) {
	t.Run("returns ErrEmptyInput when fieldName is empty", func(t *testing.T) {
		_, err := TrimAndValidateInput("   ", "")
		assert.ErrorIs(t, err, ErrEmptyInput)
	})

	t.Run("does not return ErrEmptyInput when fieldName is provided", func(t *testing.T) {
		_, err := TrimAndValidateInput("   ", "app directory")
		assert.NotErrorIs(t, err, ErrEmptyInput)
		assert.Contains(t, err.Error(), "app directory")
	})
}

func TestReplaceSpacesWithHyphens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no spaces",
			input:    "myapp",
			expected: "myapp",
		},
		{
			name:     "single space",
			input:    "my app",
			expected: "my-app",
		},
		{
			name:     "multiple spaces",
			input:    "my complex app",
			expected: "my-complex-app",
		},
		{
			name:     "consecutive spaces collapsed",
			input:    "my  app",
			expected: "my-app",
		},
		{
			name:     "mixed consecutive spaces collapsed",
			input:    "my  complex   app",
			expected: "my-complex-app",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "",
		},
		{
			name:     "spaces with hyphens",
			input:    "my-app with spaces",
			expected: "my-app-with-spaces",
		},
		{
			name:     "spaces with underscores",
			input:    "my_app with spaces",
			expected: "my_app-with-spaces",
		},
		{
			name:     "leading and trailing spaces removed",
			input:    "  my app  ",
			expected: "my-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceSpacesWithHyphens(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformRegionName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "eu-central-1",
			input:    "eu-central-1",
			expected: "euc1",
		},
		{
			name:     "us-west-2",
			input:    "us-west-2",
			expected: "usw2",
		},
		{
			name:     "eu-west-1",
			input:    "eu-west-1",
			expected: "euw1",
		},
		{
			name:     "us-east-1",
			input:    "us-east-1",
			expected: "use1",
		},
		{
			name:     "ap-southeast-2",
			input:    "ap-southeast-2",
			expected: "aps2",
		},
		{
			name:     "edge case with empty part",
			input:    "us--east-1",
			expected: "use1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformRegionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
