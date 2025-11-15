package processor

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/generator"
	"github.com/greboid/dfo/pkg/util"
)

type ProcessResult struct {
	PackageName string
}

type WritableFS = util.WritableFS

type WalkableFS = util.WalkableFS

type StatFS = fs.StatFS

func ProcessConfig(fs util.WritableFS, configPath, outputDir string) (*ProcessResult, error) {
	cfg, err := config.Load(fs, configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	packageDir := path.Join(outputDir, cfg.Package.Name)

	gen := generator.New(cfg, packageDir, fs)
	if err := gen.Generate(); err != nil {
		return nil, fmt.Errorf("generating templates: %w", err)
	}

	return &ProcessResult{PackageName: cfg.Package.Name}, nil
}

func ProcessConfigInPlace(fs util.WritableFS, configPath string) (*ProcessResult, error) {
	cfg, err := config.Load(fs, configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	outputDir := path.Dir(configPath)

	gen := generator.New(cfg, outputDir, fs)
	if err := gen.Generate(); err != nil {
		return nil, fmt.Errorf("generating templates: %w", err)
	}

	return &ProcessResult{PackageName: cfg.Package.Name}, nil
}
