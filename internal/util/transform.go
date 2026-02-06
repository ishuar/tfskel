package util

import (
	"strings"
)

// TransformRegionName converts AWS region names to shorter format
// Examples: eu-central-1 -> euc1, us-west-2 -> usw2, eu-west-1 -> euw1
func TransformRegionName(region string) string {
	parts := strings.Split(region, "-")
	var shortName strings.Builder

	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		// Check part type and append appropriately
		switch {
		case part[0] >= '0' && part[0] <= '9':
			// Numbers: keep as-is
			shortName.WriteString(part)
		case len(part) <= 2:
			// Keep short parts as-is (eu, us)
			shortName.WriteString(part)
		default:
			// Take first letter only
			shortName.WriteByte(part[0])
		}
	}

	return shortName.String()
}
