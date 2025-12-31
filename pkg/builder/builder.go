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
	imageResolver := images.NewResolver(cfg.Registry, true)

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
	type buildJob struct {
		containerName string
		index         int
	}

	type buildOutput struct {
		result *BuildResult
		err    error
		index  int
	}

	totalInLayer := len(layer)
	jobs := make(chan buildJob, totalInLayer)
	results := make(chan buildOutput, totalInLayer)

	var wg sync.WaitGroup
	workers := o.config.Concurrency
	if workers <= 0 {
		workers = 5
	}
	if workers > totalInLayer {
		workers = totalInLayer
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range jobs {
				containerName := job.containerName
				container := o.graph.Containers[containerName]
				containerDir := filepath.Dir(container.ConfigPath)
				containerfilePath := filepath.Join(containerDir, "Containerfile")

				ignorePath := filepath.Join(containerDir, "IGNORE")
				if _, err := o.fs.Stat(ignorePath); err == nil {
					slog.Info("Skipping build (IGNORE file present)",
						"container", containerName,
						"progress", fmt.Sprintf("[%d/%d]", job.index+1, totalInLayer),
					)
					results <- buildOutput{result: nil, index: job.index}
					continue
				}

				needsRebuild, err := o.cache.NeedsRebuild(containerName, container.ConfigPath)
				if err != nil {
					slog.Warn("Cache check failed, rebuilding",
						"container", containerName,
						"error", err,
					)
					needsRebuild = true
				}

				var result *BuildResult
				if !needsRebuild {
					cachedDigest, ok := o.cache.GetCachedDigest(containerName)
					if ok {
						slog.Info("Using cached build",
							"container", containerName,
							"digest", cachedDigest[:min(16, len(cachedDigest))],
							"progress", fmt.Sprintf("[%d/%d]", job.index+1, totalInLayer),
						)

						imageName := containerName
						if o.config.Registry != "" {
							imageName = fmt.Sprintf("%s/%s:latest", o.config.Registry, containerName)
						}

						result = &BuildResult{
							ContainerName: containerName,
							ImageName:     imageName,
							Digest:        cachedDigest,
							FullRef:       fmt.Sprintf("%s@%s", imageName, cachedDigest),
							Size:          0,
						}

						if o.config.Push {
							slog.Info("Pushing cached image",
								"container", containerName,
								"image", result.ImageName,
							)
							pushStart := time.Now()
							if err := o.builder.PushImage(ctx, result.ImageName); err != nil {
								slog.Error("Push failed",
									"container", containerName,
									"error", err,
								)
								results <- buildOutput{err: fmt.Errorf("%s (push): %w", containerName, err), index: job.index}
								continue
							}
							pushDuration := time.Since(pushStart)
							slog.Info("Push completed",
								"container", containerName,
								"duration", pushDuration.Round(time.Second),
							)
						}

						results <- buildOutput{result: result, index: job.index}
						continue
					}
				}

				slog.Info("Building container",
					"layer", layerIdx,
					"total_layers", totalLayers,
					"container", containerName,
					"progress", fmt.Sprintf("[%d/%d]", job.index+1, totalInLayer),
					"worker", workerID,
				)

				buildStart := time.Now()
				result, err = o.builder.BuildContainer(ctx, containerName, containerfilePath, containerDir)
				buildDuration := time.Since(buildStart)

				if err != nil {
					slog.Error("Build failed",
						"container", containerName,
						"error", err,
						"duration", buildDuration.Round(time.Second),
					)
					results <- buildOutput{err: fmt.Errorf("%s: %w", containerName, err), index: job.index}
					continue
				}

				if err := o.cache.Record(result, container.ConfigPath); err != nil {
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
					slog.Info("Pushing image",
						"container", containerName,
						"image", result.ImageName,
					)
					pushStart := time.Now()
					if err := o.builder.PushImage(ctx, result.ImageName); err != nil {
						slog.Error("Push failed",
							"container", containerName,
							"error", err,
						)
						results <- buildOutput{err: fmt.Errorf("%s (push): %w", containerName, err), index: job.index}
						continue
					}
					pushDuration := time.Since(pushStart)
					slog.Info("Push completed",
						"container", containerName,
						"duration", pushDuration.Round(time.Second),
					)
				}

				results <- buildOutput{result: result, index: job.index}
			}
		}(w)
	}

	for i, containerName := range layer {
		jobs <- buildJob{containerName: containerName, index: i}
	}
	close(jobs)

	wg.Wait()
	close(results)

	var errors []error
	for output := range results {
		if output.err != nil {
			errors = append(errors, output.err)
		} else if output.result != nil {
			o.registry.Record(output.result)
		}
	}

	if len(errors) > 0 {
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

	return nil
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
