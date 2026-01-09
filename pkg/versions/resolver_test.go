package versions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/csmith/latest/v2"
)

type mockGitTagClient struct {
	tag string
	err error
}

func (m *mockGitTagClient) GitTag(ctx context.Context, repo string, opts *latest.GitTagOptions) (string, error) {
	return m.tag, m.err
}

type mockGoReleaseClient struct {
	version  string
	url      string
	checksum string
	err      error
}

func (m *mockGoReleaseClient) GoRelease(ctx context.Context, opts *latest.GoOptions) (string, string, string, error) {
	return m.version, m.url, m.checksum, m.err
}

type mockPostgresReleaseClient struct {
	version  string
	url      string
	checksum string
	err      error
}

func (m *mockPostgresReleaseClient) PostgresRelease(ctx context.Context, opts *latest.TagOptions) (string, string, string, error) {
	return m.version, m.url, m.checksum, m.err
}

type mockAlpineReleaseClient struct {
	version  string
	url      string
	checksum string
	err      error
}

func (m *mockAlpineReleaseClient) AlpineRelease(ctx context.Context, opts *latest.AlpineReleaseOptions) (string, string, string, error) {
	return m.version, m.url, m.checksum, m.err
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	resolver := New(ctx, "user", "pass")

	if resolver.ctx != ctx {
		t.Errorf("New() ctx = %v, want %v", resolver.ctx, ctx)
	}
	if resolver.gitUser != "user" {
		t.Errorf("New() gitUser = %v, want %v", resolver.gitUser, "user")
	}
	if resolver.gitPass != "pass" {
		t.Errorf("New() gitPass = %v, want %v", resolver.gitPass, "pass")
	}
}

func TestNew_EmptyCredentials(t *testing.T) {
	ctx := context.Background()
	resolver := New(ctx, "", "")

	if resolver.ctx != ctx {
		t.Errorf("New() ctx = %v, want %v", resolver.ctx, ctx)
	}
	if resolver.gitUser != "" {
		t.Errorf("New() gitUser = %v, want empty", resolver.gitUser)
	}
	if resolver.gitPass != "" {
		t.Errorf("New() gitPass = %v, want empty", resolver.gitPass)
	}
}

func TestNewWithClients(t *testing.T) {
	ctx := context.Background()
	gitClient := &mockGitTagClient{}
	goClient := &mockGoReleaseClient{}
	postgresClient := &mockPostgresReleaseClient{}
	alpineClient := &mockAlpineReleaseClient{}

	resolver := NewWithClients(ctx, "user", "pass", gitClient.GitTag, goClient.GoRelease, postgresClient.PostgresRelease, alpineClient.AlpineRelease)

	if resolver.ctx != ctx {
		t.Errorf("NewWithClients() ctx = %v, want %v", resolver.ctx, ctx)
	}
	if resolver.gitUser != "user" {
		t.Errorf("NewWithClients() gitUser = %v, want %v", resolver.gitUser, "user")
	}
	if resolver.gitPass != "pass" {
		t.Errorf("NewWithClients() gitPass = %v, want %v", resolver.gitPass, "pass")
	}
}

func TestParsePostgresMajorVersion(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expected    int
		expectedErr string
	}{
		{
			name:        "valid major version",
			value:       "latest:15",
			expected:    15,
			expectedErr: "",
		},
		{
			name:        "valid major version 16",
			value:       "latest:16",
			expected:    16,
			expectedErr: "",
		},
		{
			name:        "no major version specified",
			value:       "latest",
			expected:    0,
			expectedErr: "",
		},
		{
			name:        "invalid major version",
			value:       "latest:abc",
			expected:    0,
			expectedErr: "invalid postgres major version",
		},
		{
			name:        "empty string",
			value:       "",
			expected:    0,
			expectedErr: "",
		},
		{
			name:        "single colon",
			value:       "latest:",
			expected:    0,
			expectedErr: "invalid postgres major version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePostgresMajorVersion(tt.value)

			if tt.expectedErr != "" {
				if err == nil {
					t.Errorf("parsePostgresMajorVersion() expected error containing %q, got nil", tt.expectedErr)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("parsePostgresMajorVersion() error = %q, want error containing %q", err.Error(), tt.expectedErr)
				}
			} else {
				if err != nil {
					t.Errorf("parsePostgresMajorVersion() unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("parsePostgresMajorVersion() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestBuildPostgresTagOptions(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectedMax int
		expectedErr string
	}{
		{
			name:        "with major version 15",
			value:       "latest:15",
			expectedMax: 15,
			expectedErr: "",
		},
		{
			name:        "with major version 16",
			value:       "latest:16",
			expectedMax: 16,
			expectedErr: "",
		},
		{
			name:        "without major version",
			value:       "latest",
			expectedMax: 0,
			expectedErr: "",
		},
		{
			name:        "invalid major version",
			value:       "latest:invalid",
			expectedMax: 0,
			expectedErr: "invalid postgres major version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := buildPostgresTagOptions(tt.value)

			if tt.expectedErr != "" {
				if err == nil {
					t.Errorf("buildPostgresTagOptions() expected error containing %q, got nil", tt.expectedErr)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("buildPostgresTagOptions() error = %q, want error containing %q", err.Error(), tt.expectedErr)
				}
			} else {
				if err != nil {
					t.Errorf("buildPostgresTagOptions() unexpected error: %v", err)
					return
				}
				if opts == nil {
					t.Errorf("buildPostgresTagOptions() returned nil options")
					return
				}
				if opts.MajorVersionMax != tt.expectedMax {
					t.Errorf("buildPostgresTagOptions() MajorVersionMax = %v, want %v", opts.MajorVersionMax, tt.expectedMax)
				}
				if len(opts.TrimPrefixes) != 1 || opts.TrimPrefixes[0] != "REL" {
					t.Errorf("buildPostgresTagOptions() TrimPrefixes = %v, want [\"REL\"]", opts.TrimPrefixes)
				}
			}
		})
	}
}

func TestFormatPostgresVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "standard version",
			version:  "15.3",
			expected: "REL_15_3",
		},
		{
			name:     "version with patch",
			version:  "16.1.2",
			expected: "REL_16_1_2",
		},
		{
			name:     "version with multiple dots",
			version:  "14.10.0",
			expected: "REL_14_10_0",
		},
		{
			name:     "already formatted version (double prefix)",
			version:  "REL_15_3",
			expected: "REL_REL_15_3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPostgresVersion(tt.version)
			if result != tt.expected {
				t.Errorf("formatPostgresVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolver_BuildGitTagOptions(t *testing.T) {
	tests := []struct {
		name           string
		gitUser        string
		gitPass        string
		expectUsername bool
		expectPassword bool
	}{
		{
			name:           "with credentials",
			gitUser:        "testuser",
			gitPass:        "testpass",
			expectUsername: true,
			expectPassword: true,
		},
		{
			name:           "without credentials",
			gitUser:        "",
			gitPass:        "",
			expectUsername: false,
			expectPassword: false,
		},
		{
			name:           "with username only",
			gitUser:        "testuser",
			gitPass:        "",
			expectUsername: true,
			expectPassword: true,
		},
		{
			name:           "with password only",
			gitUser:        "",
			gitPass:        "testpass",
			expectUsername: true,
			expectPassword: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Resolver{gitUser: tt.gitUser, gitPass: tt.gitPass}
			opts := r.buildGitTagOptions()

			if opts == nil {
				t.Errorf("buildGitTagOptions() returned nil options")
				return
			}
			if !opts.IgnorePreRelease {
				t.Errorf("buildGitTagOptions() IgnorePreRelease = false, want true")
			}
			if !opts.IgnoreErrors {
				t.Errorf("buildGitTagOptions() IgnoreErrors = false, want true")
			}

			if tt.expectUsername {
				if opts.Username != tt.gitUser {
					t.Errorf("buildGitTagOptions() Username = %v, want %v", opts.Username, tt.gitUser)
				}
			} else {
				if opts.Username != "" {
					t.Errorf("buildGitTagOptions() Username = %v, want empty", opts.Username)
				}
			}

			if tt.expectPassword {
				if opts.Password != tt.gitPass {
					t.Errorf("buildGitTagOptions() Password = %v, want %v", opts.Password, tt.gitPass)
				}
			} else {
				if opts.Password != "" {
					t.Errorf("buildGitTagOptions() Password = %v, want empty", opts.Password)
				}
			}
		})
	}
}

func TestResolver_Resolve_SpecificVersion(t *testing.T) {
	ctx := context.Background()
	r := New(ctx, "", "")

	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "specific version",
			key:      "go",
			value:    "1.21.0",
			expected: "1.21.0",
		},
		{
			name:     "specific version with prefix",
			key:      "https://github.com/owner/repo",
			value:    "v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "postgres specific version",
			key:      "postgres",
			value:    "REL_15_3",
			expected: "REL_15_3",
		},
		{
			name:     "alpine specific version",
			key:      "alpine",
			value:    "3.18.4",
			expected: "3.18.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := r.Resolve(tt.key, tt.value)

			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}
			if metadata.Version != tt.expected {
				t.Errorf("Resolve() Version = %v, want %v", metadata.Version, tt.expected)
			}
			if metadata.URL != "" {
				t.Errorf("Resolve() URL = %v, want empty", metadata.URL)
			}
			if metadata.Checksum != "" {
				t.Errorf("Resolve() Checksum = %v, want empty", metadata.Checksum)
			}
		})
	}
}

func TestResolver_Resolve_UnknownVersionType(t *testing.T) {
	ctx := context.Background()
	r := New(ctx, "", "")

	_, err := r.Resolve("unknown-package", "latest")
	if err == nil {
		t.Errorf("Resolve() expected error for unknown version type, got nil")
		return
	}
	if !strings.Contains(err.Error(), "unknown version key") {
		t.Errorf("Resolve() error = %v, want error containing 'unknown version key'", err)
	}
	if !strings.Contains(err.Error(), "unknown-package") {
		t.Errorf("Resolve() error = %v, want error containing 'unknown-package'", err)
	}
}

func TestResolver_ResolveByVersionType(t *testing.T) {
	ctx := context.Background()
	gitClient := &mockGitTagClient{}
	goClient := &mockGoReleaseClient{}
	postgresClient := &mockPostgresReleaseClient{}
	alpineClient := &mockAlpineReleaseClient{}
	r := NewWithClients(ctx, "testuser", "testpass", gitClient.GitTag, goClient.GoRelease, postgresClient.PostgresRelease, alpineClient.AlpineRelease)

	_, err := r.resolveByVersionType("unknown", "latest", "unknown")
	if err == nil {
		t.Errorf("resolveByVersionType() expected error for unknown type, got nil")
		return
	}
	if !strings.Contains(err.Error(), "unknown version key") {
		t.Errorf("resolveByVersionType() error = %v, want error containing 'unknown version key'", err)
	}
}

func TestResolver_ResolveGitTag(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		gitUser        string
		gitPass        string
		mockTag        string
		mockErr        error
		expectedTag    string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:        "successful tag resolution",
			repo:        "https://github.com/owner/repo",
			gitUser:     "user",
			gitPass:     "pass",
			mockTag:     "v1.2.3",
			mockErr:     nil,
			expectedTag: "v1.2.3",
			expectedErr: false,
		},
		{
			name:           "failed tag resolution",
			repo:           "https://github.com/owner/repo",
			gitUser:        "user",
			gitPass:        "pass",
			mockTag:        "",
			mockErr:        errors.New("network error"),
			expectedTag:    "",
			expectedErr:    true,
			expectedErrMsg: "resolving git tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitClient := &mockGitTagClient{tag: tt.mockTag, err: tt.mockErr}
			r := NewWithClients(context.Background(), tt.gitUser, tt.gitPass, gitClient.GitTag, nil, nil, nil)

			tag, err := r.resolveGitTag(tt.repo)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("resolveGitTag() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("resolveGitTag() error = %v, want error containing %q", err, tt.expectedErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolveGitTag() unexpected error: %v", err)
					return
				}
				if tag != tt.expectedTag {
					t.Errorf("resolveGitTag() = %v, want %v", tag, tt.expectedTag)
				}
			}
		})
	}
}

func TestResolver_ResolveGoVersion(t *testing.T) {
	tests := []struct {
		name           string
		mockVersion    string
		mockURL        string
		mockChecksum   string
		mockErr        error
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:         "successful go version resolution",
			mockVersion:  "1.21.0",
			mockURL:      "https://go.dev/dl/go1.21.0.linux-amd64.tar.gz",
			mockChecksum: "abc123",
			mockErr:      nil,
			expectedErr:  false,
		},
		{
			name:           "failed go version resolution",
			mockVersion:    "",
			mockURL:        "",
			mockChecksum:   "",
			mockErr:        errors.New("network error"),
			expectedErr:    true,
			expectedErrMsg: "resolving Go version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goClient := &mockGoReleaseClient{version: tt.mockVersion, url: tt.mockURL, checksum: tt.mockChecksum, err: tt.mockErr}
			r := NewWithClients(context.Background(), "", "", nil, goClient.GoRelease, nil, nil)

			metadata, err := r.resolveGoVersion()

			if tt.expectedErr {
				if err == nil {
					t.Errorf("resolveGoVersion() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("resolveGoVersion() error = %v, want error containing %q", err, tt.expectedErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolveGoVersion() unexpected error: %v", err)
					return
				}
				if metadata.Version != tt.mockVersion {
					t.Errorf("resolveGoVersion() Version = %v, want %v", metadata.Version, tt.mockVersion)
				}
				if metadata.URL != tt.mockURL {
					t.Errorf("resolveGoVersion() URL = %v, want %v", metadata.URL, tt.mockURL)
				}
				if metadata.Checksum != tt.mockChecksum {
					t.Errorf("resolveGoVersion() Checksum = %v, want %v", metadata.Checksum, tt.mockChecksum)
				}
			}
		})
	}
}

func TestResolver_ResolvePostgresVersion(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		mockVersion    string
		mockURL        string
		mockChecksum   string
		mockErr        error
		expectedVer    string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:         "successful postgres version resolution",
			value:        "latest",
			mockVersion:  "15.3",
			mockURL:      "https://postgres.org/15.3.tar.gz",
			mockChecksum: "def456",
			mockErr:      nil,
			expectedVer:  "REL_15_3",
			expectedErr:  false,
		},
		{
			name:           "failed postgres version resolution",
			value:          "latest",
			mockVersion:    "",
			mockURL:        "",
			mockChecksum:   "",
			mockErr:        errors.New("network error"),
			expectedVer:    "",
			expectedErr:    true,
			expectedErrMsg: "resolving Postgres version",
		},
		{
			name:           "invalid major version",
			value:          "latest:invalid",
			mockVersion:    "",
			mockURL:        "",
			mockChecksum:   "",
			mockErr:        nil,
			expectedVer:    "",
			expectedErr:    true,
			expectedErrMsg: "invalid postgres major version",
		},
		{
			name:         "postgres version with major constraint",
			value:        "latest:15",
			mockVersion:  "15.3",
			mockURL:      "https://postgres.org/15.3",
			mockChecksum: "def456",
			mockErr:      nil,
			expectedVer:  "REL_15_3",
			expectedErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postgresClient := &mockPostgresReleaseClient{version: tt.mockVersion, url: tt.mockURL, checksum: tt.mockChecksum, err: tt.mockErr}
			r := NewWithClients(context.Background(), "", "", nil, nil, postgresClient.PostgresRelease, nil)

			metadata, err := r.resolvePostgresVersion(tt.value)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("resolvePostgresVersion() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("resolvePostgresVersion() error = %v, want error containing %q", err, tt.expectedErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolvePostgresVersion() unexpected error: %v", err)
					return
				}
				if metadata.Version != tt.expectedVer {
					t.Errorf("resolvePostgresVersion() Version = %v, want %v", metadata.Version, tt.expectedVer)
				}
				if metadata.URL != tt.mockURL {
					t.Errorf("resolvePostgresVersion() URL = %v, want %v", metadata.URL, tt.mockURL)
				}
				if metadata.Checksum != tt.mockChecksum {
					t.Errorf("resolvePostgresVersion() Checksum = %v, want %v", metadata.Checksum, tt.mockChecksum)
				}
			}
		})
	}
}

func TestResolver_ResolveAlpineVersion(t *testing.T) {
	tests := []struct {
		name           string
		mockVersion    string
		mockURL        string
		mockChecksum   string
		mockErr        error
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:         "successful alpine version resolution",
			mockVersion:  "3.18.4",
			mockURL:      "https://alpinelinux.org/3.18.4.tar.gz",
			mockChecksum: "ghi789",
			mockErr:      nil,
			expectedErr:  false,
		},
		{
			name:           "failed alpine version resolution",
			mockVersion:    "",
			mockURL:        "",
			mockChecksum:   "",
			mockErr:        errors.New("network error"),
			expectedErr:    true,
			expectedErrMsg: "resolving Alpine version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alpineClient := &mockAlpineReleaseClient{version: tt.mockVersion, url: tt.mockURL, checksum: tt.mockChecksum, err: tt.mockErr}
			r := NewWithClients(context.Background(), "", "", nil, nil, nil, alpineClient.AlpineRelease)

			metadata, err := r.resolveAlpineVersion()

			if tt.expectedErr {
				if err == nil {
					t.Errorf("resolveAlpineVersion() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("resolveAlpineVersion() error = %v, want error containing %q", err, tt.expectedErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolveAlpineVersion() unexpected error: %v", err)
					return
				}
				if metadata.Version != tt.mockVersion {
					t.Errorf("resolveAlpineVersion() Version = %v, want %v", metadata.Version, tt.mockVersion)
				}
				if metadata.URL != tt.mockURL {
					t.Errorf("resolveAlpineVersion() URL = %v, want %v", metadata.URL, tt.mockURL)
				}
				if metadata.Checksum != tt.mockChecksum {
					t.Errorf("resolveAlpineVersion() Checksum = %v, want %v", metadata.Checksum, tt.mockChecksum)
				}
			}
		})
	}
}

func TestResolver_Resolve_LatestGit(t *testing.T) {
	gitClient := &mockGitTagClient{tag: "v2.0.0", err: nil}
	r := NewWithClients(context.Background(), "", "", gitClient.GitTag, nil, nil, nil)

	metadata, err := r.Resolve("https://github.com/owner/repo", "latest")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "v2.0.0" {
		t.Errorf("Resolve() Version = %v, want v2.0.0", metadata.Version)
	}
}

func TestResolver_Resolve_LatestGo(t *testing.T) {
	goClient := &mockGoReleaseClient{version: "1.21.5", url: "https://go.dev/dl/go1.21.5.linux-amd64.tar.gz", checksum: "abc123", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, goClient.GoRelease, nil, nil)

	metadata, err := r.Resolve("go", "latest")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "1.21.5" {
		t.Errorf("Resolve() Version = %v, want 1.21.5", metadata.Version)
	}
	if metadata.URL != "https://go.dev/dl/go1.21.5.linux-amd64.tar.gz" {
		t.Errorf("Resolve() URL = %v, want https://go.dev/dl/go1.21.5.linux-amd64.tar.gz", metadata.URL)
	}
	if metadata.Checksum != "abc123" {
		t.Errorf("Resolve() Checksum = %v, want abc123", metadata.Checksum)
	}
}

func TestResolver_Resolve_LatestPostgres(t *testing.T) {
	postgresClient := &mockPostgresReleaseClient{version: "16.1", url: "https://postgres.org/16.1.tar.gz", checksum: "def456", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, nil, postgresClient.PostgresRelease, nil)

	metadata, err := r.Resolve("postgres", "latest")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "REL_16_1" {
		t.Errorf("Resolve() Version = %v, want REL_16_1", metadata.Version)
	}
	if metadata.URL != "https://postgres.org/16.1.tar.gz" {
		t.Errorf("Resolve() URL = %v, want https://postgres.org/16.1.tar.gz", metadata.URL)
	}
	if metadata.Checksum != "def456" {
		t.Errorf("Resolve() Checksum = %v, want def456", metadata.Checksum)
	}
}

func TestResolver_Resolve_LatestAlpine(t *testing.T) {
	alpineClient := &mockAlpineReleaseClient{version: "3.19.0", url: "https://alpinelinux.org/3.19.0.tar.gz", checksum: "ghi789", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, nil, nil, alpineClient.AlpineRelease)

	metadata, err := r.Resolve("alpine", "latest")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "3.19.0" {
		t.Errorf("Resolve() Version = %v, want 3.19.0", metadata.Version)
	}
	if metadata.URL != "https://alpinelinux.org/3.19.0.tar.gz" {
		t.Errorf("Resolve() URL = %v, want https://alpinelinux.org/3.19.0.tar.gz", metadata.URL)
	}
	if metadata.Checksum != "ghi789" {
		t.Errorf("Resolve() Checksum = %v, want ghi789", metadata.Checksum)
	}
}

func TestResolver_Resolve_GitTagWithOptions(t *testing.T) {
	tests := []struct {
		name      string
		gitUser   string
		gitPass   string
		mockTag   string
		mockErr   error
		wantCreds bool
	}{
		{
			name:      "with credentials",
			gitUser:   "user",
			gitPass:   "pass",
			mockTag:   "v1.0.0",
			mockErr:   nil,
			wantCreds: true,
		},
		{
			name:      "without credentials",
			gitUser:   "",
			gitPass:   "",
			mockTag:   "v1.0.0",
			mockErr:   nil,
			wantCreds: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitClient := &mockGitTagClient{tag: tt.mockTag, err: tt.mockErr}
			r := NewWithClients(context.Background(), tt.gitUser, tt.gitPass, gitClient.GitTag, nil, nil, nil)

			metadata, err := r.Resolve("https://github.com/owner/repo", "latest")

			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}
			if metadata.Version != tt.mockTag {
				t.Errorf("Resolve() Version = %v, want %v", metadata.Version, tt.mockTag)
			}
		})
	}
}

func TestResolver_Resolve_GitTagError(t *testing.T) {
	gitClient := &mockGitTagClient{tag: "", err: fmt.Errorf("git error")}
	r := NewWithClients(context.Background(), "", "", gitClient.GitTag, nil, nil, nil)

	_, err := r.Resolve("https://github.com/owner/repo", "latest")

	if err == nil {
		t.Errorf("Resolve() expected error, got nil")
		return
	}
	if !strings.Contains(err.Error(), "resolving git tag") {
		t.Errorf("Resolve() error = %v, want error containing 'resolving git tag'", err)
	}
}

func TestResolver_Resolve_GoError(t *testing.T) {
	goClient := &mockGoReleaseClient{err: fmt.Errorf("go error")}
	r := NewWithClients(context.Background(), "", "", nil, goClient.GoRelease, nil, nil)

	_, err := r.Resolve("go", "latest")

	if err == nil {
		t.Errorf("Resolve() expected error, got nil")
		return
	}
	if !strings.Contains(err.Error(), "resolving Go version") {
		t.Errorf("Resolve() error = %v, want error containing 'resolving Go version'", err)
	}
}

func TestResolver_Resolve_PostgresError(t *testing.T) {
	postgresClient := &mockPostgresReleaseClient{err: fmt.Errorf("postgres error")}
	r := NewWithClients(context.Background(), "", "", nil, nil, postgresClient.PostgresRelease, nil)

	_, err := r.Resolve("postgres", "latest")

	if err == nil {
		t.Errorf("Resolve() expected error, got nil")
		return
	}
	if !strings.Contains(err.Error(), "resolving Postgres version") {
		t.Errorf("Resolve() error = %v, want error containing 'resolving Postgres version'", err)
	}
}

func TestResolver_Resolve_AlpineError(t *testing.T) {
	alpineClient := &mockAlpineReleaseClient{err: fmt.Errorf("alpine error")}
	r := NewWithClients(context.Background(), "", "", nil, nil, nil, alpineClient.AlpineRelease)

	_, err := r.Resolve("alpine", "latest")

	if err == nil {
		t.Errorf("Resolve() expected error, got nil")
		return
	}
	if !strings.Contains(err.Error(), "resolving Alpine version") {
		t.Errorf("Resolve() error = %v, want error containing 'resolving Alpine version'", err)
	}
}

func TestResolver_Resolve_WithMajorVersion(t *testing.T) {
	postgresClient := &mockPostgresReleaseClient{version: "15.2", url: "https://postgres.org/15.2.tar.gz", checksum: "checksum123", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, nil, postgresClient.PostgresRelease, nil)

	metadata, err := r.Resolve("postgres", "latest:15")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "REL_15_2" {
		t.Errorf("Resolve() Version = %v, want REL_15_2", metadata.Version)
	}
}

func TestResolver_Resolve_PostgresqlAlias(t *testing.T) {
	postgresClient := &mockPostgresReleaseClient{version: "15.2", url: "https://postgres.org/15.2.tar.gz", checksum: "checksum123", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, nil, postgresClient.PostgresRelease, nil)

	metadata, err := r.Resolve("postgresql", "latest")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "REL_15_2" {
		t.Errorf("Resolve() Version = %v, want REL_15_2", metadata.Version)
	}
}

func TestResolver_Resolve_GolangAlias(t *testing.T) {
	goClient := &mockGoReleaseClient{version: "1.21.5", url: "https://go.dev/dl/go1.21.5.linux-amd64.tar.gz", checksum: "abc123", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, goClient.GoRelease, nil, nil)

	metadata, err := r.Resolve("golang", "latest")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "1.21.5" {
		t.Errorf("Resolve() Version = %v, want 1.21.5", metadata.Version)
	}
}

func TestResolver_Resolve_LatestWithoutPrefix(t *testing.T) {
	goClient := &mockGoReleaseClient{version: "1.21.5", url: "https://go.dev/dl/go1.21.5.linux-amd64.tar.gz", checksum: "abc123", err: nil}
	r := NewWithClients(context.Background(), "", "", nil, goClient.GoRelease, nil, nil)

	metadata, err := r.Resolve("go", "latest/stable")

	if err != nil {
		t.Errorf("Resolve() unexpected error: %v", err)
		return
	}
	if metadata.Version != "1.21.5" {
		t.Errorf("Resolve() Version = %v, want 1.21.5", metadata.Version)
	}
}
