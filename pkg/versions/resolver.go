package versions

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/csmith/latest/v2"
)

// Resolver resolves version specifications to concrete versions
type Resolver struct {
	ctx     context.Context
	gitUser string // Optional git username for private repos
	gitPass string // Optional git password/token for private repos
}

// New creates a new version resolver
func New(ctx context.Context, gitUser, gitPass string) *Resolver {
	return &Resolver{
		ctx:     ctx,
		gitUser: gitUser,
		gitPass: gitPass,
	}
}

// Resolve resolves a version entry from the versions map
// key: the key from versions map (e.g., "go", "postgres16", "https://github.com/owner/repo")
// value: "latest", "latest:MAJOR" (for postgres), or specific version string
// Returns: resolved version string
func (r *Resolver) Resolve(key, value string) (string, error) {
	spec := VersionSpec{Key: key, Value: value}

	// If not "latest" or "latest:*", return as-is (specific version)
	if !spec.IsLatest() {
		slog.Debug("using specific version", "key", key, "version", value)
		return value, nil
	}

	// Resolve "latest" based on key type
	versionType := spec.VersionType()
	slog.Debug("resolving version", "key", key, "type", versionType, "value", value)

	var resolved string
	var err error

	switch versionType {
	case "git":
		resolved, err = r.resolveGitTag(key)
	case "go":
		resolved, err = r.resolveGoVersion()
	case "postgres":
		resolved, err = r.resolvePostgresVersion(value)
	case "alpine":
		resolved, err = r.resolveAlpineVersion()
	default:
		return "", fmt.Errorf("unknown version key %q", key)
	}

	if err != nil {
		return "", err
	}

	slog.Debug("resolved version", "key", key, "value", value, "resolved", resolved)
	return resolved, nil
}

func (r *Resolver) resolveGitTag(repo string) (string, error) {
	opts := &latest.GitTagOptions{
		TagOptions: latest.TagOptions{
			IgnorePreRelease: true,
			IgnoreErrors:     true,
		},
	}

	if r.gitUser != "" || r.gitPass != "" {
		opts.Username = r.gitUser
		opts.Password = r.gitPass
	}

	tag, err := latest.GitTag(r.ctx, repo, opts)
	if err != nil {
		return "", fmt.Errorf("resolving git tag for %s: %w", repo, err)
	}

	return tag, nil
}

func (r *Resolver) resolveGoVersion() (string, error) {
	version, _, _, err := latest.GoRelease(r.ctx, nil)
	if err != nil {
		return "", fmt.Errorf("resolving Go version: %w", err)
	}
	return version, nil
}

func (r *Resolver) resolvePostgresVersion(value string) (string, error) {
	opts := &latest.TagOptions{
		TrimPrefixes: []string{"REL"},
	}
	if parts := strings.Split(value, ":"); len(parts) == 2 {
		major, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", fmt.Errorf("invalid postgres major version %q: %w", parts[1], err)
		}
		opts.MajorVersionMax = major
	}

	version, _, _, err := latest.PostgresRelease(r.ctx, opts)
	if err != nil {
		return "", fmt.Errorf("resolving Postgres version: %w", err)
	}
	version = "REL_" + strings.ReplaceAll(version, ".", "_")
	return version, nil
}

func (r *Resolver) resolveAlpineVersion() (string, error) {
	version, _, _, err := latest.AlpineRelease(r.ctx, nil)
	if err != nil {
		return "", fmt.Errorf("resolving Alpine version: %w", err)
	}
	return version, nil
}
