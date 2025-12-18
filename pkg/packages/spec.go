package packages

import (
	"fmt"
	"strings"
)

type PackageSpec struct {
	Name    string
	Version string
}

func ParsePackageSpec(spec string) (PackageSpec, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return PackageSpec{}, fmt.Errorf("empty package specification")
	}

	if idx := strings.Index(spec, "="); idx > 0 {
		return PackageSpec{}, fmt.Errorf("package versions cannot be provided")
	}

	return PackageSpec{
		Name: spec,
	}, nil
}

func ParsePackageSpecs(specs []string) ([]PackageSpec, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	result := make([]PackageSpec, 0, len(specs))
	for i, spec := range specs {
		parsed, err := ParsePackageSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("parsing package spec at index %d: %w", i, err)
		}
		result = append(result, parsed)
	}
	return result, nil
}
