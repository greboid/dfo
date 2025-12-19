package generator

import (
	"strings"
	"testing"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/packages"
	"github.com/greboid/dfo/pkg/util"
	"github.com/greboid/dfo/pkg/versions"
)

func TestExpandVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:     "no variables",
			input:    "echo hello",
			vars:     map[string]string{},
			expected: "echo hello",
		},
		{
			name:  "single variable with braces",
			input: "echo %{VERSION}",
			vars: map[string]string{
				"VERSION": "1.0.0",
			},
			expected: "echo 1.0.0",
		},
		{
			name:  "single variable without braces - not expanded",
			input: "echo %VERSION",
			vars: map[string]string{
				"VERSION": "1.0.0",
			},
			expected: "echo %VERSION",
		},
		{
			name:  "multiple variables",
			input: "curl https://github.com/%{OWNER}/%{REPO}/releases/%{VERSION}",
			vars: map[string]string{
				"OWNER":   "myorg",
				"REPO":    "myrepo",
				"VERSION": "v1.2.3",
			},
			expected: "curl https://github.com/myorg/myrepo/releases/v1.2.3",
		},
		{
			name:  "mixed braces and no braces",
			input: "%VAR1 and %{VAR2}",
			vars: map[string]string{
				"VAR1": "first",
				"VAR2": "second",
			},
			expected: "%VAR1 and second",
		},
		{
			name:     "undefined variable",
			input:    "echo %{UNDEFINED}",
			vars:     map[string]string{},
			expected: "echo %{UNDEFINED}",
		},
		{
			name:  "overlapping variable names - no longer an issue",
			input: "%{FOO} and %{FOOBAR}",
			vars: map[string]string{
				"FOO":    "bar",
				"FOOBAR": "baz",
			},
			expected: "bar and baz",
		},
		{
			name:  "overlapping variable names with unbraced form",
			input: "%FOO and %FOOBAR",
			vars: map[string]string{
				"FOO":    "bar",
				"FOOBAR": "baz",
			},
			expected: "%FOO and %FOOBAR",
		},
		{
			name:  "prefix variables with braces",
			input: "%{APP} %{APP_VERSION} %{APP_VERSION_TAG}",
			vars: map[string]string{
				"APP":             "myapp",
				"APP_VERSION":     "v2.0",
				"APP_VERSION_TAG": "v2.0-rc1",
			},
			expected: "myapp v2.0 v2.0-rc1",
		},
		{
			name:  "bash variables should not be expanded",
			input: "export PATH=$PATH:/usr/local/bin",
			vars: map[string]string{
				"PATH": "/custom/path",
			},
			expected: "export PATH=$PATH:/usr/local/bin",
		},
		{
			name:  "only dfo variables expanded, bash unchanged",
			input: "PATH=%{CUSTOM_PATH}:/usr/local/bin:$PATH",
			vars: map[string]string{
				"CUSTOM_PATH": "/my/custom/path",
			},
			expected: "PATH=/my/custom/path:/usr/local/bin:$PATH",
		},
		{
			name:  "bash braced variables unchanged",
			input: "for DEP in $DEPS; do apk add \"${DEP}\"; done",
			vars: map[string]string{
				"DEP":  "should-not-match",
				"DEPS": "should-not-match",
			},
			expected: "for DEP in $DEPS; do apk add \"${DEP}\"; done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.ExpandVars(tt.input, tt.vars)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatRunCommand(t *testing.T) {
	g := &Generator{
		config: &config.BuildConfig{},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line command",
			input:    "echo hello",
			expected: "RUN echo hello\n",
		},
		{
			name: "multi-line with command separators",
			input: `apk update
apk add git
echo done`,
			expected: "RUN apk update; \\\n    apk add git; \\\n    echo done\n",
		},
		{
			name: "multi-line with existing backslash continuations",
			input: `echo start && \
echo middle && \
echo end`,
			expected: "RUN echo start && \\\n    echo middle && \\\n    echo end\n",
		},
		{
			name: "multi-line with trailing semicolons",
			input: `apk update;
apk add git;`,
			expected: "RUN apk update; \\\n    apk add git\n",
		},
		{
			name:     "empty lines ignored",
			input:    "echo start\n\necho end",
			expected: "RUN echo start; \\\n    echo end\n",
		},
		{
			name: "complex multi-line",
			input: `set -eux
apk add --no-cache git
go build -o /app/binary`,
			expected: "RUN set -eux; \\\n    apk add --no-cache git; \\\n    go build -o /app/binary\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.formatRunCommand(tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGenerateFetchStep(t *testing.T) {
	g := &Generator{
		config: &config.BuildConfig{
			Vars: map[string]string{
				"VERSION": "v1.0.0",
				"REPO":    "owner/repo",
			},
		},
	}

	tests := []struct {
		name     string
		fetch    *config.FetchStep
		expected string
	}{
		{
			name: "simple download without extraction",
			fetch: &config.FetchStep{
				URL:         "https://example.com/file.tar.gz",
				Destination: "/tmp/file.tar.gz",
				Extract:     false,
			},
			expected: `RUN curl -fsSL -o /tmp/file.tar.gz "https://example.com/file.tar.gz"` + "\n",
		},
		{
			name: "download with extraction",
			fetch: &config.FetchStep{
				URL:         "https://example.com/archive.tar.gz",
				Destination: "/app",
				Extract:     true,
			},
			expected: `RUN curl -fsSL "https://example.com/archive.tar.gz" | tar -xz -C "/app"` + "\n",
		},
		{
			name: "download with variable substitution",
			fetch: &config.FetchStep{
				URL:         "https://github.com/%{REPO}/releases/download/%{VERSION}/app.tar.gz",
				Destination: "/tmp/app.tar.gz",
				Extract:     false,
			},
			expected: `RUN curl -fsSL -o /tmp/app.tar.gz "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz"` + "\n",
		},
		{
			name: "default destination",
			fetch: &config.FetchStep{
				URL:     "https://example.com/file.bin",
				Extract: false,
			},
			expected: `RUN curl -fsSL -o /tmp/download "https://example.com/file.bin"` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.generateFetchStep(tt.fetch)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGeneratePackageInstallForEnv(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name     string
		env      config.Environment
		contains []string
	}{
		{
			name: "single package",
			env: config.Environment{
				Packages: []string{"git"},
			},
			contains: []string{
				"# Install packages",
				"RUN set -eux;",
				"apk add --no-cache",
				"git=",
			},
		},
		{
			name: "multiple packages",
			env: config.Environment{
				Packages: []string{"git", "ca-certificates", "musl"},
			},
			contains: []string{
				"# Install packages",
				"RUN set -eux;",
				"apk add --no-cache",
				"git=",
				"ca-certificates=",
				"musl=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := g.generatePackageInstallForEnv(tt.env)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestGenerateRootfsPackageInstallForEnv(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name     string
		env      config.Environment
		contains []string
	}{
		{
			name: "single rootfs package",
			env: config.Environment{
				RootfsPackages: []string{"busybox"},
			},
			contains: []string{
				"# Install packages into rootfs",
				"apk add --no-cache busybox=",
				"apk info -qL busybox",
				"rsync -aq --files-from=- / /rootfs/",
			},
		},
		{
			name: "multiple rootfs packages",
			env: config.Environment{
				RootfsPackages: []string{"busybox", "musl", "ca-certificates"},
			},
			contains: []string{
				"# Install packages into rootfs",
				"apk add --no-cache busybox=",
				"apk add --no-cache musl=",
				"apk add --no-cache ca-certificates=",
				"rsync -aq --files-from=- / /rootfs/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.generateRootfsPackageInstallForEnv(tt.env)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestGenerateRunWithBuildDeps(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name      string
		runCmd    string
		buildDeps []string
		contains  []string
	}{
		{
			name:      "single build dependency",
			runCmd:    "go build -o /app/binary",
			buildDeps: []string{"go"},
			contains: []string{
				"RUN apk add --no-cache --virtual .build-deps",
				"go=",
				"go build -o /app/binary",
				"apk del --no-network .build-deps",
			},
		},
		{
			name:      "multiple build dependencies",
			runCmd:    "make install",
			buildDeps: []string{"make", "gcc", "musl-dev"},
			contains: []string{
				"RUN apk add --no-cache --virtual .build-deps",
				"make=",
				"gcc=",
				"musl-dev=",
				"make install",
				"apk del --no-network .build-deps",
			},
		},
		{
			name: "multi-line run command",
			runCmd: `cd /build
make
make install`,
			buildDeps: []string{"make"},
			contains: []string{
				"RUN apk add --no-cache --virtual .build-deps",
				"make=",
				"cd /build",
				"make",
				"make install",
				"apk del --no-network .build-deps",
			},
		},
		{
			name: "multi-line with continuation backslash",
			runCmd: `rm -rf \
  /path/one \
  /path/two
echo done`,
			buildDeps: []string{"make"},
			contains: []string{
				"RUN apk add --no-cache --virtual .build-deps",
				"make=",
				"rm -rf",
				"/path/one",
				"/path/two",
				"echo done",
				"apk del --no-network .build-deps",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.generateRunWithBuildDeps(tt.runCmd, tt.buildDeps)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	cfg := &config.BuildConfig{
		Package: config.Package{Name: "test"},
	}
	outputDir := "output"
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()

	g := New(cfg, outputDir, fs, client, "3.19", "", "", "docker.io", nil)

	if g.config != cfg {
		t.Error("config not set correctly")
	}
	if g.outputDir != outputDir {
		t.Errorf("expected outputDir %q, got %q", outputDir, g.outputDir)
	}
	if g.outputFilename != "Containerfile" {
		t.Errorf("expected default filename %q, got %q", "Containerfile", g.outputFilename)
	}
}

func TestSetOutputFilename(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	g := New(&config.BuildConfig{}, "tmp", fs, client, "3.19", "", "", "docker.io", nil)
	customFilename := "Dockerfile.template"

	g.SetOutputFilename(customFilename)

	if g.outputFilename != customFilename {
		t.Errorf("expected filename %q, got %q", customFilename, g.outputFilename)
	}
}

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: []string{},
		},
		{
			name: "single key",
			input: map[string]string{
				"key1": "value1",
			},
			expected: []string{"key1"},
		},
		{
			name: "multiple keys unsorted",
			input: map[string]string{
				"zebra": "z",
				"apple": "a",
				"mango": "m",
			},
			expected: []string{"apple", "mango", "zebra"},
		},
		{
			name: "keys with numbers",
			input: map[string]string{
				"key3": "c",
				"key1": "a",
				"key2": "b",
			},
			expected: []string{"key1", "key2", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.SortedKeys(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, key := range result {
				if key != tt.expected[i] {
					t.Errorf("at index %d: expected %q, got %q", i, tt.expected[i], key)
				}
			}
		})
	}
}

func TestGeneratePipelineStep(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{
		Vars: map[string]string{
			"VERSION": "v1.0.0",
		},
	}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name     string
		step     config.PipelineStep
		expected string
		contains []string
		wantErr  bool
	}{
		{
			name: "run step without build deps",
			step: config.PipelineStep{
				Run: "echo hello",
			},
			expected: "RUN echo hello\n",
			wantErr:  false,
		},
		{
			name: "run step with build deps",
			step: config.PipelineStep{
				Run:       "go build",
				BuildDeps: []string{"go"},
			},
			contains: []string{
				"RUN apk add --no-cache --virtual .build-deps",
				"go=",
				"go build",
				"apk del --no-network .build-deps",
			},
			wantErr: false,
		},
		{
			name: "run step with variable expansion",
			step: config.PipelineStep{
				Run: "echo %{VERSION}",
			},
			expected: "RUN echo v1.0.0\n",
			wantErr:  false,
		},
		{
			name: "fetch step",
			step: config.PipelineStep{
				Fetch: &config.FetchStep{
					URL:         "https://example.com/file.tar.gz",
					Destination: "/tmp/file.tar.gz",
					Extract:     false,
				},
			},
			expected: `RUN curl -fsSL -o /tmp/file.tar.gz "https://example.com/file.tar.gz"` + "\n",
			wantErr:  false,
		},
		{
			name: "copy step without options",
			step: config.PipelineStep{
				Copy: &config.CopyStep{
					From: "app.bin",
					To:   "/usr/local/bin/app",
				},
			},
			expected: "COPY app.bin /usr/local/bin/app\n",
			wantErr:  false,
		},
		{
			name: "copy step with from-stage",
			step: config.PipelineStep{
				Copy: &config.CopyStep{
					FromStage: "builder",
					From:      "/build/app",
					To:        "/app",
				},
			},
			expected: "COPY --from=builder /build/app /app\n",
			wantErr:  false,
		},
		{
			name: "copy step with chown",
			step: config.PipelineStep{
				Copy: &config.CopyStep{
					From:  "app.conf",
					To:    "/etc/app.conf",
					Chown: "appuser:appgroup",
				},
			},
			expected: "COPY --chown=appuser:appgroup app.conf /etc/app.conf\n",
			wantErr:  false,
		},
		{
			name: "copy step with from-stage and chown",
			step: config.PipelineStep{
				Copy: &config.CopyStep{
					FromStage: "builder",
					From:      "/build/app",
					To:        "/app",
					Chown:     "root:root",
				},
			},
			expected: "COPY --from=builder --chown=root:root /build/app /app\n",
			wantErr:  false,
		},
		{
			name: "uses step - create-user",
			step: config.PipelineStep{
				Uses: "create-user",
				With: map[string]any{
					"username": "appuser",
					"uid":      1000,
					"gid":      1000,
				},
			},
			contains: []string{
				"RUN apk add --no-cache --virtual .create-user-deps",
				"busybox=",
				"# Create application user",
				"RUN addgroup -g 1000 appuser",
				"adduser -D -u 1000 -G appuser appuser",
				"RUN apk del --no-network .create-user-deps",
			},
			wantErr: false,
		},
		{
			name: "uses step - nonexistent pipeline",
			step: config.PipelineStep{
				Uses: "nonexistent-pipeline",
				With: map[string]any{},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty step",
			step:     config.PipelineStep{},
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := g.generatePipelineStep(tt.step)
			if (err != nil) != tt.wantErr {
				t.Errorf("generatePipelineStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expected != "" && result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestGenerateIncludeCall(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name     string
		step     config.PipelineStep
		wantErr  bool
		contains []string
	}{
		{
			name: "create-user pipeline",
			step: config.PipelineStep{
				Name: "Setup user",
				Uses: "create-user",
				With: map[string]any{
					"username": "testuser",
					"uid":      5000,
					"gid":      5000,
				},
			},
			wantErr: false,
			contains: []string{
				"# Create application user",
				"RUN addgroup -g 5000 testuser",
				"adduser -D -u 5000 -G testuser testuser",
			},
		},
		{
			name: "make-executable pipeline",
			step: config.PipelineStep{
				Uses: "make-executable",
				With: map[string]any{
					"path": "/usr/local/bin/myapp",
				},
			},
			wantErr: false,
			contains: []string{
				"RUN chmod +x /usr/local/bin/myapp",
			},
		},
		{
			name: "nonexistent pipeline",
			step: config.PipelineStep{
				Uses: "does-not-exist",
				With: map[string]any{},
			},
			wantErr: true,
		},
		{
			name: "pipeline with invalid parameters",
			step: config.PipelineStep{
				Uses: "create-user",
				With: map[string]any{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := g.generateIncludeCall(tt.step)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateIncludeCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, substr := range tt.contains {
					if !strings.Contains(result, substr) {
						t.Errorf("expected result to contain %q, got:\n%s", substr, result)
					}
				}
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.BuildConfig
		wantErr bool
	}{
		{
			name: "minimal single stage config",
			cfg: &config.BuildConfig{
				Package: config.Package{
					Name: "test-app",
				},
				Stages: []config.Stage{
					{
						Name: "final",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multi-stage build",
			cfg: &config.BuildConfig{
				Package: config.Package{
					Name: "multi-stage-app",
				},
				Stages: []config.Stage{
					{
						Name: "builder",
						Environment: config.Environment{
							BaseImage: "alpine",
							Packages:  []string{"go"},
						},
						Pipeline: []config.PipelineStep{
							{
								Run: "go build -o /app",
							},
						},
					},
					{
						Name: "final",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{
								Copy: &config.CopyStep{
									FromStage: "builder",
									From:      "/app",
									To:        "/usr/local/bin/app",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "generateDockerfile error - nonexistent pipeline",
			cfg: &config.BuildConfig{
				Package: config.Package{
					Name: "error-test",
				},
				Stages: []config.Stage{
					{
						Name: "final",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{
								Uses: "nonexistent-pipeline",
								With: map[string]any{},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			client := packages.NewAlpineClient()
			g := New(tt.cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

			err := g.Generate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				content, err := fs.ReadFile("output/Containerfile")
				if err != nil {
					t.Fatalf("failed to read generated file: %v", err)
				}
				if len(content) == 0 {
					t.Error("generated file is empty")
				}
			}
		})
	}
}

func TestGenerateStage(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{
		Package: config.Package{
			Name:   "test",
			Labels: map[string]string{"version": "1.0"},
		},
	}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name        string
		stage       config.Stage
		isFinal     bool
		contains    []string
		notContains []string
	}{
		{
			name: "intermediate stage with base image",
			stage: config.Stage{
				Name: "builder",
				Environment: config.Environment{
					BaseImage: "alpine",
				},
			},
			isFinal: false,
			contains: []string{
				"FROM",
				"@sha256:",
				"AS builder",
			},
			notContains: []string{
				"LABEL",
			},
		},
		{
			name: "final stage with base image",
			stage: config.Stage{
				Name: "final",
				Environment: config.Environment{
					BaseImage: "alpine",
				},
			},
			isFinal: true,
			contains: []string{
				"FROM",
				"@sha256:",
				`LABEL version="1.0"`,
			},
			notContains: []string{
				"AS final",
			},
		},
		{
			name: "stage with external image",
			stage: config.Stage{
				Name: "external",
				Environment: config.Environment{
					ExternalImage: "ubuntu:22.04",
				},
			},
			isFinal: false,
			contains: []string{
				"FROM ubuntu:22.04 AS external",
			},
			notContains: []string{
				"{{image",
			},
		},
		{
			name: "final stage with external image",
			stage: config.Stage{
				Name: "final",
				Environment: config.Environment{
					ExternalImage: "ubuntu:22.04",
				},
			},
			isFinal: true,
			contains: []string{
				"FROM ubuntu:22.04\n",
			},
			notContains: []string{
				"{{image",
				"AS final",
			},
		},
		{
			name: "stage with args and env",
			stage: config.Stage{
				Name: "builder",
				Environment: config.Environment{
					BaseImage: "alpine",
					Args: map[string]string{
						"VERSION": "1.0.0",
					},
					Environment: map[string]string{
						"APP_ENV": "production",
					},
				},
			},
			isFinal: false,
			contains: []string{
				`ARG VERSION="1.0.0"`,
				`ENV APP_ENV="production"`,
			},
		},
		{
			name: "stage with workdir and user",
			stage: config.Stage{
				Name: "app",
				Environment: config.Environment{
					BaseImage: "alpine",
					WorkDir:   "/app",
					User:      "appuser",
				},
			},
			isFinal: true,
			contains: []string{
				"WORKDIR /app",
				"USER appuser",
			},
		},
		{
			name: "stage with entrypoint and cmd",
			stage: config.Stage{
				Name: "app",
				Environment: config.Environment{
					BaseImage:  "alpine",
					Entrypoint: []string{"/usr/local/bin/app"},
					Cmd:        []string{"--help"},
				},
			},
			isFinal: true,
			contains: []string{
				`ENTRYPOINT ["/usr/local/bin/app"]`,
				`CMD ["--help"]`,
			},
		},
		{
			name: "stage with expose and volume",
			stage: config.Stage{
				Name: "web",
				Environment: config.Environment{
					BaseImage: "alpine",
					Expose:    []string{"8080", "8443"},
					Volume:    []string{"/data", "/logs"},
				},
			},
			isFinal: true,
			contains: []string{
				"EXPOSE 8080",
				"EXPOSE 8443",
				`VOLUME ["/data", "/logs"]`,
			},
		},
		{
			name: "stage with stopsignal",
			stage: config.Stage{
				Name: "app",
				Environment: config.Environment{
					BaseImage:  "alpine",
					StopSignal: "SIGTERM",
				},
			},
			isFinal: true,
			contains: []string{
				"STOPSIGNAL SIGTERM",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := g.generateStage(tt.stage, tt.isFinal)
			if err != nil {
				t.Fatalf("generateStage() error = %v", err)
			}

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(result, substr) {
					t.Errorf("expected result NOT to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}

func TestValidateVariableReferences(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.BuildConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "no variables - passes",
			cfg: &config.BuildConfig{
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "echo hello"},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "defined variable - passes",
			cfg: &config.BuildConfig{
				Vars: map[string]string{"VERSION": "1.0.0"},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "echo %{VERSION}"},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "undefined variable in run - fails",
			cfg: &config.BuildConfig{
				Vars: map[string]string{},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{
								Name: "build step",
								Run:  "echo %{UNDEFINED}",
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "undefined variable(s): %{UNDEFINED}",
		},
		{
			name: "undefined variable in fetch URL - fails",
			cfg: &config.BuildConfig{
				Vars: map[string]string{},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{
								Fetch: &config.FetchStep{
									URL:         "https://example.com/%{VERSION}/file.tar.gz",
									Destination: "/tmp/file.tar.gz",
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "undefined variable(s): %{VERSION}",
		},
		{
			name: "multiple undefined variables - all reported",
			cfg: &config.BuildConfig{
				Vars: map[string]string{},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "echo %{A} %{B} %{C}"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "%{A}",
		},
		{
			name: "some defined some undefined - fails with undefined only",
			cfg: &config.BuildConfig{
				Vars: map[string]string{"DEFINED": "value"},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "%{DEFINED} %{UNDEFINED}"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "%{UNDEFINED}",
		},
		{
			name: "bash variables ignored - passes",
			cfg: &config.BuildConfig{
				Vars: map[string]string{},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "for DEP in $DEPS; do apk add \"${DEP}\"; done"},
						},
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			client := packages.NewAlpineClient()
			g := New(tt.cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)
			err := g.validateVariableReferences()

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateWithVariableValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.BuildConfig
		wantError bool
	}{
		{
			name: "valid config generates successfully",
			cfg: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Vars:    map[string]string{"VERSION": "1.0.0"},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "echo %{VERSION}"},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "undefined variable fails before file creation",
			cfg: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Vars:    map[string]string{},
				Stages: []config.Stage{
					{
						Name: "test",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{Run: "echo %{MISSING}"},
						},
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			client := packages.NewAlpineClient()
			g := New(tt.cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)
			err := g.Generate()

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				_, statErr := fs.Stat("output/Containerfile")
				if statErr == nil {
					t.Error("output file should not be created when validation fails")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateStageContent(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{
		Package: config.Package{
			Name: "test",
		},
	}
	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	tests := []struct {
		name     string
		env      config.Environment
		pipeline []config.PipelineStep
		isFinal  bool
		contains []string
	}{
		{
			name: "environment with packages",
			env: config.Environment{
				Packages: []string{"git", "curl"},
			},
			pipeline: []config.PipelineStep{},
			isFinal:  true,
			contains: []string{
				"# Install packages",
				"apk add --no-cache",
				"git=",
				"curl=",
			},
		},
		{
			name: "environment with rootfs packages",
			env: config.Environment{
				RootfsPackages: []string{"busybox"},
			},
			pipeline: []config.PipelineStep{},
			isFinal:  true,
			contains: []string{
				"# Install packages into rootfs",
				"apk add --no-cache busybox=",
			},
		},
		{
			name: "pipeline with run command",
			env:  config.Environment{},
			pipeline: []config.PipelineStep{
				{
					Name: "Build application",
					Run:  "echo building",
				},
			},
			isFinal: true,
			contains: []string{
				"# Build application",
				"RUN echo building",
			},
		},
		{
			name: "pipeline with uses",
			env:  config.Environment{},
			pipeline: []config.PipelineStep{
				{
					Uses: "create-user",
					With: map[string]any{
						"username": "testuser",
						"uid":      1000,
						"gid":      1000,
					},
				},
			},
			isFinal: true,
			contains: []string{
				"RUN addgroup",
				"testuser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := g.generateStageContent(tt.env, tt.pipeline, tt.isFinal)
			if err != nil {
				t.Fatalf("generateStageContent() error = %v", err)
			}

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got:\n%s", substr, result)
				}
			}
		})
	}
}
func TestGenerateBOM(t *testing.T) {
	tests := []struct {
		name     string
		packages map[string]string
		images   map[string]string
		versions map[string]versions.VersionMetadata
		want     string
	}{
		{
			name:     "empty BOM",
			packages: map[string]string{},
			images:   map[string]string{},
			versions: map[string]versions.VersionMetadata{},
			want:     "",
		},
		{
			name: "packages only",
			packages: map[string]string{
				"git":             "2.43.0-r0",
				"ca-certificates": "20230506-r0",
			},
			images:   map[string]string{},
			versions: map[string]versions.VersionMetadata{},
			want:     `# BOM: {"apk:ca-certificates":"20230506-r0","apk:git":"2.43.0-r0"}`,
		},
		{
			name:     "images only",
			packages: map[string]string{},
			images: map[string]string{
				"alpine": "abc123def456",
				"golang": "xyz789uvw012",
			},
			versions: map[string]versions.VersionMetadata{},
			want:     `# BOM: {"image:alpine":"abc123def456","image:golang":"xyz789uvw012"}`,
		},
		{
			name:     "versions only",
			packages: map[string]string{},
			images:   map[string]string{},
			versions: map[string]versions.VersionMetadata{
				"prometheus": {Version: "v2.49.1"},
			},
			want: `# BOM: {"prometheus":"v2.49.1"}`,
		},
		{
			name: "all types combined",
			packages: map[string]string{
				"busybox": "1.36.1-r15",
				"musl":    "1.2.4_git20230717-r4",
			},
			images: map[string]string{
				"alpine": "abc123",
			},
			versions: map[string]versions.VersionMetadata{
				"prometheus": {Version: "v2.49.1"},
			},
			want: `# BOM: {"apk:busybox":"1.36.1-r15","apk:musl":"1.2.4_git20230717-r4","image:alpine":"abc123","prometheus":"v2.49.1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			client := packages.NewAlpineClient()
			cfg := &config.BuildConfig{}
			g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

			g.resolvedPackages = tt.packages
			g.resolvedImages = tt.images
			g.resolvedVersions = tt.versions

			result := g.generateBOM()

			if tt.want == "" {
				if result != "" {
					t.Errorf("expected empty BOM, got %q", result)
				}
			} else {
				expected := tt.want + "\n"
				if result != expected {
					t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
				}
			}
		})
	}
}

func TestGenerateWithBOM(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.BuildConfig
		contains []string
	}{
		{
			name: "simple config with packages generates BOM",
			cfg: &config.BuildConfig{
				Package: config.Package{
					Name: "test-app",
				},
				Stages: []config.Stage{
					{
						Name: "final",
						Environment: config.Environment{
							BaseImage: "alpine",
							Packages:  []string{"git"},
						},
						Pipeline: []config.PipelineStep{},
					},
				},
			},
			contains: []string{
				"# BOM:",
				`"apk:git":`,
				`"image:alpine":`,
			},
		},
		{
			name: "config with multiple packages",
			cfg: &config.BuildConfig{
				Package: config.Package{
					Name: "multi-package-app",
				},
				Stages: []config.Stage{
					{
						Name: "final",
						Environment: config.Environment{
							BaseImage: "alpine",
							Packages:  []string{"git", "ca-certificates", "curl"},
						},
						Pipeline: []config.PipelineStep{},
					},
				},
			},
			contains: []string{
				"# BOM:",
				`"apk:git":`,
				`"apk:ca-certificates":`,
				`"apk:curl":`,
				`"image:alpine":`,
			},
		},
		{
			name: "config with build deps also included in BOM",
			cfg: &config.BuildConfig{
				Package: config.Package{
					Name: "build-deps-app",
				},
				Stages: []config.Stage{
					{
						Name: "final",
						Environment: config.Environment{
							BaseImage: "alpine",
						},
						Pipeline: []config.PipelineStep{
							{
								Run:       "echo building",
								BuildDeps: []string{"make", "gcc"},
							},
						},
					},
				},
			},
			contains: []string{
				"# BOM:",
				`"apk:make":`,
				`"apk:gcc":`,
				`"image:alpine":`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := util.NewTestFS()
			client := packages.NewAlpineClient()
			g := New(tt.cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

			err := g.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			content, err := fs.ReadFile("output/Containerfile")
			if err != nil {
				t.Fatalf("failed to read generated file: %v", err)
			}

			contentStr := string(content)

			if !strings.HasPrefix(contentStr, "# BOM:") {
				t.Error("expected BOM to be at the beginning of the file")
			}

			for _, substr := range tt.contains {
				if !strings.Contains(contentStr, substr) {
					t.Errorf("expected result to contain %q", substr)
				}
			}
		})
	}
}

func TestBOMSortedOutput(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{
		Package: config.Package{Name: "test"},
		Stages: []config.Stage{
			{
				Name: "final",
				Environment: config.Environment{
					BaseImage: "alpine",
					Packages:  []string{"zlib", "git", "musl", "ca-certificates"},
				},
				Pipeline: []config.PipelineStep{},
			},
		},
	}

	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content, err := fs.ReadFile("output/Containerfile")
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)

	lines := strings.Split(contentStr, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "# BOM:") {
		t.Fatal("BOM not found at beginning of file")
	}

	bomLine := lines[0]

	caIndex := strings.Index(bomLine, `"apk:ca-certificates"`)
	gitIndex := strings.Index(bomLine, `"apk:git"`)
	muslIndex := strings.Index(bomLine, `"apk:musl"`)
	zlibIndex := strings.Index(bomLine, `"apk:zlib"`)
	alpineIndex := strings.Index(bomLine, `"image:alpine"`)

	if caIndex == -1 || gitIndex == -1 || muslIndex == -1 || zlibIndex == -1 || alpineIndex == -1 {
		t.Fatal("not all expected entries found in BOM")
	}

	if !(caIndex < gitIndex && gitIndex < muslIndex && muslIndex < zlibIndex && zlibIndex < alpineIndex) {
		t.Error("BOM entries are not in sorted order")
	}
}

func TestBOMWithMultiStage(t *testing.T) {
	fs := util.NewTestFS()
	client := packages.NewAlpineClient()
	cfg := &config.BuildConfig{
		Package: config.Package{
			Name: "multi-stage-app",
		},
		Stages: []config.Stage{
			{
				Name: "builder",
				Environment: config.Environment{
					BaseImage: "golang",
					Packages:  []string{"git"},
				},
				Pipeline: []config.PipelineStep{
					{
						Run: "go build -o /app",
					},
				},
			},
			{
				Name: "final",
				Environment: config.Environment{
					BaseImage: "alpine",
					Packages:  []string{"ca-certificates"},
				},
				Pipeline: []config.PipelineStep{
					{
						Copy: &config.CopyStep{
							FromStage: "builder",
							From:      "/app",
							To:        "/usr/local/bin/app",
						},
					},
				},
			},
		},
	}

	g := New(cfg, "output", fs, client, "3.19", "", "", "docker.io", nil)

	err := g.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content, err := fs.ReadFile("output/Containerfile")
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)

	expectedContains := []string{
		"# BOM:",
		`"apk:git":`,
		`"apk:ca-certificates":`,
		`"image:alpine":`,
		`"image:golang":`,
	}

	for _, substr := range expectedContains {
		if !strings.Contains(contentStr, substr) {
			t.Errorf("expected BOM to contain %q", substr)
		}
	}
}
