package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResolveAbsolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	return filepath.Join(cwd, path), nil
}
