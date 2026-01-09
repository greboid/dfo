package generator

import (
	"testing"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/versions"
)

func TestBuildVarsMap(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.BuildConfig
		resolvedVer map[string]versions.VersionMetadata
		expected    map[string]string
	}{
		{
			name: "only config vars",
			config: &config.BuildConfig{
				Vars: map[string]string{"VERSION": "1.0.0"},
			},
			resolvedVer: nil,
			expected:    map[string]string{"VERSION": "1.0.0"},
		},
		{
			name: "config vars with resolved versions",
			config: &config.BuildConfig{
				Vars: map[string]string{"VERSION": "1.0.0"},
			},
			resolvedVer: map[string]versions.VersionMetadata{
				"prometheus": {Version: "v2.0.0", URL: "https://...", Checksum: "abc123"},
			},
			expected: map[string]string{
				"VERSION":                      "1.0.0",
				"versions.prometheus":          "v2.0.0",
				"versions.prometheus.url":      "https://...",
				"versions.prometheus.checksum": "abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				config:           tt.config,
				resolvedVersions: tt.resolvedVer,
			}
			result := g.buildVarsMap()
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("buildVarsMap()[%q] = %q, want %q", k, result[k], v)
				}
			}
			if len(result) != len(tt.expected) {
				t.Errorf("buildVarsMap() returned %d vars, want %d", len(result), len(tt.expected))
			}
		})
	}
}
