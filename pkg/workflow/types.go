package workflow

type Workflow struct {
	Name        string            `yaml:"name"`
	On          Triggers          `yaml:"on"`
	Concurrency string            `yaml:"concurrency,omitempty"`
	Permissions map[string]string `yaml:"permissions,omitempty"`
	Jobs        map[string]Job    `yaml:"jobs"`
}

type Triggers struct {
	WorkflowDispatch *DispatchTrigger `yaml:"workflow_dispatch,omitempty"`
	WorkflowRun      *RunTrigger      `yaml:"workflow_run,omitempty"`
}

type DispatchTrigger struct{}

type RunTrigger struct {
	Workflows []string `yaml:"workflows"`
	Types     []string `yaml:"types"`
}

type Job struct {
	Name    string            `yaml:"name,omitempty"`
	RunsOn  string            `yaml:"runs-on,omitempty"`
	Needs   []string          `yaml:"needs,omitempty"`
	Uses    string            `yaml:"uses,omitempty"`
	Secrets string            `yaml:"secrets,omitempty"`
	With    map[string]string `yaml:"with,omitempty"`
	Steps   []Step            `yaml:"steps,omitempty"`
}

type Step struct {
	Name string            `yaml:"name,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
	Run  string            `yaml:"run,omitempty"`
}
