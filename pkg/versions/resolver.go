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
	ctx             context.Context
	gitUser         string
	gitPass         string
	gitTagClient    func(ctx context.Context, repo string, options *latest.GitTagOptions) (string, error)
	goReleaseClient func(ctx context.Context, options *latest.GoOptions) (latestVersion string, downloadUrl string, downloadChecksum string, err error)
	postgresClient  func(ctx context.Context, options *latest.TagOptions) (latest string, url string, checksum string, err error)
	alpineClient    func(ctx context.Context, options *latest.AlpineReleaseOptions) (latestVersion string, downloadUrl string, downloadChecksum string, err error)
}

func New(ctx context.Context, gitUser, gitPass string) *Resolver {
	return NewWithClients(ctx, gitUser, gitPass, latest.GitTag, latest.GoRelease, latest.PostgresRelease, latest.AlpineRelease)
}

func NewWithClients(ctx context.Context, gitUser, gitPass string,
	gitClient func(ctx context.Context, repo string, options *latest.GitTagOptions) (string, error),
	goClient func(ctx context.Context, options *latest.GoOptions) (latestVersion string, downloadUrl string, downloadChecksum string, err error),
	postgresClient func(ctx context.Context, options *latest.TagOptions) (latest string, url string, checksum string, err error),
	alpineClient func(ctx context.Context, options *latest.AlpineReleaseOptions) (latestVersion string, downloadUrl string, downloadChecksum string, err error)) *Resolver {
	return &Resolver{
		ctx:             ctx,
		gitUser:         gitUser,
		gitPass:         gitPass,
		gitTagClient:    gitClient,
		goReleaseClient: goClient,
		postgresClient:  postgresClient,
		alpineClient:    alpineClient,
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

	metadata, err := r.resolveByVersionType(key, value, versionType)
	if err != nil {
		return VersionMetadata{}, err
	}

	slog.Debug("resolved version", "key", key, "value", value, "resolved", metadata.Version, "url", metadata.URL, "checksum", metadata.Checksum)
	return metadata, nil
}

func (r *Resolver) resolveByVersionType(key, value, versionType string) (VersionMetadata, error) {
	switch versionType {
	case "git":
		version, err := r.resolveGitTag(key)
		if err != nil {
			return VersionMetadata{}, err
		}
		return VersionMetadata{Version: version}, nil
	case "go":
		return r.resolveGoVersion()
	case "postgres":
		return r.resolvePostgresVersion(value)
	case "alpine":
		return r.resolveAlpineVersion()
	default:
		return VersionMetadata{}, fmt.Errorf("unknown version key %q", key)
	}
}

func (r *Resolver) buildGitTagOptions() *latest.GitTagOptions {
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

	return opts
}

func (r *Resolver) resolveGitTag(repo string) (string, error) {
	opts := r.buildGitTagOptions()
	tag, err := r.gitTagClient(r.ctx, repo, opts)
	if err != nil {
		return "", fmt.Errorf("resolving git tag for %s: %w", repo, err)
	}
	return tag, nil
}

func (r *Resolver) resolveGoVersion() (VersionMetadata, error) {
	version, url, checksum, err := r.goReleaseClient(r.ctx, nil)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("resolving Go version: %w", err)
	}
	return VersionMetadata{
		Version:  version,
		URL:      url,
		Checksum: checksum,
	}, nil
}

func parsePostgresMajorVersion(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, nil
	}
	major, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid postgres major version %q: %w", parts[1], err)
	}
	return major, nil
}

func buildPostgresTagOptions(value string) (*latest.TagOptions, error) {
	opts := &latest.TagOptions{
		TrimPrefixes: []string{"REL"},
	}

	major, err := parsePostgresMajorVersion(value)
	if err != nil {
		return nil, err
	}

	if major > 0 {
		opts.MajorVersionMax = major
	}

	return opts, nil
}

func formatPostgresVersion(version string) string {
	return "REL_" + strings.ReplaceAll(version, ".", "_")
}

func (r *Resolver) resolvePostgresVersion(value string) (VersionMetadata, error) {
	opts, err := buildPostgresTagOptions(value)
	if err != nil {
		return VersionMetadata{}, err
	}

	version, url, checksum, err := r.postgresClient(r.ctx, opts)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("resolving Postgres version: %w", err)
	}

	formattedVersion := formatPostgresVersion(version)
	return VersionMetadata{
		Version:  formattedVersion,
		URL:      url,
		Checksum: checksum,
	}, nil
}

func (r *Resolver) resolveAlpineVersion() (VersionMetadata, error) {
	opts := &latest.AlpineReleaseOptions{
		Flavour: "minirootfs",
	}
	version, url, checksum, err := r.alpineClient(r.ctx, opts)
	if err != nil {
		return VersionMetadata{}, fmt.Errorf("resolving Alpine version: %w", err)
	}
	return VersionMetadata{
		Version:  version,
		URL:      url,
		Checksum: checksum,
	}, nil
}
