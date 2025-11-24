package config

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"

	"github.com/greboid/dfo/pkg/templates"
	"gopkg.in/yaml.v3"
)

func Load(fs fs.ReadFileFS, path string) (*BuildConfig, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return Parse(data)
}

func Parse(data []byte) (*BuildConfig, error) {
	var config BuildConfig

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := expandTemplates(&config); err != nil {
		return nil, err
	}

	if err := Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func expandTemplates(config *BuildConfig) error {
	var expandedStages []Stage

	for i := range config.Stages {
		stage := &config.Stages[i]

		if stage.Template == "" {
			expandedStages = append(expandedStages, *stage)
			continue
		}

		if !stage.Environment.IsEmpty() {
			return fmt.Errorf("stage %d: cannot specify both 'template' and 'environment'", i)
		}
		if len(stage.Pipeline) > 0 {
			return fmt.Errorf("stage %d: cannot specify both 'template' and 'pipeline'", i)
		}

		if err := templates.ValidateTemplateParams(stage.Template, stage.With); err != nil {
			return fmt.Errorf("stage %d with template %q: %w", i, stage.Template, err)
		}

		templateFunc, exists := templates.Registry[stage.Template]
		if !exists {
			return fmt.Errorf("stage %d: unknown template %q", i, stage.Template)
		}

		templateResult, err := templateFunc(stage.With)
		if err != nil {
			return fmt.Errorf("stage %d with template %q: %w", i, stage.Template, err)
		}

		for j, stageResult := range templateResult.Stages {
			newStage := convertStageResult(stageResult)

			if stage.Name != "" && len(templateResult.Stages) == 1 {
				newStage.Name = stage.Name
			} else if newStage.Name == "" {
				if len(templateResult.Stages) == 1 {
					newStage.Name = fmt.Sprintf("%s-%d", stage.Template, i)
				} else {
					newStage.Name = fmt.Sprintf("%s-%d-%d", stage.Template, i, j)
				}
			}
			expandedStages = append(expandedStages, newStage)
		}
	}

	config.Stages = expandedStages
	return nil
}

func convertStageResult(stageResult templates.StageResult) Stage {
	stage := Stage{
		Name: stageResult.Name,
		Environment: Environment{
			BaseImage:      stageResult.Environment.BaseImage,
			ExternalImage:  stageResult.Environment.ExternalImage,
			Packages:       stageResult.Environment.Packages,
			RootfsPackages: stageResult.Environment.RootfsPackages,
			User:           stageResult.Environment.User,
			WorkDir:        stageResult.Environment.WorkDir,
			Expose:         stageResult.Environment.Expose,
			Entrypoint:     stageResult.Environment.Entrypoint,
			Cmd:            stageResult.Environment.Cmd,
		},
		Pipeline: make([]PipelineStep, len(stageResult.Pipeline)),
	}

	for i, step := range stageResult.Pipeline {
		pipelineStep := PipelineStep{
			Uses: step.Uses,
			Run:  step.Run,
			With: step.With,
		}
		if step.Copy != nil {
			pipelineStep.Copy = &CopyStep{
				FromStage: step.Copy.FromStage,
				From:      step.Copy.From,
				To:        step.Copy.To,
				Chown:     step.Copy.Chown,
			}
		}
		stage.Pipeline[i] = pipelineStep
	}

	return stage
}

func Validate(config *BuildConfig) error {
	if config.Package.Name == "" {
		return fmt.Errorf("package.name is required")
	}

	if len(config.Stages) == 0 {
		return fmt.Errorf("at least one stage is required in the 'stages' array")
	}

	for _, stage := range config.Stages {
		if err := validateStage(stage); err != nil {
			return err
		}
	}

	if !config.Environment.IsEmpty() {
		return fmt.Errorf("cannot specify top-level environment when using stages")
	}

	return nil
}

func validateStage(stage Stage) error {
	if stage.Name == "" {
		return fmt.Errorf("stage name is required")
	}
	if strings.ContainsAny(stage.Name, " \t\n\r") {
		return fmt.Errorf("stage %q: name must be a single word (no whitespace allowed)", stage.Name)
	}

	hasBaseImage := stage.Environment.BaseImage != ""
	hasExternalImage := stage.Environment.ExternalImage != ""

	if !hasBaseImage && !hasExternalImage {
		return fmt.Errorf("stage %q: either environment.base-image or environment.external-image is required", stage.Name)
	}
	if hasBaseImage && hasExternalImage {
		return fmt.Errorf("stage %q: cannot specify both environment.base-image and environment.external-image", stage.Name)
	}

	return nil
}
