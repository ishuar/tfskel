package format

import (
	"os"
	"testing"
)

func TestShouldUseColor(t *testing.T) {
	// envVars lists all environment variables that ShouldUseColor inspects.
	// Each subtest clears these before applying its own values.
	envVars := []string{"NO_COLOR", "FORCE_COLOR", "CI"}

	tests := []struct {
		name        string
		noColorFlag bool
		env         map[string]*string // nil value = unset, non-nil = set (even if empty)
		want        bool
	}{
		// Flag-only behavior
		{
			name:        "flag false, no env - colors enabled",
			noColorFlag: false,
			want:        true,
		},
		{
			name:        "flag true, no env - colors disabled",
			noColorFlag: true,
			want:        false,
		},

		// NO_COLOR (presence-based, per https://no-color.org/)
		{
			name: "NO_COLOR=1 - colors disabled",
			env:  map[string]*string{"NO_COLOR": new("1")},
			want: false,
		},
		{
			name: "NO_COLOR empty string - colors disabled (presence is enough)",
			env:  map[string]*string{"NO_COLOR": new("")},
			want: false,
		},
		{
			name: "NO_COLOR beats FORCE_COLOR",
			env:  map[string]*string{"NO_COLOR": new("1"), "FORCE_COLOR": new("1")},
			want: false,
		},

		// FORCE_COLOR
		{
			name:        "FORCE_COLOR=1 overrides flag",
			noColorFlag: true,
			env:         map[string]*string{"FORCE_COLOR": new("1")},
			want:        true,
		},
		{
			name:        "FORCE_COLOR=true overrides flag",
			noColorFlag: true,
			env:         map[string]*string{"FORCE_COLOR": new("true")},
			want:        true,
		},
		{
			name:        "FORCE_COLOR=yes overrides flag",
			noColorFlag: true,
			env:         map[string]*string{"FORCE_COLOR": new("yes")},
			want:        true,
		},
		{
			name:        "FORCE_COLOR=0 does not override flag",
			noColorFlag: true,
			env:         map[string]*string{"FORCE_COLOR": new("0")},
			want:        false,
		},
		{
			name:        "FORCE_COLOR=false does not override flag",
			noColorFlag: true,
			env:         map[string]*string{"FORCE_COLOR": new("false")},
			want:        false,
		},

		// CI environment variable
		{
			name: "CI=true - colors disabled",
			env:  map[string]*string{"CI": new("true")},
			want: false,
		},
		{
			name: "CI=1 - colors disabled",
			env:  map[string]*string{"CI": new("1")},
			want: false,
		},
		{
			name: "CI=false - falls back to flag",
			env:  map[string]*string{"CI": new("false")},
			want: true,
		},

		// Precedence: NO_COLOR > FORCE_COLOR > CI > flag
		{
			name: "FORCE_COLOR overrides CI",
			env:  map[string]*string{"CI": new("true"), "FORCE_COLOR": new("1")},
			want: true,
		},
		{
			name: "NO_COLOR overrides FORCE_COLOR and CI",
			env:  map[string]*string{"NO_COLOR": new("1"), "FORCE_COLOR": new("1"), "CI": new("true")},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unset all relevant env vars to create a clean baseline.
			// t.Setenv registers cleanup to restore original values after the subtest.
			for _, key := range envVars {
				t.Setenv(key, "")
				os.Unsetenv(key)
			}

			// Apply test-specific env vars.
			for key, val := range tt.env {
				if val != nil {
					t.Setenv(key, *val)
				}
			}

			got := ShouldUseColor(tt.noColorFlag)
			if got != tt.want {
				t.Errorf("ShouldUseColor(%v) = %v, want %v", tt.noColorFlag, got, tt.want)
			}
		})
	}
}
