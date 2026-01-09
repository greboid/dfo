package config

import (
	"testing"
)

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

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *BuildConfig
		expectError bool
	}{
		{
			name: "valid config with single stage",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
				Stages: []Stage{{
					Name:        "build",
					Environment: Environment{BaseImage: "alpine"},
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
		},
		{
			name: "missing stages array",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
			},
			expectError: true,
		},
		{
			name: "empty stages array",
			config: &BuildConfig{
				Package: Package{Name: "test-package"},
				Stages:  []Stage{},
			},
			expectError: true,
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateStage(t *testing.T) {
	tests := []struct {
		name        string
		stage       Stage
		expectError bool
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
		},
		{
			name: "stage with neither image",
			stage: Stage{
				Name:        "no-image",
				Environment: Environment{Packages: []string{"git"}},
			},
			expectError: true,
		},
		{
			name:        "empty stage name",
			stage:       Stage{Name: "", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
		},
		{
			name:        "stage name with space",
			stage:       Stage{Name: "build stage", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
		},
		{
			name:        "stage name with tab",
			stage:       Stage{Name: "build\tstage", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
		},
		{
			name:        "stage name with newline",
			stage:       Stage{Name: "build\nstage", Environment: Environment{BaseImage: "alpine"}},
			expectError: true,
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
			if (err != nil) != tt.expectError {
				t.Errorf("validateStage() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
