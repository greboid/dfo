package apko

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LockFile represents an apko lock file
type LockFile struct {
	Version  string       `yaml:"version,omitempty"`
	Contents LockContents `yaml:"contents,omitempty"`

	// Path to the lock file
	FilePath string `yaml:"-"`
}

// LockContents contains locked package information
type LockContents struct {
	Repositories []string        `yaml:"repositories,omitempty"`
	Keyring      []string        `yaml:"keyring,omitempty"`
	Packages     []LockedPackage `yaml:"packages,omitempty"`
}

// LockedPackage represents a locked package version
type LockedPackage struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	Architecture string `yaml:"architecture,omitempty"`
	Checksum     string `yaml:"checksum,omitempty"`
}

// LoadLockFile reads and parses an apko lock file
func LoadLockFile(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Lock file doesn't exist yet
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lockFile LockFile
	if err := yaml.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	lockFile.FilePath = path
	return &lockFile, nil
}

// Exists checks if the lock file exists
func (l *LockFile) Exists() bool {
	if l == nil || l.FilePath == "" {
		return false
	}
	_, err := os.Stat(l.FilePath)
	return err == nil
}

// GetPackageVersions returns a map of package names to versions
func (l *LockFile) GetPackageVersions() map[string]string {
	if l == nil {
		return make(map[string]string)
	}

	versions := make(map[string]string)
	for _, pkg := range l.Contents.Packages {
		versions[pkg.Name] = pkg.Version
	}
	return versions
}

// Save writes the lock file
func (l *LockFile) Save() error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	if err := os.WriteFile(l.FilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// GetLockPath returns the lock file path for a spec
func GetLockPath(specPath string) string {
	return specPath + ".lock"
}
