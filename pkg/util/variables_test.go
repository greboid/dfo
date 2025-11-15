package util

import (
	"testing"
)

func TestExtractVariableReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no variables",
			input:    "echo hello world",
			expected: nil,
		},
		{
			name:     "single variable",
			input:    "echo %{VERSION}",
			expected: []string{"VERSION"},
		},
		{
			name:     "multiple variables",
			input:    "echo %{VERSION} %{NAME}",
			expected: []string{"VERSION", "NAME"},
		},
		{
			name:     "duplicate variables",
			input:    "%{VERSION} and %{VERSION} again",
			expected: []string{"VERSION"},
		},
		{
			name:     "variable with underscores",
			input:    "%{MY_VAR_NAME}",
			expected: []string{"MY_VAR_NAME"},
		},
		{
			name:     "variable with numbers",
			input:    "%{VAR123}",
			expected: []string{"VAR123"},
		},
		{
			name:     "variable starting with underscore",
			input:    "%{_PRIVATE}",
			expected: []string{"_PRIVATE"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "percent without braces ignored",
			input:    "%VERSION",
			expected: nil,
		},
		{
			name:     "incomplete brace ignored",
			input:    "%{VERSION",
			expected: nil,
		},
		{
			name:     "mixed valid and invalid",
			input:    "%{VALID} %INVALID %{ALSO_VALID}",
			expected: []string{"VALID", "ALSO_VALID"},
		},
		{
			name:     "in URL",
			input:    "https://github.com/%{OWNER}/%{REPO}/releases/download/%{VERSION}/file.tar.gz",
			expected: []string{"OWNER", "REPO", "VERSION"},
		},
		{
			name:     "bash variables ignored",
			input:    "for DEP in $DEPS; do apk add \"${DEP}\"; done",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariableReferences(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractVariableReferences(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("ExtractVariableReferences(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestValidateVariableReferences(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		vars      map[string]string
		context   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "no variables - passes",
			input:     "echo hello",
			vars:      nil,
			context:   "test",
			wantError: false,
		},
		{
			name:      "defined variable - passes",
			input:     "echo %{VERSION}",
			vars:      map[string]string{"VERSION": "1.0.0"},
			context:   "test",
			wantError: false,
		},
		{
			name:      "undefined variable - fails",
			input:     "echo %{VERSION}",
			vars:      map[string]string{},
			context:   "test step",
			wantError: true,
			errorMsg:  "test step: undefined variable(s): %{VERSION}",
		},
		{
			name:      "multiple undefined - fails with all",
			input:     "echo %{A} %{B} %{C}",
			vars:      map[string]string{},
			context:   "test",
			wantError: true,
			errorMsg:  "test: undefined variable(s): %{A}, %{B}, %{C}",
		},
		{
			name:      "some defined some not - fails with undefined only",
			input:     "%{DEFINED} %{UNDEFINED}",
			vars:      map[string]string{"DEFINED": "value"},
			context:   "step",
			wantError: true,
			errorMsg:  "step: undefined variable(s): %{UNDEFINED}",
		},
		{
			name:      "nil vars with no references - passes",
			input:     "no vars here",
			vars:      nil,
			context:   "test",
			wantError: false,
		},
		{
			name:      "nil vars with references - fails",
			input:     "%{VAR}",
			vars:      nil,
			context:   "test",
			wantError: true,
			errorMsg:  "test: undefined variable(s): %{VAR}",
		},
		{
			name:      "bash variables ignored - passes",
			input:     "for DEP in $DEPS; do apk add \"${DEP}\"; done",
			vars:      nil,
			context:   "test",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableReferences(tt.input, tt.vars, tt.context)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateVariableReferences() expected error, got nil")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("ValidateVariableReferences() error = %q, want %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateVariableReferences() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExpandVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:     "no variables",
			input:    "hello world",
			vars:     map[string]string{"FOO": "bar"},
			expected: "hello world",
		},
		{
			name:     "single replacement",
			input:    "version: %{VERSION}",
			vars:     map[string]string{"VERSION": "1.0.0"},
			expected: "version: 1.0.0",
		},
		{
			name:     "multiple replacements",
			input:    "%{A} and %{B}",
			vars:     map[string]string{"A": "first", "B": "second"},
			expected: "first and second",
		},
		{
			name:     "undefined stays unchanged",
			input:    "%{UNDEFINED}",
			vars:     map[string]string{},
			expected: "%{UNDEFINED}",
		},
		{
			name:     "nil vars",
			input:    "%{VAR}",
			vars:     nil,
			expected: "%{VAR}",
		},
		{
			name:     "same variable multiple times",
			input:    "%{V}-%{V}-%{V}",
			vars:     map[string]string{"V": "x"},
			expected: "x-x-x",
		},
		{
			name:     "bash variables unchanged",
			input:    "for DEP in $DEPS; do echo \"${DEP}\"; done",
			vars:     map[string]string{"DEPS": "should-not-match"},
			expected: "for DEP in $DEPS; do echo \"${DEP}\"; done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandVars(tt.input, tt.vars)
			if result != tt.expected {
				t.Errorf("ExpandVars(%q, %v) = %q, want %q", tt.input, tt.vars, result, tt.expected)
			}
		})
	}
}

func TestExpandVarsStrict(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		vars      map[string]string
		context   string
		expected  string
		wantError bool
	}{
		{
			name:      "all defined - expands",
			input:     "%{A} %{B}",
			vars:      map[string]string{"A": "1", "B": "2"},
			context:   "test",
			expected:  "1 2",
			wantError: false,
		},
		{
			name:      "undefined - errors",
			input:     "%{MISSING}",
			vars:      map[string]string{},
			context:   "test",
			expected:  "",
			wantError: true,
		},
		{
			name:      "no variables - passes",
			input:     "plain text",
			vars:      nil,
			context:   "test",
			expected:  "plain text",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandVarsStrict(tt.input, tt.vars, tt.context)
			if tt.wantError {
				if err == nil {
					t.Errorf("ExpandVarsStrict() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ExpandVarsStrict() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("ExpandVarsStrict() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}
