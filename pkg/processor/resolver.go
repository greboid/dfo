package processor

import (
	"fmt"
	"io/fs"
	"path"
)

const DefaultConfigFilename = "dfo.yaml"

func ResolveConfigPath(fs fs.StatFS, input string) (string, error) {
	if input == "" {
		return DefaultConfigFilename, nil
	}

	info, err := fs.Stat(input)
	if err != nil {
		return "", fmt.Errorf("accessing path: %w", err)
	}

	if info.IsDir() {
		return path.Join(input, DefaultConfigFilename), nil
	}

	return input, nil
}
