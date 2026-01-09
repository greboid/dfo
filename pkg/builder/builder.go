package builder

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/greboid/dfo/pkg/generator"
	"github.com/greboid/dfo/pkg/graph"
	"github.com/greboid/dfo/pkg/images"
	"github.com/greboid/dfo/pkg/packages"
	"github.com/greboid/dfo/pkg/util"
)

type Builder interface {
	Initialize(ctx context.Context) error

	BuildContainer(ctx context.Context, containerName string, containerfilePath string, contextDir string) (*BuildResult, error)

	PushImage(ctx context.Context, imageName string) error

	Close() error
}

type OrchestratorConfig struct {
	AlpineVersion string
	GitUser       string
	GitPass       string
	Registry      string
	OutputDir     string
	Concurrency   int
	AlpineClient  *packages.AlpineClient
	ForceRebuild  bool
	Push          bool
}

type buildJob struct {
	containerName string
	index         int
}

type buildOutput struct {
	result *BuildResult
	err    error
	index  int
}

type Orchestrator struct {
	builder       Builder
	graph         *graph.Graph
	registry      *BuildRegistry
	cache         *BuildCache
	fs            util.WritableFS
	config        OrchestratorConfig
	imageResolver *images.Resolver
}

func NewOrchestrator(
	builder Builder,
	depGraph *graph.Graph,
	fs util.WritableFS,
	cfg OrchestratorConfig,
) (*Orchestrator, error) {
	imageResolver := images.NewResolver(cfg.Registry, false)

	cache, err := NewBuildCache(cfg.OutputDir, fs)
	if err != nil {
		return nil, fmt.Errorf("initializing build cache: %w", err)
	}

	if cfg.ForceRebuild {
		slog.Info("Force rebuild enabled, invalidating build cache")
		cache.InvalidateAll()
	}

	return &Orchestrator{
		builder:       builder,
		graph:         depGraph,
		registry:      NewBuildRegistry(),
		cache:         cache,
		fs:            fs,
		config:        cfg,
		imageResolver: imageResolver,
	}, nil
}

func (o *Orchestrator) Initialize(ctx context.Context) error {
	if err := o.builder.Initialize(ctx); err != nil {
		return fmt.Errorf("initializing builder: %w", err)
	}

	slog.Info("Builder initialized successfully")
	return nil
}

func (o *Orchestrator) Close() error {
	if err := o.cache.Save(); err != nil {
		slog.Warn("Failed to save build cache", "error", err)
	}

	if err := o.builder.Close(); err != nil {
		return fmt.Errorf("closing builder: %w", err)
	}

	slog.Info("Builder closed successfully")
	return nil
}

func (o *Orchestrator) BuildLayers(ctx context.Context, layers [][]string) error {
	totalLayers := len(layers)
	totalContainers := 0
	for _, layer := range layers {
		totalContainers += len(layer)
	}

	slog.Info("Starting multi-layer build",
		"layers", totalLayers,
		"containers", totalContainers,
	)

	startTime := time.Now()

	for layerIdx, layer := range layers {
		slog.Info("Processing layer",
			"layer", layerIdx,
			"total_layers", totalLayers,
			"containers", layer,
		)

		if err := o.generateContainerfiles(ctx, layer); err != nil {
			return fmt.Errorf("generating Containerfiles for layer %d: %w", layerIdx, err)
		}

		if err := o.buildLayer(ctx, layerIdx, totalLayers, layer); err != nil {
			return fmt.Errorf("building layer %d: %w", layerIdx, err)
		}

		slog.Info("Layer completed",
			"layer", layerIdx,
			"containers_built", len(layer),
		)
	}

	duration := time.Since(startTime)
	slog.Info("✓ All containers built successfully!",
		"total_layers", totalLayers,
		"total_containers", totalContainers,
		"duration", duration.Round(time.Second),
	)

	return nil
}

func (o *Orchestrator) buildLayer(ctx context.Context, layerIdx, totalLayers int, layer []string) error {
	totalInLayer := len(layer)
	jobs := make(chan buildJob, totalInLayer)
	results := make(chan buildOutput, totalInLayer)

	workers := o.calculateWorkers(totalInLayer)

	o.startBuildWorkers(ctx, workers, jobs, results, layerIdx, totalLayers, totalInLayer)

	for i, containerName := range layer {
		jobs <- buildJob{containerName: containerName, index: i}
	}
	close(jobs)

	return o.collectAndHandleResults(results, totalInLayer, layerIdx)
}

func (o *Orchestrator) calculateWorkers(totalInLayer int) int {
	workers := o.config.Concurrency
	if workers <= 0 {
		workers = 5
	}
	if workers > totalInLayer {
		workers = totalInLayer
	}
	return workers
}

func (o *Orchestrator) startBuildWorkers(ctx context.Context, workers int, jobs chan buildJob, results chan buildOutput, layerIdx, totalLayers, totalInLayer int) {
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			o.processBuildJob(ctx, jobs, results, layerIdx, totalLayers, totalInLayer, workerID)
		}(w)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

func (o *Orchestrator) processBuildJob(ctx context.Context, jobs chan buildJob, results chan buildOutput, layerIdx, totalLayers, totalInLayer, workerID int) {
	for job := range jobs {
		result, err := o.buildContainer(ctx, job, layerIdx, totalLayers, totalInLayer, workerID)
		if err != nil {
			results <- buildOutput{err: err, index: job.index}
		} else {
			results <- buildOutput{result: result, index: job.index}
		}
	}
}

func (o *Orchestrator) buildContainer(ctx context.Context, job buildJob, layerIdx, totalLayers, totalInLayer, workerID int) (*BuildResult, error) {
	containerName := job.containerName
	container := o.graph.Containers[containerName]
	containerDir := filepath.Dir(container.ConfigPath)
	containerfilePath := filepath.Join(containerDir, "Containerfile")

	if shouldSkip, skipped := o.checkShouldSkip(containerName, containerDir, job.index+1, totalInLayer); shouldSkip {
		if skipped {
			return nil, nil
		}
		return nil, fmt.Errorf("build skipped for %s", containerName)
	}

	if result, useCache, err := o.tryUseCachedBuild(ctx, containerName, container.ConfigPath, job.index+1, totalInLayer); useCache {
		return result, err
	}

	return o.buildAndPushContainer(ctx, containerName, containerfilePath, containerDir, job.index+1, totalInLayer, layerIdx, totalLayers, workerID, container.ConfigPath)
}

func (o *Orchestrator) checkShouldSkip(containerName, containerDir string, index, totalInLayer int) (bool, bool) {
	ignorePath := filepath.Join(containerDir, "IGNORE")
	if _, err := o.fs.Stat(ignorePath); err == nil {
		slog.Info("Skipping build (IGNORE file present)",
			"container", containerName,
			"progress", fmt.Sprintf("[%d/%d]", index, totalInLayer),
		)
		return true, true
	}
	return false, false
}

func (o *Orchestrator) tryUseCachedBuild(ctx context.Context, containerName, configPath string, index, totalInLayer int) (*BuildResult, bool, error) {
	needsRebuild, err := o.cache.NeedsRebuild(containerName, configPath)
	if err != nil {
		slog.Warn("Cache check failed, rebuilding",
			"container", containerName,
			"error", err,
		)
		needsRebuild = true
	}

	if needsRebuild {
		return nil, false, nil
	}

	cachedDigest, ok := o.cache.GetCachedDigest(containerName)
	if !ok {
		return nil, false, nil
	}

	slog.Info("Using cached build",
		"container", containerName,
		"digest", cachedDigest[:min(16, len(cachedDigest))],
		"progress", fmt.Sprintf("[%d/%d]", index, totalInLayer),
	)

	imageName := util.FormatImageRefFromName(o.config.Registry, containerName)

	result := &BuildResult{
		ContainerName: containerName,
		ImageName:     imageName,
		Digest:        cachedDigest,
		FullRef:       util.FormatFullRef(imageName, cachedDigest),
		Size:          0,
	}

	if o.config.Push {
		if err := o.pushImage(ctx, result.ImageName, containerName); err != nil {
			return nil, false, err
		}
	}

	return result, true, nil
}

func (o *Orchestrator) buildAndPushContainer(ctx context.Context, containerName, containerfilePath, containerDir string, index, totalInLayer, layerIdx, totalLayers, workerID int, configPath string) (*BuildResult, error) {
	slog.Info("Building container",
		"layer", layerIdx,
		"total_layers", totalLayers,
		"container", containerName,
		"progress", fmt.Sprintf("[%d/%d]", index, totalInLayer),
		"worker", workerID,
	)

	buildStart := time.Now()
	result, err := o.builder.BuildContainer(ctx, containerName, containerfilePath, containerDir)
	buildDuration := time.Since(buildStart)

	if err != nil {
		slog.Error("Build failed",
			"container", containerName,
			"error", err,
			"duration", buildDuration.Round(time.Second),
		)
		return nil, fmt.Errorf("%s: %w", containerName, err)
	}

	if err := o.cache.Record(result, configPath); err != nil {
		slog.Warn("Failed to record build in cache",
			"container", containerName,
			"error", err,
		)
	}

	slog.Info("Build completed",
		"container", containerName,
		"digest", result.Digest[:min(16, len(result.Digest))],
		"size_mb", result.Size/(1024*1024),
		"duration", buildDuration.Round(time.Second),
	)

	if o.config.Push {
		if err := o.pushImage(ctx, result.ImageName, containerName); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (o *Orchestrator) pushImage(ctx context.Context, imageName, containerName string) error {
	slog.Info("Pushing image",
		"container", containerName,
		"image", imageName,
	)
	pushStart := time.Now()
	if err := o.builder.PushImage(ctx, imageName); err != nil {
		slog.Error("Push failed",
			"container", containerName,
			"error", err,
		)
		return fmt.Errorf("%s (push): %w", containerName, err)
	}
	pushDuration := time.Since(pushStart)
	slog.Info("Push completed",
		"container", containerName,
		"duration", pushDuration.Round(time.Second),
	)
	return nil
}

func (o *Orchestrator) collectAndHandleResults(results chan buildOutput, totalInLayer, layerIdx int) error {
	var errors []error
	for i := 0; i < totalInLayer; i++ {
		output := <-results
		if output.err != nil {
			errors = append(errors, output.err)
		} else if output.result != nil {
			o.registry.Record(output.result)
		}
	}

	if len(errors) == 0 {
		return nil
	}

	slog.Error("Layer build failed",
		"layer", layerIdx,
		"failed_builds", len(errors),
	)

	errMsg := fmt.Sprintf("failed to build %d container(s) in layer %d:", len(errors), layerIdx)
	for _, err := range errors {
		errMsg += fmt.Sprintf("\n  - %v", err)
	}
	return fmt.Errorf("%s", errMsg)
}

func (o *Orchestrator) generateContainerfiles(ctx context.Context, layer []string) error {
	builtImages := make(map[string]string)
	allBuilds := o.registry.GetAll()
	for containerName, result := range allBuilds {
		builtImages[containerName] = result.Digest
		if o.config.Registry != "" {
			imageName := fmt.Sprintf("%s/%s", o.config.Registry, containerName)
			builtImages[imageName] = result.Digest
		}
	}

	localImageNames := make([]string, 0, len(o.graph.Containers))
	for containerName := range o.graph.Containers {
		localImageNames = append(localImageNames, containerName)
		if o.config.Registry != "" {
			prefixedName := fmt.Sprintf("%s/%s", o.config.Registry, containerName)
			localImageNames = append(localImageNames, prefixedName)
		}
	}

	for _, containerName := range layer {
		container := o.graph.Containers[containerName]

		outputDir := filepath.Dir(container.ConfigPath)

		gen := generator.New(
			container.Config,
			outputDir,
			o.fs,
			o.config.AlpineClient,
			o.config.AlpineVersion,
			o.config.GitUser,
			o.config.GitPass,
			o.config.Registry,
			o.imageResolver,
		)

		if len(builtImages) > 0 {
			gen.SetBuiltImages(builtImages)
		}

		gen.SetLocalImageNames(localImageNames)

		if err := gen.Generate(); err != nil {
			return fmt.Errorf("generating Containerfile for %s: %w", containerName, err)
		}

		slog.Debug("Generated Containerfile",
			"container", containerName,
			"path", filepath.Join(outputDir, "Containerfile"),
			"built_images_count", len(builtImages),
		)
	}

	return nil
}
