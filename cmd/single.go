package cmd

import (
	"fmt"

	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
	"github.com/spf13/cobra"
)

var singleOutputDir string

var singleCmd = &cobra.Command{
	Use:   "single [directory|dfo.yaml]",
	Short: "Generate contempt template from a single YAML build file",
	RunE:  runSingle,
}

func init() {
	rootCmd.AddCommand(singleCmd)

	singleCmd.Flags().StringVarP(&singleOutputDir, "output", "o", ".", "Output directory for generated templates")
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

	result, err := processor.ProcessConfig(fs, configPath, singleOutputDir)
	if err != nil {
		return fmt.Errorf("failed to process config: %w", err)
	}

	fmt.Printf("✓ %s\n", result.PackageName)

	return nil
}
