package templates

import (
	"fmt"
)

// Volume defaults matching postgres-15 conventions
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
	"base":         base,
	"baseroot":     baseroot,
	"go-app":       goApp,
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
	return TemplateResult{
		Stages: []StageResult{
			{
				Environment: EnvironmentResult{
					BaseImage: "rust",
					Packages:  []string{"git"},
				},
				Pipeline: []PipelineStepResult{
					{
						Uses: "clone-and-build-rust",
						With: params,
					},
				},
			},
		},
	}, nil
}

func base(params map[string]any) (TemplateResult, error) {
	stage := StageResult{
		Environment: EnvironmentResult{
			BaseImage: "alpine",
		},
	}

	if packages, ok := params["packages"].([]any); ok {
		stage.Environment.Packages = make([]string, len(packages))
		for i, pkg := range packages {
			if pkgStr, ok := pkg.(string); ok {
				stage.Environment.Packages[i] = pkgStr
			} else {
				return TemplateResult{}, fmt.Errorf("package must be a string")
			}
		}
	}

	if user, ok := params["user"].(string); ok {
		stage.Environment.User = user
	}

	if workdir, ok := params["workdir"].(string); ok {
		stage.Environment.WorkDir = workdir
	}

	return TemplateResult{Stages: []StageResult{stage}}, nil
}

func baseroot(params map[string]any) (TemplateResult, error) {
	stage := StageResult{
		Environment: EnvironmentResult{
			BaseImage: "alpine",
		},
		Pipeline: []PipelineStepResult{},
	}

	if packages, ok := params["packages"].([]any); ok {
		stage.Environment.Packages = make([]string, len(packages))
		for i, pkg := range packages {
			if pkgStr, ok := pkg.(string); ok {
				stage.Environment.Packages[i] = pkgStr
			} else {
				return TemplateResult{}, fmt.Errorf("package must be a string")
			}
		}
	}

	user := "appuser"
	if u, ok := params["user"].(string); ok {
		user = u
	}

	group := user
	if g, ok := params["group"].(string); ok {
		group = g
	}

	uid := 1000
	if u, ok := params["uid"].(int); ok {
		uid = u
	}

	gid := 1000
	if g, ok := params["gid"].(int); ok {
		gid = g
	}

	stage.Pipeline = append(stage.Pipeline, PipelineStepResult{
		Uses: "setup-users-groups",
		With: map[string]any{
			"users": []map[string]any{
				{
					"name": user,
					"uid":  uid,
					"gid":  gid,
				},
			},
			"groups": []map[string]any{
				{
					"name": group,
					"gid":  gid,
				},
			},
		},
	})

	stage.Environment.User = user

	if workdir, ok := params["workdir"].(string); ok {
		stage.Environment.WorkDir = workdir
	}

	// Add volumes if specified
	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}
	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		stage.Pipeline = append(stage.Pipeline, *volumeStep)
	}

	return TemplateResult{Stages: []StageResult{stage}}, nil
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
	if ignore, ok := params["ignore"].(string); ok {
		buildParams["ignore"] = ignore
	}
	if patches, ok := params["patches"]; ok {
		buildParams["patches"] = patches
	}

	buildStage := StageResult{
		Name: "build",
		Environment: EnvironmentResult{
			BaseImage: "golang",
		},
		Pipeline: []PipelineStepResult{
			{
				Uses: "build-go-static",
				With: buildParams,
			},
		},
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

	// Add volumes if specified
	volumes, err := ParseVolumes(params)
	if err != nil {
		return TemplateResult{}, fmt.Errorf("parsing volumes: %w", err)
	}
	if volumeStep := CreateVolumesStep(volumes); volumeStep != nil {
		rootfsPipeline = append(rootfsPipeline, *volumeStep)
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

// VolumeSpec represents a volume directory specification
type VolumeSpec struct {
	Path        string
	Owner       string
	Permissions string
}

// ParseVolumes extracts volume specifications from template params.
// Each volume can specify path (required), owner (optional), and permissions (optional).
// Defaults are applied from postgres-15 conventions: owner "65532:65532", permissions "777".
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

// CreateVolumesStep generates a create-directories pipeline step from volume specs.
// Returns nil if no volumes are specified.
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
