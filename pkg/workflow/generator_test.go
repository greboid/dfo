package workflow

import (
	"strings"
	"testing"

	"github.com/greboid/dfo/pkg/graph"
)

func TestCreateWorkflowSkeleton(t *testing.T) {
	workflow := createWorkflowSkeleton()

	if workflow.Name != "Update Containers" {
		t.Errorf("workflow.Name = %q, want \"Update Containers\"", workflow.Name)
	}

	if workflow.Concurrency != "dockerfiles" {
		t.Errorf("workflow.Concurrency = %q, want \"dockerfiles\"", workflow.Concurrency)
	}

	if workflow.Permissions["contents"] != "write" {
		t.Errorf("workflow.Permissions[\"contents\"] = %q, want \"write\"", workflow.Permissions["contents"])
	}

	if workflow.On.WorkflowDispatch == nil {
		t.Error("workflow.On.WorkflowDispatch should not be nil")
	}

	if workflow.On.WorkflowRun == nil {
		t.Error("workflow.On.WorkflowRun should not be nil")
	}

	if workflow.Jobs == nil {
		t.Error("workflow.Jobs should be initialized")
	}
}

func TestAddSetupCacheJob(t *testing.T) {
	workflow := createWorkflowSkeleton()
	addSetupCacheJob(workflow)

	job, exists := workflow.Jobs["setup-cache"]
	if !exists {
		t.Fatal("setup-cache job not found")
	}

	if job.Name != "Setup cache" {
		t.Errorf("job.Name = %q, want \"Setup cache\"", job.Name)
	}

	if job.RunsOn != "ubuntu-latest" {
		t.Errorf("job.RunsOn = %q, want \"ubuntu-latest\"", job.RunsOn)
	}

	if len(job.Steps) != 2 {
		t.Errorf("len(job.Steps) = %d, want 2", len(job.Steps))
	}
}

func TestCreateContainerBuildJob(t *testing.T) {
	tests := []struct {
		name     string
		jobName  string
		needs    []string
		checkJob func(t *testing.T, job Job)
	}{
		{
			name:    "basic build job",
			jobName: "test-container",
			needs:   []string{"setup-cache"},
			checkJob: func(t *testing.T, job Job) {
				if job.Name != "Build test-container" {
					t.Errorf("job.Name = %q, want \"Build test-container\"", job.Name)
				}
				if job.RunsOn != "ubuntu-latest" {
					t.Errorf("job.RunsOn = %q, want \"ubuntu-latest\"", job.RunsOn)
				}
				if len(job.Steps) != 4 {
					t.Errorf("len(job.Steps) = %d, want 4", len(job.Steps))
				}
			},
		},
		{
			name:    "job with dependencies",
			jobName: "app-container",
			needs:   []string{"setup-cache", "base-image", "lib-image"},
			checkJob: func(t *testing.T, job Job) {
				if len(job.Needs) != 3 {
					t.Errorf("len(job.Needs) = %d, want 3", len(job.Needs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := createContainerBuildJob(tt.jobName, tt.needs)
			tt.checkJob(t, job)
		})
	}
}

func TestCreateUpdateJob(t *testing.T) {
	tests := []struct {
		name     string
		layerIdx int
		needs    []string
		layer    []string
		checkJob func(t *testing.T, job Job)
	}{
		{
			name:     "basic update job",
			layerIdx: 0,
			needs:    []string{"setup-cache", "container1", "container2"},
			layer:    []string{"container1", "container2"},
			checkJob: func(t *testing.T, job Job) {
				if job.Name != "Update layer 0 tags" {
					t.Errorf("job.Name = %q, want \"Update layer 0 tags\"", job.Name)
				}
				if job.RunsOn != "ubuntu-latest" {
					t.Errorf("job.RunsOn = %q, want \"ubuntu-latest\"", job.RunsOn)
				}
				if len(job.Needs) != 3 {
					t.Errorf("len(job.Needs) = %d, want 3", len(job.Needs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := createUpdateJob(tt.layerIdx, tt.needs, tt.layer)
			tt.checkJob(t, job)
		})
	}
}

func TestGetCommitScript(t *testing.T) {
	script := getCommitScript()

	if script == "" {
		t.Error("getCommitScript() returned empty string")
	}

	expectedParts := []string{
		"git config user.name",
		"git config user.email",
		"git add .",
		"git commit -m",
		"git pull --rebase",
		"git push",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("commit script missing expected part: %q", part)
		}
	}
}

func TestBuildNeedsArray(t *testing.T) {
	tests := []struct {
		name              string
		depGraph          *graph.Graph
		containerName     string
		previousUpdateJob string
		expected          []string
	}{
		{
			name: "no dependencies",
			depGraph: &graph.Graph{
				Containers: map[string]*graph.Container{
					"test": {Name: "test"},
				},
			},
			containerName: "test",
			expected:      []string{"setup-cache"},
		},
		{
			name: "with dependencies",
			depGraph: &graph.Graph{
				Containers: map[string]*graph.Container{
					"test": {
						Name:         "test",
						Dependencies: []string{"base", "lib"},
					},
					"base": {
						Name: "base",
					},
					"lib": {
						Name: "lib",
					},
				},
			},
			containerName: "test",
			expected:      []string{"base", "lib", "setup-cache"},
		},
		{
			name: "with previous update job",
			depGraph: &graph.Graph{
				Containers: map[string]*graph.Container{
					"test": {
						Name: "test",
					},
				},
			},
			containerName:     "test",
			previousUpdateJob: "update-layer-0",
			expected:          []string{"setup-cache", "update-layer-0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNeedsArray(tt.depGraph, tt.containerName, tt.previousUpdateJob)
			if len(result) != len(tt.expected) {
				t.Errorf("buildNeedsArray() length = %d, want %d", len(result), len(tt.expected))
			}
			for _, exp := range tt.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildNeedsArray() missing expected value: %q", exp)
				}
			}
		})
	}
}

func TestBuildUpdateScript(t *testing.T) {
	tests := []struct {
		name     string
		layer    []string
		expected []string
	}{
		{
			name:     "single container",
			layer:    []string{"app"},
			expected: []string{"set -e", "echo 'Updating app...'", "dfo single --registry"},
		},
		{
			name:     "multiple containers",
			layer:    []string{"app", "api", "worker"},
			expected: []string{"echo 'Updating app...'", "echo 'Updating api...'", "echo 'Updating worker...'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildUpdateScript(tt.layer)
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("buildUpdateScript() missing expected: %q", exp)
				}
			}
		})
	}
}

func TestBuildFinalUpdateScript(t *testing.T) {
	tests := []struct {
		name     string
		layers   [][]string
		expected []string
	}{
		{
			name:     "single layer",
			layers:   [][]string{{"app", "api"}},
			expected: []string{"dfo single --registry", "app", "api"},
		},
		{
			name: "multiple layers",
			layers: [][]string{
				{"base", "lib"},
				{"app", "api"},
			},
			expected: []string{"base", "lib", "app", "api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFinalUpdateScript(tt.layers)
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("buildFinalUpdateScript() missing expected: %q", exp)
				}
			}
		})
	}
}
