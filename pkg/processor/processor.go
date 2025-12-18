package processor

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/generator"
	"github.com/greboid/dfo/pkg/packages"
	"github.com/greboid/dfo/pkg/util"
)

type ProcessResult struct {
	PackageName string
}

type WritableFS = util.WritableFS

type WalkableFS = util.WalkableFS

type StatFS = fs.StatFS

func ProcessConfig(fs util.WritableFS, configPath, outputDir string, alpineClient *packages.AlpineClient, alpineVersion, gitUser, gitPass string) (*ProcessResult, error) {
	slog.Debug("processing config",
		"config_path", configPath,
		"output_dir", outputDir,
		"alpine_version", alpineVersion)

	cfg, err := config.Load(fs, configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	slog.Debug("loaded config", "package_name", cfg.Package.Name)

	packageDir := path.Join(outputDir, cfg.Package.Name)

	gen := generator.New(cfg, packageDir, fs, alpineClient, alpineVersion, gitUser, gitPass)
	if err := gen.Generate(); err != nil {
		return nil, fmt.Errorf("generating templates: %w", err)
	}

	slog.Debug("generated templates", "package_name", cfg.Package.Name)

	return &ProcessResult{PackageName: cfg.Package.Name}, nil
}

func ProcessConfigInPlace(fs util.WritableFS, configPath string, alpineClient *packages.AlpineClient, alpineVersion, gitUser, gitPass string) (*ProcessResult, error) {
	slog.Debug("processing config in place",
		"config_path", configPath,
		"alpine_version", alpineVersion)

	cfg, err := config.Load(fs, configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	slog.Debug("loaded config", "package_name", cfg.Package.Name)

	outputDir := path.Dir(configPath)

	gen := generator.New(cfg, outputDir, fs, alpineClient, alpineVersion, gitUser, gitPass)
	if err := gen.Generate(); err != nil {
		return nil, fmt.Errorf("generating templates: %w", err)
	}

	slog.Debug("generated templates", "package_name", cfg.Package.Name)

	return &ProcessResult{PackageName: cfg.Package.Name}, nil
}
