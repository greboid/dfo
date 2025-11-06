package melange

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// APKIndex manages the APKINDEX.tar.gz file
type APKIndex struct {
	RepoDir    string
	OutputDir  string
	Arch       string
	SigningKey string
}

// NewAPKIndex creates a new APKINDEX manager
func NewAPKIndex(repoDir, outputDir, signingKey string) *APKIndex {
	return &APKIndex{
		RepoDir:    repoDir,
		OutputDir:  outputDir,
		Arch:       "x86_64",
		SigningKey: signingKey,
	}
}

// Generate creates/updates the APKINDEX.tar.gz from built packages
func (a *APKIndex) Generate() error {
	fmt.Println("Generating APKINDEX...")

	// Ensure repo directory exists
	if err := os.MkdirAll(a.RepoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	// Get the packages directory
	packagesDir := filepath.Join(a.OutputDir, a.Arch)

	// Check if packages directory exists and has packages
	if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
		fmt.Println("No packages found, skipping APKINDEX generation")
		return nil
	}

	// Use melange to generate the index
	args := []string{
		"index",
		"--output", filepath.Join(a.RepoDir, "APKINDEX.tar.gz"),
		"--arch", a.Arch,
	}

	// Add signing key if provided
	if a.SigningKey != "" {
		args = append(args, "--signing-key", a.SigningKey)
	}

	// Find all APK files in the packages directory
	apkPattern := filepath.Join(packagesDir, "*.apk")
	apkFiles, err := filepath.Glob(apkPattern)
	if err != nil {
		return fmt.Errorf("failed to find APK files: %w", err)
	}
	if len(apkFiles) == 0 {
		fmt.Println("No APK files found, skipping APKINDEX generation")
		return nil
	}

	// Add all APK files to args
	args = append(args, apkFiles...)

	cmd := exec.Command("melange", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("melange index failed: %w", err)
	}

	// Copy public key to repo directories so melange can find it when using --repository-append
	if a.SigningKey != "" {
		pubKeyPath := a.SigningKey + ".pub"

		// Read source file
		data, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read public key: %w", err)
		}

		// Write to repo root
		destPath := filepath.Join(a.RepoDir, filepath.Base(pubKeyPath))
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to copy public key to repo: %w", err)
		}
		fmt.Printf("Copied public key to: %s\n", destPath)

		// Also write to arch-specific subdirectory
		archDestPath := filepath.Join(a.RepoDir, a.Arch, filepath.Base(pubKeyPath))
		if err := os.WriteFile(archDestPath, data, 0644); err != nil {
			return fmt.Errorf("failed to copy public key to arch directory: %w", err)
		}
		fmt.Printf("Copied public key to: %s\n", archDestPath)
	}

	fmt.Println("APKINDEX generated successfully")
	return nil
}

// Exists checks if APKINDEX exists
func (a *APKIndex) Exists() bool {
	indexPath := filepath.Join(a.RepoDir, "APKINDEX.tar.gz")
	_, err := os.Stat(indexPath)
	return err == nil
}

// GetPath returns the path to APKINDEX.tar.gz
func (a *APKIndex) GetPath() string {
	return filepath.Join(a.RepoDir, "APKINDEX.tar.gz")
}

// SetArch sets the target architecture
func (a *APKIndex) SetArch(arch string) {
	a.Arch = arch
}
