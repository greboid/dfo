package generator

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/packages"
	"github.com/greboid/dfo/pkg/pipelines"
	"github.com/greboid/dfo/pkg/util"
	"github.com/greboid/dfo/pkg/versions"
)

const (
	dirPerms  = 0755
	filePerms = 0644
)

type Generator struct {
	config           *config.BuildConfig
	outputDir        string
	outputFilename   string
	fs               util.WritableFS
	resolver         *packages.Resolver
	versionResolver  *versions.Resolver
	resolvedVersions map[string]string
}

func New(cfg *config.BuildConfig, outputDir string, fs util.WritableFS, alpineClient *packages.AlpineClient, alpineVersion, gitUser, gitPass string) *Generator {
	resolver := packages.NewResolver(alpineClient, alpineVersion)
	versionResolver := versions.New(context.Background(), gitUser, gitPass)

	return &Generator{
		config:           cfg,
		outputDir:        outputDir,
		outputFilename:   "Containerfile.gotpl",
		fs:               fs,
		resolver:         resolver,
		versionResolver:  versionResolver,
		resolvedVersions: make(map[string]string),
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

func (g *Generator) resolveVersions() error {
	if g.config.Versions == nil {
		return nil
	}

	for key, value := range g.config.Versions {
		resolved, err := g.versionResolver.Resolve(key, value)
		if err != nil {
			return fmt.Errorf("resolving version %q: %w", key, err)
		}
		g.resolvedVersions[key] = resolved
		slog.Debug("resolved version", "key", key, "value", value, "resolved", resolved)
	}

	return nil
}

func (g *Generator) buildVarsMap() map[string]string {
	vars := make(map[string]string)

	for k, v := range g.config.Vars {
		vars[k] = v
	}

	for k, v := range g.resolvedVersions {
		vars["versions."+k] = v
	}

	return vars
}

func (g *Generator) resolvePackages(pkgSpecs []string) ([]packages.ResolvedPackage, error) {
	specs, err := packages.ParsePackageSpecs(pkgSpecs)
	if err != nil {
		return nil, fmt.Errorf("parsing package specs: %w", err)
	}

	return g.resolver.Resolve(specs)
}

func (g *Generator) resolveAndFormatPackages(pkgSpecs []string, firstIndent bool, indent string) (string, error) {
	resolved, err := g.resolvePackages(pkgSpecs)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for i, pkg := range resolved {
		if i > 0 || firstIndent {
			b.WriteString(indent)
		}
		b.WriteString(fmt.Sprintf("%s=%s", pkg.Name, pkg.Version))
		if i < len(resolved)-1 {
			b.WriteString(" \\\n")
		} else {
			b.WriteString(" \\")
		}
	}
	return b.String(), nil
}

func (g *Generator) Generate() error {
	if err := g.resolveVersions(); err != nil {
		return fmt.Errorf("resolving versions: %w", err)
	}

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
	vars := g.buildVarsMap()

	for _, stage := range g.config.Stages {
		for i, step := range stage.Pipeline {
			stepContext := fmt.Sprintf("stage %q step %d", stage.Name, i+1)
			if step.Name != "" {
				stepContext = fmt.Sprintf("stage %q step %q", stage.Name, step.Name)
			}

			if step.Run != "" {
				if err := util.ValidateVariableReferences(step.Run, vars, stepContext+" (run)"); err != nil {
					return err
				}
			}

			if step.Fetch != nil && step.Fetch.URL != "" {
				if err := util.ValidateVariableReferences(step.Fetch.URL, vars, stepContext+" (fetch.url)"); err != nil {
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
		pkgInstall, err := g.generatePackageInstallForEnv(env)
		if err != nil {
			return "", fmt.Errorf("generating package install: %w", err)
		}
		b.WriteString(pkgInstall)
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

func (g *Generator) generatePackageInstallForEnv(env config.Environment) (string, error) {
	var b strings.Builder
	b.Grow(512)

	b.WriteString("# Install packages\n")
	b.WriteString("RUN set -eux; \\\n")
	b.WriteString("    apk add --no-cache \\\n")

	pkgStr, err := g.resolveAndFormatPackages(env.Packages, true, "        ")
	if err != nil {
		return "", fmt.Errorf("resolving packages: %w", err)
	}
	b.WriteString(pkgStr)
	b.WriteString("\n")
	b.WriteString("    ;\n")

	return b.String(), nil
}

func (g *Generator) generateRootfsPackageInstallForEnv(env config.Environment) string {
	var b strings.Builder
	b.Grow(512)

	b.WriteString("# Install packages into rootfs\n")

	resolved, err := g.resolvePackages(env.RootfsPackages)
	if err != nil {
		b.WriteString(fmt.Sprintf("# Error resolving packages: %v\n", err))
		return b.String()
	}

	b.WriteString("RUN \\\n")
	for _, pkg := range resolved {
		b.WriteString(fmt.Sprintf("    apk add --no-cache %s=%s; \\\n", pkg.Name, pkg.Version))
		b.WriteString(fmt.Sprintf("    apk info -qL %s | rsync -aq --files-from=- / /rootfs/; \\\n", pkg.Name))
	}

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
		vars := g.buildVarsMap()
		run := util.ExpandVars(step.Run, vars)

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

	pkgStr, err := g.resolveAndFormatPackages(buildDeps, true, "  ")
	if err != nil {
		b.WriteString(fmt.Sprintf("# Error resolving build deps: %v\n", err))
		return b.String()
	}

	b.WriteString("RUN apk add --no-cache --virtual .build-deps \\\n")
	b.WriteString(pkgStr)
	b.WriteString("\n")
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

	vars := g.buildVarsMap()
	url := util.ExpandVars(fetch.URL, vars)
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

	vars := g.buildVarsMap()

	for _, key := range []string{"tag", "commit"} {
		if value, ok := step.With[key]; ok {
			expanded, err := util.ExpandVarsStrict(value.(string), vars, "")
			if err != nil {
				return "", fmt.Errorf("variable not found in pipeline %q (step %q)", step.Uses, step.Name)
			}
			step.With[key] = expanded
		}
	}

	result, err := pipeline(step.With)
	if err != nil {
		return "", fmt.Errorf("executing pipeline %q: %w", step.Uses, err)
	}

	allBuildDeps := mergeDeps(result.BuildDeps, step.BuildDeps)

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

func (g *Generator) wrapWithBuildDeps(content string, buildDeps []string, pipelineName string) string {
	var b strings.Builder

	virtualName := fmt.Sprintf(".%s-deps", pipelineName)

	pkgStr, err := g.resolveAndFormatPackages(buildDeps, false, "    ")
	if err != nil {
		b.WriteString(fmt.Sprintf("# Error resolving build deps: %v\n", err))
		return content
	}

	b.WriteString(fmt.Sprintf("RUN apk add --no-cache --virtual %s \\\n", virtualName))
	b.WriteString("    ")
	b.WriteString(pkgStr)
	b.WriteString("\n")
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
