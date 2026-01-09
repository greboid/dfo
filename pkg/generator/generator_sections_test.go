package generator

import (
	"testing"

	"github.com/greboid/dfo/pkg/config"
)

func TestGenerateArgsSection(t *testing.T) {
	tests := []struct {
		name     string
		env      config.Environment
		expected string
	}{
		{
			name:     "empty args",
			env:      config.Environment{Args: map[string]string{}},
			expected: "",
		},
		{
			name: "single arg",
			env: config.Environment{Args: map[string]string{
				"VERSION": "1.0.0",
			}},
			expected: "ARG VERSION=\"1.0.0\"\n\n",
		},
		{
			name: "multiple args",
			env: config.Environment{Args: map[string]string{
				"VERSION": "1.0.0",
				"PORT":    "8080",
			}},
			expected: "ARG PORT=\"8080\"\nARG VERSION=\"1.0.0\"\n\n",
		},
	}

	g := &Generator{config: &config.BuildConfig{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.generateArgsSection(tt.env)
			if result != tt.expected {
				t.Errorf("generateArgsSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateLabelsSection(t *testing.T) {
	tests := []struct {
		name         string
		env          config.Environment
		config       *config.BuildConfig
		isFinalStage bool
		expected     string
	}{
		{
			name:         "no labels non-final",
			env:          config.Environment{},
			config:       &config.BuildConfig{Package: config.Package{Labels: map[string]string{}}},
			isFinalStage: false,
			expected:     "",
		},
		{
			name:         "no labels final",
			env:          config.Environment{},
			config:       &config.BuildConfig{Package: config.Package{Labels: map[string]string{}}},
			isFinalStage: true,
			expected:     "",
		},
		{
			name: "labels final stage",
			env:  config.Environment{},
			config: &config.BuildConfig{Package: config.Package{Labels: map[string]string{
				"maintainer": "test@example.com",
				"version":    "1.0.0",
			}}},
			isFinalStage: true,
			expected:     "LABEL maintainer=\"test@example.com\"\nLABEL version=\"1.0.0\"\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{config: tt.config}
			result := g.generateLabelsSection(tt.env, tt.isFinalStage)
			if result != tt.expected {
				t.Errorf("generateLabelsSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateEnvSection(t *testing.T) {
	tests := []struct {
		name     string
		env      config.Environment
		expected string
	}{
		{
			name:     "empty env",
			env:      config.Environment{Environment: map[string]string{}},
			expected: "",
		},
		{
			name: "single env var",
			env: config.Environment{Environment: map[string]string{
				"PATH": "/usr/local/bin",
			}},
			expected: "ENV PATH=\"/usr/local/bin\"\n\n",
		},
		{
			name: "multiple env vars",
			env: config.Environment{Environment: map[string]string{
				"PATH":     "/usr/local/bin",
				"APP_HOME": "/app",
			}},
			expected: "ENV APP_HOME=\"/app\"\nENV PATH=\"/usr/local/bin\"\n\n",
		},
	}

	g := &Generator{config: &config.BuildConfig{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.generateEnvSection(tt.env)
			if result != tt.expected {
				t.Errorf("generateEnvSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateWorkDirSection(t *testing.T) {
	tests := []struct {
		name     string
		env      config.Environment
		expected string
	}{
		{
			name:     "empty workdir",
			env:      config.Environment{WorkDir: ""},
			expected: "",
		},
		{
			name:     "workdir set",
			env:      config.Environment{WorkDir: "/app"},
			expected: "WORKDIR /app\n\n",
		},
	}

	g := &Generator{config: &config.BuildConfig{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.generateWorkDirSection(tt.env)
			if result != tt.expected {
				t.Errorf("generateWorkDirSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractShortDigest(t *testing.T) {
	tests := []struct {
		name     string
		digest   string
		expected string
	}{
		{
			name:     "digest with colon",
			digest:   "sha256:abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "digest without colon",
			digest:   "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "empty digest",
			digest:   "",
			expected: "",
		},
	}

	g := &Generator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.extractShortDigest(tt.digest)
			if result != tt.expected {
				t.Errorf("extractShortDigest() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSortBOMKeys(t *testing.T) {
	tests := []struct {
		name  string
		bom   map[string]string
		check func(t *testing.T, sorted map[string]string)
	}{
		{
			name: "empty bom",
			bom:  map[string]string{},
			check: func(t *testing.T, sorted map[string]string) {
				if len(sorted) != 0 {
					t.Errorf("expected empty map, got %d entries", len(sorted))
				}
			},
		},
		{
			name: "single entry",
			bom: map[string]string{
				"apk:git": "2.42.0",
			},
			check: func(t *testing.T, sorted map[string]string) {
				if len(sorted) != 1 {
					t.Errorf("expected 1 entry, got %d", len(sorted))
				}
				if val, ok := sorted["apk:git"]; !ok || val != "2.42.0" {
					t.Errorf("expected apk:git=2.42.0, got %v", sorted)
				}
			},
		},
		{
			name: "multiple unsorted entries",
			bom: map[string]string{
				"apk:busybox": "1.36.1",
				"versions:go": "1.21.0",
				"apk:git":     "2.42.0",
			},
			check: func(t *testing.T, sorted map[string]string) {
				if len(sorted) != 3 {
					t.Errorf("expected 3 entries, got %d", len(sorted))
				}
				expectedKeys := []string{"apk:busybox", "apk:git", "versions:go"}
				for _, key := range expectedKeys {
					if _, ok := sorted[key]; !ok {
						t.Errorf("expected key %q not found in sorted map", key)
					}
				}
			},
		},
	}

	g := &Generator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.sortBOMKeys(tt.bom)
			tt.check(t, result)
		})
	}
}

func TestMergeDeps(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: []string{},
		},
		{
			name:     "first empty",
			a:        []string{},
			b:        []string{"git"},
			expected: []string{"git"},
		},
		{
			name:     "second empty",
			a:        []string{"git"},
			b:        []string{},
			expected: []string{"git"},
		},
		{
			name:     "no overlap",
			a:        []string{"git", "make"},
			b:        []string{"gcc"},
			expected: []string{"git", "make", "gcc"},
		},
		{
			name:     "with overlap",
			a:        []string{"git", "make"},
			b:        []string{"make", "gcc"},
			expected: []string{"git", "make", "gcc"},
		},
		{
			name:     "all overlap",
			a:        []string{"git", "make"},
			b:        []string{"git", "make"},
			expected: []string{"git", "make"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeDeps(tt.a, tt.b)
			if len(result) != len(tt.expected) {
				t.Errorf("mergeDeps() length = %d, want %d", len(result), len(tt.expected))
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("mergeDeps()[%d] = %q, want %q", i, result[i], exp)
				}
			}
		})
	}
}
