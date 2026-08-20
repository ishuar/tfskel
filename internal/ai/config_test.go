package ai

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*viper.Viper)
		wantModel     string
		wantMaxTokens int
	}{
		{
			name:          "defaults when nothing configured",
			setup:         func(_ *viper.Viper) {},
			wantModel:     DefaultModel,
			wantMaxTokens: DefaultMaxTokens,
		},
		{
			name: "override model only",
			setup: func(v *viper.Viper) {
				v.Set("ai.model", "claude-opus-4-7")
			},
			wantModel:     "claude-opus-4-7",
			wantMaxTokens: DefaultMaxTokens,
		},
		{
			name: "override both fields",
			setup: func(v *viper.Viper) {
				v.Set("ai.model", "claude-haiku-4-5")
				v.Set("ai.max_tokens", 8192)
			},
			wantModel:     "claude-haiku-4-5",
			wantMaxTokens: 8192,
		},
		{
			name: "empty model falls back to default",
			setup: func(v *viper.Viper) {
				v.Set("ai.model", "")
			},
			wantModel:     DefaultModel,
			wantMaxTokens: DefaultMaxTokens,
		},
		{
			name: "non-positive max_tokens falls back to default",
			setup: func(v *viper.Viper) {
				v.Set("ai.max_tokens", 0)
			},
			wantModel:     DefaultModel,
			wantMaxTokens: DefaultMaxTokens,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			tt.setup(v)
			cfg := LoadConfig(v)
			assert.Equal(t, tt.wantModel, cfg.Model)
			assert.Equal(t, tt.wantMaxTokens, cfg.MaxTokens)
		})
	}
}
