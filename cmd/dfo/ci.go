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
	ciUpdate bool
	ciPush   bool
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Run in CI mode",
	Long:  "Run automated build and push for CI/CD environments",
	RunE:  runCI,
}

func init() {
	ciCmd.Flags().BoolVar(&ciUpdate, "update", false, "Check for updates before building")
	ciCmd.Flags().BoolVar(&ciPush, "push", false, "Push containers to registry after building")
	ciCmd.Flags().StringVar(&signingKeyPath, "signing-key", "", "Path to signing key")
	ciCmd.Flags().StringVarP(&outputDir, "output", "o", "repo", "Output directory for packages")
	ciCmd.Flags().StringVar(&containerOutput, "container-output", "output", "Output directory for containers")
}

func runCI(cmd *cobra.Command, args []string) error {
	fmt.Println("=== DFO CI Mode ===")

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

	if ciUpdate {
		// Run update check
		fmt.Println("\nChecking for package updates...")
		orchestrator := update.NewOrchestrator(
			repo,
			cfg,
			packageBuilder,
			apkoBuilder,
			apkIndex,
			registryClient,
		)

		result, err := orchestrator.CheckAndUpdate(false, ciPush)
		if err != nil {
			return fmt.Errorf("update check failed: %w", err)
		}

		// Print summary
		fmt.Println("\n=== Update Summary ===")
		if len(result.UpdatedPackages) > 0 {
			fmt.Println("Updated packages:")
			for _, pkg := range result.UpdatedPackages {
				info := result.VersionChanges[pkg]
				fmt.Printf("  - %s: %s -> %s\n", pkg, info.CurrentVersion, info.LatestVersion)
			}
		} else {
			fmt.Println("No packages updated")
		}

		if len(result.AffectedContainers) > 0 {
			fmt.Println("\nRebuilt containers:")
			for _, container := range result.AffectedContainers {
				fmt.Printf("  - %s\n", container)
			}
		}

	} else {
		// Build all packages
		fmt.Println("\nBuilding all packages...")
		packageSpecs, err := melange.LoadAllSpecs(repo.PackagesDir)
		if err != nil {
			return fmt.Errorf("failed to load package specs: %w", err)
		}

		for name, spec := range packageSpecs {
			fmt.Printf("Building package: %s\n", name)
			if err := packageBuilder.Build(spec); err != nil {
				return fmt.Errorf("failed to build %s: %w", name, err)
			}
		}

		// Generate APKINDEX
		fmt.Println("\nGenerating APKINDEX...")
		if err := apkIndex.Generate(); err != nil {
			return fmt.Errorf("failed to generate APKINDEX: %w", err)
		}

		// Build all containers
		fmt.Println("\nBuilding all containers...")
		containerSpecs, err := apko.LoadAllSpecs(repo.ContainersDir)
		if err != nil {
			return fmt.Errorf("failed to load container specs: %w", err)
		}

		for name, spec := range containerSpecs {
			fmt.Printf("Building container: %s\n", name)
			if err := apkoBuilder.Build(spec, "latest"); err != nil {
				return fmt.Errorf("failed to build %s: %w", name, err)
			}
		}

		// Push if requested
		if ciPush {
			fmt.Println("\nPushing containers to registry...")
			for name, spec := range containerSpecs {
				registries := cfg.GetRegistriesForContainer(name)
				if len(registries) == 0 {
					fmt.Printf("No registries configured for %s, skipping\n", name)
					continue
				}

				imagePath := apkoBuilder.GetImagePath(spec)
				if err := registryClient.PushToMultiple(imagePath, registries, name, "latest"); err != nil {
					return fmt.Errorf("failed to push %s: %w", name, err)
				}
			}
		}
	}

	fmt.Println("\n=== CI Build Complete ===")
	return nil
}
