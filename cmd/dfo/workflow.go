package main

import (
	"fmt"
	"path/filepath"

	"github.com/greboid/dfo/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	workflowOutput string
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage GitHub Actions workflows",
	Long:  "Generate and manage GitHub Actions workflows for automated builds",
}

var workflowGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate GitHub Actions workflow",
	Long:  "Generate a GitHub Actions workflow file for building and pushing containers",
	RunE:  runWorkflowGenerate,
}

func init() {
	workflowCmd.AddCommand(workflowGenerateCmd)

	workflowGenerateCmd.Flags().StringVarP(&workflowOutput, "output", "o", ".github/workflows/build.yml", "Output path for workflow file")
}

func runWorkflowGenerate(cmd *cobra.Command, args []string) error {
	fmt.Println("Generating GitHub Actions workflow...")

	// Create workflow generator
	generator := workflow.NewGenerator(repo, cfg)

	// Generate workflow YAML
	workflowYAML, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Determine output path
	outputPath := workflowOutput
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(repo.Root, outputPath)
	}

	// Write workflow file
	if err := generator.WriteToFile(workflowYAML, outputPath); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	fmt.Printf("Workflow file written to: %s\n", outputPath)
	return nil
}
