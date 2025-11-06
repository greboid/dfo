package melange

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec represents a melange package specification
type Spec struct {
	Package     PackageInfo       `yaml:"package"`
	Environment EnvironmentConfig `yaml:"environment,omitempty"`
	Pipeline    []PipelineStep    `yaml:"pipeline,omitempty"`
	Subpackages []SubpackageInfo  `yaml:"subpackages,omitempty"`
	Update      UpdateConfig      `yaml:"update,omitempty"`

	// Path to the YAML file
	FilePath string `yaml:"-"`
}

// PackageInfo contains package metadata
type PackageInfo struct {
	Name         string           `yaml:"name"`
	Version      string           `yaml:"version"`
	Epoch        int              `yaml:"epoch"`
	Description  string           `yaml:"description,omitempty"`
	Copyright    []CopyrightEntry `yaml:"copyright,omitempty"`
	Dependencies struct {
		Runtime []string `yaml:"runtime,omitempty"`
	} `yaml:"dependencies,omitempty"`
}

// CopyrightEntry represents a copyright license entry
type CopyrightEntry struct {
	License string `yaml:"license"`
}

// EnvironmentConfig contains build environment settings
type EnvironmentConfig struct {
	Contents struct {
		Packages     []string `yaml:"packages,omitempty"`
		Repositories []string `yaml:"repositories,omitempty"`
	} `yaml:"contents,omitempty"`
}

// PipelineStep represents a build step
type PipelineStep map[string]interface{}

// SubpackageInfo contains subpackage metadata
type SubpackageInfo struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Pipeline    []PipelineStep `yaml:"pipeline,omitempty"`
}

// UpdateConfig contains version update configuration
type UpdateConfig struct {
	Enabled        bool              `yaml:"enabled"`
	Shared         bool              `yaml:"shared,omitempty"`
	Git            map[string]string `yaml:"git,omitempty"`
	GitHubMonitor  map[string]string `yaml:"github,omitempty"`
	ReleaseMonitor map[string]string `yaml:"release-monitor,omitempty"`
}

// LoadSpec reads and parses a melange YAML file
func LoadSpec(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec file: %w", err)
	}

	spec.FilePath = path
	return &spec, nil
}

// GetName returns the package name without path
func (s *Spec) GetName() string {
	return s.Package.Name
}

// GetVersion returns the full version string (version-r<epoch>)
func (s *Spec) GetVersion() string {
	return fmt.Sprintf("%s-r%d", s.Package.Version, s.Package.Epoch)
}

// GetAPKName returns the expected APK filename
func (s *Spec) GetAPKName(arch string) string {
	return fmt.Sprintf("%s-%s.apk", s.Package.Name, s.GetVersion())
}

// GetRuntimeDependencies returns the list of runtime dependencies
func (s *Spec) GetRuntimeDependencies() []string {
	// Filter out alpine packages (starting with http or containing alpinelinux.org)
	var deps []string
	for _, dep := range s.Package.Dependencies.Runtime {
		if !strings.HasPrefix(dep, "http") && !strings.Contains(dep, "alpinelinux.org") {
			// Strip @local, @edge, @community, etc. suffixes
			depName := strings.Split(dep, "@")[0]
			deps = append(deps, depName)
		}
	}
	return deps
}

// GetBuildDependencies returns packages needed for build environment
func (s *Spec) GetBuildDependencies() []string {
	var deps []string
	for _, pkg := range s.Environment.Contents.Packages {
		// Filter out alpine packages
		if !strings.Contains(pkg, "alpinelinux.org") {
			// Strip @local, @edge, @community, etc. suffixes
			depName := strings.Split(pkg, "@")[0]
			deps = append(deps, depName)
		}
	}
	return deps
}

// LoadAllSpecs loads all melange specs from a directory
func LoadAllSpecs(packagesDir string) (map[string]*Spec, error) {
	specs := make(map[string]*Spec)

	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml" {
			continue
		}

		path := filepath.Join(packagesDir, entry.Name())
		spec, err := LoadSpec(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}

		specs[spec.GetName()] = spec
	}

	return specs, nil
}

// Save writes the spec back to its file
func (s *Spec) Save() error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}

	if err := os.WriteFile(s.FilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write spec file: %w", err)
	}

	return nil
}
