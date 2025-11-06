package apko

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec represents an apko container specification
type Spec struct {
	Include     string            `yaml:"include,omitempty"`
	Contents    ContentsConfig    `yaml:"contents,omitempty"`
	Accounts    AccountsConfig    `yaml:"accounts,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Paths       []PathConfig      `yaml:"paths,omitempty"`
	Entrypoint  EntrypointConfig  `yaml:"entrypoint,omitempty"`
	Cmd         string            `yaml:"cmd,omitempty"`
	Work        WorkConfig        `yaml:"work,omitempty"`

	// Path to the YAML file
	FilePath string `yaml:"-"`
}

// ContentsConfig contains package and repository configuration
type ContentsConfig struct {
	Repositories []string `yaml:"repositories,omitempty"`
	Keyring      []string `yaml:"keyring,omitempty"`
	Packages     []string `yaml:"packages,omitempty"`
}

// AccountsConfig contains user and group definitions
type AccountsConfig struct {
	RunAs  string        `yaml:"run-as,omitempty"`
	Users  []UserConfig  `yaml:"users,omitempty"`
	Groups []GroupConfig `yaml:"groups,omitempty"`
}

// UserConfig defines a user account
type UserConfig struct {
	Username string `yaml:"username"`
	UID      int    `yaml:"uid"`
	GID      int    `yaml:"gid"`
}

// GroupConfig defines a group
type GroupConfig struct {
	GroupName string `yaml:"groupname"`
	GID       int    `yaml:"gid"`
}

// PathConfig defines a filesystem path
type PathConfig struct {
	Path        string `yaml:"path"`
	Type        string `yaml:"type,omitempty"`
	UID         int    `yaml:"uid,omitempty"`
	GID         int    `yaml:"gid,omitempty"`
	Permissions int    `yaml:"permissions,omitempty"`
}

// EntrypointConfig defines the container entrypoint
type EntrypointConfig struct {
	Command  string                 `yaml:"command,omitempty"`
	Type     string                 `yaml:"type,omitempty"`
	Services map[string]interface{} `yaml:"services,omitempty"`
}

// WorkConfig defines the working directory
type WorkConfig struct {
	Dir string `yaml:"dir,omitempty"`
}

// LoadSpec reads and parses an apko YAML file
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

// GetName returns the container name (filename without extension)
func (s *Spec) GetName() string {
	base := filepath.Base(s.FilePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// GetPackages returns all packages required by this container
func (s *Spec) GetPackages() []string {
	var packages []string

	// Filter out alpine/http URLs, keep only local package names
	for _, pkg := range s.Contents.Packages {
		if !strings.HasPrefix(pkg, "http") && !strings.Contains(pkg, "alpinelinux.org") {
			packages = append(packages, pkg)
		}
	}

	return packages
}

// GetRepositories returns all repositories
func (s *Spec) GetRepositories() []string {
	return s.Contents.Repositories
}

// LoadAllSpecs loads all apko specs from a directory
func LoadAllSpecs(containersDir string) (map[string]*Spec, error) {
	specs := make(map[string]*Spec)

	entries, err := os.ReadDir(containersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read containers directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		// Skip lock files
		if strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}

		path := filepath.Join(containersDir, entry.Name())
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
