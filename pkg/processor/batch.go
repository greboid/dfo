package processor

import (
	"fmt"
	iofs "io/fs"
	"sync"

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

	configFiles, err := FindConfigFiles(fs, dir)
	if err != nil {
		return nil, err
	}

	const maxConcurrency = 5
	result := &BatchResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)

	for _, configPath := range configFiles {
		wg.Go(func() {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := processor(configPath); err != nil {
				mu.Lock()
				result.Errors++
				result.ErrorDetails = append(result.ErrorDetails, ProcessingError{
					Path: configPath,
					Err:  err,
				})
				mu.Unlock()
			} else {
				mu.Lock()
				result.Processed++
				mu.Unlock()
			}
		})
	}

	wg.Wait()

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
