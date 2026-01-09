package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	singleOutputDir     string
	singleAlpineVersion string
	singleGitUser       string
	singleGitPass       string
	singleRegistry      string
	singleStoragePath   string
	singleStorageDriver string
	singleIsolation     string
	singleConcurrency   int
	singleForceRebuild  bool
	singlePush          bool
	singleBuild         bool
	singleBuiltImages   string
)

var singleCmd = &cobra.Command{
	Use:   "single [directory|dfo.yaml]",
	Short: "Generate contempt template from a single YAML build file",
	RunE:  runSingle,
}

func init() {
	rootCmd.AddCommand(singleCmd)

	singleCmd.Flags().StringVarP(&singleOutputDir, "output", "o", ".", "Output directory for generated templates")
	singleCmd.Flags().StringVar(&singleAlpineVersion, "alpine-version", "", "Alpine Linux version to resolve packages against (default: auto-detect latest)")
	singleCmd.Flags().StringVar(&singleGitUser, "git-user", "", "Git username for private repository access")
	singleCmd.Flags().StringVar(&singleGitPass, "git-pass", "", "Git password/token for private repository access")
	singleCmd.Flags().StringVar(&singleRegistry, "registry", "", "Container registry to use for image resolution (required)")
	singleCmd.Flags().StringVar(&singleStoragePath, "storage-path", "", "Path to buildah storage (default: system default)")
	singleCmd.Flags().StringVar(&singleStorageDriver, "storage-driver", "", "Storage driver (overlay, vfs, etc.)")
	singleCmd.Flags().StringVar(&singleIsolation, "isolation", "", "Isolation mode (chroot, rootless, oci)")
	singleCmd.Flags().IntVar(&singleConcurrency, "concurrency", 5, "Number of parallel builds per layer")
	singleCmd.Flags().BoolVar(&singleForceRebuild, "force-rebuild", false, "Force rebuild container, ignoring build cache")
	singleCmd.Flags().BoolVar(&singlePush, "push", false, "Push built image to registry after successful build")
	singleCmd.Flags().BoolVar(&singleBuild, "build", false, "Build the container using buildah")
	singleCmd.Flags().StringVar(&singleBuiltImages, "built-images", "", "JSON string of built image digests (format: {\"imagename\":\"digest\"})")
	_ = singleCmd.MarkFlagRequired("registry")
}

func runSingle(_ *cobra.Command, args []string) error {
	var input string
	if len(args) > 0 {
		input = args[0]
	}

	fs := util.DefaultFS()

	configPath, err := processor.ResolveConfigPath(fs, input)
	if err != nil {
		return err
	}

	resolvedVersion, err := resolveAlpineVersion(singleAlpineVersion)
	if err != nil {
		return err
	}

	var builtImages map[string]string
	if singleBuiltImages != "" {
		if err := json.Unmarshal([]byte(singleBuiltImages), &builtImages); err != nil {
			return fmt.Errorf("parsing built-images JSON: %w", err)
		}
	}

	if singleBuild {
		cfg := &BuildConfig{
			Directory:     filepath.Dir(configPath),
			AlpineVersion: resolvedVersion,
			GitUser:       singleGitUser,
			GitPass:       singleGitPass,
			Registry:      singleRegistry,
			StoragePath:   singleStoragePath,
			StorageDriver: singleStorageDriver,
			Isolation:     singleIsolation,
			Concurrency:   singleConcurrency,
			ForceRebuild:  singleForceRebuild,
			Push:          singlePush,
		}

		graphResult, err := loadSingleConfigAndBuildGraph(configPath)
		if err != nil {
			return err
		}

		return buildContainers(cfg, graphResult)
	}

	result, err := processor.ProcessConfigWithBuiltImages(fs, configPath, singleOutputDir, alpineClient, resolvedVersion, singleGitUser, singleGitPass, singleRegistry, nil, builtImages, nil)
	if err != nil {
		return fmt.Errorf("failed to process config: %w", err)
	}

	fmt.Printf("✓ %s\n", result.PackageName)

	return nil
}
