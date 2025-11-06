package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/tarball"
	"github.com/containers/image/v5/types"
)

// Client handles container registry operations
type Client struct {
	sysCtx *types.SystemContext
}

// NewClient creates a new registry client using Docker credentials
func NewClient() *Client {
	return &Client{
		sysCtx: &types.SystemContext{
			// Use default Docker config location (~/.docker/config.json)
			// The library will automatically load credentials from there
		},
	}
}

// Push pushes a container image tar to a registry
func (c *Client) Push(imageTarPath, registry, name, tag string) error {
	// Build full image reference
	imageRef := fmt.Sprintf("%s/%s:%s", registry, name, tag)
	fmt.Printf("Pushing %s to %s\n", name, imageRef)

	ctx := context.Background()

	// Create source reference (tarball)
	srcRef, err := tarball.Transport.ParseReference(imageTarPath)
	if err != nil {
		return fmt.Errorf("failed to create tarball reference: %w", err)
	}

	// Create destination reference (registry)
	destRef, err := docker.ParseReference(fmt.Sprintf("//%s", imageRef))
	if err != nil {
		return fmt.Errorf("failed to parse destination reference: %w", err)
	}

	// Load policy (allow all)
	policy := &signature.Policy{
		Default: signature.PolicyRequirements{
			signature.NewPRInsecureAcceptAnything(),
		},
	}
	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("failed to create policy context: %w", err)
	}
	defer policyCtx.Destroy()

	// Copy image from tar to registry
	_, err = copy.Image(ctx, policyCtx, destRef, srcRef, &copy.Options{
		SourceCtx:      c.sysCtx,
		DestinationCtx: c.sysCtx,
		ReportWriter:   os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	fmt.Printf("Successfully pushed %s\n", imageRef)
	return nil
}

// PushToMultiple pushes an image to multiple registries
func (c *Client) PushToMultiple(imageTarPath string, registries []string, name, tag string) error {
	if len(registries) == 0 {
		return fmt.Errorf("no registries configured")
	}

	for _, registry := range registries {
		if err := c.Push(imageTarPath, registry, name, tag); err != nil {
			return fmt.Errorf("failed to push to %s: %w", registry, err)
		}
	}

	return nil
}
