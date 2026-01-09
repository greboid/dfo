package graph

import (
	"testing"

	"github.com/greboid/dfo/pkg/config"
)

func TestExtractDependencies(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.BuildConfig
		wantDeps []string
	}{
		{
			name: "single stage with base image",
			config: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Stages: []config.Stage{
					{
						Name:        "build",
						Environment: config.Environment{BaseImage: "alpine"},
					},
				},
			},
			wantDeps: []string{"alpine"},
		},
		{
			name: "stage with external image",
			config: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Stages: []config.Stage{
					{
						Name:        "final",
						Environment: config.Environment{ExternalImage: "ubuntu:22.04"},
					},
				},
			},
			wantDeps: nil,
		},
		{
			name: "multiple stages with same base image",
			config: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Stages: []config.Stage{
					{
						Name:        "builder",
						Environment: config.Environment{BaseImage: "golang"},
					},
					{
						Name:        "runtime",
						Environment: config.Environment{BaseImage: "golang"},
					},
				},
			},
			wantDeps: []string{"golang"},
		},
		{
			name: "multiple stages with different base images",
			config: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Stages: []config.Stage{
					{
						Name:        "builder",
						Environment: config.Environment{BaseImage: "golang"},
					},
					{
						Name:        "runtime",
						Environment: config.Environment{BaseImage: "alpine"},
					},
				},
			},
			wantDeps: []string{"golang", "alpine"},
		},
		{
			name: "no stages",
			config: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Stages:  []config.Stage{},
			},
			wantDeps: nil,
		},
		{
			name: "stages with no base image",
			config: &config.BuildConfig{
				Package: config.Package{Name: "test"},
				Stages: []config.Stage{
					{
						Name:        "final",
						Environment: config.Environment{ExternalImage: "ubuntu:22.04"},
					},
				},
			},
			wantDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDependencies(tt.config)
			if len(got) != len(tt.wantDeps) {
				t.Errorf("got %d deps, want %d", len(got), len(tt.wantDeps))
			}
			for i, dep := range got {
				if i >= len(tt.wantDeps) || dep != tt.wantDeps[i] {
					t.Errorf("deps[%d] = %q, want %q", i, dep, safeGet(tt.wantDeps, i))
				}
			}
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name           string
		configs        map[string]*config.BuildConfig
		paths          map[string]string
		wantContainers []string
	}{
		{
			name: "single container",
			configs: map[string]*config.BuildConfig{
				"test": {
					Package: config.Package{Name: "test"},
					Stages: []config.Stage{
						{
							Name:        "build",
							Environment: config.Environment{BaseImage: "alpine"},
						},
					},
				},
			},
			paths: map[string]string{
				"test": "dfo.yaml",
			},
			wantContainers: []string{"test"},
		},
		{
			name: "multiple containers",
			configs: map[string]*config.BuildConfig{
				"app1": {
					Package: config.Package{Name: "app1"},
					Stages: []config.Stage{
						{Name: "build", Environment: config.Environment{BaseImage: "alpine"}},
					},
				},
				"app2": {
					Package: config.Package{Name: "app2"},
					Stages: []config.Stage{
						{Name: "build", Environment: config.Environment{BaseImage: "alpine"}},
					},
				},
			},
			paths: map[string]string{
				"app1": "app1/dfo.yaml",
				"app2": "app2/dfo.yaml",
			},
			wantContainers: []string{"app1", "app2"},
		},
		{
			name: "container with dependencies",
			configs: map[string]*config.BuildConfig{
				"base": {
					Package: config.Package{Name: "base"},
					Stages:  []config.Stage{{Name: "base", Environment: config.Environment{BaseImage: "alpine"}}},
				},
				"app": {
					Package: config.Package{Name: "app"},
					Stages:  []config.Stage{{Name: "app", Environment: config.Environment{BaseImage: "base"}}},
				},
			},
			paths: map[string]string{
				"base": "base/dfo.yaml",
				"app":  "app/dfo.yaml",
			},
			wantContainers: []string{"base", "app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := Build(tt.configs, tt.paths)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			if len(graph.Containers) != len(tt.wantContainers) {
				t.Errorf("got %d containers, want %d", len(graph.Containers), len(tt.wantContainers))
			}

			for _, name := range tt.wantContainers {
				if _, exists := graph.Containers[name]; !exists {
					t.Errorf("container %q not found in graph", name)
				}
				container := graph.Containers[name]
				if container.Name != name {
					t.Errorf("container name = %q, want %q", container.Name, name)
				}
				if container.ConfigPath != tt.paths[name] {
					t.Errorf("ConfigPath = %q, want %q", container.ConfigPath, tt.paths[name])
				}
			}
		})
	}
}

func TestBuild_ContainerProperties(t *testing.T) {
	configs := map[string]*config.BuildConfig{
		"test": {
			Package: config.Package{Name: "test-app"},
			Stages: []config.Stage{
				{Name: "build", Environment: config.Environment{BaseImage: "golang"}},
				{Name: "runtime", Environment: config.Environment{BaseImage: "alpine"}},
			},
		},
	}
	paths := map[string]string{
		"test": "test.yaml",
	}

	graph, err := Build(configs, paths)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	container := graph.Containers["test"]
	if container == nil {
		t.Fatal("container 'test' not found in graph")
	}

	if container.Name != "test" {
		t.Errorf("Name = %q, want 'test'", container.Name)
	}

	if container.ConfigPath != "test.yaml" {
		t.Errorf("ConfigPath = %q, want 'test.yaml'", container.ConfigPath)
	}

	if container.Config == nil {
		t.Fatal("Config is nil")
	}

	if container.Config.Package.Name != "test-app" {
		t.Errorf("Package.Name = %q, want 'test-app'", container.Config.Package.Name)
	}

	if len(container.Dependencies) != 2 {
		t.Errorf("got %d dependencies, want 2", len(container.Dependencies))
	}

	if container.Dependencies[0] != "golang" {
		t.Errorf("Dependencies[0] = %q, want 'golang'", container.Dependencies[0])
	}
	if container.Dependencies[1] != "alpine" {
		t.Errorf("Dependencies[1] = %q, want 'alpine'", container.Dependencies[1])
	}
}

func safeGet[T any](slice []T, index int) T {
	if index < len(slice) {
		return slice[index]
	}
	var zero T
	return zero
}
