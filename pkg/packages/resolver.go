package packages

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/csmith/apkutils/v2"
)

type ResolvedPackage struct {
	Name    string
	Version string
}

type Resolver struct {
	client        *AlpineClient
	alpineVersion string
	repos         []string
}

func NewResolver(client *AlpineClient, alpineVersion string) *Resolver {
	return &Resolver{
		client:        client,
		alpineVersion: alpineVersion,
		repos:         []string{"main", "community"},
	}
}

func (r *Resolver) Resolve(specs []PackageSpec) ([]ResolvedPackage, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}

	slog.Debug("resolving packages",
		"alpine_version", r.alpineVersion,
		"requested_packages", names,
		"count", len(names))

	allPackages, err := r.client.GetCombinedPackages(r.alpineVersion, r.repos)
	if err != nil {
		return nil, err
	}

	// Use apkutils to flatten dependencies
	slog.Debug("flattening dependencies",
		"requested_packages", names,
		"available_packages", len(allPackages))

	flattened, err := apkutils.FlattenDependencies(allPackages, names...)
	if err != nil {
		return nil, fmt.Errorf("flattening dependencies: %w", err)
	}

	slog.Debug("dependencies flattened",
		"requested_packages", len(names),
		"total_with_deps", len(flattened))

	resolved := make([]ResolvedPackage, 0, len(flattened))
	for name, pkg := range flattened {
		resolved = append(resolved, ResolvedPackage{
			Name:    name,
			Version: pkg.Version,
		})
	}

	slices.SortFunc(resolved, func(a, b ResolvedPackage) int {
		return strings.Compare(a.Name, b.Name)
	})

	slog.Debug("package resolution complete",
		"requested", len(names),
		"resolved", len(resolved))

	return resolved, nil
}
