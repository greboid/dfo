package templates

import (
	"fmt"
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

	return TemplateResult{Stages: []StageResult{stage}}, nil
}

func goApp(params map[string]any) (TemplateResult, error) {
	repo, _ := params["repo"].(string)
	binary, _ := params["binary"].(string)

	workdir := "/src"
	if wd, ok := params["workdir"].(string); ok {
		workdir = wd
	}

	pkg := "./cmd/" + binary
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

	rootfsStage := StageResult{
		Name: "rootfs",
		Environment: EnvironmentResult{
			BaseImage: "base",
		},
		Pipeline: []PipelineStepResult{
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
		},
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
