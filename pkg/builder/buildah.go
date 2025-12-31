package builder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	imageName := containerName
	if b.registry != "" {
		imageName = fmt.Sprintf("%s/%s:latest", b.registry, containerName)
	}

	if _, err := os.Stat(containerfilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Containerfile not found: %s", containerfilePath)
	}

	slog.Debug("Building container",
		"container", containerName,
		"image_name", imageName,
		"containerfile", containerfilePath,
		"context", contextDir,
	)

	args := []string{
		"build",
		"--layers",
		"--timestamp", "0",
		"--identity-label=false",
		"-t", imageName,
		"-f", containerfilePath,
	}

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

	args = append(args, contextDir)

	cmd := exec.CommandContext(ctx, "buildah", args...)
	cmd.Dir = contextDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("building container: %w\nOutput:\n%s", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	imageID := strings.TrimSpace(lines[len(lines)-1])

	inspectArgs := []string{"inspect", "--format", "{{.FromImageDigest}}", imageID}
	if b.storageDriver != "" {
		inspectArgs = append([]string{"--storage-driver", b.storageDriver}, inspectArgs[0:]...)
	}
	if b.storagePath != "" {
		inspectArgs = append([]string{"--root", filepath.Join(b.storagePath, "storage"), "--runroot", filepath.Join(b.storagePath, "run")}, inspectArgs[0:]...)
	}

	inspectCmd := exec.CommandContext(ctx, "buildah", inspectArgs...)
	digestOutput, err := inspectCmd.CombinedOutput()
	if err != nil {
		slog.Warn("Failed to get digest, using image ID",
			"container", containerName,
			"image_id", imageID,
			"error", err,
		)
		digest := imageID
		if !strings.HasPrefix(digest, "sha256:") {
			digest = "sha256:" + digest
		}

		return &BuildResult{
			ContainerName: containerName,
			ImageName:     imageName,
			Digest:        digest,
			FullRef:       fmt.Sprintf("%s@%s", imageName, digest),
			Size:          0,
		}, nil
	}

	digest := strings.TrimSpace(string(digestOutput))
	if digest == "" || digest == "<none>" {
		digest = imageID
	}

	if !strings.HasPrefix(digest, "sha256:") {
		digest = "sha256:" + digest
	}

	fullRef := fmt.Sprintf("%s@%s", imageName, digest)

	result := &BuildResult{
		ContainerName: containerName,
		ImageName:     imageName,
		Digest:        digest,
		FullRef:       fullRef,
		Size:          0,
	}

	slog.Debug("Container build result",
		"container", containerName,
		"digest", digest[:min(16, len(digest))],
	)

	return result, nil
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
