package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	allConfig BuildConfig
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Build all containers from dfo.yaml files in a directory tree",
	Long: `Analyzes all dfo.yaml files in a directory tree and builds container images
locally using buildah with proper dependency ordering.

Builds are incremental - containers are only rebuilt if their inputs have changed.
Use --force-rebuild to rebuild all containers regardless of cache.`,
	RunE: runAll,
}

func init() {
	rootCmd.AddCommand(allCmd)

	allCmd.Flags().StringVarP(&allConfig.Directory, "directory", "d", ".", "Directory to search for dfo.yaml files")
	allCmd.Flags().StringVar(&allConfig.AlpineVersion, "alpine-version", "", "Alpine Linux version to resolve packages against (default: auto-detect latest)")
	allCmd.Flags().StringVar(&allConfig.GitUser, "git-user", "", "Git username for private repository access")
	allCmd.Flags().StringVar(&allConfig.GitPass, "git-pass", "", "Git password/token for private repository access")
	allCmd.Flags().StringVar(&allConfig.Registry, "registry", "", "Container registry to use for image resolution (required)")
	allCmd.Flags().StringVar(&allConfig.StoragePath, "storage-path", "", "Path to buildah storage (default: system default)")
	allCmd.Flags().StringVar(&allConfig.StorageDriver, "storage-driver", "", "Storage driver (overlay, vfs, etc.)")
	allCmd.Flags().StringVar(&allConfig.Isolation, "isolation", "", "Isolation mode (chroot, rootless, oci)")
	allCmd.Flags().IntVar(&allConfig.Concurrency, "concurrency", 5, "Number of parallel builds per layer")
	allCmd.Flags().BoolVar(&allConfig.ForceRebuild, "force-rebuild", false, "Force rebuild all containers, ignoring build cache")
	allCmd.Flags().BoolVar(&allConfig.Push, "push", false, "Push built images to registry after successful build")
	_ = allCmd.MarkFlagRequired("registry")
}

func runAll(_ *cobra.Command, _ []string) error {
	graphResult, err := loadConfigsAndBuildGraph(&allConfig)
	if err != nil {
		return err
	}

	if err := buildContainers(&allConfig, graphResult); err != nil {
		return fmt.Errorf("building containers: %w", err)
	}

	return nil
}
