package update

import (
	"context"
	"fmt"
	"regexp"

	"github.com/csmith/latest"
	"github.com/greboid/dfo/internal/melange"
)

// VersionInfo represents version update information
type VersionInfo struct {
	PackageName    string
	CurrentVersion string
	LatestVersion  string
	NeedsUpdate    bool
}

// Checker handles checking for package updates
type Checker struct{}

// NewChecker creates a new update checker
func NewChecker() *Checker {
	return &Checker{}
}

// CheckPackage checks if a package has updates available
func (c *Checker) CheckPackage(spec *melange.Spec) (*VersionInfo, error) {
	info := &VersionInfo{
		PackageName:    spec.GetName(),
		CurrentVersion: spec.Package.Version,
		NeedsUpdate:    false,
	}

	// If updates are not enabled, return current version
	if !spec.Update.Enabled {
		info.LatestVersion = spec.Package.Version
		return info, nil
	}

	// Determine update source and check for latest version
	latestVersion, err := c.getLatestVersion(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to check latest version for %s: %w", spec.GetName(), err)
	}

	// Apply version transformations if configured
	transformedVersion, err := c.transformVersion(spec, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to transform version for %s: %w", spec.GetName(), err)
	}

	info.LatestVersion = transformedVersion
	info.NeedsUpdate = (transformedVersion != spec.Package.Version)

	return info, nil
}

// getLatestVersion determines the latest version based on update config
func (c *Checker) getLatestVersion(spec *melange.Spec) (string, error) {
	// Special case: detect "go" package by name and use Go release checker
	if spec.GetName() == "go" && spec.Update.Git != nil {
		return c.checkGoVersion(spec)
	}

	// Check PostgreSQL
	if spec.Update.Postgres != nil {
		return c.checkPostgresVersion(spec)
	}

	// Check Go (when explicitly configured)
	if spec.Update.Go != nil {
		return c.checkGoVersion(spec)
	}

	// Check Git source
	if spec.Update.Git != nil && len(spec.Update.Git) > 0 {
		return c.checkGitVersion(spec)
	}

	// Check GitHub monitor
	if spec.Update.GitHubMonitor != nil && len(spec.Update.GitHubMonitor) > 0 {
		return c.checkGitHubVersion(spec)
	}

	// Check Release monitor
	if spec.Update.ReleaseMonitor != nil && len(spec.Update.ReleaseMonitor) > 0 {
		return c.checkReleaseMonitor(spec)
	}

	// No update source configured
	return spec.Package.Version, nil
}

// checkGitVersion checks for updates from a Git repository
func (c *Checker) checkGitVersion(spec *melange.Spec) (string, error) {
	// First, try to extract repository URL from the first git-checkout step in the pipeline
	repoURL := c.extractGitURLFromPipeline(spec)

	// Fall back to the update config if not found in pipeline
	if repoURL == "" {
		var ok bool
		repoURL, ok = spec.Update.Git["url"]
		if !ok || repoURL == "" {
			// Check if this is a PostgreSQL package with tag filtering
			if tagFilterPrefix := spec.Update.Git["tag-filter-prefix"]; tagFilterPrefix != "" {
				if len(tagFilterPrefix) >= 5 && tagFilterPrefix[:4] == "REL_" {
					// This is PostgreSQL - use GitHub mirror
					repoURL = "https://github.com/postgres/postgres"
				}
			}

			if repoURL == "" {
				return "", fmt.Errorf("git update config missing 'url' and no git-checkout step found in pipeline")
			}
		}
	}

	// Get strip prefix or tag filter prefix if configured
	stripPrefix := spec.Update.Git["strip-prefix"]
	if stripPrefix == "" {
		// Use tag-filter-prefix as strip-prefix
		tagFilterPrefix := spec.Update.Git["tag-filter-prefix"]
		if tagFilterPrefix != "" {
			stripPrefix = tagFilterPrefix
		}
	}

	// Use csmith/latest to check git tags
	ctx := context.Background()
	options := &latest.TagOptions{
		IgnoreErrors: true, // Ignore tags that don't follow semver format
	}
	if stripPrefix != "" {
		options.TrimPrefixes = []string{stripPrefix}
	}

	version, err := latest.GitTag(ctx, repoURL, options)
	if err != nil {
		return "", fmt.Errorf("failed to find git version: %w", err)
	}

	return version, nil
}

// extractGitURLFromPipeline finds the repository URL from the first git-checkout step
func (c *Checker) extractGitURLFromPipeline(spec *melange.Spec) string {
	for _, step := range spec.Pipeline {
		// Check if this step uses git-checkout
		if uses, ok := step["uses"].(string); ok && uses == "git-checkout" {
			// Extract the repository URL from the with section
			withValue, hasWith := step["with"]
			if !hasWith {
				continue
			}

			// Handle melange.PipelineStep type
			if withStep, ok := withValue.(melange.PipelineStep); ok {
				if repo, ok := withStep["repository"].(string); ok && repo != "" {
					return repo
				}
			}

			// Try map[string]interface{} (in case it's parsed differently)
			if with, ok := withValue.(map[string]interface{}); ok {
				if repo, ok := with["repository"].(string); ok && repo != "" {
					return repo
				}
			}

			// Try map[interface{}]interface{} (common with YAML parsers)
			if with, ok := withValue.(map[interface{}]interface{}); ok {
				// Check for "repository" key
				if repo, ok := with["repository"].(string); ok && repo != "" {
					return repo
				}
				// Try with interface{} key
				for key, val := range with {
					if keyStr, ok := key.(string); ok && keyStr == "repository" {
						if repo, ok := val.(string); ok && repo != "" {
							return repo
						}
					}
				}
			}
		}
	}
	return ""
}

// checkGitHubVersion checks for updates from GitHub releases
func (c *Checker) checkGitHubVersion(spec *melange.Spec) (string, error) {
	// Extract identifier from github config
	identifier, ok := spec.Update.GitHubMonitor["identifier"]
	if !ok {
		return "", fmt.Errorf("github update config missing 'identifier'")
	}

	// Get strip prefix if configured
	stripPrefix := spec.Update.GitHubMonitor["strip-prefix"]

	// Use csmith/latest to check GitHub tags
	ctx := context.Background()
	repoURL := fmt.Sprintf("https://github.com/%s", identifier)
	options := &latest.TagOptions{
		IgnoreErrors: true, // Ignore tags that don't follow semver format
	}
	if stripPrefix != "" {
		options.TrimPrefixes = []string{stripPrefix}
	}

	version, err := latest.GitTag(ctx, repoURL, options)
	if err != nil {
		return "", fmt.Errorf("failed to find github version: %w", err)
	}

	return version, nil
}

// checkReleaseMonitor checks for updates from release-monitoring.org
func (c *Checker) checkReleaseMonitor(spec *melange.Spec) (string, error) {
	// For now, we'll just return the current version
	// TODO: Implement release-monitoring.org support when needed
	return spec.Package.Version, nil
}

// checkPostgresVersion checks for the latest PostgreSQL release
func (c *Checker) checkPostgresVersion(spec *melange.Spec) (string, error) {
	ctx := context.Background()

	version, _, _, err := latest.PostgresRelease(ctx, &latest.TagOptions{
		IgnoreErrors: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get postgres version: %w", err)
	}

	return version, nil
}

// checkGoVersion checks for the latest Go release
func (c *Checker) checkGoVersion(spec *melange.Spec) (string, error) {
	ctx := context.Background()

	version, _, _, err := latest.GoRelease(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get go version: %w", err)
	}

	return version, nil
}

// transformVersion applies version transformations based on the update config
func (c *Checker) transformVersion(spec *melange.Spec, version string) (string, error) {
	// If no transformations configured, return version as-is
	if len(spec.Update.VersionTransform) == 0 {
		return version, nil
	}

	transformedVersion := version

	// Apply each transformation in order
	for _, transform := range spec.Update.VersionTransform {
		matcher, err := regexp.Compile(transform.Match)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern '%s': %w", transform.Match, err)
		}

		transformedVersion = matcher.ReplaceAllString(transformedVersion, transform.Replace)
	}

	return transformedVersion, nil
}

// CheckAllPackages checks all packages for updates
func (c *Checker) CheckAllPackages(specs map[string]*melange.Spec) (map[string]*VersionInfo, error) {
	results := make(map[string]*VersionInfo)

	for name, spec := range specs {
		info, err := c.CheckPackage(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to check package %s: %w", name, err)
		}
		results[name] = info
	}

	return results, nil
}
