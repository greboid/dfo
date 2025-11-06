package repository

import (
	"fmt"
	"os"
	"path/filepath"
)

// Repository represents a dockerfiles repository structure
type Repository struct {
	Root          string
	PackagesDir   string
	ContainersDir string
	RepoDir       string
	ConfigFile    string
}

// Discover finds and validates a repository starting from the given directory
func Discover(startDir string) (*Repository, error) {
	// If startDir is empty, use current directory
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Make absolute path
	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Look for packages and containers directories
	packagesDir := filepath.Join(absPath, "packages")
	containersDir := filepath.Join(absPath, "containers")
	repoDir := filepath.Join(absPath, "repo")
	configFile := filepath.Join(absPath, ".dfo.yaml")

	// Validate that packages and containers directories exist
	if err := validateDir(packagesDir, "packages"); err != nil {
		return nil, err
	}

	if err := validateDir(containersDir, "containers"); err != nil {
		return nil, err
	}

	// Repo directory might not exist yet, we'll create it on first use
	// Config file is optional

	return &Repository{
		Root:          absPath,
		PackagesDir:   packagesDir,
		ContainersDir: containersDir,
		RepoDir:       repoDir,
		ConfigFile:    configFile,
	}, nil
}

// validateDir checks if a directory exists
func validateDir(path string, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s directory not found at %s (are you in a dockerfiles repository?)", name, path)
		}
		return fmt.Errorf("failed to check %s directory: %w", name, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s exists but is not a directory: %s", name, path)
	}

	return nil
}

// ListPackages returns all package YAML files in the packages directory
func (r *Repository) ListPackages() ([]string, error) {
	return listYAMLFiles(r.PackagesDir)
}

// ListContainers returns all container YAML files in the containers directory
func (r *Repository) ListContainers() ([]string, error) {
	return listYAMLFiles(r.ContainersDir)
}

// listYAMLFiles finds all .yaml files in a directory (non-recursive for now)
func listYAMLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

// GetPackagePath returns the full path to a package YAML file by name
func (r *Repository) GetPackagePath(name string) string {
	return filepath.Join(r.PackagesDir, name+".yaml")
}

// GetContainerPath returns the full path to a container YAML file by name
func (r *Repository) GetContainerPath(name string) string {
	return filepath.Join(r.ContainersDir, name+".yaml")
}

// GetContainerLockPath returns the full path to a container lock file by name
func (r *Repository) GetContainerLockPath(name string) string {
	return filepath.Join(r.ContainersDir, name+".yaml.lock")
}

// EnsureRepoDir ensures the repo directory exists
func (r *Repository) EnsureRepoDir() error {
	if err := os.MkdirAll(r.RepoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}
	return nil
}

// GetAPKINDEXPath returns the path to the APKINDEX file
func (r *Repository) GetAPKINDEXPath() string {
	return filepath.Join(r.RepoDir, "APKINDEX.tar.gz")
}
