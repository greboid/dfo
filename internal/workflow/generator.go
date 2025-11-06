package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greboid/dfo/internal/apko"
	"github.com/greboid/dfo/internal/config"
	"github.com/greboid/dfo/internal/repository"
	"gopkg.in/yaml.v3"
)

// Generator generates GitHub Actions workflows
type Generator struct {
	repo   *repository.Repository
	config *config.Config
}

// NewGenerator creates a new workflow generator
func NewGenerator(repo *repository.Repository, cfg *config.Config) *Generator {
	return &Generator{
		repo:   repo,
		config: cfg,
	}
}

// Generate creates a GitHub Actions workflow YAML
func (g *Generator) Generate() (string, error) {
	// Load container specs to create matrix
	containerSpecs, err := apko.LoadAllSpecs(g.repo.ContainersDir)
	if err != nil {
		return "", fmt.Errorf("failed to load container specs: %w", err)
	}

	var containers []string
	for name := range containerSpecs {
		containers = append(containers, name)
	}

	// Create workflow structure
	workflow := map[string]interface{}{
		"name": "Build and Push Containers",
		"on": map[string]interface{}{
			"push": map[string]interface{}{
				"branches": []string{"main", "master"},
			},
			"schedule": []map[string]string{
				{"cron": "0 2 * * *"}, // Daily at 2 AM
			},
			"workflow_dispatch": nil,
		},
		"concurrency": map[string]interface{}{
			"group":              "dfo-build",
			"cancel-in-progress": false,
		},
		"jobs": map[string]interface{}{
			"check-updates": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"outputs": map[string]string{
					"updated-packages":    "${{ steps.check.outputs.packages }}",
					"affected-containers": "${{ steps.check.outputs.containers }}",
				},
				"steps": []map[string]interface{}{
					{
						"name": "Checkout",
						"uses": "actions/checkout@v4",
					},
					{
						"name": "Install tools",
						"run":  "go install github.com/greboid/dfo/cmd/dfo@latest",
					},
					{
						"name": "Check for updates",
						"id":   "check",
						"run": strings.Join([]string{
							"dfo update check --dry-run > update-report.txt",
							"echo \"packages=$(cat update-report.txt | grep 'Updated packages' -A 100 | grep -o '- [a-z0-9-]*' | cut -d' ' -f2 | tr '\n' ',' | sed 's/,$//')\" >> $GITHUB_OUTPUT",
							"echo \"containers=$(cat update-report.txt | grep 'Affected containers' -A 100 | grep -o '- [a-z0-9-]*' | cut -d' ' -f2 | tr '\n' ',' | sed 's/,$//')\" >> $GITHUB_OUTPUT",
						}, "\n"),
					},
				},
			},
			"build-packages": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"needs":   []string{"check-updates"},
				"if":      "needs.check-updates.outputs.updated-packages != ''",
				"steps": []map[string]interface{}{
					{
						"name": "Checkout",
						"uses": "actions/checkout@v4",
					},
					{
						"name": "Install melange",
						"uses": "chainguard-dev/actions/setup-melange@main",
					},
					{
						"name": "Install dfo",
						"run":  "go install github.com/greboid/dfo/cmd/dfo@latest",
					},
					{
						"name": "Setup signing key",
						"run": strings.Join([]string{
							fmt.Sprintf("mkdir -p $(dirname %s)", g.config.SigningKey),
							fmt.Sprintf("echo \"$MELANGE_SIGNING_KEY\" > %s", g.config.SigningKey),
							fmt.Sprintf("chmod 600 %s", g.config.SigningKey),
							fmt.Sprintf("openssl rsa -in %s -pubout > %s.pub", g.config.SigningKey, g.config.SigningKey),
						}, "\n"),
						"env": map[string]string{
							"MELANGE_SIGNING_KEY": "${{ secrets.MELANGE_SIGNING_KEY }}",
						},
					},
					{
						"name": "Build updated packages",
						"run":  "dfo package build --all",
					},
					{
						"name": "Cache packages",
						"uses": "actions/cache/save@v4",
						"with": map[string]string{
							"path": "repo",
							"key":  "packages-${{ github.sha }}",
						},
					},
				},
			},
			"build-containers": map[string]interface{}{
				"runs-on": "ubuntu-latest",
				"needs":   []string{"build-packages"},
				"if":      "needs.check-updates.outputs.affected-containers != ''",
				"strategy": map[string]interface{}{
					"matrix": map[string]interface{}{
						"container": containers,
					},
					"fail-fast": false,
				},
				"steps": []map[string]interface{}{
					{
						"name": "Checkout",
						"uses": "actions/checkout@v4",
					},
					{
						"name": "Install apko",
						"uses": "chainguard-dev/actions/setup-apko@main",
					},
					{
						"name": "Install dfo",
						"run":  "go install github.com/greboid/dfo/cmd/dfo@latest",
					},
					{
						"name": "Restore package cache",
						"uses": "actions/cache/restore@v4",
						"with": map[string]string{
							"path": "repo",
							"key":  "packages-${{ github.sha }}",
						},
					},
					{
						"name": "Build container",
						"run":  "dfo container build ${{ matrix.container }} --output output",
					},
					{
						"name": "Login to registries",
						"run":  "echo '${{ secrets.REGISTRY_PASSWORD }}' | docker login ${{ secrets.REGISTRY_URL }} -u ${{ secrets.REGISTRY_USERNAME }} --password-stdin",
					},
					{
						"name": "Push container",
						"run":  "dfo container push ${{ matrix.container }} --output output",
					},
				},
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow: %w", err)
	}

	return string(data), nil
}

// WriteToFile writes the workflow to a file
func (g *Generator) WriteToFile(workflowYAML string, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(workflowYAML), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
