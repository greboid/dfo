package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/greboid/dfo/pkg/workflow"
	"github.com/spf13/cobra"
)

var (
	orchestrateOutput       string
	orchestrateWorkflowOnly bool
	orchestrateConfig       BuildConfig
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
		&orchestrateConfig.Directory,
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
		&orchestrateConfig.ForceRebuild,
		"force-rebuild",
		false,
		"Force rebuild all containers, ignoring build cache",
	)
	orchestrateCmd.Flags().BoolVar(
		&orchestrateConfig.Push,
		"push",
		false,
		"Push built images to registry after successful build",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.AlpineVersion,
		"alpine-version",
		"",
		"Alpine version for package resolution (auto-detected if not specified)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.GitUser,
		"git-user",
		"",
		"Git username for private repository access",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.GitPass,
		"git-pass",
		"",
		"Git password/token for private repository access",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.Registry,
		"registry",
		"",
		"Container registry for image naming (e.g., ghcr.io/username)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.StoragePath,
		"storage-path",
		"",
		"Custom buildah storage path (default: system default, useful for rootless)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.StorageDriver,
		"storage-driver",
		"",
		"Buildah storage driver (overlay, vfs, etc. - auto-detects with vfs fallback if not specified)",
	)
	orchestrateCmd.Flags().StringVar(
		&orchestrateConfig.Isolation,
		"isolation",
		"",
		"Buildah isolation mode (chroot, oci, rootless - default: auto-detect). Use 'chroot' for simpler rootless environments (disables networking during build)",
	)
	orchestrateCmd.Flags().IntVar(
		&orchestrateConfig.Concurrency,
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
	absDir, err := filepath.Abs(orchestrateConfig.Directory)
	if err != nil {
		return fmt.Errorf("resolving directory path: %w", err)
	}

	if orchestrateOutput == "" {
		orchestrateOutput = filepath.Join(absDir, ".github", "workflows", "update-containers.yml")
	}

	graphResult, err := loadConfigsAndBuildGraph(&orchestrateConfig)
	if err != nil {
		return err
	}

	if !orchestrateWorkflowOnly {
		if err := buildContainers(&orchestrateConfig, graphResult); err != nil {
			return fmt.Errorf("building containers: %w", err)
		}
		fmt.Println()
	}

	fmt.Printf("Generating workflow to %s...\n", orchestrateOutput)
	if err := workflow.Generate(graphResult.Graph, graphResult.Layers, orchestrateOutput); err != nil {
		return fmt.Errorf("generating workflow: %w", err)
	}

	fmt.Println("Workflow generated successfully!")
	return nil
}
