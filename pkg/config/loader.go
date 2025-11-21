package config

import (
	"fmt"
	"io/fs"

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
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
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
