package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/graph"
	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
	"github.com/greboid/dfo/pkg/workflow"
	"github.com/spf13/cobra"
)

var (
	orchestrateDirectory string
	orchestrateOutput    string
)

var orchestrateCmd = &cobra.Command{
	Use:   "orchestrate",
	Short: "Generate GitHub Actions workflow from dfo.yaml dependency graph",
	Long: `Analyzes all dfo.yaml files in a directory tree and generates a GitHub
Actions workflow file with proper dependency ordering based on base-image
references. The workflow calls container-builder.yml for each container.`,
	RunE: runOrchestrate,
}

func init() {
	rootCmd.AddCommand(orchestrateCmd)

	orchestrateCmd.Flags().StringVarP(
		&orchestrateDirectory,
		"directory",
		"d",
		".",
		"Directory to search for dfo.yaml files",
	)
	orchestrateCmd.Flags().StringVarP(
		&orchestrateOutput,
		"output",
		"o",
		"",
		"Output path for workflow file (required)",
	)
	_ = orchestrateCmd.MarkFlagRequired("output")
}

func runOrchestrate(_ *cobra.Command, _ []string) error {
	fmt.Printf("Searching for dfo.yaml files in %s...\n", orchestrateDirectory)

	fs := util.DefaultFS()

	absDir, err := filepath.Abs(orchestrateDirectory)
	if err != nil {
		return fmt.Errorf("resolving directory path: %w", err)
	}

	configFiles, err := processor.FindConfigFiles(fs, absDir)
	if err != nil {
		return fmt.Errorf("finding config files: %w", err)
	}

	if len(configFiles) == 0 {
		return fmt.Errorf("no dfo.yaml files found in %s", absDir)
	}

	fmt.Printf("Found %d dfo.yaml file(s)\n", len(configFiles))

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

	fmt.Printf("Generating workflow to %s...\n", orchestrateOutput)
	if err := workflow.Generate(depGraph, layers, orchestrateOutput); err != nil {
		return fmt.Errorf("generating workflow: %w", err)
	}

	fmt.Println("Workflow generated successfully!")
	return nil
}
