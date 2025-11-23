package config

import (
	"strings"
	"testing"

	"github.com/greboid/dfo/pkg/util"
)

func checkError(t *testing.T, err error, expectError bool, errorMsg string) {
	t.Helper()
	if expectError {
		if err == nil {
			t.Errorf("expected error containing %q, got nil", errorMsg)
			return
		}
		if errorMsg != "" && !strings.Contains(err.Error(), errorMsg) {
			t.Errorf("expected error containing %q, got %q", errorMsg, err.Error())
		}
	} else {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *BuildConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with single stage",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
					Pipeline:    []PipelineStep{{Run: "echo test"}},
				}},
			},
			expectError: false,
		},
		{
			name: "valid config with multiple stages",
			config: &BuildConfig{
				Package: Package{Name: "multi-stage"},
				Stages: []Stage{
					{
						Name:        "builder",
						Environment: Environment{BaseImage: "golang"},
						Pipeline:    []PipelineStep{{Run: "go build"}},
					},
					{
						Name:        "runtime",
						Environment: Environment{ExternalImage: "alpine:3.19"},
						Pipeline:    []PipelineStep{{Run: "echo deploy"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid config with external-image",
			config: &BuildConfig{
				Package: Package{Name: "external-test"},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{ExternalImage: "ubuntu:22.04"},
					Pipeline:    []PipelineStep{{Run: "apt-get update"}},
				}},
			},
			expectError: false,
		},
		{
			name: "missing package name",
			config: &BuildConfig{
				Package: Package{Description: "no name"},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: true,
			errorMsg:    "package.name is required",
		},
		{
			name: "empty package name",
			config: &BuildConfig{
				Package: Package{Name: ""},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: true,
			errorMsg:    "package.name is required",
		},
		{
			name: "missing stages array",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
			},
			expectError: true,
			errorMsg:    "at least one stage is required",
		},
		{
			name: "empty stages array",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
				Stages:  []Stage{},
			},
			expectError: true,
			errorMsg:    "at least one stage is required",
		},
		{
			name: "stage with both base-image and external-image",
			config: &BuildConfig{
				Package: Package{Name: "conflict-test"},
				Stages: []Stage{{
					Name: "build",
					Environment: Environment{
						BaseImage:     "alpine",
						ExternalImage: "ubuntu:22.04",
					},
				}},
			},
			expectError: true,
			errorMsg:    "cannot specify both environment.base-image and environment.external-image",
		},
		{
			name: "stage with neither base-image nor external-image",
			config: &BuildConfig{
				Package: Package{Name: "missing-image"},
				Stages: []Stage{{
					Name: "build",
					Environment: Environment{
						Packages: []string{"git"},
					},
				}},
			},
			expectError: true,
			errorMsg:    "either environment.base-image or environment.external-image is required",
		},
		{
			name: "top-level environment with stages",
			config: &BuildConfig{
				Package:     Package{Name: "env-conflict"},
				Environment: Environment{BaseImage: "alpine"},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "golang"},
				}},
			},
			expectError: true,
			errorMsg:    "cannot specify top-level environment when using stages",
		},
		{
			name: "top-level environment with packages and stages",
			config: &BuildConfig{
				Package:     Package{Name: "env-packages-conflict"},
				Environment: Environment{Packages: []string{"git"}},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: true,
			errorMsg:    "cannot specify top-level environment when using stages",
		},
		{
			name: "valid config with vars",
			config: &BuildConfig{
				Package: Package{Name: "with-vars"},
				Vars:    map[string]string{"version": "1.0.0"},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: false,
		},
		{
			name: "valid config with complete package metadata",
			config: &BuildConfig{
				Package: Package{
					Name:        "complete-package",
					Description: "A complete example",
					Tags:        []string{"latest", "v1.0"},
					Labels:      map[string]string{"maintainer": "test@example.com"},
				},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: false,
		},
		{
			name: "stage name with space",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
				Stages: []Stage{{
					Name:        "build stage",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: true,
			errorMsg:    "name must be a single word",
		},
		{
			name: "empty stage name",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
				Stages: []Stage{{
					Name:        "",
					Environment: Environment{BaseImage: "alpine"},
				}},
			},
			expectError: true,
			errorMsg:    "stage name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			checkError(t, err, tt.expectError, tt.errorMsg)
		})
	}
}

func TestValidateStage(t *testing.T) {
	tests := []struct {
		name        string
		stage       Stage
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid stage with base-image",
			stage: Stage{
				Name:        "build",
				Environment: Environment{BaseImage: "alpine"},
			},
			expectError: false,
		},
		{
			name: "valid stage with external-image",
			stage: Stage{
				Name:        "runtime",
				Environment: Environment{ExternalImage: "nginx:latest"},
			},
			expectError: false,
		},
		{
			name: "stage with both images",
			stage: Stage{
				Name: "invalid",
				Environment: Environment{
					BaseImage:     "alpine",
					ExternalImage: "nginx",
				},
			},
			expectError: true,
			errorMsg:    "cannot specify both",
		},
		{
			name: "stage with neither image",
			stage: Stage{
				Name:        "no-image",
				Environment: Environment{Packages: []string{"git"}},
			},
			expectError: true,
			errorMsg:    "either environment.base-image or environment.external-image is required",
		},
		{
			name: "stage name appears in error",
			stage: Stage{
				Name:        "my-special-stage",
				Environment: Environment{},
			},
			expectError: true,
			errorMsg:    "my-special-stage",
		},
		{
			name:        "empty stage name",
			stage:       Stage{Name: "", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
			errorMsg:    "stage name is required",
		},
		{
			name:        "stage name with space",
			stage:       Stage{Name: "build stage", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
			errorMsg:    "name must be a single word",
		},
		{
			name:        "stage name with tab",
			stage:       Stage{Name: "build\tstage", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
			errorMsg:    "name must be a single word",
		},
		{
			name:        "stage name with newline",
			stage:       Stage{Name: "build\nstage", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
			errorMsg:    "name must be a single word",
		},
		{
			name:        "valid stage name with hyphen",
			stage:       Stage{Name: "build-stage", Environment: Environment{BaseImage: "alpine"}},
			expectError: false,
		},
		{
			name:        "valid stage name with underscore",
			stage:       Stage{Name: "build_stage", Environment: Environment{BaseImage: "alpine"}},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStage(tt.stage)
			checkError(t, err, tt.expectError, tt.errorMsg)
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid yaml",
			yaml: `package:
  name: test
stages:
  - name: build
    environment:
      base-image: alpine`,
			expectError: false,
		},
		{
			name:        "invalid yaml syntax",
			yaml:        `package: [unclosed`,
			expectError: true,
			errorMsg:    "parsing YAML",
		},
		{
			name: "valid yaml but invalid config",
			yaml: `package:
  name: ""
stages:
  - name: build
    environment:
      base-image: alpine`,
			expectError: true,
			errorMsg:    "package.name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			checkError(t, err, tt.expectError, tt.errorMsg)
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("loads valid config from file", func(t *testing.T) {
		fs := util.NewTestFS()

		yaml := `package:
  name: test-package
stages:
  - name: build
    environment:
      base-image: alpine
    pipeline:
      - run: echo "test"`

		_ = fs.WriteFile("config.yaml", []byte(yaml), 0644)

		config, err := Load(fs, "config.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Package.Name != "test-package" {
			t.Errorf("expected package name 'test-package', got %q", config.Package.Name)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		fs := util.NewTestFS()

		_, err := Load(fs, "nonexistent/path/to/config.yaml")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
		if !strings.Contains(err.Error(), "reading config file") {
			t.Errorf("expected error about reading file, got: %v", err)
		}
	})
}

func TestEnvironmentIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		expected bool
	}{
		{
			name:     "completely empty environment",
			env:      Environment{},
			expected: true,
		},
		{
			name:     "with base image",
			env:      Environment{BaseImage: "alpine"},
			expected: false,
		},
		{
			name:     "with external image",
			env:      Environment{ExternalImage: "alpine:3.19"},
			expected: false,
		},
		{
			name:     "with args",
			env:      Environment{Args: map[string]string{"FOO": "bar"}},
			expected: false,
		},
		{
			name:     "with packages",
			env:      Environment{Packages: []string{"git"}},
			expected: false,
		},
		{
			name:     "with rootfs packages",
			env:      Environment{RootfsPackages: []string{"busybox"}},
			expected: false,
		},
		{
			name:     "with environment variables",
			env:      Environment{Environment: map[string]string{"PATH": "/usr/bin"}},
			expected: false,
		},
		{
			name:     "with workdir",
			env:      Environment{WorkDir: "/app"},
			expected: false,
		},
		{
			name:     "with user",
			env:      Environment{User: "appuser"},
			expected: false,
		},
		{
			name:     "with entrypoint",
			env:      Environment{Entrypoint: []string{"/app/server"}},
			expected: false,
		},
		{
			name:     "with cmd",
			env:      Environment{Cmd: []string{"--help"}},
			expected: false,
		},
		{
			name:     "with expose",
			env:      Environment{Expose: []string{"8080"}},
			expected: false,
		},
		{
			name:     "with volume",
			env:      Environment{Volume: []string{"/data"}},
			expected: false,
		},
		{
			name:     "with stopsignal",
			env:      Environment{StopSignal: "SIGTERM"},
			expected: false,
		},
		{
			name: "with multiple fields",
			env: Environment{
				BaseImage: "alpine",
				Packages:  []string{"git", "curl"},
				WorkDir:   "/app",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.env.IsEmpty()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
