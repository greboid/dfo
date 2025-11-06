package main

import (
	"fmt"
	"os"

	"github.com/greboid/dfo/internal/apko"
	"github.com/greboid/dfo/internal/melange"
	"github.com/greboid/dfo/internal/registry"
	"github.com/greboid/dfo/internal/update"
	"github.com/spf13/cobra"
)

var (
	updateDryRun   bool
	updateAutoPush bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and apply package updates",
	Long:  "Check for package updates and rebuild affected packages and containers",
}

var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for package updates",
	Long:  "Check for package updates, rebuild packages and containers, optionally push to registry",
	RunE:  runUpdateCheck,
}

func init() {
	updateCmd.AddCommand(updateCheckCmd)

	updateCheckCmd.Flags().BoolVar(&updateDryRun, "dry-run", false, "Show what would be updated without making changes")
	updateCheckCmd.Flags().BoolVar(&updateAutoPush, "push", false, "Automatically push updated containers to registry")
	updateCheckCmd.Flags().StringVar(&signingKeyPath, "signing-key", "", "Path to signing key")
	updateCheckCmd.Flags().StringVarP(&outputDir, "output", "o", "repo", "Output directory for packages")
	updateCheckCmd.Flags().StringVar(&containerOutput, "container-output", "output", "Output directory for containers")
}

func runUpdateCheck(cmd *cobra.Command, args []string) error {
	// Ensure repo directory exists
	if err := repo.EnsureRepoDir(); err != nil {
		return err
	}

	// Use config signing key if CLI flag not provided
	keyPath := signingKeyPath
	if keyPath == "" && cfg.SigningKey != "" {
		keyPath = cfg.SigningKey
	}

	// Require signing key
	if keyPath == "" {
		return fmt.Errorf("signing key required: provide --signing-key flag or set signing_key in .dfo.yaml")
	}

	// Verify signing key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("signing key not found: %s", keyPath)
	} else if err != nil {
		return fmt.Errorf("failed to check signing key: %w", err)
	}

	// Verify public key exists
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("public key not found: %s", pubKeyPath)
	} else if err != nil {
		return fmt.Errorf("failed to check public key: %w", err)
	}

	// Create builders
	packageBuilder := melange.NewBuilder(repo.Root, outputDir, repo.RepoDir, keyPath)
	apkoBuilder := apko.NewBuilder(repo.Root, containerOutput, repo.RepoDir, keyPath)
	apkIndex := melange.NewAPKIndex(repo.RepoDir, outputDir, keyPath)
	registryClient := registry.NewClient()

	// Create orchestrator
	orchestrator := update.NewOrchestrator(
		repo,
		cfg,
		packageBuilder,
		apkoBuilder,
		apkIndex,
		registryClient,
	)

	// Run update check
	result, err := orchestrator.CheckAndUpdate(updateDryRun, updateAutoPush)
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	// Print summary
	fmt.Println("\n=== Update Summary ===")

	if len(result.UpdatedPackages) > 0 {
		fmt.Println("\nUpdated packages:")
		for _, pkg := range result.UpdatedPackages {
			info := result.VersionChanges[pkg]
			fmt.Printf("  - %s: %s -> %s\n", pkg, info.CurrentVersion, info.LatestVersion)
		}
	} else {
		fmt.Println("\nNo packages updated")
	}

	if len(result.AffectedContainers) > 0 {
		fmt.Println("\nAffected containers:")
		for _, container := range result.AffectedContainers {
			fmt.Printf("  - %s\n", container)
		}
	}

	if updateDryRun {
		fmt.Println("\n(Dry run mode - no changes were made)")
	}

	return nil
}
