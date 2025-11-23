package pipelines

import (
	"strings"
	"testing"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "valid parameters",
			params: map[string]any{
				"username": "appuser",
				"uid":      1000,
				"gid":      1000,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				if len(result.Steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(result.Steps))
				}
				if !strings.Contains(result.Steps[0].Content, "addgroup -g 1000 appuser") {
					t.Errorf("expected addgroup command, got: %s", result.Steps[0].Content)
				}
				if !strings.Contains(result.Steps[0].Content, "adduser -D -u 1000 -G appuser appuser") {
					t.Errorf("expected adduser command, got: %s", result.Steps[0].Content)
				}
			},
		},
		{
			name: "missing username",
			params: map[string]any{
				"uid": 1000,
				"gid": 1000,
			},
			wantErr: true,
		},
		{
			name: "missing uid",
			params: map[string]any{
				"username": "appuser",
				"gid":      1000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateUser(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestSetOwnership(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "valid parameters",
			params: map[string]any{
				"user":  "appuser",
				"group": "appgroup",
				"path":  "/app",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				if len(result.Steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(result.Steps))
				}
				if !strings.Contains(result.Steps[0].Content, "chown -R appuser:appgroup /app") {
					t.Errorf("expected chown command, got: %s", result.Steps[0].Content)
				}
			},
		},
		{
			name: "missing user",
			params: map[string]any{
				"group": "appgroup",
				"path":  "/app",
			},
			wantErr: true,
		},
		{
			name: "missing group",
			params: map[string]any{
				"user": "appuser",
				"path": "/app",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			params: map[string]any{
				"user":  "appuser",
				"group": "appgroup",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SetOwnership(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetOwnership() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestDownloadVerifyExtract(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "download and verify only",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "curl") {
					t.Errorf("expected curl command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "sha256sum") {
					t.Errorf("expected sha256sum command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "&&") {
					t.Errorf("expected commands to be chained with &&, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "download, verify, and extract tar.gz",
			params: map[string]any{
				"url":              "https://example.com/file.tar.gz",
				"destination":      "/tmp/file.tar.gz",
				"checksum":         "abc123",
				"extract-dir":      "/opt/app",
				"strip-components": 1,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "curl") {
					t.Errorf("expected curl command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "sha256sum") {
					t.Errorf("expected sha256sum command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "tar -xf") {
					t.Errorf("expected tar command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "--strip-components=1") {
					t.Errorf("expected strip-components in tar command, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "&&") {
					t.Errorf("expected commands to be chained with &&, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "download, verify, and extract zip",
			params: map[string]any{
				"url":         "https://example.com/file.zip",
				"destination": "/tmp/file.zip",
				"checksum":    "abc123",
				"extract-dir": "/opt/app",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "curl") {
					t.Errorf("expected curl command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "sha256sum") {
					t.Errorf("expected sha256sum command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "unzip") {
					t.Errorf("expected unzip command in combined step, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "&&") {
					t.Errorf("expected commands to be chained with &&, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "missing url",
			params: map[string]any{
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
			},
			wantErr: true,
		},
		{
			name: "url wrong type",
			params: map[string]any{
				"url":         123,
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
			},
			wantErr: true,
		},
		{
			name: "missing destination",
			params: map[string]any{
				"url":      "https://example.com/file.tar.gz",
				"checksum": "abc123",
			},
			wantErr: true,
		},
		{
			name: "destination wrong type",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": 123,
				"checksum":    "abc123",
			},
			wantErr: true,
		},
		{
			name: "both checksum and checksum-url (mutually exclusive)",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/tmp/file.tar.gz",
				"checksum":     "abc123",
				"checksum-url": "https://example.com/checksums.txt",
			},
			wantErr: true,
		},
		{
			name: "neither checksum nor checksum-url (at-least-one)",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
			},
			wantErr: true,
		},
		{
			name: "checksum wrong type",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
				"checksum":    123,
			},
			wantErr: true,
		},
		{
			name: "checksum-url wrong type",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/tmp/file.tar.gz",
				"checksum-url": true,
			},
			wantErr: true,
		},
		{
			name: "checksum-pattern wrong type",
			params: map[string]any{
				"url":              "https://example.com/file.tar.gz",
				"destination":      "/tmp/file.tar.gz",
				"checksum-url":     "https://example.com/checksums.txt",
				"checksum-pattern": 123,
			},
			wantErr: true,
		},
		{
			name: "extract-dir wrong type",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
				"extract-dir": 123,
			},
			wantErr: true,
		},
		{
			name: "with checksum-url and pattern",
			params: map[string]any{
				"url":              "https://example.com/file.tar.gz",
				"destination":      "/tmp/file.tar.gz",
				"checksum-url":     "https://example.com/SHA256SUMS",
				"checksum-pattern": "file.tar.gz",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "grep") {
					t.Errorf("expected grep command for pattern matching, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "with checksum-url without pattern",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/tmp/file.tar.gz",
				"checksum-url": "https://example.com/checksum.txt",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "cat") {
					t.Errorf("expected cat command for checksum file, got: %s", steps[0].Content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := DownloadVerifyExtract(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadVerifyExtract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestMakeExecutable(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "valid parameters",
			params: map[string]any{
				"path": "/usr/local/bin/app",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "chmod +x /usr/local/bin/app") {
					t.Errorf("expected chmod command, got: %s", steps[0].Content)
				}
			},
		},
		{
			name:    "missing path",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := MakeExecutable(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeExecutable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
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
		"clone-and-build-go",
		"build-go-static",
		"clone-and-build-rust",
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

func TestCloneAndBuildGo(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "minimal parameters",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "git clone") {
					t.Errorf("expected git clone in step 0, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "/src") {
					t.Errorf("expected /src workdir in step 0, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[2].Content, "/main") {
					t.Errorf("expected default output /main, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with custom output and package",
			params: map[string]any{
				"repo":    "https://github.com/example/repo",
				"package": "./cmd/app",
				"output":  "/app",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[2].Content, "/app") {
					t.Errorf("expected custom output /app, got: %s", steps[2].Content)
				}
				if !strings.Contains(steps[2].Content, "./cmd/app") {
					t.Errorf("expected custom package ./cmd/app, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with explicit tag",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
				"tag":  "v1.2.3",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "--branch v1.2.3") {
					t.Errorf("expected --branch v1.2.3, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "without tag uses github_tag function",
			params: map[string]any{
				"repo": "https://github.com/csmith/dotege",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, `{{github_tag "csmith/dotege"}}`) {
					t.Errorf("expected github_tag function, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "non-GitHub repo without tag",
			params: map[string]any{
				"repo": "https://gitlab.com/example/repo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if strings.Contains(steps[0].Content, "github_tag") {
					t.Errorf("expected no github_tag for non-GitHub repo, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "git clone --depth=1") {
					t.Errorf("expected simple clone for non-GitHub repo, got: %s", steps[0].Content)
				}
			},
		},
		{
			name:    "missing repo",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := CloneAndBuildGo(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneAndBuildGo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestBuildGoStatic(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "minimal parameters",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[2].Content, "-extldflags \"-static\"") {
					t.Errorf("expected static linking flags, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with ignore parameter",
			params: map[string]any{
				"repo":   "https://github.com/example/repo",
				"ignore": "modernc.org/mathutil",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[3].Content, "--ignore modernc.org/mathutil") {
					t.Errorf("expected ignore parameter, got: %s", steps[3].Content)
				}
			},
		},
		{
			name: "with custom workdir",
			params: map[string]any{
				"repo":    "https://github.com/example/repo",
				"workdir": "/custom",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "/custom") {
					t.Errorf("expected custom workdir, got: %s", steps[0].Content)
				}
			},
		},
		{
			name:    "missing repo",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := BuildGo(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestCloneAndBuildRust(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "minimal parameters",
			params: map[string]any{
				"repo": "https://github.com/example/rustapp",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 3 {
					t.Errorf("expected 3 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "git clone") {
					t.Errorf("expected git clone in step 0, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "/src") {
					t.Errorf("expected /src workdir in step 0, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[1].Content, "cargo build --release --target x86_64-unknown-linux-musl") {
					t.Errorf("expected cargo build with musl target, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[2].Content, "find") && !strings.Contains(steps[2].Content, "/main") {
					t.Errorf("expected copy to /main, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with features",
			params: map[string]any{
				"repo":     "https://github.com/example/rustapp",
				"features": "sqlite,enable_mimalloc",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[1].Content, "--features sqlite,enable_mimalloc") {
					t.Errorf("expected features in build command, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with patches as array",
			params: map[string]any{
				"repo": "https://github.com/example/rustapp",
				"patches": []any{
					"fix1.diff",
					"fix2.diff",
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 5 {
					t.Errorf("expected 5 steps (clone + 2 patches + build + copy), got %d", len(steps))
				}
				if !strings.Contains(steps[1].Content, "fix1.diff") {
					t.Errorf("expected first patch step, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[2].Content, "fix2.diff") {
					t.Errorf("expected second patch step, got: %s", steps[2].Content)
				}
				if !strings.Contains(steps[1].Content, "patch -p1") {
					t.Errorf("expected patch command, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with single patch as string",
			params: map[string]any{
				"repo":    "https://github.com/example/rustapp",
				"patches": "single.diff",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps (clone + patch + build + copy), got %d", len(steps))
				}
				if !strings.Contains(steps[1].Content, "single.diff") {
					t.Errorf("expected patch step, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with custom workdir and output",
			params: map[string]any{
				"repo":    "https://github.com/example/rustapp",
				"workdir": "/build",
				"output":  "/app/binary",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "/build") {
					t.Errorf("expected custom workdir /build, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[2].Content, "/app/binary") {
					t.Errorf("expected custom output /app/binary, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with explicit tag",
			params: map[string]any{
				"repo": "https://github.com/example/rustapp",
				"tag":  "v2.0.0",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "--branch v2.0.0") {
					t.Errorf("expected --branch v2.0.0, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "without tag uses github_tag function",
			params: map[string]any{
				"repo": "https://github.com/dani-garcia/vaultwarden",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, `{{github_tag "dani-garcia/vaultwarden"}}`) {
					t.Errorf("expected github_tag function, got: %s", steps[0].Content)
				}
			},
		},
		{
			name:    "missing repo",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := CloneAndBuildRust(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneAndBuildRust() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestCloneAndBuildMake(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "minimal parameters with default strip",
			params: map[string]any{
				"repo": "https://github.com/example/makerepo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 2 {
					t.Errorf("expected 2 steps (clone + strip), got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "git clone") {
					t.Errorf("expected git clone in step 0, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[1].Content, "find /src/example/makerepo -type f -executable -exec strip") {
					t.Errorf("expected strip command in step 1, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with make-steps",
			params: map[string]any{
				"repo": "https://github.com/example/makerepo",
				"make-steps": []any{
					"make",
					"make install",
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 3 {
					t.Errorf("expected 3 steps (clone + make + strip), got %d", len(steps))
				}
				if !strings.Contains(steps[1].Content, "make") {
					t.Errorf("expected make command, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[1].Content, "make install") {
					t.Errorf("expected make install command, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with single make-step as string",
			params: map[string]any{
				"repo":       "https://github.com/example/makerepo",
				"make-steps": "make",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 3 {
					t.Errorf("expected 3 steps (clone + make + strip), got %d", len(steps))
				}
				if !strings.Contains(steps[1].Content, "make") {
					t.Errorf("expected make command, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with strip disabled",
			params: map[string]any{
				"repo":  "https://github.com/example/makerepo",
				"strip": false,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step (clone only), got %d", len(steps))
				}
				if strings.Contains(steps[0].Content, "strip") {
					t.Errorf("unexpected strip command when strip=false, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "with strip enabled explicitly",
			params: map[string]any{
				"repo":  "https://github.com/example/makerepo",
				"strip": true,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				hasStripStep := false
				for _, step := range steps {
					if strings.Contains(step.Content, "strip") {
						hasStripStep = true
						break
					}
				}
				if !hasStripStep {
					t.Error("expected strip step when strip=true")
				}
			},
		},
		{
			name: "with custom workdir",
			params: map[string]any{
				"repo":    "https://github.com/example/makerepo",
				"workdir": "/build",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "/build") {
					t.Errorf("expected custom workdir /build, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "with tag",
			params: map[string]any{
				"repo": "https://github.com/example/makerepo",
				"tag":  "v1.0.0",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "--branch v1.0.0") {
					t.Errorf("expected --branch v1.0.0, got: %s", steps[0].Content)
				}
			},
		},
		{
			name:    "missing repo",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := CloneAndBuildMake(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneAndBuildMake() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestCloneAndBuildAutoconf(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "minimal parameters with defaults",
			params: map[string]any{
				"repo": "https://github.com/example/autoconfrepo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 3 {
					t.Errorf("expected 3 steps (clone + configure + strip), got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "git clone") {
					t.Errorf("expected git clone in step 0, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[1].Content, "./configure") {
					t.Errorf("expected configure in step 1, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[2].Content, "strip") {
					t.Errorf("expected strip in step 2, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with configure options",
			params: map[string]any{
				"repo": "https://github.com/example/autoconfrepo",
				"configure-options": []any{
					"--prefix=/usr",
					"--enable-static",
					"--disable-shared",
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 3 {
					t.Errorf("expected 3 steps, got %d", len(steps))
				}
				configStep := steps[1].Content
				if !strings.Contains(configStep, "./configure --prefix=/usr --enable-static --disable-shared") {
					t.Errorf("expected configure with options, got: %s", configStep)
				}
			},
		},
		{
			name: "with single configure option as string",
			params: map[string]any{
				"repo":              "https://github.com/example/autoconfrepo",
				"configure-options": "--prefix=/usr",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[1].Content, "./configure --prefix=/usr") {
					t.Errorf("expected configure with option, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with make steps",
			params: map[string]any{
				"repo": "https://github.com/example/autoconfrepo",
				"make-steps": []any{
					"make -j$(nproc)",
					"make install",
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps (clone + configure + make + strip), got %d", len(steps))
				}
				makeStep := steps[2].Content
				if !strings.Contains(makeStep, "make -j$(nproc)") {
					t.Errorf("expected make -j in step 2, got: %s", makeStep)
				}
				if !strings.Contains(makeStep, "make install") {
					t.Errorf("expected make install in step 2, got: %s", makeStep)
				}
			},
		},
		{
			name: "with single make step as string",
			params: map[string]any{
				"repo":       "https://github.com/example/autoconfrepo",
				"make-steps": "make install",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[2].Content, "make install") {
					t.Errorf("expected make install, got: %s", steps[2].Content)
				}
			},
		},
		{
			name: "with strip disabled",
			params: map[string]any{
				"repo":  "https://github.com/example/autoconfrepo",
				"strip": false,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 2 {
					t.Errorf("expected 2 steps (clone + configure), got %d", len(steps))
				}
				for _, step := range steps {
					if strings.Contains(step.Content, "strip") {
						t.Errorf("unexpected strip command when strip=false, got: %s", step.Content)
					}
				}
			},
		},
		{
			name: "with strip enabled explicitly",
			params: map[string]any{
				"repo":  "https://github.com/example/autoconfrepo",
				"strip": true,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				hasStripStep := false
				for _, step := range steps {
					if strings.Contains(step.Content, "strip") {
						hasStripStep = true
						break
					}
				}
				if !hasStripStep {
					t.Error("expected strip step when strip=true")
				}
			},
		},
		{
			name: "with custom workdir",
			params: map[string]any{
				"repo":    "https://github.com/example/autoconfrepo",
				"workdir": "/build/custom",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "/build/custom") {
					t.Errorf("expected custom workdir, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[1].Content, "WORKDIR /build/custom") {
					t.Errorf("expected WORKDIR /build/custom in configure step, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "with tag",
			params: map[string]any{
				"repo": "https://github.com/example/autoconfrepo",
				"tag":  "v2.5.1",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "--branch v2.5.1") {
					t.Errorf("expected --branch v2.5.1, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "complete example with all options",
			params: map[string]any{
				"repo":    "https://github.com/example/autoconfrepo",
				"workdir": "/build",
				"tag":     "v1.0.0",
				"configure-options": []any{
					"--prefix=/usr",
					"--enable-ssl",
				},
				"make-steps": []any{
					"make -j4",
					"make install",
				},
				"strip": true,
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "--branch v1.0.0") {
					t.Errorf("expected tag in clone, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[1].Content, "./configure --prefix=/usr --enable-ssl") {
					t.Errorf("expected configure options, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[2].Content, "make -j4") {
					t.Errorf("expected make steps, got: %s", steps[2].Content)
				}
				if !strings.Contains(steps[3].Content, "strip") {
					t.Errorf("expected strip step, got: %s", steps[3].Content)
				}
			},
		},
		{
			name:    "missing repo",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := CloneAndBuildAutoconf(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneAndBuildAutoconf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestClone(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "basic clone without tag or commit",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "git clone") {
					t.Errorf("expected git clone command, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, `{{github_tag "example/repo"}}`) {
					t.Errorf("expected github_tag function for GitHub repo, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "clone with tag",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
				"tag":  "v1.0.0",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "--branch v1.0.0") {
					t.Errorf("expected --branch v1.0.0, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "clone with commit",
			params: map[string]any{
				"repo":   "https://github.com/example/repo",
				"commit": "abc123def456",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "git checkout abc123def456") {
					t.Errorf("expected git checkout with commit, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "clone with custom workdir",
			params: map[string]any{
				"repo":    "https://github.com/example/repo",
				"workdir": "/custom/path",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if !strings.Contains(steps[0].Content, "/custom/path") {
					t.Errorf("expected custom workdir, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "both tag and commit specified - error",
			params: map[string]any{
				"repo":   "https://github.com/example/repo",
				"tag":    "v1.0.0",
				"commit": "abc123",
			},
			wantErr: true,
		},
		{
			name:    "missing repo",
			params:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := Clone(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestSetupUsersGroups(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "users and groups with rootfs",
			params: map[string]any{
				"rootfs": "/rootfs",
				"groups": []any{
					map[string]any{"name": "root", "gid": 0},
					map[string]any{"name": "nonroot", "gid": 65532},
				},
				"users": []any{
					map[string]any{
						"username": "root",
						"uid":      0,
						"gid":      0,
						"home":     "/root",
						"shell":    "/sbin/nologin",
					},
					map[string]any{
						"username": "nonroot",
						"uid":      65532,
						"gid":      65532,
						"home":     "/home/nonroot",
						"shell":    "/sbin/nologin",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				content := steps[0].Content
				if !strings.Contains(content, `echo "root:x:0:" >> /rootfs/etc/group`) {
					t.Errorf("expected root group creation, got: %s", content)
				}
				if !strings.Contains(content, `echo "nonroot:x:65532:" >> /rootfs/etc/group`) {
					t.Errorf("expected nonroot group creation, got: %s", content)
				}
				if !strings.Contains(content, `echo "root:x:0:0:root:/root:/sbin/nologin" >> /rootfs/etc/passwd`) {
					t.Errorf("expected root user creation, got: %s", content)
				}
				if !strings.Contains(content, `echo "nonroot:x:65532:65532:nonroot:/home/nonroot:/sbin/nologin" >> /rootfs/etc/passwd`) {
					t.Errorf("expected nonroot user creation, got: %s", content)
				}
			},
		},
		{
			name: "users with defaults (no home specified)",
			params: map[string]any{
				"users": []any{
					map[string]any{
						"username": "testuser",
						"uid":      1000,
						"gid":      1000,
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				content := steps[0].Content
				if !strings.Contains(content, `echo "testuser:x:1000:1000:testuser:/nonexistent:/sbin/nologin" >> /etc/passwd`) {
					t.Errorf("expected user with defaults, got: %s", content)
				}
				if strings.Contains(content, "mkdir -p /nonexistent") || strings.Contains(content, "mkdir -p  /nonexistent") {
					t.Errorf("should not create /nonexistent directory, got: %s", content)
				}
			},
		},
		{
			name: "home directories created automatically",
			params: map[string]any{
				"rootfs": "/rootfs",
				"users": []any{
					map[string]any{
						"username": "appuser",
						"uid":      1000,
						"gid":      1000,
						"home":     "/home/appuser",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				content := steps[0].Content
				if !strings.Contains(content, "mkdir -p /rootfs/home/appuser") {
					t.Errorf("expected home directory creation, got: %s", content)
				}
				if !strings.Contains(content, "chown 1000:1000 /rootfs/home/appuser") {
					t.Errorf("expected ownership change for home directory, got: %s", content)
				}
			},
		},
		{
			name:    "no users or groups",
			params:  map[string]any{},
			wantErr: true,
		},
		{
			name: "invalid groups - not an array",
			params: map[string]any{
				"groups": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid users - not an array",
			params: map[string]any{
				"users": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid group - missing name",
			params: map[string]any{
				"groups": []any{
					map[string]any{"gid": 1000},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid group - missing gid",
			params: map[string]any{
				"groups": []any{
					map[string]any{"name": "testgroup"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid user - missing username",
			params: map[string]any{
				"users": []any{
					map[string]any{"uid": 1000, "gid": 1000},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid user - missing uid",
			params: map[string]any{
				"users": []any{
					map[string]any{"username": "testuser", "gid": 1000},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid user - missing gid",
			params: map[string]any{
				"users": []any{
					map[string]any{"username": "testuser", "uid": 1000},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := SetupUsersGroups(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetupUsersGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestCreateDirectories(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "single directory without permissions",
			params: map[string]any{
				"directories": []any{
					map[string]any{"path": "/tmp"},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "mkdir -p /tmp") {
					t.Errorf("expected mkdir command, got: %s", steps[0].Content)
				}
				if strings.Contains(steps[0].Content, "chmod") {
					t.Errorf("unexpected chmod command, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "single directory with permissions",
			params: map[string]any{
				"directories": []any{
					map[string]any{
						"path":        "/tmp",
						"permissions": "01777",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "mkdir -p /tmp") {
					t.Errorf("expected mkdir command, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "chmod 01777 /tmp") {
					t.Errorf("expected chmod command with permissions, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "multiple directories without permissions",
			params: map[string]any{
				"directories": []any{
					map[string]any{"path": "/rootfs/tmp"},
					map[string]any{"path": "/rootfs/proc"},
					map[string]any{"path": "/rootfs/dev"},
					map[string]any{"path": "/rootfs/sys"},
					map[string]any{"path": "/rootfs/etc"},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "mkdir -p /rootfs/tmp /rootfs/proc /rootfs/dev /rootfs/sys /rootfs/etc") {
					t.Errorf("expected all directories in mkdir command, got: %s", steps[0].Content)
				}
				if strings.Contains(steps[0].Content, "chmod") {
					t.Errorf("unexpected chmod command, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "multiple directories with mixed permissions",
			params: map[string]any{
				"directories": []any{
					map[string]any{
						"path":        "/rootfs/tmp",
						"permissions": "01777",
					},
					map[string]any{"path": "/rootfs/proc"},
					map[string]any{"path": "/rootfs/dev"},
					map[string]any{"path": "/rootfs/sys"},
					map[string]any{"path": "/rootfs/etc"},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				content := steps[0].Content
				if !strings.Contains(content, "mkdir -p /rootfs/tmp /rootfs/proc /rootfs/dev /rootfs/sys /rootfs/etc") {
					t.Errorf("expected all directories in mkdir command, got: %s", content)
				}
				if !strings.Contains(content, "chmod 01777 /rootfs/tmp") {
					t.Errorf("expected chmod command for /rootfs/tmp, got: %s", content)
				}
				chmodCount := strings.Count(content, "chmod")
				if chmodCount != 1 {
					t.Errorf("expected exactly 1 chmod command, got %d", chmodCount)
				}
			},
		},
		{
			name: "multiple directories with multiple permissions",
			params: map[string]any{
				"directories": []any{
					map[string]any{
						"path":        "/var/tmp",
						"permissions": "01777",
					},
					map[string]any{
						"path":        "/etc/config",
						"permissions": "0755",
					},
					map[string]any{"path": "/var/log"},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				content := steps[0].Content
				if !strings.Contains(content, "mkdir -p /var/tmp /etc/config /var/log") {
					t.Errorf("expected all directories in mkdir command, got: %s", content)
				}
				if !strings.Contains(content, "chmod 01777 /var/tmp") {
					t.Errorf("expected chmod command for /var/tmp, got: %s", content)
				}
				if !strings.Contains(content, "chmod 0755 /etc/config") {
					t.Errorf("expected chmod command for /etc/config, got: %s", content)
				}
			},
		},
		{
			name:    "missing directories parameter",
			params:  map[string]any{},
			wantErr: true,
		},
		{
			name: "empty directories array",
			params: map[string]any{
				"directories": []any{},
			},
			wantErr: true,
		},
		{
			name: "directories not an array",
			params: map[string]any{
				"directories": "invalid",
			},
			wantErr: true,
		},
		{
			name: "directory missing path",
			params: map[string]any{
				"directories": []any{
					map[string]any{"permissions": "0755"},
				},
			},
			wantErr: true,
		},
		{
			name: "directory with empty path",
			params: map[string]any{
				"directories": []any{
					map[string]any{"path": ""},
				},
			},
			wantErr: true,
		},
		{
			name: "directory not a map",
			params: map[string]any{
				"directories": []any{
					"/tmp",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := CreateDirectories(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDirectories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}

func TestTypeErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		pipeline string
		params   map[string]any
		errMsg   string
	}{
		{
			name:     "clone - tag wrong type",
			pipeline: "clone",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
				"tag":  123,
			},
			errMsg: "parameter \"tag\" must be a string, got int",
		},
		{
			name:     "clone - workdir wrong type",
			pipeline: "clone",
			params: map[string]any{
				"repo":    "https://github.com/example/repo",
				"workdir": true,
			},
			errMsg: "parameter \"workdir\" must be a string, got bool",
		},
		{
			name:     "clone-and-build-make - strip wrong type",
			pipeline: "clone-and-build-make",
			params: map[string]any{
				"repo":  "https://github.com/example/repo",
				"strip": "yes",
			},
			errMsg: "parameter \"strip\" must be a boolean, got string",
		},
		{
			name:     "clone-and-build-autoconf - strip wrong type",
			pipeline: "clone-and-build-autoconf",
			params: map[string]any{
				"repo":  "https://github.com/example/repo",
				"strip": 1,
			},
			errMsg: "parameter \"strip\" must be a boolean, got int",
		},
		{
			name:     "download-verify-extract - strip-components wrong type",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"url":              "https://example.com/file.tar.gz",
				"destination":      "/tmp/file.tar.gz",
				"checksum":         "abc123",
				"extract-dir":      "/opt",
				"strip-components": "1",
			},
			errMsg: "parameter \"strip-components\" must be an integer, got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, exists := Registry[tt.pipeline]
			if !exists {
				t.Fatalf("pipeline %q not found in registry", tt.pipeline)
			}
			_, err := pipeline(tt.params)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.errMsg)
				return
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestArchiveFormatValidation(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "unsupported archive format",
			params: map[string]any{
				"url":         "https://example.com/file.rar",
				"destination": "/tmp/file.rar",
				"checksum":    "abc123",
				"extract-dir": "/opt",
			},
			wantErr:     true,
			errContains: "unsupported archive format",
		},
		{
			name: "unsupported archive format - exe",
			params: map[string]any{
				"url":         "https://example.com/file.exe",
				"destination": "/tmp/file.exe",
				"checksum":    "abc123",
				"extract-dir": "/opt",
			},
			wantErr:     true,
			errContains: "unsupported archive format",
		},
		{
			name: "valid tar.gz",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
				"extract-dir": "/opt",
			},
			wantErr: false,
		},
		{
			name: "valid tgz",
			params: map[string]any{
				"url":         "https://example.com/file.tgz",
				"destination": "/tmp/file.tgz",
				"checksum":    "abc123",
				"extract-dir": "/opt",
			},
			wantErr: false,
		},
		{
			name: "valid tar.xz",
			params: map[string]any{
				"url":         "https://example.com/file.tar.xz",
				"destination": "/tmp/file.tar.xz",
				"checksum":    "abc123",
				"extract-dir": "/opt",
			},
			wantErr: false,
		},
		{
			name: "valid zip",
			params: map[string]any{
				"url":         "https://example.com/file.zip",
				"destination": "/tmp/file.zip",
				"checksum":    "abc123",
				"extract-dir": "/opt",
			},
			wantErr: false,
		},
		{
			name: "no extraction - any format allowed",
			params: map[string]any{
				"url":         "https://example.com/file.rar",
				"destination": "/tmp/file.rar",
				"checksum":    "abc123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DownloadVerifyExtract(tt.params)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGeneratePatchSteps(t *testing.T) {
	tests := []struct {
		name    string
		patches []string
		workdir string
		check   func(*testing.T, []Step)
	}{
		{
			name:    "empty patches",
			patches: []string{},
			workdir: "/src",
			check: func(t *testing.T, steps []Step) {
				if len(steps) != 0 {
					t.Errorf("expected 0 steps for empty patches, got %d", len(steps))
				}
			},
		},
		{
			name:    "single patch",
			patches: []string{"fix.patch"},
			workdir: "/src/project",
			check: func(t *testing.T, steps []Step) {
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Name, "Apply patch fix.patch") {
					t.Errorf("expected patch name in step name, got: %s", steps[0].Name)
				}
				if !strings.Contains(steps[0].Content, "COPY fix.patch /src/project/") {
					t.Errorf("expected COPY command, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "cd /src/project && patch -p1 < fix.patch") {
					t.Errorf("expected patch command, got: %s", steps[0].Content)
				}
			},
		},
		{
			name:    "multiple patches",
			patches: []string{"fix1.diff", "fix2.diff", "fix3.diff"},
			workdir: "/build",
			check: func(t *testing.T, steps []Step) {
				if len(steps) != 3 {
					t.Errorf("expected 3 steps, got %d", len(steps))
				}
				for i, patch := range []string{"fix1.diff", "fix2.diff", "fix3.diff"} {
					if !strings.Contains(steps[i].Name, patch) {
						t.Errorf("step %d: expected patch name %s in step name, got: %s", i, patch, steps[i].Name)
					}
					if !strings.Contains(steps[i].Content, "COPY "+patch+" /build/") {
						t.Errorf("step %d: expected COPY command for %s, got: %s", i, patch, steps[i].Content)
					}
					if !strings.Contains(steps[i].Content, "patch -p1 < "+patch) {
						t.Errorf("step %d: expected patch command for %s, got: %s", i, patch, steps[i].Content)
					}
				}
			},
		},
		{
			name:    "patches with subdirectory paths",
			patches: []string{"patches/security-fix.patch", "local/build-fix.diff"},
			workdir: "/src",
			check: func(t *testing.T, steps []Step) {
				if len(steps) != 2 {
					t.Errorf("expected 2 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "COPY patches/security-fix.patch /src/") {
					t.Errorf("expected COPY with full path, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "patch -p1 < patches/security-fix.patch") {
					t.Errorf("expected patch with full path, got: %s", steps[0].Content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := generatePatchSteps(tt.patches, tt.workdir)
			tt.check(t, steps)
		})
	}
}

func TestCloneAndBuildGoWithPatches(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "with patches as array",
			params: map[string]any{
				"repo": "https://github.com/example/gorepo",
				"patches": []any{
					"fix1.patch",
					"fix2.patch",
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 6 {
					t.Errorf("expected 6 steps (clone + 2 patches + mod download + build + license), got %d", len(steps))
				}
				if !strings.Contains(steps[1].Content, "fix1.patch") {
					t.Errorf("expected first patch step, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[2].Content, "fix2.patch") {
					t.Errorf("expected second patch step, got: %s", steps[2].Content)
				}
				if !strings.Contains(steps[1].Content, "patch -p1") {
					t.Errorf("expected patch command, got: %s", steps[1].Content)
				}
				// Verify patch is in build deps
				hasPatch := false
				for _, dep := range result.BuildDeps {
					if dep == "patch" {
						hasPatch = true
						break
					}
				}
				if !hasPatch {
					t.Errorf("expected 'patch' in BuildDeps, got: %v", result.BuildDeps)
				}
			},
		},
		{
			name: "with single patch as string",
			params: map[string]any{
				"repo":    "https://github.com/example/gorepo",
				"patches": "single.patch",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 5 {
					t.Errorf("expected 5 steps (clone + patch + mod download + build + license), got %d", len(steps))
				}
				if !strings.Contains(steps[1].Content, "single.patch") {
					t.Errorf("expected patch step, got: %s", steps[1].Content)
				}
			},
		},
		{
			name: "without patches - no patch in deps",
			params: map[string]any{
				"repo": "https://github.com/example/gorepo",
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 4 {
					t.Errorf("expected 4 steps (clone + mod download + build + license), got %d", len(steps))
				}
				// Verify patch is NOT in build deps
				for _, dep := range result.BuildDeps {
					if dep == "patch" {
						t.Errorf("unexpected 'patch' in BuildDeps when no patches specified")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CloneAndBuildGo(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloneAndBuildGo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestCopyFiles(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		check   func(*testing.T, PipelineResult)
	}{
		{
			name: "single file without options",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from": "config.yaml",
						"to":   "/etc/config.yaml",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "COPY config.yaml /etc/config.yaml") {
					t.Errorf("expected COPY command, got: %s", steps[0].Content)
				}
				if strings.Contains(steps[0].Content, "--chown") {
					t.Errorf("unexpected --chown flag, got: %s", steps[0].Content)
				}
				if strings.Contains(steps[0].Content, "--chmod") {
					t.Errorf("unexpected --chmod flag, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "single file with chown",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from":  "app.conf",
						"to":    "/etc/app.conf",
						"chown": "appuser:appgroup",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "COPY --chown=appuser:appgroup app.conf /etc/app.conf") {
					t.Errorf("expected COPY with --chown, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "single file with chmod",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from":  "entrypoint.sh",
						"to":    "/usr/local/bin/entrypoint.sh",
						"chmod": "0755",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "COPY --chmod=0755 entrypoint.sh /usr/local/bin/entrypoint.sh") {
					t.Errorf("expected COPY with --chmod, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "single file with both chown and chmod",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from":  "script.sh",
						"to":    "/app/script.sh",
						"chown": "root:root",
						"chmod": "0700",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 1 {
					t.Errorf("expected 1 step, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "--chown=root:root") {
					t.Errorf("expected --chown flag, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "--chmod=0700") {
					t.Errorf("expected --chmod flag, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[0].Content, "script.sh /app/script.sh") {
					t.Errorf("expected file paths, got: %s", steps[0].Content)
				}
			},
		},
		{
			name: "multiple files",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from": "config.yaml",
						"to":   "/etc/config.yaml",
					},
					map[string]any{
						"from":  "entrypoint.sh",
						"to":    "/usr/local/bin/entrypoint.sh",
						"chmod": "0755",
					},
					map[string]any{
						"from":  "app.conf",
						"to":    "/etc/app.conf",
						"chown": "appuser:appgroup",
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result PipelineResult) {
				steps := result.Steps
				if len(steps) != 3 {
					t.Errorf("expected 3 steps, got %d", len(steps))
				}
				if !strings.Contains(steps[0].Content, "COPY config.yaml /etc/config.yaml") {
					t.Errorf("expected first COPY command, got: %s", steps[0].Content)
				}
				if !strings.Contains(steps[1].Content, "COPY --chmod=0755 entrypoint.sh /usr/local/bin/entrypoint.sh") {
					t.Errorf("expected second COPY command with chmod, got: %s", steps[1].Content)
				}
				if !strings.Contains(steps[2].Content, "COPY --chown=appuser:appgroup app.conf /etc/app.conf") {
					t.Errorf("expected third COPY command with chown, got: %s", steps[2].Content)
				}
			},
		},
		{
			name:    "missing files parameter",
			params:  map[string]any{},
			wantErr: true,
		},
		{
			name: "empty files array",
			params: map[string]any{
				"files": []any{},
			},
			wantErr: true,
		},
		{
			name: "files not an array",
			params: map[string]any{
				"files": "invalid",
			},
			wantErr: true,
		},
		{
			name: "file missing from field",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"to": "/etc/config.yaml",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "file missing to field",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from": "config.yaml",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "file with empty from field",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from": "",
						"to":   "/etc/config.yaml",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "file with empty to field",
			params: map[string]any{
				"files": []any{
					map[string]any{
						"from": "config.yaml",
						"to":   "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "file not a map",
			params: map[string]any{
				"files": []any{
					"config.yaml",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := CopyFiles(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, steps)
			}
		})
	}
}
