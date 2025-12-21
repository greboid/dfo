package workflow

import (
	"fmt"
	"os"
	"sort"

	"github.com/greboid/dfo/pkg/graph"
	"gopkg.in/yaml.v3"
)

func Generate(
	depGraph *graph.Graph,
	layers [][]string,
	outputPath string,
) error {
	workflow := &Workflow{
		Name:        "Update Containers",
		Concurrency: "dockerfiles",
		Permissions: map[string]string{
			"contents": "write",
		},
		On: Triggers{
			WorkflowDispatch: &DispatchTrigger{},
			WorkflowRun: &RunTrigger{
				Workflows: []string{"Update Workflow"},
				Types:     []string{"completed"},
			},
		},
		Jobs: make(map[string]Job),
	}

	workflow.Jobs["setup-cache"] = Job{
		Name:   "Setup cache",
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{
				Name: "Install latest dfo",
				Uses: "mattdowdell/go-installer@v0.3.0",
				With: map[string]string{
					"package": "github.com/greboid/dfo",
				},
			},
		},
	}

	for _, layer := range layers {
		for _, containerName := range layer {
			needs := buildNeedsArray(depGraph, containerName)

			workflow.Jobs[containerName] = Job{
				Name:    fmt.Sprintf("Build %s", containerName),
				Needs:   needs,
				Uses:    "./.github/workflows/container-builder.yml",
				Secrets: "inherit",
				With: map[string]string{
					"PROJECT_NAME": containerName,
				},
			}
		}
	}

	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("marshaling workflow: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing workflow file: %w", err)
	}

	return nil
}

func buildNeedsArray(depGraph *graph.Graph, containerName string) []string {
	needs := []string{"setup-cache"}

	container := depGraph.Containers[containerName]

	for _, dep := range container.Dependencies {
		if _, exists := depGraph.Containers[dep]; exists {
			needs = append(needs, dep)
		}
	}

	if len(needs) > 1 {
		sort.Strings(needs[1:])
	}

	return needs
}
