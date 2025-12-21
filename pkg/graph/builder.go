package graph

import (
	"log/slog"

	"github.com/greboid/dfo/pkg/config"
)

func Build(
	configs map[string]*config.BuildConfig,
	paths map[string]string,
) (*Graph, error) {
	graph := &Graph{
		Containers: make(map[string]*Container),
	}

	for containerName, cfg := range configs {
		container := &Container{
			Name:         containerName,
			ConfigPath:   paths[containerName],
			Config:       cfg,
			Dependencies: extractDependencies(cfg),
		}

		graph.Containers[containerName] = container

		slog.Debug("added container to graph",
			"name", containerName,
			"dependencies", container.Dependencies)
	}

	for _, container := range graph.Containers {
		for _, dep := range container.Dependencies {
			if _, exists := graph.Containers[dep]; !exists {
				slog.Warn("dependency not found in graph",
					"container", container.Name,
					"dependency", dep,
					"note", "might be external image")
			}
		}
	}

	return graph, nil
}

func extractDependencies(cfg *config.BuildConfig) []string {
	seen := make(map[string]bool)
	var deps []string

	for _, stage := range cfg.Stages {
		if stage.Environment.BaseImage != "" {
			baseImage := stage.Environment.BaseImage

			if !seen[baseImage] {
				seen[baseImage] = true
				deps = append(deps, baseImage)
			}
		}
	}

	return deps
}
