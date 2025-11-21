package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAbsolutePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := []struct {
		name        string
		inputPath   string
		expected    string
		expectError bool
	}{
		{
			name:        "absolute path unchanged",
			inputPath:   "/usr/local/bin",
			expected:    "/usr/local/bin",
			expectError: false,
		},
		{
			name:        "relative path converted",
			inputPath:   "relative/path",
			expected:    filepath.Join(cwd, "relative/path"),
			expectError: false,
		},
		{
			name:        "current directory",
			inputPath:   ".",
			expected:    filepath.Join(cwd, "."),
			expectError: false,
		},
		{
			name:        "parent directory",
			inputPath:   "..",
			expected:    filepath.Join(cwd, ".."),
			expectError: false,
		},
		{
			name:        "relative path with parent reference",
			inputPath:   "../sibling/path",
			expected:    filepath.Join(cwd, "../sibling/path"),
			expectError: false,
		},
		{
			name:        "empty path",
			inputPath:   "",
			expected:    cwd,
			expectError: false,
		},
		{
			name:        "path with spaces",
			inputPath:   "path with spaces",
			expected:    filepath.Join(cwd, "path with spaces"),
			expectError: false,
		},
		{
			name:        "absolute path with trailing slash",
			inputPath:   "/usr/local/bin/",
			expected:    "/usr/local/bin/",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveAbsolutePath(tt.inputPath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestResolveAbsolutePathWithTempDir(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	result, err := ResolveAbsolutePath("test.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := filepath.Join(tmpDir, "test.txt")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolveAbsolutePathSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	realDir := filepath.Join(tmpDir, "real")
	if err := os.Mkdir(realDir, 0755); err != nil {
		t.Fatalf("failed to create real directory: %v", err)
	}

	symlinkPath := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realDir, symlinkPath); err != nil {
		t.Skipf("skipping symlink test: %v", err)
	}

	result, err := ResolveAbsolutePath(symlinkPath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != symlinkPath {
		t.Errorf("expected %q, got %q", symlinkPath, result)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	result, err = ResolveAbsolutePath("link")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := filepath.Join(tmpDir, "link")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
