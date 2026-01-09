package packages

import (
	"strings"
	"testing"
)

func TestParsePackageSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		wantName    string
		wantVersion string
		wantErr     bool
		errMsg      string
	}{
		{
			name:     "simple package name",
			spec:     "git",
			wantName: "git",
			wantErr:  false,
		},
		{
			name:     "package with hyphen",
			spec:     "ca-certificates",
			wantName: "ca-certificates",
			wantErr:  false,
		},
		{
			name:    "package with version equals sign",
			spec:    "package=1.0",
			wantErr: true,
			errMsg:  "package versions cannot be provided",
		},
		{
			name:    "empty spec",
			spec:    "",
			wantErr: true,
			errMsg:  "empty package specification",
		},
		{
			name:    "whitespace only",
			spec:    "   ",
			wantErr: true,
			errMsg:  "empty package specification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePackageSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" {
					if err.Error() != tt.errMsg {
						t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", got.Version, tt.wantVersion)
			}
		})
	}
}

func TestParsePackageSpecs(t *testing.T) {
	tests := []struct {
		name        string
		specs       []string
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:      "single package",
			specs:     []string{"git"},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiple packages",
			specs:     []string{"git", "ca-certificates", "curl"},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "empty slice",
			specs:     []string{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "nil slice",
			specs:     nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:        "one invalid spec",
			specs:       []string{"git", "bad=1.0"},
			wantErr:     true,
			errContains: "parsing package spec at index 1",
		},
		{
			name:        "first spec invalid",
			specs:       []string{"bad=1.0", "git"},
			wantErr:     true,
			errContains: "parsing package spec at index 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePackageSpecs(tt.specs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" {
					if err.Error() != tt.errContains && !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("error = %q, want contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != tt.wantCount {
				t.Errorf("got %d packages, want %d", len(got), tt.wantCount)
			}
		})
	}
}
