package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/graph"
	"github.com/greboid/dfo/pkg/images"
	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	allDirectory     string
	allAlpineVersion string
	allGitUser       string
	allGitPass       string
	allRegistry      string
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate contempt template from all dfo.yaml files in a directory tree",
	RunE:  runAll,
}

func init() {
	rootCmd.AddCommand(allCmd)

	allCmd.Flags().StringVarP(&allDirectory, "directory", "d", ".", "Directory to search for dfo.yaml files")
	allCmd.Flags().StringVar(&allAlpineVersion, "alpine-version", "", "Alpine Linux version to resolve packages against (default: auto-detect latest)")
	allCmd.Flags().StringVar(&allGitUser, "git-user", "", "Git username for private repository access")
	allCmd.Flags().StringVar(&allGitPass, "git-pass", "", "Git password/token for private repository access")
	allCmd.Flags().StringVar(&allRegistry, "registry", "", "Container registry to use for image resolution (required)")
	_ = allCmd.MarkFlagRequired("registry")
}

func runAll(_ *cobra.Command, _ []string) error {
	fmt.Printf("Searching for dfo.yaml files in %s...\n", allDirectory)

	fs := util.DefaultFS()

	absDir, err := filepath.Abs(allDirectory)
	if err != nil {
		return fmt.Errorf("resolving directory path: %w", err)
	}

	// Find all config files
	configFiles, err := processor.FindConfigFiles(fs, absDir)
	if err != nil {
		return fmt.Errorf("finding config files: %w", err)
	}

	if len(configFiles) == 0 {
		fmt.Println("No dfo.yaml files found.")
		return nil
	}

	fmt.Printf("Found %d dfo.yaml file(s)\n", len(configFiles))

	// Load all configs and build dependency graph
	configs := make(map[string]*config.BuildConfig)
	containerPaths := make(map[string]string)

	for _, configPath := range configFiles {
		cfg, err := config.Load(fs, configPath)
		if err != nil {
			return fmt.Errorf("loading %s: %w", configPath, err)
		}

		containerName := filepath.Base(filepath.Dir(configPath))
		configs[containerName] = cfg
		containerPaths[containerName] = configPath
	}

	fmt.Println("Building dependency graph...")
	depGraph, err := graph.Build(configs, containerPaths)
	if err != nil {
		return fmt.Errorf("building dependency graph: %w", err)
	}

	fmt.Printf("Graph contains %d container(s)\n", len(depGraph.Containers))

	fmt.Println("Resolving build order...")
	layers, err := depGraph.TopologicalSort()
	if err != nil {
		return fmt.Errorf("resolving dependencies: %w", err)
	}

	fmt.Printf("Resolved into %d layer(s):\n", len(layers))
	for i, layer := range layers {
		fmt.Printf("  Layer %d: %v\n", i, layer)
	}

	// Resolve Alpine version
	resolvedVersion := allAlpineVersion
	if resolvedVersion == "" {
		latest, err := alpineClient.GetLatestStableVersion()
		if err != nil {
			return fmt.Errorf("failed to detect latest Alpine version: %w", err)
		}
		resolvedVersion = latest
		fmt.Printf("Auto-detected Alpine version: %s\n", resolvedVersion)
	}

	sharedImageResolver := images.NewResolver(allRegistry, true)

	// Process layers in order
	fmt.Println("\nGenerating Containerfiles in dependency order...")
	totalProcessed := 0
	totalErrors := 0

	for layerIdx, layer := range layers {
		fmt.Printf("\nProcessing layer %d: %v\n", layerIdx, layer)

		// Process this layer (can be done in parallel within the layer)
		var layerMu sync.Mutex
		var layerWg sync.WaitGroup
		const maxConcurrency = 5
		semaphore := make(chan struct{}, maxConcurrency)

		for _, containerName := range layer {
			layerWg.Add(1)
			go func(cName string) {
				defer layerWg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				container := depGraph.Containers[cName]
				configPath := container.ConfigPath

				result, err := processor.ProcessConfigInPlace(
					fs,
					configPath,
					alpineClient,
					resolvedVersion,
					allGitUser,
					allGitPass,
					allRegistry,
					sharedImageResolver,
				)

				layerMu.Lock()
				defer layerMu.Unlock()

				if err != nil {
					totalErrors++
					fmt.Fprintf(os.Stderr, "✗ %s: %v\n", cName, err)
				} else {
					totalProcessed++
					fmt.Printf("✓ %s\n", result.PackageName)
				}
			}(containerName)
		}

		layerWg.Wait()
	}

	// Print summary
	fmt.Printf("\nSummary: %d file(s) processed", totalProcessed)
	if totalErrors > 0 {
		fmt.Printf(", %d error(s)", totalErrors)
	}
	fmt.Println()

	if totalErrors > 0 {
		return fmt.Errorf("%d file(s) failed to process", totalErrors)
	}

	return nil
}
