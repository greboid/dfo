package versions

import (
	"testing"
)

func TestVersionSpec_IsLatest(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "latest prefix",
			value:    "latest",
			expected: true,
		},
		{
			name:     "latest with qualifier",
			value:    "latest/stable",
			expected: true,
		},
		{
			name:     "specific version",
			value:    "v1.0.0",
			expected: false,
		},
		{
			name:     "semver",
			value:    "1.2.3",
			expected: false,
		},
		{
			name:     "empty string",
			value:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := VersionSpec{Value: tt.value}
			result := spec.IsLatest()
			if result != tt.expected {
				t.Errorf("VersionSpec.IsLatest() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionSpec_IsGitRepo(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "https URL",
			key:      "https://github.com/owner/repo",
			expected: true,
		},
		{
			name:     "http URL",
			key:      "http://github.com/owner/repo",
			expected: true,
		},
		{
			name:     "non-URL key",
			key:      "postgres",
			expected: false,
		},
		{
			name:     "empty key",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := VersionSpec{Key: tt.key}
			result := spec.IsGitRepo()
			if result != tt.expected {
				t.Errorf("VersionSpec.IsGitRepo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionSpec_VersionType(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "https URL is git",
			key:      "https://github.com/owner/repo",
			expected: "git",
		},
		{
			name:     "http URL is git",
			key:      "http://gitlab.com/owner/repo",
			expected: "git",
		},
		{
			name:     "go key",
			key:      "go",
			expected: "go",
		},
		{
			name:     "golang key",
			key:      "golang",
			expected: "go",
		},
		{
			name:     "postgres key",
			key:      "postgres",
			expected: "postgres",
		},
		{
			name:     "postgresql key",
			key:      "postgresql",
			expected: "postgres",
		},
		{
			name:     "postgres15 key",
			key:      "postgres15",
			expected: "postgres",
		},
		{
			name:     "alpine key",
			key:      "alpine",
			expected: "alpine",
		},
		{
			name:     "unknown key",
			key:      "unknown-package",
			expected: "unknown",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := VersionSpec{Key: tt.key}
			result := spec.VersionType()
			if result != tt.expected {
				t.Errorf("VersionSpec.VersionType() = %q, want %q", result, tt.expected)
			}
		})
	}
}
