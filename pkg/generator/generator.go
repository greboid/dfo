package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/greboid/dfo/pkg/config"
	"github.com/greboid/dfo/pkg/images"
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
	imageResolver    *images.Resolver
	resolvedVersions map[string]versions.VersionMetadata
	resolvedPackages map[string]string
	resolvedImages   map[string]string
	builtImages      map[string]string
	localImageNames  map[string]bool
	mu               sync.Mutex
}

func New(cfg *config.BuildConfig, outputDir string, fs util.WritableFS, alpineClient *packages.AlpineClient, alpineVersion, gitUser, gitPass, registry string, sharedImageResolver *images.Resolver) *Generator {
	resolver := packages.NewResolver(alpineClient, alpineVersion)
	versionResolver := versions.New(context.Background(), gitUser, gitPass)

	var imageResolver *images.Resolver
	if sharedImageResolver != nil {
		imageResolver = sharedImageResolver
	} else {
		imageResolver = images.NewResolver(registry, false)
	}

	return &Generator{
		config:           cfg,
		outputDir:        outputDir,
		outputFilename:   "Containerfile",
		fs:               fs,
		resolver:         resolver,
		versionResolver:  versionResolver,
		imageResolver:    imageResolver,
		resolvedVersions: make(map[string]versions.VersionMetadata),
		resolvedPackages: make(map[string]string),
		resolvedImages:   make(map[string]string),
		builtImages:      make(map[string]string),
		localImageNames:  make(map[string]bool),
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

func (g *Generator) SetBuiltImages(builtImages map[string]string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for imageName, digest := range builtImages {
		g.builtImages[imageName] = digest
	}
}

func (g *Generator) SetLocalImageNames(localNames []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, name := range localNames {
		g.localImageNames[name] = true
	}
}

func (g *Generator) resolveVersions() error {
	if g.config.Versions == nil {
		return nil
	}

	const maxConcurrency = 10
	type versionResult struct {
		key      string
		value    string
		resolved versions.VersionMetadata
		err      error
	}

	results := make(chan versionResult, len(g.config.Versions))
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for key, value := range g.config.Versions {
		wg.Go(func() {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resolved, err := g.versionResolver.Resolve(key, value)
			results <- versionResult{key: key, value: value, resolved: resolved, err: err}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			return fmt.Errorf("resolving version %q: %w", result.key, result.err)
		}
		g.resolvedVersions[result.key] = result.resolved
		slog.Debug("resolved version", "key", result.key, "value", result.value, "resolved", result.resolved)
	}

	return nil
}

func (g *Generator) resolveImage(imageName string) (*images.ResolvedImage, error) {
	if resolved, ok := g.tryGetBuiltImage(imageName); ok {
		return resolved, nil
	}

	if resolved, ok := g.tryGetBuiltImageWithPrefix(imageName); ok {
		return resolved, nil
	}

	if err := g.validateLocalImageStatus(imageName); err != nil {
		return nil, err
	}

	return g.resolveExternalImage(imageName)
}

func (g *Generator) tryGetBuiltImage(imageName string) (*images.ResolvedImage, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	builtDigest, ok := g.builtImages[imageName]
	if !ok {
		return nil, false
	}

	slog.Debug("Using built image digest",
		"image", imageName,
		"digest", builtDigest[:min(16, len(builtDigest))],
	)

	return &images.ResolvedImage{
		Name:    imageName,
		Digest:  builtDigest,
		FullRef: util.FormatFullRef(imageName, builtDigest),
	}, true
}

func (g *Generator) tryGetBuiltImageWithPrefix(imageName string) (*images.ResolvedImage, bool) {
	registry := g.imageResolver.GetRegistry()
	if registry == "" || strings.Contains(imageName, "/") {
		return nil, false
	}

	prefixedName := fmt.Sprintf("%s/%s", registry, imageName)

	g.mu.Lock()
	defer g.mu.Unlock()

	builtDigest, ok := g.builtImages[prefixedName]
	if !ok {
		return nil, false
	}

	slog.Debug("Using built image digest (with registry prefix)",
		"image", prefixedName,
		"digest", builtDigest[:min(16, len(builtDigest))],
	)

	return &images.ResolvedImage{
		Name:    prefixedName,
		Digest:  builtDigest,
		FullRef: util.FormatFullRef(prefixedName, builtDigest),
	}, true
}

func (g *Generator) validateLocalImageStatus(imageName string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	registry := g.imageResolver.GetRegistry()
	prefixedName := ""
	if registry != "" && !strings.Contains(imageName, "/") {
		prefixedName = fmt.Sprintf("%s/%s", registry, imageName)
	}

	isLocal := g.localImageNames[imageName]
	if !isLocal && prefixedName != "" {
		isLocal = g.localImageNames[prefixedName]
	}

	if isLocal {
		return fmt.Errorf("local image %q has not been built yet - check build order", imageName)
	}

	return nil
}

func (g *Generator) resolveExternalImage(imageName string) (*images.ResolvedImage, error) {
	slog.Debug("Resolving external image from registry", "image", imageName)

	resolved, err := g.imageResolver.Resolve(context.Background(), imageName)
	if err != nil {
		return nil, fmt.Errorf("resolving external image %q from registry: %w", imageName, err)
	}

	return resolved, nil
}

func (g *Generator) buildVarsMap() map[string]string {
	vars := make(map[string]string)

	for k, v := range g.config.Vars {
		vars[k] = v
	}

	for k, v := range g.resolvedVersions {
		vars["versions."+k] = v.Version

		if v.URL != "" {
			vars["versions."+k+".url"] = v.URL
		}

		if v.Checksum != "" {
			vars["versions."+k+".checksum"] = v.Checksum
		}
	}

	return vars
}

func (g *Generator) resolvePackages(pkgSpecs []string) ([]packages.ResolvedPackage, error) {
	specs, err := packages.ParsePackageSpecs(pkgSpecs)
	if err != nil {
		return nil, fmt.Errorf("parsing package specs: %w", err)
	}

	resolved, err := g.resolver.Resolve(specs)
	if err != nil {
		return nil, err
	}

	g.mu.Lock()
	for _, pkg := range resolved {
		g.resolvedPackages[pkg.Name] = pkg.Version
	}
	g.mu.Unlock()

	return resolved, nil
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

	var output strings.Builder
	bom := g.generateBOM()
	if bom != "" {
		output.WriteString(bom)
		output.WriteString("\n")
	}
	output.WriteString(b.String())

	outputPath := path.Join(g.outputDir, g.outputFilename)
	if err := g.fs.WriteFile(outputPath, []byte(output.String()), filePerms); err != nil {
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
		resolvedImage, err := g.resolveImage(stage.Environment.BaseImage)
		if err != nil {
			return "", fmt.Errorf("resolving base image: %w", err)
		}

		if isFinalStage {
			b.WriteString(fmt.Sprintf("FROM %s\n\n", resolvedImage.FullRef))
		} else {
			b.WriteString(fmt.Sprintf("FROM %s AS %s\n\n", resolvedImage.FullRef, stage.Name))
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

	b.WriteString(g.generateArgsSection(env))
	b.WriteString(g.generateLabelsSection(env, isFinalStage))
	b.WriteString(g.generateEnvSection(env))

	if err := g.appendPackageSections(env, &b); err != nil {
		return "", err
	}

	b.WriteString(g.generateWorkDirSection(env))

	if err := g.appendPipelineSections(pipeline, &b); err != nil {
		return "", err
	}

	b.WriteString(g.generateMetadataSections(env))
	return b.String(), nil
}

func (g *Generator) generateArgsSection(env config.Environment) string {
	if len(env.Args) == 0 {
		return ""
	}
	var b strings.Builder
	for _, key := range util.SortedKeys(env.Args) {
		b.WriteString(fmt.Sprintf("ARG %s=\"%s\"\n", key, env.Args[key]))
	}
	b.WriteString("\n")
	return b.String()
}

func (g *Generator) generateLabelsSection(env config.Environment, isFinalStage bool) string {
	if len(g.config.Package.Labels) == 0 || !isFinalStage {
		return ""
	}
	return util.FormatMapDirectives("LABEL", g.config.Package.Labels)
}

func (g *Generator) generateEnvSection(env config.Environment) string {
	if len(env.Environment) == 0 {
		return ""
	}
	return util.FormatMapDirectives("ENV", env.Environment)
}

func (g *Generator) appendPackageSections(env config.Environment, b *strings.Builder) error {
	if len(env.Packages) > 0 {
		pkgInstall, err := g.generatePackageInstallForEnv(env)
		if err != nil {
			return err
		}
		b.WriteString(pkgInstall)
		b.WriteString("\n")
	}
	if len(env.RootfsPackages) > 0 {
		content := g.generateRootfsPackageInstallForEnv(env)
		b.WriteString(g.wrapWithBuildDeps(content, []string{"busybox", "rsync"}, "rootfs-packages"))
		b.WriteString("\n")
	}
	return nil
}

func (g *Generator) generateWorkDirSection(env config.Environment) string {
	if env.WorkDir == "" {
		return ""
	}
	return fmt.Sprintf("WORKDIR %s\n\n", env.WorkDir)
}

func (g *Generator) appendPipelineSections(pipeline []config.PipelineStep, b *strings.Builder) error {
	for _, step := range pipeline {
		stepContent, err := g.generatePipelineStep(step)
		if err != nil {
			return err
		}
		if stepContent != "" {
			if step.Name != "" {
				b.WriteString(fmt.Sprintf("# %s\n", step.Name))
			}
			b.WriteString(stepContent)
			b.WriteString("\n")
		}
	}
	return nil
}

func (g *Generator) generateMetadataSections(env config.Environment) string {
	var b strings.Builder

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

	return b.String()
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

	return b.String()[:b.Len()-3] + "\n"
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
	pipeline, err := g.getPipeline(step.Uses, step.Name)
	if err != nil {
		return "", err
	}

	if err := pipelines.ValidateParams(step.Uses, step.With); err != nil {
		return "", err
	}

	expandedWith, err := g.expandPipelineParams(step.With, step.Uses, step.Name)
	if err != nil {
		return "", err
	}

	result, err := pipeline(expandedWith)
	if err != nil {
		return "", fmt.Errorf("executing pipeline %q: %w", step.Uses, err)
	}

	return g.formatPipelineResult(&result, step.BuildDeps, step.Uses), nil
}

func (g *Generator) getPipeline(pipelineName, stepName string) (pipelines.Pipeline, error) {
	pipeline, exists := pipelines.Registry[pipelineName]
	if !exists {
		return nil, fmt.Errorf("pipeline %q not found (referenced in step %q)", pipelineName, stepName)
	}
	return pipeline, nil
}

func (g *Generator) expandPipelineParams(with map[string]any, pipelineName, stepName string) (map[string]any, error) {
	vars := g.buildVarsMap()
	expandedWith := make(map[string]any)

	for key, value := range with {
		if strValue, ok := value.(string); ok {
			expanded, err := util.ExpandVarsStrict(strValue, vars, "")
			if err != nil {
				return nil, fmt.Errorf("variable %q not found in pipeline %q (step %q)", strValue, pipelineName, stepName)
			}
			expandedWith[key] = expanded
		} else {
			expandedWith[key] = value
		}
	}

	return expandedWith, nil
}

func (g *Generator) formatPipelineResult(result *pipelines.PipelineResult, buildDeps []string, pipelineName string) string {
	var stepsContent strings.Builder
	for _, pipelineStep := range result.Steps {
		if pipelineStep.Name != "" {
			stepsContent.WriteString(fmt.Sprintf("# %s\n", pipelineStep.Name))
		}
		stepsContent.WriteString(pipelineStep.Content)
	}

	allBuildDeps := mergeDeps(result.BuildDeps, buildDeps)
	if len(allBuildDeps) > 0 {
		return g.wrapWithBuildDeps(stepsContent.String(), allBuildDeps, pipelineName)
	}

	return stepsContent.String()
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

func (g *Generator) generateBOM() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	bom := g.collectBOMEntries()
	if len(bom) == 0 {
		return ""
	}

	sortedBOM := g.sortBOMKeys(bom)
	return g.formatBOMAsComment(sortedBOM)
}

func (g *Generator) collectBOMEntries() map[string]string {
	bom := make(map[string]string)

	for pkg, version := range g.resolvedPackages {
		bom[fmt.Sprintf("apk:%s", pkg)] = version
	}

	for key, metadata := range g.resolvedVersions {
		bom[key] = metadata.Version
	}

	for image, digest := range g.resolvedImages {
		bom[fmt.Sprintf("image:%s", image)] = digest
	}

	for imageName, digest := range g.builtImages {
		shortDigest := g.extractShortDigest(digest)
		bom[fmt.Sprintf("built:%s", imageName)] = shortDigest
	}

	return bom
}

func (g *Generator) extractShortDigest(digest string) string {
	if idx := strings.Index(digest, ":"); idx != -1 {
		return digest[idx+1:]
	}
	return digest
}

func (g *Generator) sortBOMKeys(bom map[string]string) map[string]string {
	keys := make([]string, 0, len(bom))
	for key := range bom {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	sortedBOM := make(map[string]string)
	for _, key := range keys {
		sortedBOM[key] = bom[key]
	}

	return sortedBOM
}

func (g *Generator) formatBOMAsComment(bom map[string]string) string {
	jsonBytes, err := json.Marshal(bom)
	if err != nil {
		slog.Warn("failed to generate BOM", "error", err)
		return ""
	}

	return fmt.Sprintf("# BOM: %s\n", string(jsonBytes))
}
