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
	workflow := createWorkflowSkeleton()

	addSetupCacheJob(workflow)
	addContainerBuildJobs(workflow, depGraph, layers)
	addUpdateJobs(workflow, layers)
	addCommitJob(workflow, layers)

	return writeWorkflowFile(workflow, outputPath)
}

func createWorkflowSkeleton() *Workflow {
	return &Workflow{
		Name:        "Update Containers",
		Concurrency: "dockerfiles",
		Permissions: map[string]string{"contents": "write"},
		On: Triggers{
			WorkflowDispatch: &DispatchTrigger{},
			WorkflowRun: &RunTrigger{
				Workflows: []string{"Update Workflow"},
				Types:     []string{"completed"},
			},
		},
		Jobs: make(map[string]Job),
	}
}

func addSetupCacheJob(workflow *Workflow) {
	workflow.Jobs["setup-cache"] = Job{
		Name:   "Setup cache",
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{Name: "Setup Go", Uses: "actions/setup-go@v6", With: map[string]string{"go-version": "stable", "cache": "false"}},
			{Name: "Install latest dfo", Uses: "mattdowdell/go-installer@v0.3.0", With: map[string]string{"package": "github.com/greboid/dfo"}},
		},
	}
}

func addContainerBuildJobs(workflow *Workflow, depGraph *graph.Graph, layers [][]string) {
	var previousUpdateJob string

	for layerIdx, layer := range layers {
		for _, containerName := range layer {
			needs := buildNeedsArray(depGraph, containerName, previousUpdateJob)
			workflow.Jobs[containerName] = createContainerBuildJob(containerName, needs)
		}
		previousUpdateJob = fmt.Sprintf("update-layer-%d", layerIdx)
	}
}

func createContainerBuildJob(containerName string, needs []string) Job {
	return Job{
		Name:   fmt.Sprintf("Build %s", containerName),
		Needs:  needs,
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{Name: "Checkout code", Uses: "actions/checkout@v6"},
			{Name: "Setup Go", Uses: "actions/setup-go@v6", With: map[string]string{"go-version": "stable", "cache": "false"}},
			{Name: "Install latest dfo", Uses: "mattdowdell/go-installer@v0.3.0", With: map[string]string{"package": "github.com/greboid/dfo"}},
			{Name: fmt.Sprintf("Build %s", containerName), Run: fmt.Sprintf("dfo single --build --registry \"${{ secrets.REGISTRY }}\" %s", containerName)},
		},
	}
}

func addUpdateJobs(workflow *Workflow, layers [][]string) {
	for layerIdx, layer := range layers {
		updateJobName := fmt.Sprintf("update-layer-%d", layerIdx)
		updateNeeds := append([]string{"setup-cache"}, layer...)
		workflow.Jobs[updateJobName] = createUpdateJob(layerIdx, updateNeeds, layer)
	}
}

func createUpdateJob(layerIdx int, needs []string, layer []string) Job {
	return Job{
		Name:   fmt.Sprintf("Update layer %d tags", layerIdx),
		Needs:  needs,
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{Name: "Checkout code", Uses: "actions/checkout@v6", With: map[string]string{"token": "${{ secrets.GITHUB_TOKEN }}"}},
			{Name: "Setup Go", Uses: "actions/setup-go@v6", With: map[string]string{"go-version": "stable", "cache": "false"}},
			{Name: "Install latest dfo", Uses: "mattdowdell/go-installer@v0.3.0", With: map[string]string{"package": "github.com/greboid/dfo"}},
			{Name: fmt.Sprintf("Update layer %d Containerfiles", layerIdx), Run: buildUpdateScript(layer)},
		},
	}
}

func addCommitJob(workflow *Workflow, layers [][]string) {
	if len(layers) == 0 {
		return
	}

	workflow.Jobs["commit-changes"] = Job{
		Name:   "Commit updated files",
		Needs:  []string{fmt.Sprintf("update-layer-%d", len(layers)-1)},
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{Name: "Checkout code", Uses: "actions/checkout@v6", With: map[string]string{"token": "${{ secrets.GITHUB_TOKEN }}"}},
			{Name: "Setup Go", Uses: "actions/setup-go@v6", With: map[string]string{"go-version": "stable", "cache": "false"}},
			{Name: "Install latest dfo", Uses: "mattdowdell/go-installer@v0.3.0", With: map[string]string{"package": "github.com/greboid/dfo"}},
			{Name: "Update all Containerfiles with built image digests", Run: buildFinalUpdateScript(layers)},
			{Name: "Commit and push changes", Run: getCommitScript()},
		},
	}
}

func getCommitScript() string {
	return `git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"
git add .
if git diff --staged --quiet; then
  echo "No changes to commit"
else
  git commit -m "Update Containerfiles and BOMs with built image digests"
  git pull --rebase origin $(git branch --show-current)
  git push
fi`
}

func writeWorkflowFile(workflow *Workflow, outputPath string) error {
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
		script += fmt.Sprintf("dfo single --registry \"${{ secrets.REGISTRY }}\" %s\n", containerName)
	}
	return script
}

func buildFinalUpdateScript(layers [][]string) string {
	script := "set -e\necho 'Updating all Containerfiles with built image digests...'\n"
	for _, layer := range layers {
		for _, containerName := range layer {
			script += fmt.Sprintf("dfo single --registry \"${{ secrets.REGISTRY }}\" %s\n", containerName)
		}
	}
	return script
}
