package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/greboid/dfo/pkg/builder"
	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/graph"
	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
)

type BuildConfig struct {
	Directory     string
	AlpineVersion string
	GitUser       string
	GitPass       string
	Registry      string
	StoragePath   string
	StorageDriver string
	Isolation     string
	Concurrency   int
	ForceRebuild  bool
	Push          bool
}

type GraphResult struct {
	Graph   *graph.Graph
	Layers  [][]string
	Configs map[string]*config.BuildConfig
	Paths   map[string]string
}

func loadConfigsAndBuildGraph(cfg *BuildConfig) (*GraphResult, error) {
	fmt.Printf("Searching for dfo.yaml files in %s...\n", cfg.Directory)

	fs := util.DefaultFS()

	absDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return nil, fmt.Errorf("resolving directory path: %w", err)
	}

	configFiles, err := processor.FindConfigFiles(fs, absDir)
	if err != nil {
		return nil, fmt.Errorf("finding config files: %w", err)
	}

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no dfo.yaml files found in %s", absDir)
	}

	fmt.Printf("Found %d dfo.yaml file(s)\n", len(configFiles))

	return loadConfigFilesAndBuildGraph(fs, configFiles)
}

func loadSingleConfigAndBuildGraph(configPath string) (*GraphResult, error) {
	fs := util.DefaultFS()

	fmt.Printf("Loading config from %s...\n", configPath)

	return loadConfigFilesAndBuildGraph(fs, []string{configPath})
}

func loadConfigFilesAndBuildGraph(fs util.WritableFS, configFiles []string) (*GraphResult, error) {
	configs := make(map[string]*config.BuildConfig)
	containerPaths := make(map[string]string)

	for _, configPath := range configFiles {
		cfgFile, err := config.Load(fs, configPath)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", configPath, err)
		}

		containerName := filepath.Base(filepath.Dir(configPath))
		configs[containerName] = cfgFile
		containerPaths[containerName] = configPath
	}

	fmt.Println("Building dependency graph...")
	depGraph, err := graph.Build(configs, containerPaths)
	if err != nil {
		return nil, fmt.Errorf("building dependency graph: %w", err)
	}

	fmt.Printf("Graph contains %d container(s)\n", len(depGraph.Containers))

	fmt.Println("Resolving build order...")
	layers, err := depGraph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("resolving dependencies: %w", err)
	}

	fmt.Printf("Resolved into %d layer(s):\n", len(layers))
	for i, layer := range layers {
		fmt.Printf("  Layer %d: %v\n", i, layer)
	}

	return &GraphResult{
		Graph:   depGraph,
		Layers:  layers,
		Configs: configs,
		Paths:   containerPaths,
	}, nil
}

func buildContainers(cfg *BuildConfig, graphResult *GraphResult) error {
	fs := util.DefaultFS()

	resolvedVersion := cfg.AlpineVersion
	if resolvedVersion == "" {
		latest, err := alpineClient.GetLatestStableVersion()
		if err != nil {
			return fmt.Errorf("failed to detect latest Alpine version: %w", err)
		}
		resolvedVersion = latest
		fmt.Printf("Auto-detected Alpine version: %s\n", resolvedVersion)
	}

	fmt.Println("\nBuilding containers with buildah...")

	buildConfig := builder.OrchestratorConfig{
		AlpineVersion: resolvedVersion,
		GitUser:       cfg.GitUser,
		GitPass:       cfg.GitPass,
		Registry:      cfg.Registry,
		OutputDir:     cfg.Directory,
		Concurrency:   cfg.Concurrency,
		AlpineClient:  alpineClient,
		ForceRebuild:  cfg.ForceRebuild,
		Push:          cfg.Push,
	}

	buildahBuilder := builder.NewBuildahBuilder(cfg.Registry, cfg.StoragePath, cfg.StorageDriver, cfg.Isolation)

	orch, err := builder.NewOrchestrator(
		buildahBuilder,
		graphResult.Graph,
		fs,
		buildConfig,
	)
	if err != nil {
		return fmt.Errorf("creating orchestrator: %w", err)
	}

	defer func() {
		if closeErr := orch.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close builder: %v\n", closeErr)
		}
	}()

	ctx := context.Background()

	if err = orch.Initialize(ctx); err != nil {
		return fmt.Errorf("initializing builder: %w", err)
	}

	if err = orch.BuildLayers(ctx, graphResult.Layers); err != nil {
		return fmt.Errorf("building layers: %w", err)
	}

	return nil
}

func resolveAlpineVersion(version string) (string, error) {
	if version != "" {
		return version, nil
	}

	latest, err := alpineClient.GetLatestStableVersion()
	if err != nil {
		return "", fmt.Errorf("failed to detect latest Alpine version: %w", err)
	}

	fmt.Printf("Auto-detected Alpine version: %s\n", latest)
	return latest, nil
}
