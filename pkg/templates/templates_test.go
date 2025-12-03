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

func TestGoAppWithExtraCopies(t *testing.T) {
	params := map[string]any{
		"repo":   "https://github.com/tailscale/golink",
		"binary": "golink",
		"extra-copies": []any{
			map[string]any{
				"from": "/src/tailscale/golink/static/",
				"to":   "/static/",
			},
			map[string]any{
				"from": "/src/tailscale/golink/templates/",
				"to":   "/templates/",
			},
		},
	}

	result, err := goApp(params)
	if err != nil {
		t.Fatalf("goApp() error = %v", err)
	}

	if len(result.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(result.Stages))
	}

	rootfsStage := result.Stages[1]
	if rootfsStage.Name != "rootfs" {
		t.Errorf("expected rootfs stage name 'rootfs', got %q", rootfsStage.Name)
	}

	// Should have: binary copy, notices copy, 2 extra-copies = 4 total
	if len(rootfsStage.Pipeline) != 4 {
		t.Errorf("expected 4 copy steps in rootfs (binary + notices + 2 extra-copies), got %d", len(rootfsStage.Pipeline))
	}

	// Verify extra-copies are present
	foundStatic := false
	foundTemplates := false
	for _, step := range rootfsStage.Pipeline {
		if step.Copy != nil {
			if step.Copy.From == "/src/tailscale/golink/static/" && step.Copy.To == "/rootfs/static/" {
				foundStatic = true
			}
			if step.Copy.From == "/src/tailscale/golink/templates/" && step.Copy.To == "/rootfs/templates/" {
				foundTemplates = true
			}
		}
	}
	if !foundStatic {
		t.Error("expected extra-copy for static/ not found")
	}
	if !foundTemplates {
		t.Error("expected extra-copy for templates/ not found")
	}
}

func TestMultiGoAppSameRepo(t *testing.T) {
	params := map[string]any{
		"binaries": []any{
			map[string]any{
				"repo":       "https://github.com/emersion/soju",
				"package":    "./cmd/soju",
				"binary":     "soju",
				"go-tags":    "moderncsqlite",
				"entrypoint": true,
			},
			map[string]any{
				"repo":    "https://github.com/emersion/soju",
				"package": "./cmd/sojuctl",
				"binary":  "sojuctl",
			},
			map[string]any{
				"repo":    "https://github.com/emersion/soju",
				"package": "./cmd/sojudb",
				"binary":  "sojudb",
				"go-tags": "moderncsqlite",
			},
		},
		"expose": []any{"6697"},
	}

	result, err := multiGoApp(params)
	if err != nil {
		t.Fatalf("multiGoApp() error = %v", err)
	}

	if len(result.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(result.Stages))
	}

	buildStage := result.Stages[0]
	if buildStage.Name != "build" {
		t.Errorf("expected build stage name 'build', got %q", buildStage.Name)
	}

	// Should have 1 clone + 3 builds = 4 steps (same repo cloned once)
	cloneCount := 0
	buildCount := 0
	for _, step := range buildStage.Pipeline {
		if step.Uses == "clone" {
			cloneCount++
		}
		if step.Uses == "build-go-only" {
			buildCount++
		}
	}
	if cloneCount != 1 {
		t.Errorf("expected 1 clone step (same repo), got %d", cloneCount)
	}
	if buildCount != 3 {
		t.Errorf("expected 3 build steps, got %d", buildCount)
	}

	rootfsStage := result.Stages[1]
	// Should have 3 binary copies + 3 notices copies = 6
	if len(rootfsStage.Pipeline) != 6 {
		t.Errorf("expected 6 copy steps in rootfs (3 binaries + 3 notices), got %d", len(rootfsStage.Pipeline))
	}

	finalStage := result.Stages[2]
	// Entrypoint should be soju (marked with entrypoint: true)
	if len(finalStage.Environment.Entrypoint) == 0 || finalStage.Environment.Entrypoint[0] != "/soju" {
		t.Errorf("expected entrypoint /soju, got %v", finalStage.Environment.Entrypoint)
	}

	if len(finalStage.Environment.Expose) != 1 || finalStage.Environment.Expose[0] != "6697" {
		t.Errorf("expected expose 6697, got %v", finalStage.Environment.Expose)
	}
}

func TestMultiGoAppDifferentRepos(t *testing.T) {
	params := map[string]any{
		"binaries": []any{
			map[string]any{
				"repo":   "https://github.com/ergochat/ergo",
				"binary": "ergo",
			},
			map[string]any{
				"repo":       "https://github.com/csmith/certwrapper",
				"binary":     "certwrapper",
				"entrypoint": true,
			},
		},
		"extra-copies": []any{
			map[string]any{
				"from": "/src/ergochat/ergo/languages/",
				"to":   "/ircd-bin/languages/",
			},
		},
	}

	result, err := multiGoApp(params)
	if err != nil {
		t.Fatalf("multiGoApp() error = %v", err)
	}

	buildStage := result.Stages[0]

	// Should have 2 clones (different repos) + 2 builds = 4 steps
	cloneCount := 0
	buildCount := 0
	for _, step := range buildStage.Pipeline {
		if step.Uses == "clone" {
			cloneCount++
		}
		if step.Uses == "build-go-only" {
			buildCount++
		}
	}
	if cloneCount != 2 {
		t.Errorf("expected 2 clone steps (different repos), got %d", cloneCount)
	}
	if buildCount != 2 {
		t.Errorf("expected 2 build steps, got %d", buildCount)
	}

	rootfsStage := result.Stages[1]
	// Should have 2 binary copies + 2 notices copies + 1 extra-copy = 5
	if len(rootfsStage.Pipeline) != 5 {
		t.Errorf("expected 5 copy steps in rootfs, got %d", len(rootfsStage.Pipeline))
	}

	// Verify extra-copy is present
	foundLanguages := false
	for _, step := range rootfsStage.Pipeline {
		if step.Copy != nil && step.Copy.From == "/src/ergochat/ergo/languages/" {
			foundLanguages = true
			if step.Copy.To != "/rootfs/ircd-bin/languages/" {
				t.Errorf("expected extra-copy to /rootfs/ircd-bin/languages/, got %s", step.Copy.To)
			}
		}
	}
	if !foundLanguages {
		t.Error("expected extra-copy for languages/ not found")
	}

	finalStage := result.Stages[2]
	// Entrypoint should be certwrapper (marked with entrypoint: true)
	if len(finalStage.Environment.Entrypoint) == 0 || finalStage.Environment.Entrypoint[0] != "/certwrapper" {
		t.Errorf("expected entrypoint /certwrapper, got %v", finalStage.Environment.Entrypoint)
	}
}

func TestMultiGoAppDefaultEntrypoint(t *testing.T) {
	params := map[string]any{
		"binaries": []any{
			map[string]any{
				"repo":   "https://github.com/example/app1",
				"binary": "app1",
			},
			map[string]any{
				"repo":   "https://github.com/example/app2",
				"binary": "app2",
			},
		},
	}

	result, err := multiGoApp(params)
	if err != nil {
		t.Fatalf("multiGoApp() error = %v", err)
	}

	finalStage := result.Stages[2]
	// Default entrypoint should be first binary (app1)
	if len(finalStage.Environment.Entrypoint) == 0 || finalStage.Environment.Entrypoint[0] != "/app1" {
		t.Errorf("expected default entrypoint /app1, got %v", finalStage.Environment.Entrypoint)
	}
}

func TestParseExtraCopies(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		want    []ExtraCopySpec
		wantErr bool
	}{
		{
			name:   "no extra-copies",
			params: map[string]any{},
			want:   nil,
		},
		{
			name: "single extra-copy",
			params: map[string]any{
				"extra-copies": []any{
					map[string]any{
						"from": "/src/static/",
						"to":   "/static/",
					},
				},
			},
			want: []ExtraCopySpec{
				{From: "/src/static/", To: "/static/"},
			},
		},
		{
			name: "multiple extra-copies",
			params: map[string]any{
				"extra-copies": []any{
					map[string]any{"from": "/a", "to": "/b"},
					map[string]any{"from": "/c", "to": "/d"},
				},
			},
			want: []ExtraCopySpec{
				{From: "/a", To: "/b"},
				{From: "/c", To: "/d"},
			},
		},
		{
			name: "missing from",
			params: map[string]any{
				"extra-copies": []any{
					map[string]any{"to": "/dest"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing to",
			params: map[string]any{
				"extra-copies": []any{
					map[string]any{"from": "/src"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExtraCopies(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExtraCopies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseExtraCopies() got %d items, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i].From != tt.want[i].From || got[i].To != tt.want[i].To {
						t.Errorf("ParseExtraCopies()[%d] = %+v, want %+v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestParseBinaries(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		want    []BinarySpec
		wantErr bool
	}{
		{
			name:    "missing binaries",
			params:  map[string]any{},
			wantErr: true,
		},
		{
			name: "empty binaries",
			params: map[string]any{
				"binaries": []any{},
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			params: map[string]any{
				"binaries": []any{
					map[string]any{"binary": "app"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing binary name",
			params: map[string]any{
				"binaries": []any{
					map[string]any{"repo": "https://github.com/example/app"},
				},
			},
			wantErr: true,
		},
		{
			name: "minimal valid binary",
			params: map[string]any{
				"binaries": []any{
					map[string]any{
						"repo":   "https://github.com/example/app",
						"binary": "app",
					},
				},
			},
			want: []BinarySpec{
				{Repo: "https://github.com/example/app", Binary: "app", Package: "."},
			},
		},
		{
			name: "full binary spec",
			params: map[string]any{
				"binaries": []any{
					map[string]any{
						"repo":       "https://github.com/example/app",
						"binary":     "app",
						"package":    "./cmd/app",
						"go-tags":    "netgo",
						"ignore":     "example.com/private",
						"entrypoint": true,
						"cgo":        true,
					},
				},
			},
			want: []BinarySpec{
				{
					Repo:       "https://github.com/example/app",
					Binary:     "app",
					Package:    "./cmd/app",
					GoTags:     "netgo",
					Ignore:     "example.com/private",
					Entrypoint: true,
					Cgo:        true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBinaries(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBinaries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseBinaries() got %d items, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i].Repo != tt.want[i].Repo ||
						got[i].Binary != tt.want[i].Binary ||
						got[i].Package != tt.want[i].Package ||
						got[i].GoTags != tt.want[i].GoTags ||
						got[i].Ignore != tt.want[i].Ignore ||
						got[i].Entrypoint != tt.want[i].Entrypoint ||
						got[i].Cgo != tt.want[i].Cgo {
						t.Errorf("ParseBinaries()[%d] = %+v, want %+v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}
