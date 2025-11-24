package templates

import (
	"testing"

	"github.com/greboid/dfo/pkg/pipelines"
)

func TestValidateTemplateParams(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		params       map[string]any
		wantErr      bool
	}{
		{
			name:         "go-builder with required params",
			templateName: "go-builder",
			params: map[string]any{
				"repo":   "https://github.com/owner/repo",
				"output": "/app/binary",
			},
			wantErr: false,
		},
		{
			name:         "go-builder missing required param",
			templateName: "go-builder",
			params: map[string]any{
				"repo": "https://github.com/owner/repo",
			},
			wantErr: true,
		},
		{
			name:         "base with optional params",
			templateName: "base",
			params: map[string]any{
				"packages": []any{"ca-certificates"},
				"user":     "appuser",
			},
			wantErr: false,
		},
		{
			name:         "base with no params",
			templateName: "base",
			params:       map[string]any{},
			wantErr:      false,
		},
		{
			name:         "unknown template",
			templateName: "nonexistent",
			params:       map[string]any{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplateParams(tt.templateName, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGoBuilder(t *testing.T) {
	params := map[string]any{
		"repo":   "https://github.com/owner/repo",
		"output": "/app/binary",
	}

	result, err := goBuilder(params)
	if err != nil {
		t.Fatalf("goBuilder() error = %v", err)
	}

	if len(result.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(result.Stages))
	}

	stage := result.Stages[0]
	if stage.Environment.BaseImage != "golang" {
		t.Errorf("expected base image golang, got %s", stage.Environment.BaseImage)
	}

	if len(stage.Pipeline) != 1 {
		t.Errorf("expected 1 pipeline step, got %d", len(stage.Pipeline))
	}

	if stage.Pipeline[0].Uses != "clone-and-build-go" {
		t.Errorf("expected pipeline to use clone-and-build-go, got %s", stage.Pipeline[0].Uses)
	}
}

func TestBase(t *testing.T) {
	params := map[string]any{
		"packages": []any{"ca-certificates", "tzdata"},
		"user":     "appuser",
		"workdir":  "/app",
	}

	result, err := base(params)
	if err != nil {
		t.Fatalf("base() error = %v", err)
	}

	if len(result.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(result.Stages))
	}

	stage := result.Stages[0]
	if stage.Environment.BaseImage != "alpine" {
		t.Errorf("expected base image alpine, got %s", stage.Environment.BaseImage)
	}

	if len(stage.Environment.Packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(stage.Environment.Packages))
	}

	if stage.Environment.User != "appuser" {
		t.Errorf("expected user appuser, got %s", stage.Environment.User)
	}

	if stage.Environment.WorkDir != "/app" {
		t.Errorf("expected workdir /app, got %s", stage.Environment.WorkDir)
	}
}

func TestBaseroot(t *testing.T) {
	params := map[string]any{
		"user":    "myuser",
		"group":   "mygroup",
		"uid":     1500,
		"gid":     1500,
		"workdir": "/app",
	}

	result, err := baseroot(params)
	if err != nil {
		t.Fatalf("baseroot() error = %v", err)
	}

	if len(result.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(result.Stages))
	}

	stage := result.Stages[0]
	if stage.Environment.BaseImage != "alpine" {
		t.Errorf("expected base image alpine, got %s", stage.Environment.BaseImage)
	}

	if len(stage.Pipeline) != 1 {
		t.Errorf("expected 1 pipeline step, got %d", len(stage.Pipeline))
	}

	if stage.Pipeline[0].Uses != "setup-users-groups" {
		t.Errorf("expected pipeline to use setup-users-groups, got %s", stage.Pipeline[0].Uses)
	}

	if stage.Environment.User != "myuser" {
		t.Errorf("expected user myuser, got %s", stage.Environment.User)
	}

	if stage.Environment.WorkDir != "/app" {
		t.Errorf("expected workdir /app, got %s", stage.Environment.WorkDir)
	}
}

func TestGoApp(t *testing.T) {
	params := map[string]any{
		"repo":   "https://github.com/tailscale/golink",
		"binary": "golink",
		"expose": []any{"8080"},
		"cmd":    []any{"--sqlitedb", "/home/nonroot/golink.db"},
	}

	result, err := goApp(params)
	if err != nil {
		t.Fatalf("goApp() error = %v", err)
	}

	if len(result.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(result.Stages))
	}

	buildStage := result.Stages[0]
	if buildStage.Name != "build" {
		t.Errorf("expected build stage name 'build', got %q", buildStage.Name)
	}
	if buildStage.Environment.BaseImage != "golang" {
		t.Errorf("expected golang base image, got %q", buildStage.Environment.BaseImage)
	}

	rootfsStage := result.Stages[1]
	if rootfsStage.Name != "rootfs" {
		t.Errorf("expected rootfs stage name 'rootfs', got %q", rootfsStage.Name)
	}
	if len(rootfsStage.Pipeline) != 2 {
		t.Errorf("expected 2 copy steps in rootfs, got %d", len(rootfsStage.Pipeline))
	}

	finalStage := result.Stages[2]
	if finalStage.Name != "final" {
		t.Errorf("expected final stage name 'final', got %q", finalStage.Name)
	}
	if len(finalStage.Environment.Entrypoint) == 0 || finalStage.Environment.Entrypoint[0] != "/golink" {
		t.Errorf("expected entrypoint /golink, got %v", finalStage.Environment.Entrypoint)
	}
	if len(finalStage.Environment.Expose) != 1 || finalStage.Environment.Expose[0] != "8080" {
		t.Errorf("expected expose 8080, got %v", finalStage.Environment.Expose)
	}
	if len(finalStage.Environment.Cmd) != 2 {
		t.Errorf("expected 2 cmd args, got %d", len(finalStage.Environment.Cmd))
	}
}

func TestValidateSignature(t *testing.T) {
	tests := []struct {
		name        string
		sig         pipelines.PipelineSignature
		params      map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "all required params present",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name": {Type: pipelines.TypeString, Required: true},
					"age":  {Type: pipelines.TypeInt, Required: true},
				},
			},
			params: map[string]any{
				"name": "test",
				"age":  25,
			},
			wantErr: false,
		},
		{
			name: "missing required param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name": {Type: pipelines.TypeString, Required: true},
				},
			},
			params:      map[string]any{},
			wantErr:     true,
			errContains: "required parameter \"name\" is missing",
		},
		{
			name: "nil required param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name": {Type: pipelines.TypeString, Required: true},
				},
			},
			params: map[string]any{
				"name": nil,
			},
			wantErr:     true,
			errContains: "required parameter \"name\" is missing",
		},
		{
			name: "wrong type for string param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name": {Type: pipelines.TypeString, Required: true},
				},
			},
			params: map[string]any{
				"name": 123,
			},
			wantErr:     true,
			errContains: "parameter \"name\" must be a string",
		},
		{
			name: "wrong type for int param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"age": {Type: pipelines.TypeInt, Required: true},
				},
			},
			params: map[string]any{
				"age": "not a number",
			},
			wantErr:     true,
			errContains: "parameter \"age\" must be an integer",
		},
		{
			name: "wrong type for bool param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"enabled": {Type: pipelines.TypeBool, Required: true},
				},
			},
			params: map[string]any{
				"enabled": "true",
			},
			wantErr:     true,
			errContains: "parameter \"enabled\" must be a boolean",
		},
		{
			name: "wrong type for string array param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"tags": {Type: pipelines.TypeStringArray, Required: true},
				},
			},
			params: map[string]any{
				"tags": "not an array",
			},
			wantErr:     true,
			errContains: "parameter \"tags\" must be an array",
		},
		{
			name: "correct string array param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"tags": {Type: pipelines.TypeStringArray, Required: true},
				},
			},
			params: map[string]any{
				"tags": []any{"tag1", "tag2"},
			},
			wantErr: false,
		},
		{
			name: "wrong type for object array param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"items": {Type: pipelines.TypeObjectArray, Required: true},
				},
			},
			params: map[string]any{
				"items": "not an array",
			},
			wantErr:     true,
			errContains: "parameter \"items\" must be an array",
		},
		{
			name: "correct object array param",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"items": {Type: pipelines.TypeObjectArray, Required: true},
				},
			},
			params: map[string]any{
				"items": []any{map[string]any{"key": "value"}},
			},
			wantErr: false,
		},
		{
			name: "optional param with correct type",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name":        {Type: pipelines.TypeString, Required: true},
					"description": {Type: pipelines.TypeString, Required: false},
				},
			},
			params: map[string]any{
				"name":        "test",
				"description": "test desc",
			},
			wantErr: false,
		},
		{
			name: "optional param with wrong type",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name":  {Type: pipelines.TypeString, Required: true},
					"count": {Type: pipelines.TypeInt, Required: false},
				},
			},
			params: map[string]any{
				"name":  "test",
				"count": "not a number",
			},
			wantErr:     true,
			errContains: "parameter \"count\" must be an integer",
		},
		{
			name: "optional param not provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name":        {Type: pipelines.TypeString, Required: true},
					"description": {Type: pipelines.TypeString, Required: false},
				},
			},
			params: map[string]any{
				"name": "test",
			},
			wantErr: false,
		},
		{
			name: "optional param is nil",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name":        {Type: pipelines.TypeString, Required: true},
					"description": {Type: pipelines.TypeString, Required: false},
				},
			},
			params: map[string]any{
				"name":        "test",
				"description": nil,
			},
			wantErr: false,
		},
		{
			name: "mutually exclusive - both provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"branch": {Type: pipelines.TypeString, Required: false},
					"tag":    {Type: pipelines.TypeString, Required: false},
				},
				MutuallyExclusive: [][]string{{"branch", "tag"}},
			},
			params: map[string]any{
				"branch": "main",
				"tag":    "v1.0.0",
			},
			wantErr:     true,
			errContains: "cannot specify both branch and tag",
		},
		{
			name: "mutually exclusive - only one provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"branch": {Type: pipelines.TypeString, Required: false},
					"tag":    {Type: pipelines.TypeString, Required: false},
				},
				MutuallyExclusive: [][]string{{"branch", "tag"}},
			},
			params: map[string]any{
				"branch": "main",
			},
			wantErr: false,
		},
		{
			name: "mutually exclusive - empty string not counted",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"branch": {Type: pipelines.TypeString, Required: false},
					"tag":    {Type: pipelines.TypeString, Required: false},
				},
				MutuallyExclusive: [][]string{{"branch", "tag"}},
			},
			params: map[string]any{
				"branch": "",
				"tag":    "v1.0.0",
			},
			wantErr: false,
		},
		{
			name: "mutually exclusive - three params, two provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"branch": {Type: pipelines.TypeString, Required: false},
					"tag":    {Type: pipelines.TypeString, Required: false},
					"commit": {Type: pipelines.TypeString, Required: false},
				},
				MutuallyExclusive: [][]string{{"branch", "tag", "commit"}},
			},
			params: map[string]any{
				"branch": "main",
				"tag":    "v1.0.0",
			},
			wantErr:     true,
			errContains: "cannot specify both branch and tag",
		},
		{
			name: "at-least-one - none provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"checksum":     {Type: pipelines.TypeString, Required: false},
					"checksum-url": {Type: pipelines.TypeString, Required: false},
				},
				AtLeastOne: [][]string{{"checksum", "checksum-url"}},
			},
			params:      map[string]any{},
			wantErr:     true,
			errContains: "at least one of checksum, checksum-url is required",
		},
		{
			name: "at-least-one - one provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"checksum":     {Type: pipelines.TypeString, Required: false},
					"checksum-url": {Type: pipelines.TypeString, Required: false},
				},
				AtLeastOne: [][]string{{"checksum", "checksum-url"}},
			},
			params: map[string]any{
				"checksum": "abc123",
			},
			wantErr: false,
		},
		{
			name: "at-least-one - both provided",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"checksum":     {Type: pipelines.TypeString, Required: false},
					"checksum-url": {Type: pipelines.TypeString, Required: false},
				},
				AtLeastOne: [][]string{{"checksum", "checksum-url"}},
			},
			params: map[string]any{
				"checksum":     "abc123",
				"checksum-url": "https://example.com/checksum",
			},
			wantErr: false,
		},
		{
			name: "at-least-one - empty string not counted",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"checksum":     {Type: pipelines.TypeString, Required: false},
					"checksum-url": {Type: pipelines.TypeString, Required: false},
				},
				AtLeastOne: [][]string{{"checksum", "checksum-url"}},
			},
			params: map[string]any{
				"checksum":     "",
				"checksum-url": "",
			},
			wantErr:     true,
			errContains: "at least one of checksum, checksum-url is required",
		},
		{
			name: "multiple errors combined",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"name":  {Type: pipelines.TypeString, Required: true},
					"age":   {Type: pipelines.TypeInt, Required: true},
					"email": {Type: pipelines.TypeString, Required: false},
				},
			},
			params: map[string]any{
				"age":   "not a number",
				"email": 123,
			},
			wantErr:     true,
			errContains: "required parameter \"name\" is missing",
		},
		{
			name: "mutually exclusive and at-least-one combined",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"checksum":     {Type: pipelines.TypeString, Required: false},
					"checksum-url": {Type: pipelines.TypeString, Required: false},
				},
				MutuallyExclusive: [][]string{{"checksum", "checksum-url"}},
				AtLeastOne:        [][]string{{"checksum", "checksum-url"}},
			},
			params: map[string]any{
				"checksum": "abc123",
			},
			wantErr: false,
		},
		{
			name: "mutually exclusive and at-least-one both violated",
			sig: pipelines.PipelineSignature{
				Parameters: map[string]pipelines.ParamSpec{
					"checksum":     {Type: pipelines.TypeString, Required: false},
					"checksum-url": {Type: pipelines.TypeString, Required: false},
				},
				MutuallyExclusive: [][]string{{"checksum", "checksum-url"}},
				AtLeastOne:        [][]string{{"checksum", "checksum-url"}},
			},
			params: map[string]any{
				"checksum":     "abc123",
				"checksum-url": "https://example.com/checksum",
			},
			wantErr:     true,
			errContains: "cannot specify both checksum and checksum-url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSignature(tt.sig, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateSignature() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr ||
			containsString(s[1:], substr))))
}
