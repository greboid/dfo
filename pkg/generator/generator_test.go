package generator

import (
	"testing"
)

func TestBuildFetchCommand(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		dest     string
		extract  bool
		expected string
	}{
		{
			name:     "download without extraction",
			url:      "https://example.com/file.tar.gz",
			dest:     "/tmp/file.tar.gz",
			extract:  false,
			expected: "RUN curl -fsSL -o /tmp/file.tar.gz \"https://example.com/file.tar.gz\"\n",
		},
		{
			name:     "download with extraction",
			url:      "https://example.com/archive.tar.gz",
			dest:     "/app",
			extract:  true,
			expected: "RUN curl -fsSL \"https://example.com/archive.tar.gz\" | tar -xz -C \"/app\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFetchCommand(tt.url, tt.dest, tt.extract)
			if result != tt.expected {
				t.Errorf("buildFetchCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}
