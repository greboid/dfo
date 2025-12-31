package workflow

import (
	"fmt"
	"os"
	"path/filepath"
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

	var previousUpdateJob string

	for layerIdx, layer := range layers {
		for _, containerName := range layer {
			needs := buildNeedsArray(depGraph, containerName, previousUpdateJob)

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

		updateJobName := fmt.Sprintf("update-layer-%d", layerIdx)
		updateNeeds := []string{"setup-cache"}
		updateNeeds = append(updateNeeds, layer...)

		workflow.Jobs[updateJobName] = Job{
			Name:   fmt.Sprintf("Update layer %d tags", layerIdx),
			Needs:  updateNeeds,
			RunsOn: "ubuntu-latest",
			Steps: []Step{
				{
					Name: "Checkout code",
					Uses: "actions/checkout@v4",
					With: map[string]string{
						"token": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
				{
					Name: "Install latest dfo",
					Uses: "mattdowdell/go-installer@v0.3.0",
					With: map[string]string{
						"package": "github.com/greboid/dfo",
					},
				},
				{
					Name: fmt.Sprintf("Update layer %d Containerfiles", layerIdx),
					Run:  buildUpdateScript(layer),
				},
			},
		}

		previousUpdateJob = updateJobName
	}

	if len(layers) > 0 {
		commitNeeds := []string{fmt.Sprintf("update-layer-%d", len(layers)-1)}
		workflow.Jobs["commit-changes"] = Job{
			Name:   "Commit updated files",
			Needs:  commitNeeds,
			RunsOn: "ubuntu-latest",
			Steps: []Step{
				{
					Name: "Checkout code",
					Uses: "actions/checkout@v4",
					With: map[string]string{
						"token": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
				{
					Name: "Install latest dfo",
					Uses: "mattdowdell/go-installer@v0.3.0",
					With: map[string]string{
						"package": "github.com/greboid/dfo",
					},
				},
				{
					Name: "Update all Containerfiles with built image digests",
					Run:  buildFinalUpdateScript(layers),
				},
				{
					Name: "Commit and push changes",
					Run: `git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"
git add .
if git diff --staged --quiet; then
  echo "No changes to commit"
else
  git commit -m "Update Containerfiles and BOMs with built image digests"
  git push
fi`,
				},
			},
		}
	}

	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("marshaling workflow: %w", err)
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing workflow file: %w", err)
	}

	return nil
}

func buildNeedsArray(depGraph *graph.Graph, containerName string, previousUpdateJob string) []string {
	needs := []string{"setup-cache"}

	if previousUpdateJob != "" {
		needs = append(needs, previousUpdateJob)
	}

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

func buildUpdateScript(layer []string) string {
	script := "set -e\n"
	for _, containerName := range layer {
		script += fmt.Sprintf("echo 'Updating %s...'\n", containerName)
		script += fmt.Sprintf("dfo single -d %s\n", containerName)
	}
	return script
}

func buildFinalUpdateScript(layers [][]string) string {
	script := "set -e\necho 'Updating all Containerfiles with built image digests...'\n"
	for _, layer := range layers {
		for _, containerName := range layer {
			script += fmt.Sprintf("dfo single -d %s\n", containerName)
		}
	}
	return script
}
