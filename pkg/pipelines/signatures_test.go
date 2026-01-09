package pipelines

import (
	"strings"
	"testing"
)

func TestValidateRequiredParams(t *testing.T) {
	sig := PipelineSignature{
		Name: "test-pipeline",
		Parameters: map[string]ParamSpec{
			"required_param": {Type: TypeString, Required: true, Description: "Required param"},
			"optional_param": {Type: TypeString, Required: false, Description: "Optional param"},
		},
	}

	tests := []struct {
		name          string
		params        map[string]any
		expectError   bool
		errorContains []string
	}{
		{
			name: "all required params present",
			params: map[string]any{
				"required_param": "value",
			},
			expectError: false,
		},
		{
			name:          "missing required param",
			params:        map[string]any{},
			expectError:   true,
			errorContains: []string{"required_param"},
		},
		{
			name: "required param is nil",
			params: map[string]any{
				"required_param": nil,
			},
			expectError:   true,
			errorContains: []string{"required_param"},
		},
		{
			name: "optional param missing is ok",
			params: map[string]any{
				"required_param": "value",
			},
			expectError: false,
		},
		{
			name: "both params present",
			params: map[string]any{
				"required_param": "value1",
				"optional_param": "value2",
			},
			expectError: false,
		},
		{
			name: "multiple required params missing",
			params: map[string]any{
				"required_param": nil,
			},
			expectError:   true,
			errorContains: []string{"required_param"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateRequiredParams(sig, tt.params)
			hasError := len(errors) > 0

			if hasError != tt.expectError {
				t.Errorf("validateRequiredParams() error = %v, expectError %v", errors, tt.expectError)
			}

			if len(tt.errorContains) > 0 {
				for _, expected := range tt.errorContains {
					found := false
					for _, err := range errors {
						if strings.Contains(err, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error to contain %q, got %v", expected, errors)
					}
				}
			}
		})
	}
}

func TestValidateParamTypes(t *testing.T) {
	sig := PipelineSignature{
		Name: "test-pipeline",
		Parameters: map[string]ParamSpec{
			"string_param": {Type: TypeString, Required: false},
			"int_param":    {Type: TypeInt, Required: false},
			"bool_param":   {Type: TypeBool, Required: false},
			"array_param":  {Type: TypeStringArray, Required: false},
			"object_param": {Type: TypeObjectArray, Required: false},
		},
	}

	tests := []struct {
		name        string
		params      map[string]any
		expectError bool
	}{
		{
			name: "all valid types",
			params: map[string]any{
				"string_param": "hello",
				"int_param":    42,
				"bool_param":   true,
				"array_param":  []string{"a", "b"},
				"object_param": []map[string]any{{"key": "value"}},
			},
			expectError: false,
		},
		{
			name: "int as float64",
			params: map[string]any{
				"int_param": 42.0,
			},
			expectError: false,
		},
		{
			name: "string array as string",
			params: map[string]any{
				"array_param": "single-string",
			},
			expectError: false,
		},
		{
			name: "string array as []any with strings",
			params: map[string]any{
				"array_param": []any{"a", "b"},
			},
			expectError: false,
		},
		{
			name: "object array as []map[string]any",
			params: map[string]any{
				"object_param": []map[string]any{{"key": "value"}},
			},
			expectError: false,
		},
		{
			name: "string param wrong type",
			params: map[string]any{
				"string_param": 123,
			},
			expectError: true,
		},
		{
			name: "int param wrong type",
			params: map[string]any{
				"int_param": "not an int",
			},
			expectError: true,
		},
		{
			name: "bool param wrong type",
			params: map[string]any{
				"bool_param": "true",
			},
			expectError: true,
		},
		{
			name: "array param wrong type - element not string",
			params: map[string]any{
				"array_param": []any{1, 2, 3},
			},
			expectError: true,
		},
		{
			name: "object param wrong type - element not object",
			params: map[string]any{
				"object_param": []any{"string", "not object"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateParamTypes(sig, tt.params)
			hasError := len(errors) > 0

			if hasError != tt.expectError {
				t.Errorf("validateParamTypes() error = %v, expectError %v", errors, tt.expectError)
			}
		})
	}
}

func TestValidateMutuallyExclusive(t *testing.T) {
	tests := []struct {
		name          string
		groups        [][]string
		params        map[string]any
		expectError   bool
		errorContains []string
	}{
		{
			name:        "no params present",
			groups:      [][]string{{"param1", "param2"}},
			params:      map[string]any{},
			expectError: false,
		},
		{
			name:        "only one param present",
			groups:      [][]string{{"param1", "param2"}},
			params:      map[string]any{"param1": "value"},
			expectError: false,
		},
		{
			name:   "both params present",
			groups: [][]string{{"param1", "param2"}},
			params: map[string]any{
				"param1": "value1",
				"param2": "value2",
			},
			expectError:   true,
			errorContains: []string{"param1", "param2"},
		},
		{
			name:   "three params, two present",
			groups: [][]string{{"param1", "param2", "param3"}},
			params: map[string]any{
				"param1": "value1",
				"param3": "value3",
			},
			expectError: true,
		},
		{
			name: "multiple groups, all valid",
			groups: [][]string{
				{"a1", "a2"},
				{"b1", "b2"},
			},
			params: map[string]any{
				"a1": "value",
				"b1": "value",
			},
			expectError: false,
		},
		{
			name: "multiple groups, one invalid",
			groups: [][]string{
				{"a1", "a2"},
				{"b1", "b2"},
			},
			params: map[string]any{
				"a1": "value1",
				"a2": "value2",
				"b1": "value",
			},
			expectError: true,
		},
		{
			name:   "empty string not considered present",
			groups: [][]string{{"param1", "param2"}},
			params: map[string]any{
				"param1": "value",
				"param2": "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateMutuallyExclusive(tt.groups, tt.params)
			hasError := len(errors) > 0

			if hasError != tt.expectError {
				t.Errorf("validateMutuallyExclusive() error = %v, expectError %v", errors, tt.expectError)
			}

			if len(tt.errorContains) > 0 && hasError {
				for _, expected := range tt.errorContains {
					found := false
					for _, err := range errors {
						if strings.Contains(err, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error to contain %q, got %v", expected, errors)
					}
				}
			}
		})
	}
}

func TestValidateAtLeastOne(t *testing.T) {
	tests := []struct {
		name        string
		groups      [][]string
		params      map[string]any
		expectError bool
	}{
		{
			name:        "one param present",
			groups:      [][]string{{"param1", "param2"}},
			params:      map[string]any{"param1": "value"},
			expectError: false,
		},
		{
			name:        "multiple params present",
			groups:      [][]string{{"param1", "param2", "param3"}},
			params:      map[string]any{"param1": "v1", "param3": "v3"},
			expectError: false,
		},
		{
			name:        "no params present",
			groups:      [][]string{{"param1", "param2"}},
			params:      map[string]any{},
			expectError: true,
		},
		{
			name:        "nil param not counted",
			groups:      [][]string{{"param1", "param2"}},
			params:      map[string]any{"param1": nil},
			expectError: true,
		},
		{
			name:        "empty string not counted",
			groups:      [][]string{{"param1", "param2"}},
			params:      map[string]any{"param1": ""},
			expectError: true,
		},
		{
			name: "multiple groups, all satisfied",
			groups: [][]string{
				{"a1", "a2"},
				{"b1", "b2"},
			},
			params: map[string]any{
				"a1": "value",
				"b2": "value",
			},
			expectError: false,
		},
		{
			name: "multiple groups, one unsatisfied",
			groups: [][]string{
				{"a1", "a2"},
				{"b1", "b2"},
			},
			params: map[string]any{
				"a1": "value",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateAtLeastOne(tt.groups, tt.params)
			hasError := len(errors) > 0

			if hasError != tt.expectError {
				t.Errorf("validateAtLeastOne() error = %v, expectError %v", errors, tt.expectError)
			}
		})
	}
}

func TestIsParamPresent(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		param    string
		expected bool
	}{
		{
			name:     "param exists and has value",
			params:   map[string]any{"key": "value"},
			param:    "key",
			expected: true,
		},
		{
			name:     "param does not exist",
			params:   map[string]any{},
			param:    "key",
			expected: false,
		},
		{
			name:     "param exists but is nil",
			params:   map[string]any{"key": nil},
			param:    "key",
			expected: false,
		},
		{
			name:     "param exists but is empty string",
			params:   map[string]any{"key": ""},
			param:    "key",
			expected: false,
		},
		{
			name:     "param exists with false bool",
			params:   map[string]any{"key": false},
			param:    "key",
			expected: true,
		},
		{
			name:     "param exists with zero int",
			params:   map[string]any{"key": 0},
			param:    "key",
			expected: true,
		},
		{
			name:     "param exists with zero float",
			params:   map[string]any{"key": 0.0},
			param:    "key",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isParamPresent(tt.params, tt.param)
			if result != tt.expected {
				t.Errorf("isParamPresent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCheckType(t *testing.T) {
	tests := []struct {
		name         string
		paramName    string
		value        any
		expectedType ParamType
		expectError  bool
	}{
		{
			name:         "valid string",
			paramName:    "test",
			value:        "hello",
			expectedType: TypeString,
			expectError:  false,
		},
		{
			name:         "invalid string",
			paramName:    "test",
			value:        123,
			expectedType: TypeString,
			expectError:  true,
		},
		{
			name:         "valid int",
			paramName:    "test",
			value:        42,
			expectedType: TypeInt,
			expectError:  false,
		},
		{
			name:         "int as float64",
			paramName:    "test",
			value:        42.0,
			expectedType: TypeInt,
			expectError:  false,
		},
		{
			name:         "invalid int",
			paramName:    "test",
			value:        "42",
			expectedType: TypeInt,
			expectError:  true,
		},
		{
			name:         "valid bool",
			paramName:    "test",
			value:        true,
			expectedType: TypeBool,
			expectError:  false,
		},
		{
			name:         "invalid bool",
			paramName:    "test",
			value:        "true",
			expectedType: TypeBool,
			expectError:  true,
		},
		{
			name:         "valid string array",
			paramName:    "test",
			value:        []string{"a", "b"},
			expectedType: TypeStringArray,
			expectError:  false,
		},
		{
			name:         "valid object array",
			paramName:    "test",
			value:        []map[string]any{{"key": "value"}},
			expectedType: TypeObjectArray,
			expectError:  false,
		},
		{
			name:         "unknown type",
			paramName:    "test",
			value:        "anything",
			expectedType: "unknown",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckType(tt.paramName, tt.value, tt.expectedType)
			hasError := err != nil

			if hasError != tt.expectError {
				t.Errorf("CheckType() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateSignature(t *testing.T) {
	sig := PipelineSignature{
		Name: "test-pipeline",
		Parameters: map[string]ParamSpec{
			"required_string": {Type: TypeString, Required: true},
			"optional_int":    {Type: TypeInt, Required: false},
			"optional_bool":   {Type: TypeBool, Required: false},
		},
		MutuallyExclusive: [][]string{{"opt_a", "opt_b"}},
		AtLeastOne:        [][]string{{"req_a", "req_b"}},
	}

	tests := []struct {
		name        string
		params      map[string]any
		expectError bool
	}{
		{
			name: "valid params",
			params: map[string]any{
				"required_string": "value",
				"req_a":           "a",
			},
			expectError: false,
		},
		{
			name:        "missing required param",
			params:      map[string]any{},
			expectError: true,
		},
		{
			name: "wrong type for param",
			params: map[string]any{
				"required_string": 123,
			},
			expectError: true,
		},
		{
			name: "mutually exclusive params both present",
			params: map[string]any{
				"required_string": "value",
				"req_a":           "a",
				"opt_a":           "a",
				"opt_b":           "b",
			},
			expectError: true,
		},
		{
			name: "at least one group unsatisfied",
			params: map[string]any{
				"required_string": "value",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignature(sig, tt.params)
			hasError := err != nil

			if hasError != tt.expectError {
				t.Errorf("ValidateSignature() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name         string
		pipelineName string
		params       map[string]any
		expectError  bool
	}{
		{
			name:         "existing pipeline with valid params",
			pipelineName: "create-user",
			params: map[string]any{
				"username": "testuser",
				"uid":      1000,
				"gid":      1000,
			},
			expectError: false,
		},
		{
			name:         "existing pipeline with missing required param",
			pipelineName: "create-user",
			params: map[string]any{
				"username": "testuser",
			},
			expectError: true,
		},
		{
			name:         "existing pipeline with wrong type",
			pipelineName: "create-user",
			params: map[string]any{
				"username": "testuser",
				"uid":      "not-an-int",
				"gid":      1000,
			},
			expectError: true,
		},
		{
			name:         "non-existing pipeline returns nil",
			pipelineName: "non-existent-pipeline",
			params:       map[string]any{},
			expectError:  false,
		},
		{
			name:         "download-verify-extract with checksum",
			pipelineName: "download-verify-extract",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/opt/file.tar.gz",
				"checksum":    "abc123",
			},
			expectError: false,
		},
		{
			name:         "download-verify-extract with checksum-url",
			pipelineName: "download-verify-extract",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/opt/file.tar.gz",
				"checksum-url": "https://example.com/checksum.txt",
			},
			expectError: false,
		},
		{
			name:         "download-verify-extract with both checksum options",
			pipelineName: "download-verify-extract",
			params: map[string]any{
				"url":          "https://example.com/file.tar.gz",
				"destination":  "/opt/file.tar.gz",
				"checksum":     "abc123",
				"checksum-url": "https://example.com/checksum.txt",
			},
			expectError: true,
		},
		{
			name:         "download-verify-extract with no checksum option",
			pipelineName: "download-verify-extract",
			params: map[string]any{
				"url":         "https://example.com/file.tar.gz",
				"destination": "/opt/file.tar.gz",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(tt.pipelineName, tt.params)
			hasError := err != nil

			if hasError != tt.expectError {
				t.Errorf("ValidateParams() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
