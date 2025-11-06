package main

import (
	"fmt"
	"os"

	"github.com/greboid/dfo/internal/apko"
	"github.com/greboid/dfo/internal/graph"
	"github.com/greboid/dfo/internal/melange"
	"github.com/greboid/dfo/internal/registry"
	"github.com/spf13/cobra"
)

var (
	containerBuildAll bool
	containerPushAll  bool
	containerTag      string
	containerOutput   string
)

var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage apko containers",
	Long:  "Build and manage apko containers",
}

var containerBuildCmd = &cobra.Command{
	Use:   "build [container-name]",
	Short: "Build a container and its dependencies",
	Long:  "Build an apko container and all required packages",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runContainerBuild,
}

var containerPushCmd = &cobra.Command{
	Use:   "push [container-name]",
	Short: "Push a container to registry",
	Long:  "Push a built container to configured registries",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runContainerPush,
}

func init() {
	containerCmd.AddCommand(containerBuildCmd)
	containerCmd.AddCommand(containerPushCmd)

	containerBuildCmd.Flags().BoolVar(&containerBuildAll, "all", false, "Build all containers")
	containerBuildCmd.Flags().StringVarP(&containerTag, "tag", "t", "latest", "Container tag")
	containerBuildCmd.Flags().StringVarP(&containerOutput, "output", "o", "output", "Output directory for built containers")

	containerPushCmd.Flags().BoolVar(&containerPushAll, "all", false, "Push all containers")
	containerPushCmd.Flags().StringVarP(&containerTag, "tag", "t", "latest", "Container tag")
	containerPushCmd.Flags().StringVarP(&containerOutput, "output", "o", "output", "Directory containing built containers")
}

func runContainerBuild(cmd *cobra.Command, args []string) error {
	// Determine what to build
	if !containerBuildAll && len(args) == 0 {
		return fmt.Errorf("specify a container name or use --all to build all containers")
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

	// Create builders
	packageBuilder := melange.NewBuilder(repo.Root, outputDir, repo.RepoDir, keyPath)
	apkoBuilder := apko.NewBuilder(repo.Root, containerOutput, repo.RepoDir, keyPath)

	// Load all specs
	containerSpecs, err := apko.LoadAllSpecs(repo.ContainersDir)
	if err != nil {
		return fmt.Errorf("failed to load container specs: %w", err)
	}

	packageSpecs, err := melange.LoadAllSpecs(repo.PackagesDir)
	if err != nil {
		return fmt.Errorf("failed to load package specs: %w", err)
	}

	// Determine containers to build
	var containersToBuild []string
	if containerBuildAll {
		for name := range containerSpecs {
			containersToBuild = append(containersToBuild, name)
		}
	} else {
		containerName := args[0]
		if _, exists := containerSpecs[containerName]; !exists {
			return fmt.Errorf("container not found: %s", containerName)
		}
		containersToBuild = append(containersToBuild, containerName)
	}

	// For each container, ensure its packages are built
	for _, containerName := range containersToBuild {
		spec := containerSpecs[containerName]
		packages := spec.GetPackages()

		fmt.Printf("\nBuilding container: %s\n", containerName)
		fmt.Printf("Required packages: %v\n", packages)

		// Build package dependency graph
		g := graph.New()
		for name, pkgSpec := range packageSpecs {
			g.AddNode(name)
			for _, dep := range pkgSpec.GetRuntimeDependencies() {
				if _, exists := packageSpecs[dep]; exists {
					g.AddDependency(name, dep)
				}
			}
		}

		// Build all required packages
		packagesToBuild := make(map[string]bool)
		for _, pkg := range packages {
			if _, exists := packageSpecs[pkg]; exists {
				deps, err := g.GetDependencies(pkg)
				if err != nil {
					return fmt.Errorf("failed to resolve dependencies for %s: %w", pkg, err)
				}
				for _, dep := range deps {
					packagesToBuild[dep] = true
				}
			}
		}

		// Build packages if needed
		if len(packagesToBuild) > 0 {
			var buildOrder []string
			for pkg := range packagesToBuild {
				buildOrder = append(buildOrder, pkg)
			}
			fmt.Printf("Building packages: %v\n", buildOrder)

			for _, pkgName := range buildOrder {
				pkgSpec := packageSpecs[pkgName]
				needsBuild, err := packageBuilder.CheckNeedsBuild(pkgSpec)
				if err != nil {
					return err
				}
				if needsBuild {
					if err := packageBuilder.Build(pkgSpec); err != nil {
						return fmt.Errorf("failed to build package %s: %w", pkgName, err)
					}
				} else {
					fmt.Printf("Package %s already built, skipping\n", pkgName)
				}
			}

			// Update APKINDEX
			apkIndex := melange.NewAPKIndex(repo.RepoDir, outputDir, keyPath)
			if err := apkIndex.Generate(); err != nil {
				return fmt.Errorf("failed to generate APKINDEX: %w", err)
			}
		}

		// Build container
		if err := apkoBuilder.Build(spec, containerTag); err != nil {
			return fmt.Errorf("failed to build container %s: %w", containerName, err)
		}
	}

	fmt.Println("\nContainer build complete!")
	return nil
}

func runContainerPush(cmd *cobra.Command, args []string) error {
	// Determine what to push
	if !containerPushAll && len(args) == 0 {
		return fmt.Errorf("specify a container name or use --all to push all containers")
	}

	// Load container specs
	containerSpecs, err := apko.LoadAllSpecs(repo.ContainersDir)
	if err != nil {
		return fmt.Errorf("failed to load container specs: %w", err)
	}

	// Determine containers to push
	var containersToPush []string
	if containerPushAll {
		for name := range containerSpecs {
			containersToPush = append(containersToPush, name)
		}
	} else {
		containerName := args[0]
		if _, exists := containerSpecs[containerName]; !exists {
			return fmt.Errorf("container not found: %s", containerName)
		}
		containersToPush = append(containersToPush, containerName)
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

	// Create registry client
	registryClient := registry.NewClient()
	apkoBuilder := apko.NewBuilder(repo.Root, containerOutput, repo.RepoDir, keyPath)

	// Push each container
	for _, containerName := range containersToPush {
		spec := containerSpecs[containerName]

		// Get registries for this container
		registries := cfg.GetRegistriesForContainer(containerName)
		if len(registries) == 0 {
			fmt.Printf("No registries configured for %s, skipping\n", containerName)
			continue
		}

		fmt.Printf("\nPushing %s to registries: %v\n", containerName, registries)

		// Get image path
		imagePath := apkoBuilder.GetImagePath(spec)

		// Push to all registries
		if err := registryClient.PushToMultiple(imagePath, registries, containerName, containerTag); err != nil {
			return fmt.Errorf("failed to push %s: %w", containerName, err)
		}
	}

	fmt.Println("\nPush complete!")
	return nil
}
