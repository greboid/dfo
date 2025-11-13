package update

import (
	"fmt"

	"github.com/greboid/dfo/internal/apko"
	"github.com/greboid/dfo/internal/config"
	"github.com/greboid/dfo/internal/graph"
	"github.com/greboid/dfo/internal/melange"
	"github.com/greboid/dfo/internal/registry"
	"github.com/greboid/dfo/internal/repository"
)

// Orchestrator coordinates update detection and rebuilding
type Orchestrator struct {
	repo           *repository.Repository
	config         *config.Config
	packageBuilder *melange.Builder
	apkoBuilder    *apko.Builder
	apkIndex       *melange.APKIndex
	registryClient *registry.Client
	checker        *Checker
}

// NewOrchestrator creates a new update orchestrator
func NewOrchestrator(
	repo *repository.Repository,
	cfg *config.Config,
	packageBuilder *melange.Builder,
	apkoBuilder *apko.Builder,
	apkIndex *melange.APKIndex,
	registryClient *registry.Client,
) *Orchestrator {
	return &Orchestrator{
		repo:           repo,
		config:         cfg,
		packageBuilder: packageBuilder,
		apkoBuilder:    apkoBuilder,
		apkIndex:       apkIndex,
		registryClient: registryClient,
		checker:        NewChecker(),
	}
}

// UpdateResult contains the result of an update check
type UpdateResult struct {
	UpdatedPackages    []string
	AffectedContainers []string
	VersionChanges     map[string]*VersionInfo
}

// CheckAndUpdate checks for package updates and rebuilds as needed
func (o *Orchestrator) CheckAndUpdate(dryRun bool, autoPush bool, force bool) (*UpdateResult, error) {
	result := &UpdateResult{
		UpdatedPackages:    []string{},
		AffectedContainers: []string{},
		VersionChanges:     make(map[string]*VersionInfo),
	}

	// Load all package specs
	packageSpecs, err := melange.LoadAllSpecs(o.repo.PackagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load package specs: %w", err)
	}

	// Check all packages for updates
	fmt.Println("Checking for package updates...")
	versionInfo, err := o.checker.CheckAllPackages(packageSpecs)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	// Find packages that need updates
	var packagesToUpdate []string
	for name, info := range versionInfo {
		result.VersionChanges[name] = info
		// With force flag, update all packages regardless of version
		// Without force, only update if version changed
		if force || info.NeedsUpdate {
			if info.NeedsUpdate {
				fmt.Printf("Update available: %s %s -> %s\n", name, info.CurrentVersion, info.LatestVersion)
			} else if force {
				// Show if the version will change due to transform/cleanup
				if info.CurrentVersion != info.LatestVersion {
					fmt.Printf("Force updating: %s %s -> %s (cleaning version format)\n", name, info.CurrentVersion, info.LatestVersion)
				} else {
					fmt.Printf("Force updating: %s %s (already at latest)\n", name, info.CurrentVersion)
				}
			}
			packagesToUpdate = append(packagesToUpdate, name)
		}
	}

	if len(packagesToUpdate) == 0 {
		fmt.Println("No updates available")
		return result, nil
	}

	if dryRun {
		fmt.Println("\nDry run mode: would update packages:", packagesToUpdate)
		result.UpdatedPackages = packagesToUpdate
		return result, nil
	}

	// Update package versions in YAML files
	for _, pkgName := range packagesToUpdate {
		spec := packageSpecs[pkgName]
		info := versionInfo[pkgName]
		spec.Package.Version = info.LatestVersion
		spec.Package.Epoch = 0 // Reset epoch on version bump
		if err := spec.Save(); err != nil {
			return nil, fmt.Errorf("failed to save updated spec for %s: %w", pkgName, err)
		}
		fmt.Printf("Updated %s to version %s\n", pkgName, info.LatestVersion)
	}

	// Build updated packages
	if err := o.buildPackages(packagesToUpdate, packageSpecs); err != nil {
		return nil, fmt.Errorf("failed to build packages: %w", err)
	}
	result.UpdatedPackages = packagesToUpdate

	// Update APKINDEX
	if err := o.apkIndex.Generate(); err != nil {
		return nil, fmt.Errorf("failed to generate APKINDEX: %w", err)
	}

	// Find containers that use updated packages
	affectedContainers, err := o.findAffectedContainers(packagesToUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to find affected containers: %w", err)
	}
	result.AffectedContainers = affectedContainers

	if len(affectedContainers) > 0 {
		fmt.Printf("\nRebuilding affected containers: %v\n", affectedContainers)
		if err := o.buildContainers(affectedContainers); err != nil {
			return nil, fmt.Errorf("failed to build containers: %w", err)
		}

		// Push containers if requested
		if autoPush {
			if err := o.pushContainers(affectedContainers); err != nil {
				return nil, fmt.Errorf("failed to push containers: %w", err)
			}
		}
	}

	return result, nil
}

// buildPackages builds a list of packages in dependency order
func (o *Orchestrator) buildPackages(packageNames []string, allSpecs map[string]*melange.Spec) error {
	// Build dependency graph
	g := graph.New()
	for name, spec := range allSpecs {
		g.AddNode(name)
		for _, dep := range spec.GetRuntimeDependencies() {
			if _, exists := allSpecs[dep]; exists {
				g.AddDependency(name, dep)
			}
		}
	}

	// Get build order for requested packages
	var buildOrder []string
	visited := make(map[string]bool)

	for _, pkgName := range packageNames {
		deps, err := g.GetDependencies(pkgName)
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies for %s: %w", pkgName, err)
		}
		for _, dep := range deps {
			if !visited[dep] {
				buildOrder = append(buildOrder, dep)
				visited[dep] = true
			}
		}
	}

	// Build packages
	for _, pkgName := range buildOrder {
		spec := allSpecs[pkgName]
		if err := o.packageBuilder.Build(spec); err != nil {
			return fmt.Errorf("failed to build %s: %w", pkgName, err)
		}
	}

	return nil
}

// findAffectedContainers finds containers that depend on updated packages
func (o *Orchestrator) findAffectedContainers(updatedPackages []string) ([]string, error) {
	containerSpecs, err := apko.LoadAllSpecs(o.repo.ContainersDir)
	if err != nil {
		return nil, err
	}

	updateMap := make(map[string]bool)
	for _, pkg := range updatedPackages {
		updateMap[pkg] = true
	}

	var affected []string
	for name, spec := range containerSpecs {
		for _, pkg := range spec.GetPackages() {
			if updateMap[pkg] {
				affected = append(affected, name)
				break
			}
		}
	}

	return affected, nil
}

// buildContainers builds a list of containers
func (o *Orchestrator) buildContainers(containerNames []string) error {
	for _, name := range containerNames {
		specPath := o.repo.GetContainerPath(name)
		spec, err := apko.LoadSpec(specPath)
		if err != nil {
			return fmt.Errorf("failed to load container spec %s: %w", name, err)
		}

		tag := "latest" // TODO: use version from config or elsewhere
		if err := o.apkoBuilder.Build(spec, tag); err != nil {
			return fmt.Errorf("failed to build container %s: %w", name, err)
		}
	}
	return nil
}

// pushContainers pushes containers to configured registries
func (o *Orchestrator) pushContainers(containerNames []string) error {
	for _, name := range containerNames {
		registries := o.config.GetRegistriesForContainer(name)
		if len(registries) == 0 {
			fmt.Printf("No registries configured for %s, skipping push\n", name)
			continue
		}

		imagePath := o.apkoBuilder.GetImagePath(&apko.Spec{FilePath: o.repo.GetContainerPath(name)})
		tag := "latest"

		if err := o.registryClient.PushToMultiple(imagePath, registries, name, tag); err != nil {
			return fmt.Errorf("failed to push %s: %w", name, err)
		}
	}
	return nil
}
