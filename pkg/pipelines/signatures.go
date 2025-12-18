package pipelines

import (
	"fmt"
	"strings"
)

type ParamType string

const (
	TypeString      ParamType = "string"
	TypeInt         ParamType = "int"
	TypeBool        ParamType = "bool"
	TypeStringArray ParamType = "string_array"
	TypeObjectArray ParamType = "object_array"
)

type ParamSpec struct {
	Type        ParamType
	Required    bool
	Description string
}

type PipelineSignature struct {
	Name              string
	Description       string
	Parameters        map[string]ParamSpec
	MutuallyExclusive [][]string
	AtLeastOne        [][]string
}

var Signatures = map[string]PipelineSignature{
	"create-user": {
		Name:        "create-user",
		Description: "Create a user and group in the container",
		Parameters: map[string]ParamSpec{
			"username": {Type: TypeString, Required: true, Description: "Username to create"},
			"uid":      {Type: TypeInt, Required: true, Description: "User ID"},
			"gid":      {Type: TypeInt, Required: true, Description: "Group ID"},
		},
	},
	"set-ownership": {
		Name:        "set-ownership",
		Description: "Change ownership of a path",
		Parameters: map[string]ParamSpec{
			"user":  {Type: TypeString, Required: true, Description: "User name or ID"},
			"group": {Type: TypeString, Required: true, Description: "Group name or ID"},
			"path":  {Type: TypeString, Required: true, Description: "Path to change ownership of"},
		},
	},
	"download-verify-extract": {
		Name:        "download-verify-extract",
		Description: "Download a file, verify its checksum, and optionally extract it",
		Parameters: map[string]ParamSpec{
			"url":              {Type: TypeString, Required: true, Description: "URL to download"},
			"destination":      {Type: TypeString, Required: true, Description: "Destination path for downloaded file"},
			"checksum":         {Type: TypeString, Required: false, Description: "Expected SHA256 checksum"},
			"checksum-url":     {Type: TypeString, Required: false, Description: "URL to fetch checksum from"},
			"checksum-pattern": {Type: TypeString, Required: false, Description: "Pattern to extract checksum from checksum file"},
			"extract-dir":      {Type: TypeString, Required: false, Description: "Directory to extract archive to"},
			"strip-components": {Type: TypeInt, Required: false, Description: "Number of path components to strip during extraction"},
		},
		MutuallyExclusive: [][]string{{"checksum", "checksum-url"}},
		AtLeastOne:        [][]string{{"checksum", "checksum-url"}},
	},
	"make-executable": {
		Name:        "make-executable",
		Description: "Make a file executable",
		Parameters: map[string]ParamSpec{
			"path": {Type: TypeString, Required: true, Description: "Path to make executable"},
		},
	},
	"clone": {
		Name:        "clone",
		Description: "Clone a git repository",
		Parameters: map[string]ParamSpec{
			"repo":    {Type: TypeString, Required: true, Description: "Repository URL"},
			"workdir": {Type: TypeString, Required: false, Description: "Working directory for clone (default: /src)"},
			"tag":     {Type: TypeString, Required: false, Description: "Tag or branch to checkout"},
			"commit":  {Type: TypeString, Required: false, Description: "Specific commit to checkout"},
		},
		MutuallyExclusive: [][]string{{"tag", "commit"}},
	},
	"clone-and-build-go": {
		Name:        "clone-and-build-go",
		Description: "Clone a Go repository and build it",
		Parameters: map[string]ParamSpec{
			"repo":    {Type: TypeString, Required: true, Description: "Repository URL"},
			"package": {Type: TypeString, Required: false, Description: "Go package to build (default: .)"},
			"output":  {Type: TypeString, Required: false, Description: "Output binary path (default: /main)"},
			"tag":     {Type: TypeString, Required: false, Description: "Tag or branch to checkout"},
			"go-tags": {Type: TypeString, Required: false, Description: "Additional Go build tags (default: netgo,osusergo)"},
			"cgo":     {Type: TypeBool, Required: false, Description: "Enable CGO (default: true)"},
			"ignore":  {Type: TypeStringArray, Required: false, Description: "Packages to ignore for license generation"},
			"patches": {Type: TypeStringArray, Required: false, Description: "Patch files to apply"},
		},
	},
	"build-go-static": {
		Name:        "build-go-static",
		Description: "Clone and build a statically linked Go binary",
		Parameters: map[string]ParamSpec{
			"repo":    {Type: TypeString, Required: true, Description: "Repository URL"},
			"workdir": {Type: TypeString, Required: false, Description: "Working directory (default: /src)"},
			"package": {Type: TypeString, Required: false, Description: "Go package to build (default: .)"},
			"output":  {Type: TypeString, Required: false, Description: "Output binary path (default: /main)"},
			"ignore":  {Type: TypeStringArray, Required: false, Description: "Packages to ignore for license generation"},
			"tag":     {Type: TypeString, Required: false, Description: "Tag or branch to checkout"},
			"go-tags": {Type: TypeString, Required: false, Description: "Additional Go build tags (default: netgo,osusergo)"},
			"cgo":     {Type: TypeBool, Required: false, Description: "Enable CGO (default: true)"},
			"patches": {Type: TypeStringArray, Required: false, Description: "Patch files to apply"},
		},
	},
	"build-go-only": {
		Name:        "build-go-only",
		Description: "Build a statically linked Go binary (without cloning - repo must already be cloned)",
		Parameters: map[string]ParamSpec{
			"workdir": {Type: TypeString, Required: true, Description: "Working directory where repo is already cloned"},
			"package": {Type: TypeString, Required: false, Description: "Go package to build (default: .)"},
			"output":  {Type: TypeString, Required: false, Description: "Output binary path (default: /main)"},
			"ignore":  {Type: TypeStringArray, Required: false, Description: "Packages to ignore for license generation"},
			"go-tags": {Type: TypeString, Required: false, Description: "Additional Go build tags (default: netgo,osusergo)"},
			"cgo":     {Type: TypeBool, Required: false, Description: "Enable CGO (default: false)"},
		},
	},
	"clone-and-build-rust": {
		Name:        "clone-and-build-rust",
		Description: "Clone a Rust repository and build it",
		Parameters: map[string]ParamSpec{
			"repo":     {Type: TypeString, Required: true, Description: "Repository URL"},
			"workdir":  {Type: TypeString, Required: false, Description: "Working directory (default: /src)"},
			"features": {Type: TypeString, Required: false, Description: "Cargo features to enable"},
			"output":   {Type: TypeString, Required: false, Description: "Output binary path (default: /main)"},
			"tag":      {Type: TypeString, Required: false, Description: "Tag or branch to checkout"},
			"patches":  {Type: TypeStringArray, Required: false, Description: "Patch files to apply"},
		},
	},
	"clone-and-build-make": {
		Name:        "clone-and-build-make",
		Description: "Clone a repository and build with make",
		Parameters: map[string]ParamSpec{
			"repo":       {Type: TypeString, Required: true, Description: "Repository URL"},
			"workdir":    {Type: TypeString, Required: false, Description: "Working directory (default: /src)"},
			"tag":        {Type: TypeString, Required: false, Description: "Tag or branch to checkout"},
			"make-steps": {Type: TypeStringArray, Required: false, Description: "Make commands to run"},
			"strip":      {Type: TypeBool, Required: false, Description: "Strip binaries after build (default: true)"},
		},
	},
	"clone-and-build-autoconf": {
		Name:        "clone-and-build-autoconf",
		Description: "Clone a repository and build with autoconf/configure",
		Parameters: map[string]ParamSpec{
			"repo":              {Type: TypeString, Required: true, Description: "Repository URL"},
			"workdir":           {Type: TypeString, Required: false, Description: "Working directory (default: /src)"},
			"tag":               {Type: TypeString, Required: false, Description: "Tag or branch to checkout"},
			"configure-options": {Type: TypeStringArray, Required: false, Description: "Options to pass to configure"},
			"make-steps":        {Type: TypeStringArray, Required: false, Description: "Make commands to run"},
			"strip":             {Type: TypeBool, Required: false, Description: "Strip binaries after build (default: true)"},
		},
	},
	"setup-users-groups": {
		Name:        "setup-users-groups",
		Description: "Set up users and groups in a rootfs",
		Parameters: map[string]ParamSpec{
			"rootfs": {Type: TypeString, Required: false, Description: "Root filesystem path"},
			"groups": {Type: TypeObjectArray, Required: false, Description: "Groups to create (name, gid)"},
			"users":  {Type: TypeObjectArray, Required: false, Description: "Users to create (username, uid, gid, home, shell)"},
		},
		AtLeastOne: [][]string{{"groups", "users"}},
	},
	"create-directories": {
		Name:        "create-directories",
		Description: "Create directories with optional permissions",
		Parameters: map[string]ParamSpec{
			"directories": {Type: TypeObjectArray, Required: true, Description: "Directories to create (path, permissions)"},
		},
	},
	"copy-files": {
		Name:        "copy-files",
		Description: "Copy files into the container",
		Parameters: map[string]ParamSpec{
			"files": {Type: TypeObjectArray, Required: true, Description: "Files to copy (from, to, chown, chmod)"},
		},
	},
}

func ValidateParams(pipelineName string, params map[string]any) error {
	sig, exists := Signatures[pipelineName]
	if !exists {
		return nil
	}

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
			errors = append(errors, fmt.Sprintf("cannot specify both %s", strings.Join(present, " and ")))
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
			errors = append(errors, fmt.Sprintf("at least one of %s is required", strings.Join(group, ", ")))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("pipeline %q: %s", pipelineName, strings.Join(errors, "; "))
	}
	return nil
}

func checkType(paramName string, value any, expectedType ParamType) error {
	switch expectedType {
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %q must be a string, got %T", paramName, value)
		}
	case TypeInt:
		if _, ok := value.(float64); ok {
			return nil
		}
		if _, ok := value.(int); ok {
			return nil
		}
		return fmt.Errorf("parameter %q must be an integer, got %T", paramName, value)
	case TypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter %q must be a boolean, got %T", paramName, value)
		}
	case TypeStringArray:
		switch v := value.(type) {
		case string:
			return nil
		case []string:
			return nil
		case []any:
			for i, item := range v {
				if _, ok := item.(string); !ok {
					return fmt.Errorf("parameter %q[%d] must be a string, got %T", paramName, i, item)
				}
			}
		default:
			return fmt.Errorf("parameter %q must be a string or array of strings, got %T", paramName, value)
		}
	case TypeObjectArray:
		switch v := value.(type) {
		case []any:
			for i, item := range v {
				if _, ok := item.(map[string]any); !ok {
					return fmt.Errorf("parameter %q[%d] must be an object, got %T", paramName, i, item)
				}
			}
		case []map[string]any:
			// Already the correct type, no validation needed
		default:
			return fmt.Errorf("parameter %q must be an array of objects, got %T", paramName, value)
		}
	}
	return nil
}
