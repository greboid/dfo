package templates

import (
	"fmt"

	"github.com/greboid/dfo/pkg/pipelines"
)

const (
	DefaultVolumeOwner       = "65532:65532"
	DefaultVolumePermissions = "777"
)

type TemplateResult struct {
	Stages []StageResult
}

type StageResult struct {
	Name        string
	Environment EnvironmentResult
	Pipeline    []PipelineStepResult
}

type EnvironmentResult struct {
	BaseImage      string
	ExternalImage  string
	Packages       []string
	RootfsPackages []string
	User           string
	WorkDir        string
	Expose         []string
	Entrypoint     []string
	Cmd            []string
}

type PipelineStepResult struct {
	Uses string
	Run  string
	With map[string]any
	Copy *CopyStepResult
}

type CopyStepResult struct {
	FromStage string
	From      string
	To        string
	Chown     string
}

type TemplateFunc func(params map[string]any) (TemplateResult, error)

var Registry = map[string]TemplateFunc{
	"go-builder":   goBuilder,
	"rust-builder": rustBuilder,
	"go-app":       goApp,
	"multi-go-app": multiGoApp,
	"rust-app":     rustApp,
}

func goBuilder(params map[string]any) (TemplateResult, error) {
	return TemplateResult{
		Stages: []StageResult{
			{
				Environment: EnvironmentResult{
					BaseImage: "golang",
				},
				Pipeline: []PipelineStepResult{
					{
						Uses: "clone-and-build-go",
						With: params,
					},
				},
			},
		},
	}, nil
}

func rustBuilder(params map[string]any) (TemplateResult, error) {
	packages := []string{"git"}
	if pkgs, ok := params["packages"].([]any); ok {
		for _, pkg := range pkgs {
			if pkgStr, ok := pkg.(string); ok {
				packages = append(packages, pkgStr)
			}
		}
	}

	pipelineParams := make(map[string]any)
	for k, v := range params {
		if k != "packages" {
			pipelineParams[k] = v
		}
	}

	return TemplateResult{
		Stages: []StageResult{
			{
				Environment: EnvironmentResult{
					BaseImage: "rust",
					Packages:  packages,
				},
				Pipeline: []PipelineStepResult{
					{
						Uses: "clone-and-build-rust",
						With: pipelineParams,
					},
				},
			},
		},
	}, nil
}

func goApp(params map[string]any) (TemplateResult, error) {
	binary, _ := params["binary"].(string)

	buildParams := prepareGoBuildParams(params)

	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}

	extraCopies, err := ParseExtraCopies(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing extra-copies: %w", err)
	}

	buildStage := createGoBuildStage(buildParams, volumes)
	rootfsStage := createGoRootfsStage(binary, volumes, extraCopies)
	finalStage := createFinalStage(binary, params)

	return TemplateResult{
		Stages: []StageResult{buildStage, rootfsStage, finalStage},
	}, nil
}

func prepareGoBuildParams(params map[string]any) map[string]any {
	workdir := getStringOrDefault(params, "workdir", "/src")
	pkg := getStringOrDefault(params, "package", ".")
	repo, _ := params["repo"].(string)

	buildParams := map[string]any{
		"workdir": workdir,
		"repo":    repo,
		"package": pkg,
	}

	if tag, ok := params["tag"].(string); ok {
		buildParams["tag"] = tag
	}
	if ignore, ok := params["ignore"].([]any); ok {
		buildParams["ignore"] = ignore
	}
	if patches, ok := params["patches"]; ok {
		buildParams["patches"] = patches
	}
	if goTags, ok := params["go-tags"].(string); ok {
		buildParams["go-tags"] = goTags
	}
	if goExperiment, ok := params["go-experiment"].(string); ok {
		buildParams["go-experiment"] = goExperiment
	}
	if packages, ok := params["packages"].([]any); ok {
		buildParams["packages"] = packages
	}
	if goGenerate, ok := params["go-generate"].([]any); ok {
		buildParams["go-generate"] = goGenerate
	}
	if goInstall, ok := params["go-install"].([]any); ok {
		buildParams["go-install"] = goInstall
	}

	return buildParams
}

func createGoBuildStage(buildParams map[string]any, volumes []VolumeSpec) StageResult {
	buildPipeline := []PipelineStepResult{
		{
			Uses: "build-go-static",
			With: buildParams,
		},
	}

	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		buildPipeline = append(buildPipeline, *volumeStep)
	}

	return StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "golang",
		},
		Pipeline: buildPipeline,
	}
}

func createGoRootfsStage(binary string, volumes []VolumeSpec, extraCopies []ExtraCopySpec) StageResult {
	rootfsPipeline := []PipelineStepResult{
		{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      "/main",
				To:        "/rootfs/" + binary,
			},
		},
		{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      "/notices",
				To:        "/rootfs/notices",
			},
		},
	}

	for _, vol := range volumes {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      vol.Path,
				To:        "/rootfs" + vol.Path,
			},
		})
	}

	for _, ec := range extraCopies {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      ec.From,
				To:        "/rootfs" + ec.To,
			},
		})
	}

	return StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: rootfsPipeline,
	}
}

func createFinalStage(binary string, params map[string]any) StageResult {
	finalStage := StageResult{
		Name: "final",
		Environment: EnvironmentResult{
			BaseImage:  "base",
			Entrypoint: []string{"/" + binary},
		},
		Pipeline: []PipelineStepResult{
			{
				Copy: &CopyStepResult{
					FromStage: "rootfs",
					From:      "/rootfs/",
					To:        "/",
				},
			},
		},
	}

	if expose, ok := params["expose"].([]any); ok {
		finalStage.Environment.Expose = convertStringArray(expose)
	}

	if entrypoint, ok := params["entrypoint"].([]any); ok {
		finalStage.Environment.Entrypoint = convertStringArray(entrypoint)
	}

	if cmd, ok := params["cmd"].([]any); ok {
		finalStage.Environment.Cmd = convertStringArray(cmd)
	}

	return finalStage
}

func getStringOrDefault(params map[string]any, key, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

func convertStringArray(arr []any) []string {
	result := make([]string, len(arr))
	for i, item := range arr {
		if str, ok := item.(string); ok {
			result[i] = str
		}
	}
	return result
}

func rustApp(params map[string]any) (TemplateResult, error) {
	binary, _ := params["binary"].(string)

	buildParams := prepareRustBuildParams(params)
	packages := preparePackages(params)

	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}

	buildStage := createRustBuildStage(buildParams, packages, volumes)
	rootfsStage := createRustRootfsStage(binary, volumes)
	finalStage := createFinalStage(binary, params)

	return TemplateResult{
		Stages: []StageResult{buildStage, rootfsStage, finalStage},
	}, nil
}

func prepareRustBuildParams(params map[string]any) map[string]any {
	repo, _ := params["repo"].(string)

	buildParams := map[string]any{
		"repo": repo,
	}

	if workdir, ok := params["workdir"].(string); ok {
		buildParams["workdir"] = workdir
	}
	if features, ok := params["features"].(string); ok {
		buildParams["features"] = features
	}
	if patches, ok := params["patches"]; ok {
		buildParams["patches"] = patches
	}
	if tag, ok := params["tag"].(string); ok {
		buildParams["tag"] = tag
	}

	return buildParams
}

func preparePackages(params map[string]any) []string {
	packages := []string{"git"}
	if pkgs, ok := params["packages"].([]any); ok {
		for _, pkg := range pkgs {
			if pkgStr, ok := pkg.(string); ok {
				packages = append(packages, pkgStr)
			}
		}
	}
	return packages
}

func createRustBuildStage(buildParams map[string]any, packages []string, volumes []VolumeSpec) StageResult {
	buildPipeline := []PipelineStepResult{
		{
			Uses: "clone-and-build-rust",
			With: buildParams,
		},
	}

	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		buildPipeline = append(buildPipeline, *volumeStep)
	}

	return StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "rust",
			Packages:  packages,
		},
		Pipeline: buildPipeline,
	}
}

func createRustRootfsStage(binary string, volumes []VolumeSpec) StageResult {
	rootfsPipeline := []PipelineStepResult{
		{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      "/main",
				To:        "/rootfs/" + binary,
			},
		},
	}

	for _, vol := range volumes {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      vol.Path,
				To:        "/rootfs" + vol.Path,
			},
		})
	}

	return StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: rootfsPipeline,
	}
}

type BinarySpec struct {
	Repo         string
	Tag          string
	Package      string
	Binary       string
	GoTags       string
	GoExperiment string
	Ignore       []string
	Patches      []string
	Entrypoint   bool
	Cgo          bool
}

func ParseBinaries(params map[string]any) ([]BinarySpec, error) {
	binariesList, err := parseBinariesList(params)
	if err != nil {
		return nil, err
	}

	binaries := make([]BinarySpec, 0, len(binariesList))
	for i, b := range binariesList {
		spec, err := parseBinarySpec(b, i)
		if err != nil {
			return nil, err
		}
		binaries = append(binaries, spec)
	}

	return binaries, nil
}

func parseBinariesList(params map[string]any) ([]any, error) {
	binariesParam, ok := params["binaries"]
	if !ok {
		return nil, fmt.Errorf("binaries parameter is required")
	}

	binariesList, ok := binariesParam.([]any)
	if !ok {
		return nil, fmt.Errorf("binaries must be a list")
	}

	if len(binariesList) == 0 {
		return nil, fmt.Errorf("at least one binary must be specified")
	}

	return binariesList, nil
}

func parseBinarySpec(b any, index int) (BinarySpec, error) {
	binaryMap, ok := b.(map[string]any)
	if !ok {
		return BinarySpec{}, fmt.Errorf("binary at index %d must be a map", index)
	}

	repo, err := getStringField(binaryMap, "repo", index)
	if err != nil {
		return BinarySpec{}, err
	}

	binary, err := getStringField(binaryMap, "binary", index)
	if err != nil {
		return BinarySpec{}, err
	}

	spec := BinarySpec{
		Repo:   repo,
		Binary: binary,
	}

	if pkg, ok := binaryMap["package"].(string); ok {
		spec.Package = pkg
	} else {
		spec.Package = "."
	}

	if goTags, ok := binaryMap["go-tags"].(string); ok {
		spec.GoTags = goTags
	}

	if goExperiment, ok := binaryMap["go-experiment"].(string); ok {
		spec.GoExperiment = goExperiment
	}

	if ignore, ok := binaryMap["ignore"].([]any); ok {
		for _, ig := range ignore {
			if igs, ok := ig.(string); ok {
				spec.Ignore = append(spec.Ignore, igs)
			}
		}
	}

	if patches, ok := binaryMap["patches"].([]any); ok {
		for _, p := range patches {
			if ps, ok := p.(string); ok {
				spec.Patches = append(spec.Patches, ps)
			}
		}
	}

	if entrypoint, ok := binaryMap["entrypoint"].(bool); ok {
		spec.Entrypoint = entrypoint
	}

	if cgo, ok := binaryMap["cgo"].(bool); ok {
		spec.Cgo = cgo
	}

	if tag, ok := binaryMap["tag"].(string); ok {
		spec.Tag = tag
	}

	return spec, nil
}

func getStringField(m map[string]any, field string, index int) (string, error) {
	val, ok := m[field].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("binary at index %d must have a %s", index, field)
	}
	return val, nil
}

func multiGoApp(params map[string]any) (TemplateResult, error) {
	binaries, err := ParseBinaries(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing binaries: %w", err)
	}

	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}

	extraCopies, err := ParseExtraCopies(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing extra-copies: %w", err)
	}

	buildPipeline := createMultiBuildPipeline(binaries)
	buildStage := createMultiBuildStage(buildPipeline, volumes)
	rootfsStage := createMultiRootfsStage(binaries, volumes, extraCopies)
	finalStage := createMultiFinalStage(binaries, params)

	return TemplateResult{
		Stages: []StageResult{buildStage, rootfsStage, finalStage},
	}, nil
}

func createMultiBuildPipeline(binaries []BinarySpec) []PipelineStepResult {
	clonedRepos := make(map[string]string)
	var buildPipeline []PipelineStepResult

	for _, bin := range binaries {
		workdir := getWorkdirForBin(bin, clonedRepos)
		buildPipeline = append(buildPipeline, createCloneStep(bin, workdir)...)
		buildPipeline = append(buildPipeline, createBuildOnlyStep(bin, workdir))
	}

	return buildPipeline
}

func getWorkdirForBin(bin BinarySpec, clonedRepos map[string]string) string {
	workdir, alreadyCloned := clonedRepos[bin.Repo]
	if !alreadyCloned {
		workdir = "/src"
		if ownerRepo := pipelines.ExtractGitHubOwnerRepo(bin.Repo); ownerRepo != "" {
			workdir = "/src/" + ownerRepo
		}
		clonedRepos[bin.Repo] = workdir
	}
	return workdir
}

func createCloneStep(bin BinarySpec, workdir string) []PipelineStepResult {
	return []PipelineStepResult{
		{
			Uses: "clone",
			With: map[string]any{
				"repo":    bin.Repo,
				"workdir": workdir,
				"tag":     bin.Tag,
			},
		},
	}
}

func createBuildOnlyStep(bin BinarySpec, workdir string) PipelineStepResult {
	buildParams := map[string]any{
		"workdir": workdir,
		"package": bin.Package,
		"output":  "/" + bin.Binary,
	}

	if len(bin.Ignore) > 0 {
		buildParams["ignore"] = bin.Ignore
	}
	if bin.GoTags != "" {
		buildParams["go-tags"] = bin.GoTags
	}
	if bin.GoExperiment != "" {
		buildParams["go-experiment"] = bin.GoExperiment
	}
	if bin.Cgo {
		buildParams["cgo"] = bin.Cgo
	}

	return PipelineStepResult{
		Uses: "build-go-only",
		With: buildParams,
	}
}

func createMultiBuildStage(buildPipeline []PipelineStepResult, volumes []VolumeSpec) StageResult {
	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		buildPipeline = append(buildPipeline, *volumeStep)
	}

	return StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "golang",
		},
		Pipeline: buildPipeline,
	}
}

func createMultiRootfsStage(binaries []BinarySpec, volumes []VolumeSpec, extraCopies []ExtraCopySpec) StageResult {
	var rootfsPipeline []PipelineStepResult

	for _, bin := range binaries {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      "/" + bin.Binary,
				To:        "/rootfs/" + bin.Binary,
			},
		})
	}

	for _, bin := range binaries {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      "/notices/" + bin.Binary,
				To:        "/rootfs/notices/" + bin.Binary,
			},
		})
	}

	for _, vol := range volumes {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      vol.Path,
				To:        "/rootfs" + vol.Path,
			},
		})
	}

	for _, ec := range extraCopies {
		rootfsPipeline = append(rootfsPipeline, PipelineStepResult{
			Copy: &CopyStepResult{
				FromStage: "build",
				From:      ec.From,
				To:        "/rootfs" + ec.To,
			},
		})
	}

	return StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: rootfsPipeline,
	}
}

func createMultiFinalStage(binaries []BinarySpec, params map[string]any) StageResult {
	entrypointBinary := findEntrypointBinary(binaries)

	finalStage := StageResult{
		Name: "final",
		Environment: EnvironmentResult{
			BaseImage:  "base",
			Entrypoint: []string{"/" + entrypointBinary},
		},
		Pipeline: []PipelineStepResult{
			{
				Copy: &CopyStepResult{
					FromStage: "rootfs",
					From:      "/rootfs/",
					To:        "/",
				},
			},
		},
	}

	if expose, ok := params["expose"].([]any); ok {
		finalStage.Environment.Expose = convertStringArray(expose)
	}

	if entrypoint, ok := params["entrypoint"].([]any); ok {
		finalStage.Environment.Entrypoint = convertStringArray(entrypoint)
	}

	if cmd, ok := params["cmd"].([]any); ok {
		finalStage.Environment.Cmd = convertStringArray(cmd)
	}

	return finalStage
}

func findEntrypointBinary(binaries []BinarySpec) string {
	for _, bin := range binaries {
		if bin.Entrypoint {
			return bin.Binary
		}
	}
	if len(binaries) > 0 {
		return binaries[0].Binary
	}
	return ""
}

type VolumeSpec struct {
	Path        string
	Owner       string
	Permissions string
}

type ExtraCopySpec struct {
	From string
	To   string
}

func ParseVolumes(params map[string]any) ([]VolumeSpec, error) {
	volumesParam, ok := params["volumes"]
	if !ok {
		return nil, nil
	}

	volumesList, ok := volumesParam.([]any)
	if !ok {
		return nil, fmt.Errorf("volumes must be a list")
	}

	volumes := make([]VolumeSpec, 0, len(volumesList))
	for i, v := range volumesList {
		volumeMap, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("volume at index %d must be a map", i)
		}

		path, ok := volumeMap["path"].(string)
		if !ok || path == "" {
			return nil, fmt.Errorf("volume at index %d must have a path", i)
		}

		owner := DefaultVolumeOwner
		if o, ok := volumeMap["owner"].(string); ok && o != "" {
			owner = o
		}

		permissions := DefaultVolumePermissions
		if p, ok := volumeMap["permissions"].(string); ok && p != "" {
			permissions = p
		}

		volumes = append(volumes, VolumeSpec{
			Path:        path,
			Owner:       owner,
			Permissions: permissions,
		})
	}

	return volumes, nil
}

func CreateVolumesStep(volumes []VolumeSpec) *PipelineStepResult {
	if len(volumes) == 0 {
		return nil
	}

	directories := make([]map[string]any, len(volumes))
	for i, vol := range volumes {
		directories[i] = map[string]any{
			"path":        vol.Path,
			"owner":       vol.Owner,
			"permissions": vol.Permissions,
		}
	}

	return &PipelineStepResult{
		Uses: "create-directories",
		With: map[string]any{
			"directories": directories,
		},
	}
}

func ParseExtraCopies(params map[string]any) ([]ExtraCopySpec, error) {
	copiesParam, ok := params["extra-copies"]
	if !ok {
		return nil, nil
	}

	copiesList, ok := copiesParam.([]any)
	if !ok {
		return nil, fmt.Errorf("extra-copies must be a list")
	}

	copies := make([]ExtraCopySpec, 0, len(copiesList))
	for i, c := range copiesList {
		copyMap, ok := c.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("extra-copy at index %d must be a map", i)
		}

		from, ok := copyMap["from"].(string)
		if !ok || from == "" {
			return nil, fmt.Errorf("extra-copy at index %d must have a 'from' path", i)
		}

		to, ok := copyMap["to"].(string)
		if !ok || to == "" {
			return nil, fmt.Errorf("extra-copy at index %d must have a 'to' path", i)
		}

		copies = append(copies, ExtraCopySpec{
			From: from,
			To:   to,
		})
	}

	return copies, nil
}
