package pipelines

import (
	"strings"
	"testing"
)

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name         string
		pipeline     string
		params       map[string]any
		wantError    bool
		errorContain string
	}{
		{
			name:     "create-user - all required params",
			pipeline: "create-user",
			params: map[string]any{
				"username": "appuser",
				"uid":      1000,
				"gid":      1000,
			},
			wantError: false,
		},
		{
			name:     "create-user - float64 for int (YAML behavior)",
			pipeline: "create-user",
			params: map[string]any{
				"username": "appuser",
				"uid":      float64(1000),
				"gid":      float64(1000),
			},
			wantError: false,
		},
		{
			name:     "create-user - missing username",
			pipeline: "create-user",
			params: map[string]any{
				"uid": 1000,
				"gid": 1000,
			},
			wantError:    true,
			errorContain: "required parameter \"username\" is missing",
		},
		{
			name:     "create-user - wrong type for uid",
			pipeline: "create-user",
			params: map[string]any{
				"username": "appuser",
				"uid":      "not-an-int",
				"gid":      1000,
			},
			wantError:    true,
			errorContain: "must be an integer",
		},
		{
			name:     "download-verify-extract - with checksum",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
			},
			wantError: false,
		},
		{
			name:     "download-verify-extract - with checksum-url",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/tmp/file.tar.gz",
				"checksum-url": "https://example.com/checksums.txt",
			},
			wantError: false,
		},
		{
			name:     "download-verify-extract - both checksums (mutually exclusive)",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/tmp/file.tar.gz",
				"checksum":     "abc123",
				"checksum-url": "https://example.com/checksums.txt",
			},
			wantError:    true,
			errorContain: "cannot specify both",
		},
		{
			name:     "download-verify-extract - neither checksum (at-least-one)",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/tmp/file.tar.gz",
			},
			wantError:    true,
			errorContain: "at least one of",
		},
		{
			name:     "download-verify-extract - empty strings for checksums (at-least-one)",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/tmp/file.tar.gz",
				"checksum":     "",
				"checksum-url": "",
			},
			wantError:    true,
			errorContain: "at least one of",
		},
		{
			name:     "download-verify-extract - missing url",
			pipeline: "download-verify-extract",
			params: map[string]any{
				"destination": "/tmp/file.tar.gz",
				"checksum":    "abc123",
			},
			wantError:    true,
			errorContain: "required parameter \"url\" is missing",
		},
		{
			name:     "clone - repo only",
			pipeline: "clone",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
			},
			wantError: false,
		},
		{
			name:     "clone - with tag",
			pipeline: "clone",
			params: map[string]any{
				"repo": "https://github.com/example/repo",
				"tag":  "v1.0.0",
			},
			wantError: false,
		},
		{
			name:     "clone - with commit",
			pipeline: "clone",
			params: map[string]any{
				"repo":   "https://github.com/example/repo",
				"commit": "abc123",
			},
			wantError: false,
		},
		{
			name:     "clone - both tag and commit (mutually exclusive)",
			pipeline: "clone",
			params: map[string]any{
				"repo":   "https://github.com/example/repo",
				"tag":    "v1.0.0",
				"commit": "abc123",
			},
			wantError:    true,
			errorContain: "cannot specify both",
		},
		{
			name:     "clone - tag with empty commit (not mutually exclusive)",
			pipeline: "clone",
			params: map[string]any{
				"repo":   "https://github.com/example/repo",
				"tag":    "v1.0.0",
				"commit": "",
			},
			wantError: false,
		},
		{
			name:     "setup-users-groups - with users",
			pipeline: "setup-users-groups",
			params: map[string]any{
				"rootfs": "/rootfs",
				"users": []any{
					map[string]any{"username": "app", "uid": 1000, "gid": 1000},
				},
			},
			wantError: false,
		},
		{
			name:     "setup-users-groups - with groups",
			pipeline: "setup-users-groups",
			params: map[string]any{
				"rootfs": "/rootfs",
				"groups": []any{
					map[string]any{"name": "app", "gid": 1000},
				},
			},
			wantError: false,
		},
		{
			name:     "setup-users-groups - neither users nor groups",
			pipeline: "setup-users-groups",
			params: map[string]any{
				"rootfs": "/rootfs",
			},
			wantError:    true,
			errorContain: "at least one of",
		},
		{
			name:     "clone-and-build-rust - patches as array",
			pipeline: "clone-and-build-rust",
			params: map[string]any{
				"repo":    "https://github.com/example/repo",
				"patches": []any{"patch1.patch", "patch2.patch"},
			},
			wantError: false,
		},
		{
			name:     "clone-and-build-rust - patches as string",
			pipeline: "clone-and-build-rust",
			params: map[string]any{
				"repo":    "https://github.com/example/repo",
				"patches": "single.patch",
			},
			wantError: false,
		},
		{
			name:     "clone-and-build-make - strip as bool",
			pipeline: "clone-and-build-make",
			params: map[string]any{
				"repo":  "https://github.com/example/repo",
				"strip": false,
			},
			wantError: false,
		},
		{
			name:     "clone-and-build-make - strip as string (wrong type)",
			pipeline: "clone-and-build-make",
			params: map[string]any{
				"repo":  "https://github.com/example/repo",
				"strip": "false",
			},
			wantError:    true,
			errorContain: "must be a boolean",
		},
		{
			name:     "create-directories - valid",
			pipeline: "create-directories",
			params: map[string]any{
				"directories": []any{
					map[string]any{"path": "/data", "permissions": "755"},
				},
			},
			wantError: false,
		},
		{
			name:         "create-directories - missing required",
			pipeline:     "create-directories",
			params:       map[string]any{},
			wantError:    true,
			errorContain: "required parameter \"directories\" is missing",
		},
		{
			name:     "create-directories - wrong type (not array)",
			pipeline: "create-directories",
			params: map[string]any{
				"directories": "not-an-array",
			},
			wantError:    true,
			errorContain: "must be an array of objects",
		},
		{
			name:      "unknown pipeline - no validation",
			pipeline:  "unknown-pipeline",
			params:    map[string]any{"anything": "goes"},
			wantError: false,
		},
		{
			name:     "make-executable - empty path",
			pipeline: "make-executable",
			params: map[string]any{
				"path": "",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(tt.pipeline, tt.params)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateParams() expected error, got nil")
					return
				}
				if tt.errorContain != "" && !strings.Contains(err.Error(), tt.errorContain) {
					t.Errorf("ValidateParams() error = %q, want error containing %q", err.Error(), tt.errorContain)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateParams() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckType(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		value     any
		expected  ParamType
		wantError bool
	}{
		{"string - valid", "test", "hello", TypeString, false},
		{"string - invalid", "test", 123, TypeString, true},
		{"int - valid int", "test", 42, TypeInt, false},
		{"int - valid float64", "test", float64(42), TypeInt, false},
		{"int - invalid", "test", "not-int", TypeInt, true},
		{"bool - valid", "test", true, TypeBool, false},
		{"bool - invalid", "test", "true", TypeBool, true},
		{"string_array - string", "test", "single", TypeStringArray, false},
		{"string_array - []string", "test", []string{"a", "b"}, TypeStringArray, false},
		{"string_array - []any strings", "test", []any{"a", "b"}, TypeStringArray, false},
		{"string_array - []any mixed", "test", []any{"a", 1}, TypeStringArray, true},
		{"string_array - invalid type", "test", 123, TypeStringArray, true},
		{"object_array - valid", "test", []any{map[string]any{"a": "b"}}, TypeObjectArray, false},
		{"object_array - invalid", "test", "not-array", TypeObjectArray, true},
		{"object_array - non-object item", "test", []any{map[string]any{"a": "b"}, "not-an-object"}, TypeObjectArray, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkType(tt.paramName, tt.value, tt.expected)
			if tt.wantError && err == nil {
				t.Errorf("checkType() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("checkType() unexpected error: %v", err)
			}
		})
	}
}

func TestAllPipelinesHaveSignatures(t *testing.T) {
	for name := range Registry {
		if _, exists := Signatures[name]; !exists {
			t.Errorf("Pipeline %q is registered but has no signature defined", name)
		}
	}
}

func TestAllSignaturesHavePipelines(t *testing.T) {
	for name := range Signatures {
		if _, exists := Registry[name]; !exists {
			t.Errorf("Signature defined for %q but no pipeline is registered", name)
		}
	}
}
