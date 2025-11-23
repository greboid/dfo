package generator

import (
	"fmt"
	"path"
	"strings"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/pipelines"
	"github.com/greboid/dfo/pkg/util"
)

const (
	dirPerms  = 0755
	filePerms = 0644
)

type Generator struct {
	config         *config.BuildConfig
	outputDir      string
	outputFilename string
	fs             util.WritableFS
}

func New(cfg *config.BuildConfig, outputDir string, fs util.WritableFS) *Generator {
	return &Generator{
		config:         cfg,
		outputDir:      outputDir,
		outputFilename: "Containerfile.gotpl",
		fs:             fs,
	}
}

func buildFetchCommand(url, dest string, extract bool) string {
	if extract {
		return util.WrapRun(fmt.Sprintf("curl -fsSL %q | tar -xz -C %q", url, dest))
	}
	return util.WrapRun(fmt.Sprintf("curl -fsSL -o %s %q", dest, url))
}

func (g *Generator) SetOutputFilename(filename string) {
	g.outputFilename = filename
}

func (g *Generator) Generate() error {
	if err := g.validateVariableReferences(); err != nil {
		return fmt.Errorf("variable validation: %w", err)
	}

	if err := g.fs.MkdirAll(g.outputDir, dirPerms); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := g.generateDockerfile(); err != nil {
		return fmt.Errorf("generating Dockerfile: %w", err)
	}

	return nil
}

func (g *Generator) validateVariableReferences() error {
	for _, stage := range g.config.Stages {
		for i, step := range stage.Pipeline {
			stepContext := fmt.Sprintf("stage %q step %d", stage.Name, i+1)
			if step.Name != "" {
				stepContext = fmt.Sprintf("stage %q step %q", stage.Name, step.Name)
			}

			if step.Run != "" {
				if err := util.ValidateVariableReferences(step.Run, g.config.Vars, stepContext+" (run)"); err != nil {
					return err
				}
			}

			if step.Fetch != nil && step.Fetch.URL != "" {
				if err := util.ValidateVariableReferences(step.Fetch.URL, g.config.Vars, stepContext+" (fetch.url)"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *Generator) generateDockerfile() error {
	var b strings.Builder
	b.Grow(4096)

	for i, stage := range g.config.Stages {
		isFinalStage := i == len(g.config.Stages)-1
		stageContent, err := g.generateStage(stage, isFinalStage)
		if err != nil {
			return fmt.Errorf("generating stage %q: %w", stage.Name, err)
		}
		b.WriteString(stageContent)
		b.WriteString("\n")
	}

	outputPath := path.Join(g.outputDir, g.outputFilename)
	if err := g.fs.WriteFile(outputPath, []byte(b.String()), filePerms); err != nil {
		return fmt.Errorf("writing %s: %w", g.outputFilename, err)
	}

	return nil
}

func (g *Generator) generateStage(stage config.Stage, isFinalStage bool) (string, error) {
	var b strings.Builder
	b.Grow(2048)

	if stage.Environment.ExternalImage != "" {
		if isFinalStage {
			b.WriteString(fmt.Sprintf("FROM %s\n\n", stage.Environment.ExternalImage))
		} else {
			b.WriteString(fmt.Sprintf("FROM %s AS %s\n\n", stage.Environment.ExternalImage, stage.Name))
		}
	} else {
		if isFinalStage {
			b.WriteString(fmt.Sprintf("FROM {{image %q}}\n\n", stage.Environment.BaseImage))
		} else {
			b.WriteString(fmt.Sprintf("FROM {{image %q}} AS %s\n\n", stage.Environment.BaseImage, stage.Name))
		}
	}

	content, err := g.generateStageContent(stage.Environment, stage.Pipeline, isFinalStage)
	if err != nil {
		return "", err
	}
	b.WriteString(content)

	return b.String(), nil
}

func (g *Generator) generateStageContent(env config.Environment, pipeline []config.PipelineStep, isFinalStage bool) (string, error) {
	var b strings.Builder
	b.Grow(1024)

	if len(env.Args) > 0 {
		for _, key := range util.SortedKeys(env.Args) {
			b.WriteString(fmt.Sprintf("ARG %s=\"%s\"\n", key, env.Args[key]))
		}
		b.WriteString("\n")
	}

	if len(g.config.Package.Labels) > 0 && isFinalStage {
		b.WriteString(util.FormatMapDirectives("LABEL", g.config.Package.Labels))
	}

	if len(env.Environment) > 0 {
		b.WriteString(util.FormatMapDirectives("ENV", env.Environment))
	}

	if len(env.Packages) > 0 {
		b.WriteString(g.generatePackageInstallForEnv(env))
		b.WriteString("\n")
	}

	if len(env.RootfsPackages) > 0 {
		b.WriteString(g.generateRootfsPackageInstallForEnv(env))
		b.WriteString("\n")
	}

	if env.WorkDir != "" {
		b.WriteString(fmt.Sprintf("WORKDIR %s\n\n", env.WorkDir))
	}

	for _, step := range pipeline {
		stepContent, err := g.generatePipelineStep(step)
		if err != nil {
			return "", err
		}
		if stepContent != "" {
			if step.Name != "" {
				b.WriteString(fmt.Sprintf("# %s\n", step.Name))
			}
			b.WriteString(stepContent)
			b.WriteString("\n")
		}
	}

	if len(env.Expose) > 0 {
		for _, port := range env.Expose {
			b.WriteString(fmt.Sprintf("EXPOSE %s\n", port))
		}
		b.WriteString("\n")
	}

	b.WriteString(util.FormatDockerfileArray("VOLUME", env.Volume))

	if env.StopSignal != "" {
		b.WriteString(fmt.Sprintf("STOPSIGNAL %s\n\n", env.StopSignal))
	}

	if env.User != "" {
		b.WriteString(fmt.Sprintf("USER %s\n\n", env.User))
	}

	b.WriteString(util.FormatDockerfileArray("ENTRYPOINT", env.Entrypoint))
	b.WriteString(util.FormatDockerfileArray("CMD", env.Cmd))

	return b.String(), nil
}

func (g *Generator) generatePackageInstallForEnv(env config.Environment) string {
	var b strings.Builder
	b.Grow(512)

	b.WriteString("# Install packages\n")
	b.WriteString("RUN set -eux; \\\n")
	b.WriteString("    apk add --no-cache \\\n")

	pkgList := util.BuildPackageList(env.Packages)
	b.WriteString(fmt.Sprintf("    {{- range $key, $value := alpine_packages %s}}\n", pkgList))
	b.WriteString("        {{$key}}={{$value}} \\\n")
	b.WriteString("    {{- end}}\n")
	b.WriteString("    ;\n")

	return b.String()
}

func (g *Generator) generateRootfsPackageInstallForEnv(env config.Environment) string {
	var b strings.Builder
	b.Grow(512)

	b.WriteString("# Install packages into rootfs\n")
	b.WriteString("RUN \\\n")

	pkgList := util.BuildPackageList(env.RootfsPackages)
	b.WriteString(fmt.Sprintf("{{- range $key, $value := alpine_packages %s}}\n", pkgList))
	b.WriteString("    apk add --no-cache {{$key}}={{$value}}; \\\n")
	b.WriteString("    apk info -qL {{$key}} | rsync -aq --files-from=- / /rootfs/; \\\n")
	b.WriteString("{{- end}}\n")
	b.WriteString("    true\n")

	return b.String()
}

func (g *Generator) generatePipelineStep(step config.PipelineStep) (string, error) {
	var b strings.Builder

	if step.Uses != "" {
		content, err := g.generateIncludeCall(step)
		if err != nil {
			return "", err
		}
		b.WriteString(content)
		return b.String(), nil
	}

	if step.Run != "" {
		run := util.ExpandVars(step.Run, g.config.Vars)

		if len(step.BuildDeps) > 0 {
			b.WriteString(g.generateRunWithBuildDeps(run, step.BuildDeps))
		} else {
			b.WriteString(g.formatRunCommand(run))
		}
		return b.String(), nil
	}

	if step.Fetch != nil {
		b.WriteString(g.generateFetchStep(step.Fetch))
		return b.String(), nil
	}

	if step.Copy != nil {
		copyCmd := "COPY"
		if step.Copy.FromStage != "" {
			copyCmd += fmt.Sprintf(" --from=%s", step.Copy.FromStage)
		}
		if step.Copy.Chown != "" {
			copyCmd += fmt.Sprintf(" --chown=%s", step.Copy.Chown)
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", copyCmd, step.Copy.From, step.Copy.To))
		return b.String(), nil
	}

	return "", nil
}

func (g *Generator) generateRunWithBuildDeps(runCmd string, buildDeps []string) string {
	var b strings.Builder

	pkgList := util.BuildPackageList(buildDeps)
	b.WriteString("RUN apk add --no-cache --virtual .build-deps \\\n")
	b.WriteString(fmt.Sprintf("  {{- range $key, $value := alpine_packages %s}}\n", pkgList))
	b.WriteString("  {{$key}}={{$value}} \\\n")
	b.WriteString("  {{- end}}\n")
	b.WriteString("  ; \\\n")

	lines := strings.Split(strings.TrimSpace(runCmd), "\n")
	for _, line := range lines {
		b.WriteString(util.FormatShellLineWithContinuation(line, "  "))
	}

	b.WriteString("  apk del --no-network .build-deps\n")

	return b.String()
}

func (g *Generator) generateFetchStep(fetch *config.FetchStep) string {
	dest := fetch.Destination
	if dest == "" {
		dest = "/tmp/download"
	}

	url := util.ExpandVars(fetch.URL, g.config.Vars)
	return buildFetchCommand(url, dest, fetch.Extract)
}

func (g *Generator) generateIncludeCall(step config.PipelineStep) (string, error) {
	pipeline, exists := pipelines.Registry[step.Uses]
	if !exists {
		return "", fmt.Errorf("pipeline %q not found (referenced in step %q)", step.Uses, step.Name)
	}

	if err := pipelines.ValidateParams(step.Uses, step.With); err != nil {
		return "", err
	}

	result, err := pipeline(step.With)
	if err != nil {
		return "", fmt.Errorf("executing pipeline %q: %w", step.Uses, err)
	}

	// Merge any user-specified build-deps with pipeline-declared build-deps
	allBuildDeps := mergeDeps(result.BuildDeps, step.BuildDeps)

	// Generate the step content
	var stepsContent strings.Builder
	for _, pipelineStep := range result.Steps {
		if pipelineStep.Name != "" {
			stepsContent.WriteString(fmt.Sprintf("# %s\n", pipelineStep.Name))
		}
		stepsContent.WriteString(pipelineStep.Content)
	}

	// If there are build-deps, wrap the content
	if len(allBuildDeps) > 0 {
		return g.wrapWithBuildDeps(stepsContent.String(), allBuildDeps, step.Uses), nil
	}

	return stepsContent.String(), nil
}

// mergeDeps combines two slices of dependencies, removing duplicates
func mergeDeps(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, dep := range a {
		if !seen[dep] {
			seen[dep] = true
			result = append(result, dep)
		}
	}
	for _, dep := range b {
		if !seen[dep] {
			seen[dep] = true
			result = append(result, dep)
		}
	}

	return result
}

// wrapWithBuildDeps wraps dockerfile content with build dependency installation and cleanup
func (g *Generator) wrapWithBuildDeps(content string, buildDeps []string, pipelineName string) string {
	var b strings.Builder

	virtualName := fmt.Sprintf(".%s-deps", pipelineName)
	pkgList := util.BuildPackageList(buildDeps)
	b.WriteString(fmt.Sprintf("RUN apk add --no-cache --virtual %s \\\n", virtualName))
	b.WriteString(fmt.Sprintf("    {{- range $key, $value := alpine_packages %s}}\n", pkgList))
	b.WriteString("    {{$key}}={{$value}} \\\n")
	b.WriteString("    {{- end}}\n")
	b.WriteString("    ;\n\n")

	b.WriteString(content)

	b.WriteString(fmt.Sprintf("RUN apk del --no-network %s\n", virtualName))

	return b.String()
}

func (g *Generator) formatRunCommand(run string) string {
	lines := strings.Split(run, "\n")

	var nonEmptyLines []string
	for _, line := range lines {
		normalized, _ := util.NormalizeShellLine(line)
		if normalized != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return ""
	}

	if len(nonEmptyLines) == 1 {
		normalized, _ := util.NormalizeShellLine(nonEmptyLines[0])
		return fmt.Sprintf("RUN %s\n", normalized)
	}

	var b strings.Builder
	b.Grow(256)

	for i, line := range nonEmptyLines {
		if i == 0 {
			b.WriteString(util.FormatShellLineWithContinuation(line, "RUN "))
		} else if i < len(nonEmptyLines)-1 {
			b.WriteString(util.FormatShellLineWithContinuation(line, "    "))
		} else {
			normalized, _ := util.NormalizeShellLine(line)
			b.WriteString(fmt.Sprintf("    %s\n", normalized))
		}
	}

	return b.String()
}
