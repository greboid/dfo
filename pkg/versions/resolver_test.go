package versions

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	resolver := New(ctx, "user", "pass")

	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}

	if resolver.ctx != ctx {
		t.Error("context not set correctly")
	}

	if resolver.gitUser != "user" {
		t.Errorf("gitUser = %v, want user", resolver.gitUser)
	}

	if resolver.gitPass != "pass" {
		t.Errorf("gitPass = %v, want pass", resolver.gitPass)
	}
}

func TestResolve_SpecificVersion(t *testing.T) {
	ctx := context.Background()
	resolver := New(ctx, "", "")

	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "specific go version",
			key:     "go",
			value:   "1.22.0",
			wantErr: false,
		},
		{
			name:    "specific postgres version",
			key:     "postgres",
			value:   "16.1",
			wantErr: false,
		},
		{
			name:    "specific git tag",
			key:     "https://github.com/owner/repo",
			value:   "v1.2.3",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.value {
				t.Errorf("Resolve() = %v, want %v (specific version should be returned as-is)", got, tt.value)
			}
		})
	}
}

func TestResolve_UnknownKey(t *testing.T) {
	ctx := context.Background()
	resolver := New(ctx, "", "")

	_, err := resolver.Resolve("unknownkey", "latest")
	if err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

// Note: We don't test actual resolution of "latest" versions here because:
// 1. It requires network access to external services
// 2. Results change over time
// 3. It's better tested through integration tests
// The resolver relies on the github.com/csmith/latest library which is well-tested.
