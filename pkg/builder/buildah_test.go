package builder

import (
	"slices"
	"testing"
)

func TestBuildImageName(t *testing.T) {
	tests := []struct {
		name          string
		builder       *BuildahBuilder
		containerName string
		expected      string
	}{
		{
			name:          "no registry",
			builder:       &BuildahBuilder{registry: ""},
			containerName: "myapp",
			expected:      "myapp",
		},
		{
			name:          "with registry",
			builder:       &BuildahBuilder{registry: "ghcr.io/username"},
			containerName: "myapp",
			expected:      "ghcr.io/username/myapp:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder.buildImageName(tt.containerName)
			if result != tt.expected {
				t.Errorf("buildImageName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildStorageArgs(t *testing.T) {
	tests := []struct {
		name     string
		builder  *BuildahBuilder
		expected int
	}{
		{
			name:     "no storage config",
			builder:  &BuildahBuilder{isolation: "", storageDriver: "", storagePath: ""},
			expected: 0,
		},
		{
			name:     "with isolation",
			builder:  &BuildahBuilder{isolation: "chroot", storageDriver: "", storagePath: ""},
			expected: 2,
		},
		{
			name:     "with storage driver",
			builder:  &BuildahBuilder{isolation: "", storageDriver: "overlay", storagePath: ""},
			expected: 2,
		},
		{
			name:     "with storage path",
			builder:  &BuildahBuilder{isolation: "", storageDriver: "", storagePath: "/var/buildah"},
			expected: 4,
		},
		{
			name:     "with all options",
			builder:  &BuildahBuilder{isolation: "chroot", storageDriver: "overlay", storagePath: "/var/buildah"},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder.buildStorageArgs()
			if len(result) != tt.expected {
				t.Errorf("buildStorageArgs() length = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestBuildInspectArgs(t *testing.T) {
	tests := []struct {
		name     string
		builder  *BuildahBuilder
		imageID  string
		contains []string
	}{
		{
			name:     "basic inspect",
			builder:  &BuildahBuilder{storageDriver: "", storagePath: ""},
			imageID:  "abc123",
			contains: []string{"inspect", "--format", "{{.FromImageDigest}}", "abc123"},
		},
		{
			name:     "with storage driver",
			builder:  &BuildahBuilder{storageDriver: "overlay", storagePath: ""},
			imageID:  "abc123",
			contains: []string{"--storage-driver", "overlay", "inspect", "--format", "{{.FromImageDigest}}", "abc123"},
		},
		{
			name:     "with storage path",
			builder:  &BuildahBuilder{storageDriver: "", storagePath: "/var/buildah"},
			imageID:  "abc123",
			contains: []string{"--root", "/var/buildah/storage", "--runroot", "/var/buildah/run", "inspect", "--format", "{{.FromImageDigest}}", "abc123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder.buildInspectArgs(tt.imageID)
			for _, exp := range tt.contains {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildInspectArgs() missing expected: %q", exp)
				}
			}
		})
	}
}

func TestBuildBuildArgs(t *testing.T) {
	tests := []struct {
		name              string
		builder           *BuildahBuilder
		imageName         string
		containerfilePath string
		contextDir        string
		check             func(t *testing.T, args []string)
	}{
		{
			name:              "minimal args",
			builder:           &BuildahBuilder{isolation: "", storageDriver: "", storagePath: ""},
			imageName:         "myapp",
			containerfilePath: "/path/to/Containerfile",
			contextDir:        "/context",
			check: func(t *testing.T, args []string) {
				if args[0] != "build" {
					t.Errorf("expected first arg to be 'build', got %q", args[0])
				}
				if !slices.Contains(args, "--layers") {
					t.Error("missing --layers flag")
				}
				if !slices.Contains(args, "-t") {
					t.Error("missing -t flag")
				}
				if !slices.Contains(args, "myapp") {
					t.Error("missing image name")
				}
				if !slices.Contains(args, "/path/to/Containerfile") {
					t.Error("missing containerfile path")
				}
				if args[len(args)-1] != "/context" {
					t.Errorf("expected last arg to be context dir, got %q", args[len(args)-1])
				}
			},
		},
		{
			name:              "with storage options",
			builder:           &BuildahBuilder{isolation: "chroot", storageDriver: "overlay", storagePath: "/var/buildah"},
			imageName:         "myapp",
			containerfilePath: "/path/to/Containerfile",
			contextDir:        "/context",
			check: func(t *testing.T, args []string) {
				if !slices.Contains(args, "--isolation") {
					t.Error("missing --isolation flag")
				}
				if !slices.Contains(args, "chroot") {
					t.Error("missing isolation value")
				}
				if !slices.Contains(args, "--storage-driver") {
					t.Error("missing --storage-driver flag")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder.buildBuildArgs(tt.imageName, tt.containerfilePath, tt.contextDir)
			tt.check(t, result)
		})
	}
}
