package melange

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Builder handles building melange packages
type Builder struct {
	WorkDir    string
	OutputDir  string
	SigningKey string
	RepoDir    string
	Arch       string
}

// NewBuilder creates a new package builder
func NewBuilder(workDir, outputDir, repoDir, signingKey string) *Builder {
	return &Builder{
		WorkDir:    workDir,
		OutputDir:  outputDir,
		SigningKey: signingKey,
		RepoDir:    repoDir,
		Arch:       "x86_64", // Default architecture
	}
}

// Build builds a melange package
func (b *Builder) Build(spec *Spec) error {
	fmt.Printf("Building package: %s (%s)\n", spec.GetName(), spec.GetVersion())

	// Ensure output directory exists
	if err := os.MkdirAll(b.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Prepare melange command
	args := []string{
		"build",
		spec.FilePath,
		"--arch", b.Arch,
		"--out-dir", b.OutputDir,
	}

	args = append(args, "--signing-key", b.SigningKey)
	pubKeyPath := b.SigningKey + ".pub"
	args = append(args, "--keyring-append", pubKeyPath)

	// Add repository directory for dependencies
	if b.RepoDir != "" {
		args = append(args, "--repository-append", b.RepoDir)
	}

	fmt.Println("melange", args, b.WorkDir)
	// Execute melange
	cmd := exec.Command("melange", args...)
	cmd.Dir = b.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("melange build failed: %w", err)
	}

	fmt.Printf("Successfully built: %s\n", spec.GetName())
	return nil
}

// CheckNeedsBuild determines if a package needs to be built
func (b *Builder) CheckNeedsBuild(spec *Spec) (bool, error) {
	// Check if APK file exists in output directory
	apkPath := filepath.Join(b.OutputDir, b.Arch, spec.GetAPKName(b.Arch))

	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		return true, nil // Package doesn't exist, needs build
	} else if err != nil {
		return false, fmt.Errorf("failed to check package: %w", err)
	}

	// Package exists, no need to rebuild unless forced
	return false, nil
}

// GetPackagePath returns the path to a built package
func (b *Builder) GetPackagePath(spec *Spec) string {
	return filepath.Join(b.OutputDir, b.Arch, spec.GetAPKName(b.Arch))
}

// ListBuiltPackages returns all APK files in the output directory
func (b *Builder) ListBuiltPackages() ([]string, error) {
	archDir := filepath.Join(b.OutputDir, b.Arch)

	if _, err := os.Stat(archDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(archDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	var packages []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".apk") {
			packages = append(packages, filepath.Join(archDir, entry.Name()))
		}
	}

	return packages, nil
}

// SetArch sets the target architecture
func (b *Builder) SetArch(arch string) {
	b.Arch = arch
}
