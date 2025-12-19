package cmd

import (
	"fmt"
	"os"
	"path"
	"sync"

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

	var outputMu sync.Mutex

	fileProcessor := func(configPath string) error {
		result, err := processor.ProcessConfigInPlace(fs, configPath, alpineClient, resolvedVersion, allGitUser, allGitPass, allRegistry, sharedImageResolver)
		if err != nil {
			return err
		}
		outputMu.Lock()
		fmt.Printf("✓ %s\n", result.PackageName)
		outputMu.Unlock()
		return nil
	}

	result, err := processor.WalkAndProcess(fs, allDirectory, fileProcessor)
	if err != nil {
		return err
	}

	for _, pe := range result.ErrorDetails {
		_, _ = fmt.Fprintf(os.Stderr, "✗ %s: %v\n", path.Base(path.Dir(pe.Path)), pe.Err)
	}

	fmt.Printf("Summary: %d file(s) processed", result.Processed)
	if result.Errors > 0 {
		fmt.Printf(", %d error(s)", result.Errors)
	}
	fmt.Println()

	if result.Processed == 0 {
		fmt.Println("\nNo dfo.yaml files found.")
	}

	if result.Errors > 0 {
		return fmt.Errorf("%d file(s) failed to process", result.Errors)
	}

	return nil
}
