package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
