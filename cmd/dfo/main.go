package main

import (
	"fmt"
	"os"

	"github.com/greboid/dfo/internal/config"
	"github.com/greboid/dfo/internal/repository"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	repoPath string
	verbose  bool

	// Shared state
	repo *repository.Repository
	cfg  *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "dfo",
	Short: "Docker Filesystem Objects - Manage apko containers and melange packages",
	Long: `dfo is a CLI tool for managing container builds using apko and melange.
It handles dependency resolution, version tracking, and registry operations.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Discover repository
		var err error
		repo, err = repository.Discover(repoPath)
		if err != nil {
			return fmt.Errorf("failed to discover repository: %w", err)
		}

		// Load config
		cfg, err = config.Load(repo.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Repository root: %s\n", repo.Root)
			fmt.Fprintf(os.Stderr, "Config loaded from: %s\n", repo.ConfigFile)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "r", "", "Path to repository (default: current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add subcommands
	rootCmd.AddCommand(packageCmd)
	rootCmd.AddCommand(containerCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(ciCmd)
	rootCmd.AddCommand(workflowCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
