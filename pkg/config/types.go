package config

type BuildConfig struct {
	Package     Package           `yaml:"package"`
	Stages      []Stage           `yaml:"stages,omitempty"`
	Environment Environment       `yaml:"environment"`
	Vars        map[string]string `yaml:"vars,omitempty"`
	Versions    map[string]string `yaml:"versions,omitempty"`
}

type Stage struct {
	Name        string         `yaml:"name,omitempty"`
	Template    string         `yaml:"template,omitempty"`
	With        map[string]any `yaml:"with,omitempty"`
	Environment Environment    `yaml:"environment,omitempty"`
	Pipeline    []PipelineStep `yaml:"pipeline,omitempty"`
}

type Package struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

type Environment struct {
	BaseImage      string            `yaml:"base-image,omitempty"`
	ExternalImage  string            `yaml:"external-image,omitempty"`
	Args           map[string]string `yaml:"args,omitempty"`
	Packages       []string          `yaml:"packages,omitempty"`
	RootfsPackages []string          `yaml:"rootfs-packages,omitempty"`
	Environment    map[string]string `yaml:"environment,omitempty"`
	WorkDir        string            `yaml:"workdir,omitempty"`
	User           string            `yaml:"user,omitempty"`
	Entrypoint     []string          `yaml:"entrypoint,omitempty"`
	Cmd            []string          `yaml:"cmd,omitempty"`
	Expose         []string          `yaml:"expose,omitempty"`
	Volume         []string          `yaml:"volume,omitempty"`
	StopSignal     string            `yaml:"stopsignal,omitempty"`
}

type PipelineStep struct {
	Name      string         `yaml:"name,omitempty"`
	Uses      string         `yaml:"uses,omitempty"`
	Run       string         `yaml:"run,omitempty"`
	BuildDeps []string       `yaml:"build-deps,omitempty"`
	Fetch     *FetchStep     `yaml:"fetch,omitempty"`
	Copy      *CopyStep      `yaml:"copy,omitempty"`
	With      map[string]any `yaml:"with,omitempty"`
}

type FetchStep struct {
	URL         string `yaml:"url"`
	Destination string `yaml:"destination,omitempty"`
	Extract     bool   `yaml:"extract,omitempty"`
}

type CopyStep struct {
	FromStage string `yaml:"from-stage,omitempty"`
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Chown     string `yaml:"chown,omitempty"`
}

func (e Environment) IsEmpty() bool {
	return e.BaseImage == "" &&
		e.ExternalImage == "" &&
		len(e.Args) == 0 &&
		len(e.Packages) == 0 &&
		len(e.RootfsPackages) == 0 &&
		len(e.Environment) == 0 &&
		e.WorkDir == "" &&
		e.User == "" &&
		len(e.Entrypoint) == 0 &&
		len(e.Cmd) == 0 &&
		len(e.Expose) == 0 &&
		len(e.Volume) == 0 &&
		e.StopSignal == ""
}
