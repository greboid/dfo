package apko

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Builder handles building apko containers
type Builder struct {
	WorkDir    string
	OutputDir  string
	RepoDir    string
	SigningKey string
	Arch       string
}

// NewBuilder creates a new container builder
func NewBuilder(workDir, outputDir, repoDir, signingKey string) *Builder {
	return &Builder{
		WorkDir:    workDir,
		OutputDir:  outputDir,
		RepoDir:    repoDir,
		SigningKey: signingKey,
		Arch:       "x86_64",
	}
}

// ensureLockFile generates a lock file if it doesn't exist
func (b *Builder) ensureLockFile(spec *Spec) (string, error) {
	lockFile := spec.FilePath + ".lock"

	// Check if lock file already exists
	if _, err := os.Stat(lockFile); err == nil {
		fmt.Printf("Using existing lock file: %s\n", lockFile)
		return lockFile, nil
	}

	// Generate lock file
	fmt.Printf("Generating lock file: %s\n", lockFile)
	args := []string{
		"lock",
		spec.FilePath,
		"--arch", b.Arch,
		"--output", lockFile,
	}

	// Add keyring for signature verification
	pubKeyPath := b.SigningKey + ".pub"
	args = append(args, "--keyring-append", pubKeyPath)

	// Add local repository if it exists
	if b.RepoDir != "" {
		repoPath := filepath.Join(b.RepoDir, "APKINDEX.tar.gz")
		if _, err := os.Stat(repoPath); err == nil {
			args = append(args, "--repository-append", b.RepoDir)
		}
	}

	cmd := exec.Command("apko", args...)
	cmd.Dir = b.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("apko lock failed: %w", err)
	}

	fmt.Printf("Lock file generated: %s\n", lockFile)
	return lockFile, nil
}

// Build builds an apko container and generates a lock file
func (b *Builder) Build(spec *Spec, tag string) error {
	fmt.Printf("Building container: %s (tag: %s)\n", spec.GetName(), tag)

	// Ensure output directory exists
	if err := os.MkdirAll(b.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Ensure lock file exists
	lockFile, err := b.ensureLockFile(spec)
	if err != nil {
		return err
	}

	// Output image path
	outputImage := filepath.Join(b.OutputDir, spec.GetName()+".tar")

	// Prepare apko command
	args := []string{
		"build",
		spec.FilePath,
		tag,
		outputImage,
		"--arch", b.Arch,
		"--lockfile", lockFile,
	}

	// Add keyring for signature verification
	pubKeyPath := b.SigningKey + ".pub"
	args = append(args, "--keyring-append", pubKeyPath)

	// Add local repository if it exists
	if b.RepoDir != "" {
		repoPath := filepath.Join(b.RepoDir, "APKINDEX.tar.gz")
		if _, err := os.Stat(repoPath); err == nil {
			args = append(args, "--repository-append", b.RepoDir)
		}
	}

	// Execute apko
	cmd := exec.Command("apko", args...)
	cmd.Dir = b.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apko build failed: %w", err)
	}

	fmt.Printf("Successfully built container: %s\n", spec.GetName())
	return nil
}

// Publish builds and publishes a container directly to a registry
func (b *Builder) Publish(spec *Spec, tag string) error {
	fmt.Printf("Building and publishing container: %s (tag: %s)\n", spec.GetName(), tag)

	// Ensure lock file exists
	lockFile, err := b.ensureLockFile(spec)
	if err != nil {
		return err
	}

	// Prepare apko command for direct publish
	args := []string{
		"publish",
		spec.FilePath,
		tag,
		"--arch", b.Arch,
		"--lockfile", lockFile,
	}

	// Add keyring for signature verification
	pubKeyPath := b.SigningKey + ".pub"
	args = append(args, "--keyring-append", pubKeyPath)

	// Add local repository if it exists
	if b.RepoDir != "" {
		repoPath := filepath.Join(b.RepoDir, "APKINDEX.tar.gz")
		if _, err := os.Stat(repoPath); err == nil {
			args = append(args, "--repository-append", b.RepoDir)
		}
	}

	// Execute apko publish
	cmd := exec.Command("apko", args...)
	cmd.Dir = b.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apko publish failed: %w", err)
	}

	fmt.Printf("Successfully published container: %s\n", spec.GetName())
	return nil
}

// GetImagePath returns the path to a built container image
func (b *Builder) GetImagePath(spec *Spec) string {
	return filepath.Join(b.OutputDir, spec.GetName()+".tar")
}

// SetArch sets the target architecture
func (b *Builder) SetArch(arch string) {
	b.Arch = arch
}
