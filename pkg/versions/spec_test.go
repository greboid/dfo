package versions

import "testing"

func TestVersionSpec_IsLatest(t *testing.T) {
	tests := []struct {
		name string
		spec VersionSpec
		want bool
	}{
		{
			name: "latest version",
			spec: VersionSpec{Key: "go", Value: "latest"},
			want: true,
		},
		{
			name: "latest with major version",
			spec: VersionSpec{Key: "postgres16", Value: "latest:16"},
			want: true,
		},
		{
			name: "specific version",
			spec: VersionSpec{Key: "go", Value: "1.22.0"},
			want: false,
		},
		{
			name: "empty value",
			spec: VersionSpec{Key: "go", Value: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.IsLatest(); got != tt.want {
				t.Errorf("IsLatest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionSpec_IsGitRepo(t *testing.T) {
	tests := []struct {
		name string
		spec VersionSpec
		want bool
	}{
		{
			name: "https git repo",
			spec: VersionSpec{Key: "https://github.com/owner/repo", Value: "latest"},
			want: true,
		},
		{
			name: "http git repo",
			spec: VersionSpec{Key: "http://github.com/owner/repo", Value: "latest"},
			want: true,
		},
		{
			name: "not a git repo",
			spec: VersionSpec{Key: "go", Value: "latest"},
			want: false,
		},
		{
			name: "empty key",
			spec: VersionSpec{Key: "", Value: "latest"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.IsGitRepo(); got != tt.want {
				t.Errorf("IsGitRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionSpec_VersionType(t *testing.T) {
	tests := []struct {
		name string
		spec VersionSpec
		want string
	}{
		{
			name: "git repo",
			spec: VersionSpec{Key: "https://github.com/owner/repo", Value: "latest"},
			want: "git",
		},
		{
			name: "go",
			spec: VersionSpec{Key: "go", Value: "latest"},
			want: "go",
		},
		{
			name: "golang",
			spec: VersionSpec{Key: "golang", Value: "latest"},
			want: "go",
		},
		{
			name: "postgres",
			spec: VersionSpec{Key: "postgres", Value: "latest"},
			want: "postgres",
		},
		{
			name: "postgres16",
			spec: VersionSpec{Key: "postgres16", Value: "latest:16"},
			want: "postgres",
		},
		{
			name: "postgresql",
			spec: VersionSpec{Key: "postgresql", Value: "latest"},
			want: "postgres",
		},
		{
			name: "alpine",
			spec: VersionSpec{Key: "alpine", Value: "latest"},
			want: "alpine",
		},
		{
			name: "unknown",
			spec: VersionSpec{Key: "customapp", Value: "latest"},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.VersionType(); got != tt.want {
				t.Errorf("VersionType() = %v, want %v", got, tt.want)
			}
		})
	}
}
