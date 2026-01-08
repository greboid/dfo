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
	repo, _ := params["repo"].(string)
	binary, _ := params["binary"].(string)

	workdir := "/src"
	if wd, ok := params["workdir"].(string); ok {
		workdir = wd
	}

	pkg := "."
	if p, ok := params["package"].(string); ok {
		pkg = p
	}

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

	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}

	extraCopies, err := ParseExtraCopies(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing extra-copies: %w", err)
	}

	buildPipeline := []PipelineStepResult{
		{
			Uses: "build-go-static",
			With: buildParams,
		},
	}

	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		buildPipeline = append(buildPipeline, *volumeStep)
	}

	buildStage := StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "golang",
		},
		Pipeline: buildPipeline,
	}

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

	rootfsStage := StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: rootfsPipeline,
	}

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
		finalStage.Environment.Expose = make([]string, len(expose))
		for i, port := range expose {
			if portStr, ok := port.(string); ok {
				finalStage.Environment.Expose[i] = portStr
			}
		}
	}

	if entrypoint, ok := params["entrypoint"].([]any); ok {
		finalStage.Environment.Entrypoint = make([]string, len(entrypoint))
		for i, arg := range entrypoint {
			if argStr, ok := arg.(string); ok {
				finalStage.Environment.Entrypoint[i] = argStr
			}
		}
	}

	if cmd, ok := params["cmd"].([]any); ok {
		finalStage.Environment.Cmd = make([]string, len(cmd))
		for i, arg := range cmd {
			if argStr, ok := arg.(string); ok {
				finalStage.Environment.Cmd[i] = argStr
			}
		}
	}

	return TemplateResult{
		Stages: []StageResult{buildStage, rootfsStage, finalStage},
	}, nil
}

func rustApp(params map[string]any) (TemplateResult, error) {
	repo, _ := params["repo"].(string)
	binary, _ := params["binary"].(string)

	packages := []string{"git"}
	if pkgs, ok := params["packages"].([]any); ok {
		for _, pkg := range pkgs {
			if pkgStr, ok := pkg.(string); ok {
				packages = append(packages, pkgStr)
			}
		}
	}

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

	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}

	buildPipeline := []PipelineStepResult{
		{
			Uses: "clone-and-build-rust",
			With: buildParams,
		},
	}

	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		buildPipeline = append(buildPipeline, *volumeStep)
	}

	buildStage := StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "rust",
			Packages:  packages,
		},
		Pipeline: buildPipeline,
	}

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

	rootfsStage := StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: rootfsPipeline,
	}

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
		finalStage.Environment.Expose = make([]string, len(expose))
		for i, port := range expose {
			if portStr, ok := port.(string); ok {
				finalStage.Environment.Expose[i] = portStr
			}
		}
	}

	if entrypoint, ok := params["entrypoint"].([]any); ok {
		finalStage.Environment.Entrypoint = make([]string, len(entrypoint))
		for i, arg := range entrypoint {
			if argStr, ok := arg.(string); ok {
				finalStage.Environment.Entrypoint[i] = argStr
			}
		}
	}

	if cmd, ok := params["cmd"].([]any); ok {
		finalStage.Environment.Cmd = make([]string, len(cmd))
		for i, arg := range cmd {
			if argStr, ok := arg.(string); ok {
				finalStage.Environment.Cmd[i] = argStr
			}
		}
	}

	return TemplateResult{
		Stages: []StageResult{buildStage, rootfsStage, finalStage},
	}, nil
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

	binaries := make([]BinarySpec, 0, len(binariesList))
	for i, b := range binariesList {
		binaryMap, ok := b.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("binary at index %d must be a map", i)
		}

		repo, ok := binaryMap["repo"].(string)
		if !ok || repo == "" {
			return nil, fmt.Errorf("binary at index %d must have a repo", i)
		}

		binary, ok := binaryMap["binary"].(string)
		if !ok || binary == "" {
			return nil, fmt.Errorf("binary at index %d must have a binary name", i)
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

		binaries = append(binaries, spec)
	}

	return binaries, nil
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

	clonedRepos := make(map[string]string)

	var buildPipeline []PipelineStepResult

	for _, bin := range binaries {
		workdir, alreadyCloned := clonedRepos[bin.Repo]
		if !alreadyCloned {
			workdir = "/src"
			if ownerRepo := pipelines.ExtractGitHubOwnerRepo(bin.Repo); ownerRepo != "" {
				workdir = "/src/" + ownerRepo
			}
			clonedRepos[bin.Repo] = workdir

			cloneWith := map[string]any{
				"repo":    bin.Repo,
				"workdir": workdir,
			}
			if bin.Tag != "" {
				cloneWith["tag"] = bin.Tag
			}
			buildPipeline = append(buildPipeline, PipelineStepResult{
				Uses: "clone",
				With: cloneWith,
			})
		}

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

		buildPipeline = append(buildPipeline, PipelineStepResult{
			Uses: "build-go-only",
			With: buildParams,
		})
	}

	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		buildPipeline = append(buildPipeline, *volumeStep)
	}

	buildStage := StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "golang",
		},
		Pipeline: buildPipeline,
	}

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

	rootfsStage := StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: rootfsPipeline,
	}

	entrypointBinary := binaries[0].Binary
	for _, bin := range binaries {
		if bin.Entrypoint {
			entrypointBinary = bin.Binary
			break
		}
	}

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
		finalStage.Environment.Expose = make([]string, len(expose))
		for i, port := range expose {
			if portStr, ok := port.(string); ok {
				finalStage.Environment.Expose[i] = portStr
			}
		}
	}

	if entrypoint, ok := params["entrypoint"].([]any); ok {
		finalStage.Environment.Entrypoint = make([]string, len(entrypoint))
		for i, arg := range entrypoint {
			if argStr, ok := arg.(string); ok {
				finalStage.Environment.Entrypoint[i] = argStr
			}
		}
	}

	if cmd, ok := params["cmd"].([]any); ok {
		finalStage.Environment.Cmd = make([]string, len(cmd))
		for i, arg := range cmd {
			if argStr, ok := arg.(string); ok {
				finalStage.Environment.Cmd[i] = argStr
			}
		}
	}

	return TemplateResult{
		Stages: []StageResult{buildStage, rootfsStage, finalStage},
	}, nil
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
