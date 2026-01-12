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

	addContainerBuildJobs(workflow, depGraph, layers)
	addCommitJob(workflow, depGraph)

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

func addContainerBuildJobs(workflow *Workflow, depGraph *graph.Graph, layers [][]string) {
	for _, layer := range layers {
		for _, containerName := range layer {
			needs := buildNeedsArray(depGraph, containerName)
			workflow.Jobs[containerName] = createContainerBuildJob(containerName, needs)
		}
	}
}

func createContainerBuildJob(containerName string, needs []string) Job {
	return Job{
		Name:   fmt.Sprintf("Build %s", containerName),
		Needs:  needs,
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{Name: "Checkout code", Uses: "actions/checkout@v6"},
			{Name: "Set up Go", Uses: "actions/setup-go@v6", With: map[string]string{"go-version": "stable", "cache": "false"}},
			{Name: "Install latest dfo", Uses: "mattdowdell/go-installer@v0.3.0", With: map[string]string{"package": "github.com/greboid/dfo"}},
			{Name: "Login to registry", Uses: "redhat-actions/podman-login@v1", With: map[string]string{"registry": "${{ secrets.REGISTRY }}", "username": "${{ secrets.REGISTRY_USER }}", "password": "${{ secrets.REGISTRY_PASS }}"}},
			{Name: fmt.Sprintf("Build %s", containerName), Run: fmt.Sprintf("dfo single --build --push --registry \"${{ secrets.REGISTRY }}\" %s", containerName)},
		},
	}
}

func addCommitJob(workflow *Workflow, depGraph *graph.Graph) {
	if len(depGraph.Containers) == 0 {
		return
	}

	var allContainers []string
	for name := range depGraph.Containers {
		allContainers = append(allContainers, name)
	}
	sort.Strings(allContainers)

	workflow.Jobs["commit-changes"] = Job{
		Name:   "Commit updated files",
		Needs:  allContainers,
		RunsOn: "ubuntu-latest",
		Steps: []Step{
			{Name: "Checkout code", Uses: "actions/checkout@v6", With: map[string]string{"token": "${{ secrets.GITHUB_TOKEN }}"}},
			{Name: "Setup Go", Uses: "actions/setup-go@v6", With: map[string]string{"go-version": "stable", "cache": "false"}},
			{Name: "Install latest dfo", Uses: "mattdowdell/go-installer@v0.3.0", With: map[string]string{"package": "github.com/greboid/dfo"}},
			{Name: "Update all Containerfiles with built image digests", Run: buildFinalUpdateScript(depGraph)},
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

func buildNeedsArray(depGraph *graph.Graph, containerName string) []string {
	needs := []string{}
	container := depGraph.Containers[containerName]

	for _, dep := range container.Dependencies {
		if _, exists := depGraph.Containers[dep]; exists {
			needs = append(needs, dep)
		}
	}

	if len(needs) > 0 {
		sort.Strings(needs)
	}

	return needs
}

func buildFinalUpdateScript(depGraph *graph.Graph) string {
	script := "set -e\necho 'Updating all Containerfiles with built image digests...'\n"
	var allContainers []string
	for name := range depGraph.Containers {
		allContainers = append(allContainers, name)
	}
	sort.Strings(allContainers)
	for _, containerName := range allContainers {
		script += fmt.Sprintf("dfo single --registry \"${{ secrets.REGISTRY }}\" %s\n", containerName)
	}
	return script
}
