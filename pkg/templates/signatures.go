package templates

import (
	"fmt"

	"github.com/greboid/dfo/pkg/pipelines"
)

var Signatures = map[string]pipelines.PipelineSignature{
	"go-builder": {
		Name:        "go-builder",
		Description: "Stage for cloning and building Go projects",
		Parameters: map[string]pipelines.ParamSpec{
			"repo":        {Type: pipelines.TypeString, Required: true},
			"output":      {Type: pipelines.TypeString, Required: true},
			"branch":      {Type: pipelines.TypeString, Required: false},
			"tag":         {Type: pipelines.TypeString, Required: false},
			"commit":      {Type: pipelines.TypeString, Required: false},
			"go-version":  {Type: pipelines.TypeString, Required: false},
			"build-flags": {Type: pipelines.TypeString, Required: false},
		},
		MutuallyExclusive: [][]string{{"branch", "tag", "commit"}},
	},
	"rust-builder": {
		Name:        "rust-builder",
		Description: "Stage for cloning and building Rust projects",
		Parameters: map[string]pipelines.ParamSpec{
			"repo":     {Type: pipelines.TypeString, Required: true},
			"output":   {Type: pipelines.TypeString, Required: false},
			"branch":   {Type: pipelines.TypeString, Required: false},
			"tag":      {Type: pipelines.TypeString, Required: false},
			"commit":   {Type: pipelines.TypeString, Required: false},
			"profile":  {Type: pipelines.TypeString, Required: false},
			"features": {Type: pipelines.TypeString, Required: false},
			"patches":  {Type: pipelines.TypeStringArray, Required: false},
			"workdir":  {Type: pipelines.TypeString, Required: false},
			"packages": {Type: pipelines.TypeStringArray, Required: false},
		},
		MutuallyExclusive: [][]string{{"branch", "tag", "commit"}},
	},
	"go-app": {
		Name:        "go-app",
		Description: "Complete Go application with build, rootfs, and final stages",
		Parameters: map[string]pipelines.ParamSpec{
			"repo":         {Type: pipelines.TypeString, Required: true},
			"package":      {Type: pipelines.TypeString, Required: false},
			"binary":       {Type: pipelines.TypeString, Required: true},
			"workdir":      {Type: pipelines.TypeString, Required: false},
			"ignore":       {Type: pipelines.TypeString, Required: false},
			"expose":       {Type: pipelines.TypeStringArray, Required: false},
			"cmd":          {Type: pipelines.TypeStringArray, Required: false},
			"extra-copies": {Type: pipelines.TypeObjectArray, Required: false},
		},
	},
	"multi-go-app": {
		Name:        "multi-go-app",
		Description: "Complete Go application with multiple binaries",
		Parameters: map[string]pipelines.ParamSpec{
			"binaries":     {Type: pipelines.TypeObjectArray, Required: true},
			"extra-copies": {Type: pipelines.TypeObjectArray, Required: false},
			"volumes":      {Type: pipelines.TypeObjectArray, Required: false},
			"expose":       {Type: pipelines.TypeStringArray, Required: false},
			"cmd":          {Type: pipelines.TypeStringArray, Required: false},
			"entrypoint":   {Type: pipelines.TypeStringArray, Required: false},
		},
	},
	"rust-app": {
		Name:        "rust-app",
		Description: "Complete Rust application with build, rootfs, and final stages",
		Parameters: map[string]pipelines.ParamSpec{
			"repo":     {Type: pipelines.TypeString, Required: true},
			"binary":   {Type: pipelines.TypeString, Required: true},
			"workdir":  {Type: pipelines.TypeString, Required: false},
			"features": {Type: pipelines.TypeString, Required: false},
			"patches":  {Type: pipelines.TypeStringArray, Required: false},
			"packages": {Type: pipelines.TypeStringArray, Required: false},
			"tag":      {Type: pipelines.TypeString, Required: false},
			"expose":   {Type: pipelines.TypeStringArray, Required: false},
			"cmd":      {Type: pipelines.TypeStringArray, Required: false},
		},
	},
}

func ValidateTemplateParams(templateName string, params map[string]any) error {
	sig, exists := Signatures[templateName]
	if !exists {
		return fmt.Errorf("unknown template: %s", templateName)
	}

	return validateSignature(sig, params)
}

func validateSignature(sig pipelines.PipelineSignature, params map[string]any) error {
	var errors []string

	for paramName, spec := range sig.Parameters {
		if spec.Required {
			val, exists := params[paramName]
			if !exists || val == nil {
				errors = append(errors, fmt.Sprintf("required parameter %q is missing", paramName))
				continue
			}

			if err := checkType(paramName, val, spec.Type); err != nil {
				errors = append(errors, err.Error())
			}
		} else {
			if val, exists := params[paramName]; exists && val != nil {
				if err := checkType(paramName, val, spec.Type); err != nil {
					errors = append(errors, err.Error())
				}
			}
		}
	}

	for _, group := range sig.MutuallyExclusive {
		var present []string
		for _, param := range group {
			if val, exists := params[param]; exists && val != nil {
				if str, ok := val.(string); ok && str == "" {
					continue
				}
				present = append(present, param)
			}
		}
		if len(present) > 1 {
			errors = append(errors, fmt.Sprintf("cannot specify both %s", joinStrings(present, " and ")))
		}
	}

	for _, group := range sig.AtLeastOne {
		hasAny := false
		for _, param := range group {
			if val, exists := params[param]; exists && val != nil {
				if str, ok := val.(string); ok && str == "" {
					continue
				}
				hasAny = true
				break
			}
		}
		if !hasAny {
			errors = append(errors, fmt.Sprintf("at least one of %s is required", joinStrings(group, ", ")))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", joinStrings(errors, "; "))
	}

	return nil
}

func checkType(paramName string, val any, expectedType pipelines.ParamType) error {
	switch expectedType {
	case pipelines.TypeString:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("parameter %q must be a string", paramName)
		}
	case pipelines.TypeInt:
		if _, ok := val.(int); !ok {
			return fmt.Errorf("parameter %q must be an integer", paramName)
		}
	case pipelines.TypeBool:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("parameter %q must be a boolean", paramName)
		}
	case pipelines.TypeStringArray:
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("parameter %q must be an array", paramName)
		}
	case pipelines.TypeObjectArray:
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("parameter %q must be an array", paramName)
		}
	}
	return nil
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, str := range strs {
		if i > 0 {
			result += sep
		}
		result += str
	}
	return result
}
