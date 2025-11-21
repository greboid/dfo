package util

import (
	"testing"
)

func TestFormatDockerfileArray(t *testing.T) {
	tests := []struct {
		name      string
		directive string
		values    []string
		expected  string
	}{
		{
			name:      "single value",
			directive: "ENTRYPOINT",
			values:    []string{"/app/server"},
			expected:  "ENTRYPOINT [\"/app/server\"]\n\n",
		},
		{
			name:      "multiple values",
			directive: "CMD",
			values:    []string{"echo", "hello", "world"},
			expected:  "CMD [\"echo\", \"hello\", \"world\"]\n\n",
		},
		{
			name:      "empty slice returns empty string",
			directive: "ENTRYPOINT",
			values:    []string{},
			expected:  "",
		},
		{
			name:      "nil slice returns empty string",
			directive: "ENTRYPOINT",
			values:    nil,
			expected:  "",
		},
		{
			name:      "values with special characters",
			directive: "CMD",
			values:    []string{"echo", "hello \"world\"", "with\nnewline"},
			expected:  "CMD [\"echo\", \"hello \\\"world\\\"\", \"with\\nnewline\"]\n\n",
		},
		{
			name:      "empty string in values",
			directive: "CMD",
			values:    []string{"", "test"},
			expected:  "CMD [\"\", \"test\"]\n\n",
		},
		{
			name:      "value with spaces",
			directive: "CMD",
			values:    []string{"echo", "hello world"},
			expected:  "CMD [\"echo\", \"hello world\"]\n\n",
		},
		{
			name:      "long directive name",
			directive: "HEALTHCHECK CMD",
			values:    []string{"curl", "-f", "http://localhost/"},
			expected:  "HEALTHCHECK CMD [\"curl\", \"-f\", \"http://localhost/\"]\n\n",
		},
		{
			name:      "value with backslash",
			directive: "CMD",
			values:    []string{"C:\\Program Files\\app.exe"},
			expected:  "CMD [\"C:\\\\Program Files\\\\app.exe\"]\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDockerfileArray(tt.directive, tt.values)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestFormatMapDirectives(t *testing.T) {
	tests := []struct {
		name      string
		directive string
		values    map[string]string
		expected  string
	}{
		{
			name:      "single key-value pair",
			directive: "ENV",
			values:    map[string]string{"APP_NAME": "myapp"},
			expected:  "ENV APP_NAME=\"myapp\"\n\n",
		},
		{
			name:      "multiple key-value pairs sorted by key",
			directive: "LABEL",
			values: map[string]string{
				"version":     "1.0",
				"maintainer":  "team@example.com",
				"description": "My application",
			},
			expected: "LABEL description=\"My application\"\nLABEL maintainer=\"team@example.com\"\nLABEL version=\"1.0\"\n\n",
		},
		{
			name:      "empty map returns empty string",
			directive: "ENV",
			values:    map[string]string{},
			expected:  "",
		},
		{
			name:      "nil map returns empty string",
			directive: "ENV",
			values:    nil,
			expected:  "",
		},
		{
			name:      "value with special characters",
			directive: "ENV",
			values:    map[string]string{"MESSAGE": "hello \"world\""},
			expected:  "ENV MESSAGE=\"hello \\\"world\\\"\"\n\n",
		},
		{
			name:      "value with newline",
			directive: "ENV",
			values:    map[string]string{"MULTI": "line1\nline2"},
			expected:  "ENV MULTI=\"line1\\nline2\"\n\n",
		},
		{
			name:      "empty value",
			directive: "ENV",
			values:    map[string]string{"EMPTY": ""},
			expected:  "ENV EMPTY=\"\"\n\n",
		},
		{
			name:      "value with spaces",
			directive: "ENV",
			values:    map[string]string{"PATH": "/usr/bin:/usr/local/bin"},
			expected:  "ENV PATH=\"/usr/bin:/usr/local/bin\"\n\n",
		},
		{
			name:      "ARG directive",
			directive: "ARG",
			values: map[string]string{
				"VERSION": "1.0",
				"BUILD":   "123",
			},
			expected: "ARG BUILD=\"123\"\nARG VERSION=\"1.0\"\n\n",
		},
		{
			name:      "keys are sorted alphabetically",
			directive: "ENV",
			values: map[string]string{
				"ZEBRA":  "z",
				"ALPHA":  "a",
				"MIDDLE": "m",
			},
			expected: "ENV ALPHA=\"a\"\nENV MIDDLE=\"m\"\nENV ZEBRA=\"z\"\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMapDirectives(tt.directive, tt.values)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestWrapRun(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple command",
			command:  "apt-get update",
			expected: "RUN apt-get update\n",
		},
		{
			name:     "command with arguments",
			command:  "apt-get install -y curl wget",
			expected: "RUN apt-get install -y curl wget\n",
		},
		{
			name:     "empty command returns empty string",
			command:  "",
			expected: "",
		},
		{
			name:     "command with pipes",
			command:  "cat file.txt | grep pattern",
			expected: "RUN cat file.txt | grep pattern\n",
		},
		{
			name:     "command with &&",
			command:  "apt-get update && apt-get install -y curl",
			expected: "RUN apt-get update && apt-get install -y curl\n",
		},
		{
			name:     "multiline command with backslash",
			command:  "apt-get update \\\n  && apt-get install -y curl",
			expected: "RUN apt-get update \\\n  && apt-get install -y curl\n",
		},
		{
			name:     "command with quotes",
			command:  "echo \"hello world\"",
			expected: "RUN echo \"hello world\"\n",
		},
		{
			name:     "command with environment variable",
			command:  "export PATH=$PATH:/usr/local/bin",
			expected: "RUN export PATH=$PATH:/usr/local/bin\n",
		},
		{
			name:     "whitespace only command",
			command:  "   ",
			expected: "RUN    \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapRun(tt.command)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildPackageList(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{
			name:     "single package",
			packages: []string{"curl"},
			expected: "\"curl\"",
		},
		{
			name:     "multiple packages",
			packages: []string{"curl", "wget", "git"},
			expected: "\"curl\" \"wget\" \"git\"",
		},
		{
			name:     "empty slice returns empty string",
			packages: []string{},
			expected: "",
		},
		{
			name:     "nil slice returns empty string",
			packages: nil,
			expected: "",
		},
		{
			name:     "package with version",
			packages: []string{"curl=7.68.0-1"},
			expected: "\"curl=7.68.0-1\"",
		},
		{
			name:     "package with special characters",
			packages: []string{"lib-dev", "lib++"},
			expected: "\"lib-dev\" \"lib++\"",
		},
		{
			name:     "empty string in packages",
			packages: []string{"", "curl"},
			expected: "\"\" \"curl\"",
		},
		{
			name:     "package with spaces (should be quoted)",
			packages: []string{"package name"},
			expected: "\"package name\"",
		},
		{
			name:     "many packages",
			packages: []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5"},
			expected: "\"pkg1\" \"pkg2\" \"pkg3\" \"pkg4\" \"pkg5\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPackageList(tt.packages)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
