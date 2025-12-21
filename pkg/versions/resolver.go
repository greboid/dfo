package versions

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/csmith/latest/v2"
)

type Resolver struct {
	ctx     context.Context
	gitUser string
	gitPass string
}

func New(ctx context.Context, gitUser, gitPass string) *Resolver {
	return &Resolver{
		ctx:     ctx,
		gitUser: gitUser,
		gitPass: gitPass,
	}
}

func (r *Resolver) Resolve(key, value string) (VersionMetadata, error) {
	spec := VersionSpec{Key: key, Value: value}

	if !spec.IsLatest() {
		slog.Debug("using specific version", "key", key, "version", value)
		return VersionMetadata{Version: value}, nil
	}

	versionType := spec.VersionType()
	slog.Debug("resolving version", "key", key, "type", versionType, "value", value)

	var metadata VersionMetadata
	var err error

	switch versionType {
	case "git":
		version, gitErr := r.resolveGitTag(key)
		if gitErr != nil {
			err = gitErr
		} else {
			metadata = VersionMetadata{Version: version}
		}
	case "go":
		metadata, err = r.resolveGoVersion()
	case "postgres":
		metadata, err = r.resolvePostgresVersion(value)
	case "alpine":
		metadata, err = r.resolveAlpineVersion()
	default:
		return VersionMetadata{}, fmt.Errorf("unknown version key %q", key)
	}

	if err != nil {
		return VersionMetadata{}, err
	}

	slog.Debug("resolved version", "key", key, "value", value, "resolved", metadata.Version, "url", metadata.URL, "checksum", metadata.Checksum)
	return metadata, nil
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

func (r *Resolver) resolveGoVersion() (VersionMetadata, error) {
	version, url, checksum, err := latest.GoRelease(r.ctx, nil)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("resolving Go version: %w", err)
	}
	return VersionMetadata{
		Version:  version,
		URL:      url,
		Checksum: checksum,
	}, nil
}

func (r *Resolver) resolvePostgresVersion(value string) (VersionMetadata, error) {
	opts := &latest.TagOptions{
		TrimPrefixes: []string{"REL"},
	}
	if parts := strings.Split(value, ":"); len(parts) == 2 {
		major, err := strconv.Atoi(parts[1])
		if err != nil {
			return VersionMetadata{}, fmt.Errorf("invalid postgres major version %q: %w", parts[1], err)
		}
		opts.MajorVersionMax = major
	}

	version, url, checksum, err := latest.PostgresRelease(r.ctx, opts)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("resolving Postgres version: %w", err)
	}
	version = "REL_" + strings.ReplaceAll(version, ".", "_")
	return VersionMetadata{
		Version:  version,
		URL:      url,
		Checksum: checksum,
	}, nil
}

func (r *Resolver) resolveAlpineVersion() (VersionMetadata, error) {
	opts := &latest.AlpineReleaseOptions{
		Flavour: "minirootfs",
	}
	version, url, checksum, err := latest.AlpineRelease(r.ctx, opts)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("resolving Alpine version: %w", err)
	}
	return VersionMetadata{
		Version:  version,
		URL:      url,
		Checksum: checksum,
	}, nil
}
