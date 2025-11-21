package processor

import (
	"fmt"
	iofs "io/fs"

	"github.com/greboid/dfo/pkg/util"
)

type BatchResult struct {
	Processed    int
	Errors       int
	ErrorDetails []ProcessingError
}

type ProcessingError struct {
	Path string
	Err  error
}

func (e ProcessingError) Error() string {
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}

type FileProcessor func(configPath string) error

func WalkAndProcess(fs util.WalkableFS, dir string, processor FileProcessor) (*BatchResult, error) {
	info, err := fs.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("accessing directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	result := &BatchResult{}

	_ = fs.WalkDir(dir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			result.Errors++
			result.ErrorDetails = append(result.ErrorDetails, ProcessingError{
				Path: path,
				Err:  err,
			})
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if d.Name() != DefaultConfigFilename {
			return nil
		}

		if err = processor(path); err != nil {
			result.Errors++
			result.ErrorDetails = append(result.ErrorDetails, ProcessingError{
				Path: path,
				Err:  err,
			})
			return nil
		}

		result.Processed++
		return nil
	})

	return result, nil
}

func FindConfigFiles(fs util.WalkableFS, dir string) ([]string, error) {
	info, err := fs.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("accessing directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	var files []string

	_ = fs.WalkDir(dir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if d.Name() == DefaultConfigFilename {
			files = append(files, path)
		}

		return nil
	})

	return files, nil
}
