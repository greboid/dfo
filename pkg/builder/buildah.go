package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/greboid/dfo/pkg/util"
)

type BuildahBuilder struct {
	registry      string
	storagePath   string
	storageDriver string
	isolation     string
}

func NewBuildahBuilder(registry, storagePath, storageDriver, isolation string) *BuildahBuilder {
	return &BuildahBuilder{
		registry:      registry,
		storagePath:   storagePath,
		storageDriver: storageDriver,
		isolation:     isolation,
	}
}

func (b *BuildahBuilder) Initialize(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "buildah", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buildah not found or not executable: %w\nOutput: %s", err, string(output))
	}

	slog.Info("Buildah CLI initialized", "version", strings.TrimSpace(strings.Split(string(output), "\n")[0]))
	return nil
}

func (b *BuildahBuilder) BuildContainer(ctx context.Context, containerName, containerfilePath, contextDir string) (*BuildResult, error) {
	imageName := b.buildImageName(containerName)

	if err := b.validateContainerfilePath(containerfilePath, contextDir); err != nil {
		return nil, err
	}

	slog.Debug("Building container",
		"container", containerName,
		"image_name", imageName,
		"containerfile", containerfilePath,
		"context", contextDir,
	)

	imageID, err := b.executeBuild(ctx, imageName, containerfilePath, contextDir)
	if err != nil {
		return nil, err
	}

	digest, err := b.getImageDigest(ctx, imageID, containerName)
	if err != nil {
		slog.Warn("Failed to get digest, using image ID",
			"container", containerName,
			"image_id", imageID,
			"error", err,
		)
		digest = util.NormalizeDigest(imageID)
	}

	return b.createBuildResult(containerName, imageName, digest), nil
}

func (b *BuildahBuilder) buildImageName(containerName string) string {
	if b.registry != "" {
		return fmt.Sprintf("%s/%s:latest", b.registry, containerName)
	}
	return containerName
}

func (b *BuildahBuilder) validateContainerfilePath(containerfilePath, contextDir string) error {
	fullPath := filepath.Join(contextDir, containerfilePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("containerfile not found: %s", fullPath)
	}
	return nil
}

func (b *BuildahBuilder) executeBuild(ctx context.Context, imageName, containerfilePath, contextDir string) (string, error) {
	args := b.buildBuildArgs(imageName, containerfilePath, contextDir)

	cmd := exec.CommandContext(ctx, "buildah", args...)
	cmd.Dir = contextDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("building container: %w\nOutput:\n%s", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

func (b *BuildahBuilder) buildBuildArgs(imageName, containerfilePath, contextDir string) []string {
	args := []string{
		"build",
		"--layers",
		"--timestamp", "0",
		"--identity-label=false",
		"-t", imageName,
		"-f", containerfilePath,
	}

	args = append(args, b.buildStorageArgs()...)
	args = append(args, ".")

	return args
}

func (b *BuildahBuilder) buildStorageArgs() []string {
	var args []string

	if b.isolation != "" {
		args = append(args, "--isolation", b.isolation)
	}
	if b.storageDriver != "" {
		args = append(args, "--storage-driver", b.storageDriver)
	}
	if b.storagePath != "" {
		args = append(args, "--root", filepath.Join(b.storagePath, "storage"))
		args = append(args, "--runroot", filepath.Join(b.storagePath, "run"))
	}

	return args
}

func (b *BuildahBuilder) getImageDigest(ctx context.Context, imageID, containerName string) (string, error) {
	inspectArgs := b.buildInspectArgs(imageID)

	inspectCmd := exec.CommandContext(ctx, "buildah", inspectArgs...)
	digestOutput, err := inspectCmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	digest := strings.TrimSpace(string(digestOutput))
	if digest == "" || digest == "<none>" {
		return imageID, nil
	}

	return util.NormalizeDigest(digest), nil
}

func (b *BuildahBuilder) buildInspectArgs(imageID string) []string {
	args := []string{"inspect", "--format", "{{.FromImageDigest}}", imageID}
	return append(b.buildStorageArgs(), args...)
}

func (b *BuildahBuilder) createBuildResult(containerName, imageName, digest string) *BuildResult {
	result := &BuildResult{
		ContainerName: containerName,
		ImageName:     imageName,
		Digest:        digest,
		FullRef:       util.FormatFullRef(imageName, digest),
		Size:          0,
	}

	slog.Debug("Container build result",
		"container", containerName,
		"digest", digest[:min(16, len(digest))],
	)

	return result
}

func (b *BuildahBuilder) PushImage(ctx context.Context, imageName string) error {
	slog.Debug("Pushing image to registry", "image", imageName)

	args := []string{"push"}

	if b.storageDriver != "" {
		args = append(args, "--storage-driver", b.storageDriver)
	}
	if b.storagePath != "" {
		args = append(args, "--root", filepath.Join(b.storagePath, "storage"))
		args = append(args, "--runroot", filepath.Join(b.storagePath, "run"))
	}

	args = append(args, imageName)

	cmd := exec.CommandContext(ctx, "buildah", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pushing image %s: %w\nOutput:\n%s", imageName, err, string(output))
	}

	slog.Debug("Image pushed successfully", "image", imageName)
	return nil
}

func (b *BuildahBuilder) Close() error {
	slog.Debug("Buildah CLI closed")
	return nil
}
