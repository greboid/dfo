package processor

import (
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/greboid/dfo/pkg/util"
)

const validConfig = `package:
  name: test-package
stages:
  - name: build
    environment:
      base-image: alpine
    pipeline:
      - run: echo "test"`

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		input       string
		wantSuffix  string
		expectError bool
		errorMsg    string
	}{
		{
			name:       "empty input returns default filename",
			setupFiles: map[string]string{},
			input:      "",
			wantSuffix: "dfo.yaml",
		},
		{
			name: "file path returns the path",
			setupFiles: map[string]string{
				"config.yaml": "content",
			},
			input:      "config.yaml",
			wantSuffix: "config.yaml",
		},
		{
			name: "directory path appends dfo.yaml",
			setupFiles: map[string]string{
				"project/other.txt": "content",
			},
			input:      "project",
			wantSuffix: "project/dfo.yaml",
		},
		{
			name:        "nonexistent path returns error",
			setupFiles:  map[string]string{},
			input:       "nonexistent",
			expectError: true,
			errorMsg:    "accessing path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()

			for relPath, content := range tt.setupFiles {
				dir := path.Dir(relPath)
				if dir != "." && dir != "" {
					_ = fs.MkdirAll(dir, 0755)
				}
				_ = fs.WriteFile(relPath, []byte(content), 0644)
			}

			for relPath := range tt.setupFiles {
				dir := path.Dir(relPath)
				if dir != "." && dir != "" {
					_ = fs.MkdirAll(dir, 0755)
				}
			}

			result, err := ResolveConfigPath(fs, tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if !strings.HasSuffix(result, tt.wantSuffix) {
					t.Errorf("expected result ending with %q, got %q", tt.wantSuffix, result)
				}
			}
		})
	}
}

func TestResolveConfigPathWithFile(t *testing.T) {
	fs := util.NewTestFS()
	_ = fs.WriteFile("test.yaml", []byte("content"), 0644)

	result, err := ResolveConfigPath(fs, "test.yaml")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "test.yaml" {
		t.Errorf("expected %q, got %q", "test.yaml", result)
	}
}

func TestProcessConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid config",
			configYAML:  validConfig,
			expectError: false,
		},
		{
			name: "invalid config - missing package name",
			configYAML: `package:
  description: no name
stages:
  - name: build
    environment:
      base-image: alpine`,
			expectError: true,
			errorMsg:    "loading config",
		},
		{
			name: "invalid YAML syntax",
			configYAML: `package:
  name: test
  invalid: [unclosed`,
			expectError: true,
			errorMsg:    "loading config",
		},
		{
			name: "generator error - undefined variable reference",
			configYAML: `package:
  name: test-package
stages:
  - name: build
    environment:
      base-image: alpine
    pipeline:
      - run: echo %{UNDEFINED_VAR}`,
			expectError: true,
			errorMsg:    "generating templates",
		},
		{
			name: "generator error - unknown pipeline",
			configYAML: `package:
  name: test-package
stages:
  - name: build
    environment:
      base-image: alpine
    pipeline:
      - uses: nonexistent-pipeline`,
			expectError: true,
			errorMsg:    "generating templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			_ = fs.WriteFile("dfo.yaml", []byte(tt.configYAML), 0644)
			_ = fs.MkdirAll("output", 0755)

			result, err := ProcessConfig(fs, "dfo.yaml", "output")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Error("expected non-nil result")
					return
				}
				if result.PackageName == "" {
					t.Error("expected non-empty package name")
				}
			}
		})
	}
}

func TestProcessConfigCreatesOutput(t *testing.T) {
	fs := util.NewTestFS()
	_ = fs.WriteFile("dfo.yaml", []byte(validConfig), 0644)
	_ = fs.MkdirAll("output", 0755)

	result, err := ProcessConfig(fs, "dfo.yaml", "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PackageName != "test-package" {
		t.Errorf("expected package name %q, got %q", "test-package", result.PackageName)
	}

	_, err = fs.Stat("output/test-package")
	if err != nil {
		t.Error("expected package directory to be created")
	}
}

func TestProcessConfigInPlace(t *testing.T) {
	fs := util.NewTestFS()
	_ = fs.MkdirAll("project", 0755)
	_ = fs.WriteFile("project/dfo.yaml", []byte(validConfig), 0644)

	result, err := ProcessConfigInPlace(fs, "project/dfo.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PackageName != "test-package" {
		t.Errorf("expected package name %q, got %q", "test-package", result.PackageName)
	}
}

func TestProcessConfigInPlaceErrors(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "loading error - missing package name",
			configYAML: `package:
  name: ""
stages:
  - name: build`,
			expectError: true,
			errorMsg:    "loading config",
		},
		{
			name: "generator error - undefined variable reference",
			configYAML: `package:
  name: test-package
stages:
  - name: build
    environment:
      base-image: alpine
    pipeline:
      - run: echo %{UNDEFINED_VAR}`,
			expectError: true,
			errorMsg:    "generating templates",
		},
		{
			name: "generator error - unknown pipeline",
			configYAML: `package:
  name: test-package
stages:
  - name: build
    environment:
      base-image: alpine
    pipeline:
      - uses: nonexistent-pipeline`,
			expectError: true,
			errorMsg:    "generating templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			_ = fs.WriteFile("dfo.yaml", []byte(tt.configYAML), 0644)

			_, err := ProcessConfigInPlace(fs, "dfo.yaml")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWalkAndProcess(t *testing.T) {
	fs := util.NewTestFS()

	files := map[string]string{
		"project1/dfo.yaml": validConfig,
		"project2/dfo.yaml": validConfig,
		"nested/deep/dfo.yaml": `package:
  name: nested-package
stages:
  - name: build
    environment:
      base-image: alpine`,
		"other/readme.txt": "not a config file",
	}

	_ = fs.MkdirAll(".", 0755)

	for relPath, content := range files {
		dir := path.Dir(relPath)
		if dir != "." && dir != "" {
			_ = fs.MkdirAll(dir, 0755)
		}
		_ = fs.WriteFile(relPath, []byte(content), 0644)
	}

	var processedPaths []string
	processor := func(p string) error {
		processedPaths = append(processedPaths, p)
		return nil
	}

	result, err := WalkAndProcess(fs, ".", processor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 3 {
		t.Errorf("expected 3 processed files, got %d", result.Processed)
	}

	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}

	if len(processedPaths) != 3 {
		t.Errorf("expected 3 processed paths, got %d", len(processedPaths))
	}
}

func TestWalkAndProcessWithErrors(t *testing.T) {
	fs := util.NewTestFS()

	_ = fs.MkdirAll("good", 0755)
	_ = fs.MkdirAll("bad", 0755)
	_ = fs.WriteFile("good/dfo.yaml", []byte(validConfig), 0644)
	_ = fs.WriteFile("bad/dfo.yaml", []byte(validConfig), 0644)

	processor := func(p string) error {
		if strings.Contains(p, "bad") {
			return errors.New("simulated failure")
		}
		return nil
	}

	result, err := WalkAndProcess(fs, ".", processor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("expected 1 processed file, got %d", result.Processed)
	}

	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}

	if len(result.ErrorDetails) != 1 {
		t.Errorf("expected 1 error detail, got %d", len(result.ErrorDetails))
	}
}

func TestWalkAndProcessInvalidDirectory(t *testing.T) {
	fs := util.NewTestFS()

	_, err := WalkAndProcess(fs, "nonexistent", func(string) error { return nil })
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
	if !strings.Contains(err.Error(), "accessing directory") {
		t.Errorf("expected error about accessing directory, got: %v", err)
	}
}

func TestWalkAndProcessNotADirectory(t *testing.T) {
	fs := util.NewTestFS()
	_ = fs.WriteFile("file.txt", []byte("content"), 0644)

	_, err := WalkAndProcess(fs, "file.txt", func(string) error { return nil })
	if err == nil {
		t.Fatal("expected error for file instead of directory")
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Errorf("expected error about not being a directory, got: %v", err)
	}
}

func TestFindConfigFiles(t *testing.T) {
	fs := util.NewTestFS()

	files := map[string]string{
		"project1/dfo.yaml":    validConfig,
		"project2/dfo.yaml":    validConfig,
		"nested/deep/dfo.yaml": validConfig,
		"other/readme.txt":     "not a config file",
	}

	for relPath, content := range files {
		dir := path.Dir(relPath)
		if dir != "." && dir != "" {
			_ = fs.MkdirAll(dir, 0755)
		}
		_ = fs.WriteFile(relPath, []byte(content), 0644)
	}

	found, err := FindConfigFiles(fs, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(found) != 3 {
		t.Errorf("expected 3 config files, got %d", len(found))
	}

	for _, p := range found {
		if path.Base(p) != "dfo.yaml" {
			t.Errorf("expected dfo.yaml, got %s", path.Base(p))
		}
	}
}

func TestFindConfigFilesEmpty(t *testing.T) {
	fs := util.NewTestFS()
	_ = fs.MkdirAll(".", 0755)
	_ = fs.WriteFile("other.txt", []byte("content"), 0644)

	found, err := FindConfigFiles(fs, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(found) != 0 {
		t.Errorf("expected 0 config files, got %d", len(found))
	}
}

func TestFindConfigFilesInvalidDirectory(t *testing.T) {
	fs := util.NewTestFS()

	_, err := FindConfigFiles(fs, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestProcessingErrorString(t *testing.T) {
	pe := ProcessingError{
		Path: "/path/to/file",
		Err:  errors.New("test error"),
	}

	expected := "/path/to/file: test error"
	if pe.Error() != expected {
		t.Errorf("expected %q, got %q", expected, pe.Error())
	}
}
