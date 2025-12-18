package cmd

import (
	"fmt"

	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	singleOutputDir     string
	singleAlpineVersion string
	singleGitUser       string
	singleGitPass       string
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

	resolvedVersion := singleAlpineVersion
	if resolvedVersion == "" {
		latest, err := alpineClient.GetLatestStableVersion()
		if err != nil {
			return fmt.Errorf("failed to detect latest Alpine version: %w", err)
		}
		resolvedVersion = latest
		fmt.Printf("Auto-detected Alpine version: %s\n", resolvedVersion)
	}

	result, err := processor.ProcessConfig(fs, configPath, singleOutputDir, alpineClient, resolvedVersion, singleGitUser, singleGitPass)
	if err != nil {
		return fmt.Errorf("failed to process config: %w", err)
	}

	fmt.Printf("✓ %s\n", result.PackageName)

	return nil
}
