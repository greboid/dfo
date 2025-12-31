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
	"github.com/greboid/dfo/pkg/workflow"
	"github.com/spf13/cobra"
)

var (
	orchestrateDirectory     string
	orchestrateOutput        string
	orchestrateAlpineVersion string
	orchestrateGitUser       string
	orchestrateGitPass       string
	orchestrateRegistry      string
	orchestrateStoragePath   string
	orchestrateStorageDriver string
	orchestrateIsolation     string
	orchestrateConcurrency   int
	orchestrateForceRebuild  bool
	orchestratePush          bool
	orchestrateWorkflowOnly  bool
)

var orchestrateCmd = &cobra.Command{
	Use:   "orchestrate",
	Short: "Build containers and generate GitHub Actions workflow from dfo.yaml dependency graph",
	Long: `Analyzes all dfo.yaml files in a directory tree, builds container images
locally using buildah with proper dependency ordering, updates Containerfiles
and BOMs with actual build digests, and generates a GitHub Actions workflow file.

Builds are incremental - containers are only rebuilt if their inputs have changed.
Use --force-rebuild to rebuild all containers regardless of cache.

Note: Networking is automatically disabled during builds in rootless mode (non-root user)
to avoid permission errors. Most builds don't need networking since packages come from
base images.`,
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
		"Output path for workflow file (default: <directory>/.github/workflows/update-containers.yml)",
	)

	orchestrateCmd.Flags().BoolVar(
		&orchestrateForceRebuild,
		"force-rebuild",
		false,
		"Force rebuild all containers, ignoring build cache",
	)
	orchestrateCmd.Flags().BoolVar(
		&orchestratePush,
		"push",
		false,
		"Push built images to registry after successful build",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateAlpineVersion,
		"alpine-version",
		"",
		"Alpine version for package resolution (auto-detected if not specified)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateGitUser,
		"git-user",
		"",
		"Git username for private repository access",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateGitPass,
		"git-pass",
		"",
		"Git password/token for private repository access",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateRegistry,
		"registry",
		"",
		"Container registry for image naming (e.g., ghcr.io/username)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateStoragePath,
		"storage-path",
		"",
		"Custom buildah storage path (default: system default, useful for rootless)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateStorageDriver,
		"storage-driver",
		"",
		"Buildah storage driver (overlay, vfs, etc. - auto-detects with vfs fallback if not specified)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateIsolation,
		"isolation",
		"",
		"Buildah isolation mode (chroot, oci, rootless - default: auto-detect). Use 'chroot' for simpler rootless environments (disables networking during build)",
	)
	orchestrateCmd.Flags().IntVar(
		&orchestrateConcurrency,
		"concurrency",
		5,
		"Maximum parallel builds per layer",
	)
	orchestrateCmd.Flags().BoolVar(
		&orchestrateWorkflowOnly,
		"workflow",
		false,
		"Only generate workflow file without building containers locally",
	)
}

func runOrchestrate(_ *cobra.Command, _ []string) error {
	fmt.Printf("Searching for dfo.yaml files in %s...\n", orchestrateDirectory)

	fs := util.DefaultFS()

	absDir, err := filepath.Abs(orchestrateDirectory)
	if err != nil {
		return fmt.Errorf("resolving directory path: %w", err)
	}

	if orchestrateOutput == "" {
		orchestrateOutput = filepath.Join(absDir, ".github", "workflows", "update-containers.yml")
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

	if !orchestrateWorkflowOnly {
		fmt.Println("\nBuilding containers with buildah...")
		if err := buildContainers(depGraph, layers); err != nil {
			return fmt.Errorf("building containers: %w", err)
		}
		fmt.Println()
	}

	fmt.Printf("Generating workflow to %s...\n", orchestrateOutput)
	if err := workflow.Generate(depGraph, layers, orchestrateOutput); err != nil {
		return fmt.Errorf("generating workflow: %w", err)
	}

	fmt.Println("Workflow generated successfully!")
	return nil
}

func buildContainers(depGraph *graph.Graph, layers [][]string) error {
	fs := util.DefaultFS()

	resolvedVersion := orchestrateAlpineVersion
	if resolvedVersion == "" {
		latest, err := alpineClient.GetLatestStableVersion()
		if err != nil {
			return fmt.Errorf("failed to detect latest Alpine version: %w", err)
		}
		resolvedVersion = latest
		fmt.Printf("Auto-detected Alpine version: %s\n", resolvedVersion)
	}

	buildConfig := builder.OrchestratorConfig{
		AlpineVersion: resolvedVersion,
		GitUser:       orchestrateGitUser,
		GitPass:       orchestrateGitPass,
		Registry:      orchestrateRegistry,
		OutputDir:     orchestrateDirectory,
		Concurrency:   orchestrateConcurrency,
		AlpineClient:  alpineClient,
		ForceRebuild:  orchestrateForceRebuild,
		Push:          orchestratePush,
	}

	buildahBuilder := builder.NewBuildahBuilder(orchestrateRegistry, orchestrateStoragePath, orchestrateStorageDriver, orchestrateIsolation)

	orch, err := builder.NewOrchestrator(
		buildahBuilder,
		depGraph,
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

	if err = orch.BuildLayers(ctx, layers); err != nil {
		return fmt.Errorf("building layers: %w", err)
	}

	return nil
}
