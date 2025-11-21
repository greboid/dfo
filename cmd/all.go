package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/greboid/dfo/pkg/processor"
	"github.com/greboid/dfo/pkg/util"
	"github.com/spf13/cobra"
)

var allDirectory string

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate contempt template from all dfo.yaml files in a directory tree",
	RunE:  runAll,
}

func init() {
	rootCmd.AddCommand(allCmd)

	allCmd.Flags().StringVarP(&allDirectory, "directory", "d", ".", "Directory to search for dfo.yaml files")
}

func runAll(_ *cobra.Command, _ []string) error {
	fmt.Printf("Searching for dfo.yaml files in %s...\n", allDirectory)

	fs := util.DefaultFS()

	fileProcessor := func(configPath string) error {
		result, err := processor.ProcessConfigInPlace(fs, configPath)
		if err != nil {
			return err
		}
		fmt.Printf("✓ %s\n", result.PackageName)
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
