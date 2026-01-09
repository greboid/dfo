package pipelines

import (
	"strings"
	"testing"
)

func TestExtractGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "https GitHub URL",
			repoURL:  "https://github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "http GitHub URL",
			repoURL:  "http://github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "GitHub URL with .git suffix",
			repoURL:  "https://github.com/owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "GitHub URL with trailing slash",
			repoURL:  "https://github.com/owner/repo/",
			expected: "owner/repo",
		},
		{
			name:     "git@ syntax",
			repoURL:  "git@github.com:owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "github.com/ prefix without protocol",
			repoURL:  "github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "gitlab URL returns empty",
			repoURL:  "https://gitlab.com/owner/repo",
			expected: "",
		},
		{
			name:     "non-GitHub URL returns empty",
			repoURL:  "https://example.com/owner/repo",
			expected: "",
		},
		{
			name:     "empty string",
			repoURL:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractGitHubOwnerRepo(tt.repoURL)
			if result != tt.expected {
				t.Errorf("ExtractGitHubOwnerRepo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsTarArchive(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{filename: "file.tar", expected: true},
		{filename: "file.tar.gz", expected: true},
		{filename: "file.tgz", expected: true},
		{filename: "file.tar.bz2", expected: true},
		{filename: "file.tbz2", expected: true},
		{filename: "file.tar.xz", expected: true},
		{filename: "file.txz", expected: true},
		{filename: "file.zip", expected: false},
		{filename: "file.rar", expected: false},
		{filename: "file.txt", expected: false},
		{filename: "archive.tar.gz", expected: true},
		{filename: "data.tgz", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isTarArchive(tt.filename)
			if result != tt.expected {
				t.Errorf("isTarArchive(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestValidateArchiveFormat(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectError bool
	}{
		{
			name:        "valid tar.gz",
			filename:    "file.tar.gz",
			expectError: false,
		},
		{
			name:        "valid tgz",
			filename:    "file.tgz",
			expectError: false,
		},
		{
			name:        "valid tar.xz",
			filename:    "file.tar.xz",
			expectError: false,
		},
		{
			name:        "valid zip",
			filename:    "file.zip",
			expectError: false,
		},
		{
			name:        "unsupported rar",
			filename:    "file.rar",
			expectError: true,
		},
		{
			name:        "unsupported exe",
			filename:    "file.exe",
			expectError: true,
		},
		{
			name:        "unsupported 7z",
			filename:    "file.7z",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArchiveFormat(tt.filename)
			if (err != nil) != tt.expectError {
				t.Errorf("validateArchiveFormat(%q) error = %v, expectError %v", tt.filename, err, tt.expectError)
			}
		})
	}
}

func TestBuildExtractCommand(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		extractDir  string
		strip       int
		contains    string
	}{
		{
			name:        "zip archive",
			destination: "file.zip",
			extractDir:  "/opt",
			strip:       0,
			contains:    "unzip -q",
		},
		{
			name:        "tar.gz with strip",
			destination: "file.tar.gz",
			extractDir:  "/opt",
			strip:       1,
			contains:    "tar -xf",
		},
		{
			name:        "tar.xz no strip",
			destination: "file.tar.xz",
			extractDir:  "/opt",
			strip:       0,
			contains:    "tar -xf",
		},
		{
			name:        "unsupported format",
			destination: "file.rar",
			extractDir:  "/opt",
			strip:       0,
			contains:    "Unsupported archive format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildExtractCommand(tt.destination, tt.extractDir, tt.strip)
			if tt.contains != "" {
				if result == "" {
					t.Fatal("buildExtractCommand() returned empty result")
				}
				if !strings.Contains(result, tt.contains) {
					t.Errorf("buildExtractCommand() = %q, want to contain %q", result, tt.contains)
				}
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	expectedPipelines := []string{
		"create-user",
		"set-ownership",
		"download-verify-extract",
		"make-executable",
		"clone",
		"clone-and-build-go",
		"build-go-static",
		"build-go-only",
		"clone-and-build-rust",
		"clone-and-build-make",
		"clone-and-build-autoconf",
		"setup-users-groups",
		"create-directories",
		"copy-files",
	}

	for _, name := range expectedPipelines {
		if _, exists := Registry[name]; !exists {
			t.Errorf("expected pipeline %q to be in registry", name)
		}
	}
}
