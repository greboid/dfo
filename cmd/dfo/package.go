package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greboid/dfo/internal/graph"
	"github.com/greboid/dfo/internal/melange"
	"github.com/spf13/cobra"
)

var (
	buildAll       bool
	forceBuild     bool
	signingKeyPath string
	outputDir      string
)

var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Manage melange packages",
	Long:  "Build and manage melange packages",
}

var packageBuildCmd = &cobra.Command{
	Use:   "build [package-name]",
	Short: "Build a package and its dependencies",
	Long:  "Build a melange package and all its dependencies in the correct order",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPackageBuild,
}

func init() {
	packageCmd.AddCommand(packageBuildCmd)

	packageBuildCmd.Flags().BoolVar(&buildAll, "all", false, "Build all packages")
	packageBuildCmd.Flags().BoolVar(&forceBuild, "force", false, "Force rebuild even if package already exists")
	packageBuildCmd.Flags().StringVar(&signingKeyPath, "signing-key", "", "Path to signing key (required unless set in config)")
	packageBuildCmd.Flags().StringVarP(&outputDir, "output", "o", "repo", "Output directory for built packages")
}

func runPackageBuild(cmd *cobra.Command, args []string) error {
	// Determine what to build
	if !buildAll && len(args) == 0 {
		return fmt.Errorf("specify a package name or use --all to build all packages")
	}

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

	// Create builder
	builder := melange.NewBuilder(repo.Root, outputDir, repo.RepoDir, keyPath)

	// Load all package specs
	packageSpecs, err := melange.LoadAllSpecs(repo.PackagesDir)
	if err != nil {
		return fmt.Errorf("failed to load package specs: %w", err)
	}

	// Build dependency graph
	g := graph.New()
	for name, spec := range packageSpecs {
		g.AddNode(name)
		for _, dep := range spec.GetRuntimeDependencies() {
			// Skip self-dependencies (e.g., go@local in go package)
			if dep == name {
				continue
			}
			if _, exists := packageSpecs[dep]; exists {
				g.AddDependency(name, dep)
			}
		}
		for _, dep := range spec.GetBuildDependencies() {
			// Skip self-dependencies (e.g., go@local in go package)
			if dep == name {
				continue
			}
			if _, exists := packageSpecs[dep]; exists {
				g.AddDependency(name, dep)
			}
		}
	}

	// Determine build list
	var buildOrder []string
	if buildAll {
		// Build all packages in topological order
		buildOrder, err = g.TopologicalSort()
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies: %w", err)
		}
	} else {
		// Build specific package and its dependencies
		packageName := args[0]
		if _, exists := packageSpecs[packageName]; !exists {
			return fmt.Errorf("package not found: %s", packageName)
		}

		buildOrder, err = g.GetDependencies(packageName)
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies: %w", err)
		}
	}

	fmt.Printf("Building %d package(s) in order: %v\n", len(buildOrder), buildOrder)

	// Build packages
	var builtCount, skippedCount int
	for _, pkgName := range buildOrder {
		spec := packageSpecs[pkgName]

		// Check if package needs to be built
		if !forceBuild {
			needsBuild, err := builder.CheckNeedsBuild(spec)
			if err != nil {
				return fmt.Errorf("failed to check build status for %s: %w", pkgName, err)
			}

			if !needsBuild {
				fmt.Printf("Skipping %s (%s) - already built\n", spec.GetName(), spec.GetVersion())
				skippedCount++
				continue
			}
		}

		if err := builder.Build(spec); err != nil {
			return fmt.Errorf("failed to build %s: %w", pkgName, err)
		}
		builtCount++
	}

	fmt.Printf("\nBuild summary: %d built, %d skipped\n", builtCount, skippedCount)

	// Generate APKINDEX
	fmt.Println("\nGenerating APKINDEX...")
	apkIndex := melange.NewAPKIndex(repo.RepoDir, outputDir, keyPath)
	if err := apkIndex.Generate(); err != nil {
		return fmt.Errorf("failed to generate APKINDEX: %w", err)
	}

	// Print summary
	fmt.Println("\nBuild complete!")
	fmt.Printf("Packages: %s\n", filepath.Join(outputDir, builder.Arch))
	fmt.Printf("APKINDEX: %s\n", apkIndex.GetPath())

	return nil
}
