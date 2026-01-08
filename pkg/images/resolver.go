package images

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Resolver struct {
	registry       string
	checkLocal     bool
	defaultOptions []remote.Option
	cache          map[string]*ResolvedImage
	cacheMu        sync.RWMutex
}

type ResolvedImage struct {
	Registry string
	Name     string
	Digest   string
	FullRef  string
}

func NewResolver(registry string, checkLocal bool) *Resolver {
	return &Resolver{
		registry:   registry,
		checkLocal: checkLocal,
		defaultOptions: []remote.Option{
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
		},
		cache: make(map[string]*ResolvedImage),
	}
}

func (r *Resolver) GetRegistry() string {
	return r.registry
}

func (r *Resolver) Resolve(ctx context.Context, imageName string) (*ResolvedImage, error) {
	ref, err := r.parseImageReference(imageName)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference %q: %w", imageName, err)
	}

	cacheKey := ref.String()

	r.cacheMu.RLock()
	if cached, ok := r.cache[cacheKey]; ok {
		r.cacheMu.RUnlock()
		slog.Debug("resolved image from cache", "image", imageName, "digest", cached.Digest)
		return cached, nil
	}
	r.cacheMu.RUnlock()

	if r.checkLocal {
		if resolved, err := r.resolveFromLocal(ctx, ref); err == nil {
			slog.Debug("resolved image from local daemon", "image", imageName, "digest", resolved.Digest)
			return resolved, nil
		} else {
			slog.Debug("image not found locally, checking registry", "image", imageName, "error", err)
		}
	}

	resolved, err := r.resolveFromRegistry(ctx, ref)
	if err != nil {
		return nil, err
	}

	r.cacheMu.Lock()
	r.cache[cacheKey] = resolved
	r.cacheMu.Unlock()

	return resolved, nil
}

func (r *Resolver) parseImageReference(imageName string) (name.Reference, error) {
	if r.registry != "" && !strings.Contains(imageName, "/") {
		imageName = r.registry + "/" + imageName
	}

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err
	}

	return ref, nil
}

func (r *Resolver) resolveFromLocal(ctx context.Context, ref name.Reference) (*ResolvedImage, error) {
	img, err := daemon.Image(ref)
	if err != nil {
		return nil, err
	}

	digest, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("getting image digest: %w", err)
	}

	return &ResolvedImage{
		Registry: ref.Context().RegistryStr(),
		Name:     ref.Context().RepositoryStr(),
		Digest:   digest.String(),
		FullRef:  fmt.Sprintf("%s@%s", ref.Context().Name(), digest.String()),
	}, nil
}

func (r *Resolver) resolveFromRegistry(ctx context.Context, ref name.Reference) (*ResolvedImage, error) {
	desc, err := remote.Get(ref, r.defaultOptions...)
	if err != nil {
		return nil, fmt.Errorf("fetching image from registry: %w", err)
	}

	slog.Debug("resolved image from registry", "image", ref.String(), "digest", desc.Digest.String())

	return &ResolvedImage{
		Registry: ref.Context().RegistryStr(),
		Name:     ref.Context().RepositoryStr(),
		Digest:   desc.Digest.String(),
		FullRef:  fmt.Sprintf("%s@%s", ref.Context().Name(), desc.Digest.String()),
	}, nil
}
