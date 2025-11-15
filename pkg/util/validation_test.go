package util

import (
	"testing"
)

func TestValidateStringParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		key         string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid string parameter",
			params:      map[string]any{"url": "https://example.com"},
			key:         "url",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]any{"other": "value"},
			key:         "url",
			expectError: true,
			errorMsg:    "url is required",
		},
		{
			name:        "empty string parameter",
			params:      map[string]any{"url": ""},
			key:         "url",
			expectError: true,
			errorMsg:    "url cannot be empty",
		},
		{
			name:        "wrong type - integer",
			params:      map[string]any{"url": 123},
			key:         "url",
			expectError: true,
			errorMsg:    "url must be a string, got int",
		},
		{
			name:        "wrong type - boolean",
			params:      map[string]any{"url": true},
			key:         "url",
			expectError: true,
			errorMsg:    "url must be a string, got bool",
		},
		{
			name:        "wrong type - array",
			params:      map[string]any{"url": []string{"value"}},
			key:         "url",
			expectError: true,
			errorMsg:    "url must be a string, got []string",
		},
		{
			name:        "wrong type - map",
			params:      map[string]any{"url": map[string]string{"key": "value"}},
			key:         "url",
			expectError: true,
			errorMsg:    "url must be a string, got map[string]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateStringParam(tt.params, tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateIntParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		key         string
		expected    int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid int parameter",
			params:      map[string]any{"port": 8080},
			key:         "port",
			expected:    8080,
			expectError: false,
		},
		{
			name:        "valid float64 parameter",
			params:      map[string]any{"port": 8080.0},
			key:         "port",
			expected:    8080,
			expectError: false,
		},
		{
			name:        "float64 with decimal conversion",
			params:      map[string]any{"port": 8080.7},
			key:         "port",
			expected:    8080,
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]any{"other": 123},
			key:         "port",
			expectError: true,
			errorMsg:    "port is required",
		},
		{
			name:        "wrong type - string",
			params:      map[string]any{"port": "8080"},
			key:         "port",
			expectError: true,
			errorMsg:    "port must be an integer, got string",
		},
		{
			name:        "wrong type - boolean",
			params:      map[string]any{"port": true},
			key:         "port",
			expectError: true,
			errorMsg:    "port must be an integer, got bool",
		},
		{
			name:        "wrong type - array",
			params:      map[string]any{"port": []int{8080}},
			key:         "port",
			expectError: true,
			errorMsg:    "port must be an integer, got []int",
		},
		{
			name:        "zero value",
			params:      map[string]any{"port": 0},
			key:         "port",
			expected:    0,
			expectError: false,
		},
		{
			name:        "negative value",
			params:      map[string]any{"port": -1},
			key:         "port",
			expected:    -1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateIntParam(tt.params, tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateOptionalStringParam(t *testing.T) {
	tests := []struct {
		name       string
		params     map[string]any
		key        string
		defaultVal string
		expected   string
	}{
		{
			name:       "valid string parameter",
			params:     map[string]any{"name": "test"},
			key:        "name",
			defaultVal: "default",
			expected:   "test",
		},
		{
			name:       "missing parameter returns default",
			params:     map[string]any{"other": "value"},
			key:        "name",
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "empty string returns default",
			params:     map[string]any{"name": ""},
			key:        "name",
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "wrong type returns default",
			params:     map[string]any{"name": 123},
			key:        "name",
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "empty default value",
			params:     map[string]any{"other": "value"},
			key:        "name",
			defaultVal: "",
			expected:   "",
		},
		{
			name:       "nil params map",
			params:     nil,
			key:        "name",
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "boolean value returns default",
			params:     map[string]any{"name": true},
			key:        "name",
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "array value returns default",
			params:     map[string]any{"name": []string{"test"}},
			key:        "name",
			defaultVal: "default",
			expected:   "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateOptionalStringParam(tt.params, tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateOptionalStringParamStrict(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		key         string
		defaultVal  string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid string parameter",
			params:      map[string]any{"name": "test"},
			key:         "name",
			defaultVal:  "default",
			expected:    "test",
			expectError: false,
		},
		{
			name:        "missing parameter returns default",
			params:      map[string]any{"other": "value"},
			key:         "name",
			defaultVal:  "default",
			expected:    "default",
			expectError: false,
		},
		{
			name:        "nil value returns default",
			params:      map[string]any{"name": nil},
			key:         "name",
			defaultVal:  "default",
			expected:    "default",
			expectError: false,
		},
		{
			name:        "empty string returns default",
			params:      map[string]any{"name": ""},
			key:         "name",
			defaultVal:  "default",
			expected:    "default",
			expectError: false,
		},
		{
			name:        "wrong type - integer returns error",
			params:      map[string]any{"name": 123},
			key:         "name",
			defaultVal:  "default",
			expectError: true,
			errorMsg:    "name must be a string, got int",
		},
		{
			name:        "wrong type - boolean returns error",
			params:      map[string]any{"name": true},
			key:         "name",
			defaultVal:  "default",
			expectError: true,
			errorMsg:    "name must be a string, got bool",
		},
		{
			name:        "wrong type - array returns error",
			params:      map[string]any{"name": []string{"test"}},
			key:         "name",
			defaultVal:  "default",
			expectError: true,
			errorMsg:    "name must be a string, got []string",
		},
		{
			name:        "wrong type - float returns error",
			params:      map[string]any{"name": 3.14},
			key:         "name",
			defaultVal:  "default",
			expectError: true,
			errorMsg:    "name must be a string, got float64",
		},
		{
			name:        "empty default value with missing key",
			params:      map[string]any{"other": "value"},
			key:         "name",
			defaultVal:  "",
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateOptionalStringParamStrict(tt.params, tt.key, tt.defaultVal)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.errorMsg)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateOptionalBoolParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		key         string
		defaultVal  bool
		expected    bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid true parameter",
			params:      map[string]any{"enabled": true},
			key:         "enabled",
			defaultVal:  false,
			expected:    true,
			expectError: false,
		},
		{
			name:        "valid false parameter",
			params:      map[string]any{"enabled": false},
			key:         "enabled",
			defaultVal:  true,
			expected:    false,
			expectError: false,
		},
		{
			name:        "missing parameter returns default true",
			params:      map[string]any{"other": "value"},
			key:         "enabled",
			defaultVal:  true,
			expected:    true,
			expectError: false,
		},
		{
			name:        "missing parameter returns default false",
			params:      map[string]any{"other": "value"},
			key:         "enabled",
			defaultVal:  false,
			expected:    false,
			expectError: false,
		},
		{
			name:        "nil value returns default",
			params:      map[string]any{"enabled": nil},
			key:         "enabled",
			defaultVal:  true,
			expected:    true,
			expectError: false,
		},
		{
			name:        "wrong type - string returns error",
			params:      map[string]any{"enabled": "true"},
			key:         "enabled",
			defaultVal:  false,
			expectError: true,
			errorMsg:    "enabled must be a boolean, got string",
		},
		{
			name:        "wrong type - integer returns error",
			params:      map[string]any{"enabled": 1},
			key:         "enabled",
			defaultVal:  false,
			expectError: true,
			errorMsg:    "enabled must be a boolean, got int",
		},
		{
			name:        "wrong type - float returns error",
			params:      map[string]any{"enabled": 1.0},
			key:         "enabled",
			defaultVal:  false,
			expectError: true,
			errorMsg:    "enabled must be a boolean, got float64",
		},
		{
			name:        "wrong type - array returns error",
			params:      map[string]any{"enabled": []bool{true}},
			key:         "enabled",
			defaultVal:  false,
			expectError: true,
			errorMsg:    "enabled must be a boolean, got []bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateOptionalBoolParam(tt.params, tt.key, tt.defaultVal)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.errorMsg)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateOptionalIntParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		key         string
		defaultVal  int
		expected    int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid int parameter",
			params:      map[string]any{"count": 42},
			key:         "count",
			defaultVal:  0,
			expected:    42,
			expectError: false,
		},
		{
			name:        "valid float64 parameter (from JSON)",
			params:      map[string]any{"count": 42.0},
			key:         "count",
			defaultVal:  0,
			expected:    42,
			expectError: false,
		},
		{
			name:        "float64 with decimal truncates",
			params:      map[string]any{"count": 42.9},
			key:         "count",
			defaultVal:  0,
			expected:    42,
			expectError: false,
		},
		{
			name:        "missing parameter returns default",
			params:      map[string]any{"other": "value"},
			key:         "count",
			defaultVal:  10,
			expected:    10,
			expectError: false,
		},
		{
			name:        "nil value returns default",
			params:      map[string]any{"count": nil},
			key:         "count",
			defaultVal:  10,
			expected:    10,
			expectError: false,
		},
		{
			name:        "zero value is valid",
			params:      map[string]any{"count": 0},
			key:         "count",
			defaultVal:  10,
			expected:    0,
			expectError: false,
		},
		{
			name:        "negative value is valid",
			params:      map[string]any{"count": -5},
			key:         "count",
			defaultVal:  0,
			expected:    -5,
			expectError: false,
		},
		{
			name:        "wrong type - string returns error",
			params:      map[string]any{"count": "42"},
			key:         "count",
			defaultVal:  0,
			expectError: true,
			errorMsg:    "count must be an integer, got string",
		},
		{
			name:        "wrong type - boolean returns error",
			params:      map[string]any{"count": true},
			key:         "count",
			defaultVal:  0,
			expectError: true,
			errorMsg:    "count must be an integer, got bool",
		},
		{
			name:        "wrong type - array returns error",
			params:      map[string]any{"count": []int{1, 2, 3}},
			key:         "count",
			defaultVal:  0,
			expectError: true,
			errorMsg:    "count must be an integer, got []int",
		},
		{
			name:        "negative default with missing key",
			params:      map[string]any{"other": "value"},
			key:         "count",
			defaultVal:  -1,
			expected:    -1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateOptionalIntParam(tt.params, tt.key, tt.defaultVal)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.errorMsg)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestValidateMutuallyExclusiveRequired(t *testing.T) {
	tests := []struct {
		name        string
		hasA        bool
		hasB        bool
		nameA       string
		nameB       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "only A present - valid",
			hasA:        true,
			hasB:        false,
			nameA:       "base-image",
			nameB:       "external-image",
			expectError: false,
		},
		{
			name:        "only B present - valid",
			hasA:        false,
			hasB:        true,
			nameA:       "base-image",
			nameB:       "external-image",
			expectError: false,
		},
		{
			name:        "both missing - error",
			hasA:        false,
			hasB:        false,
			nameA:       "base-image",
			nameB:       "external-image",
			expectError: true,
			errorMsg:    "either base-image or external-image is required",
		},
		{
			name:        "both present - error",
			hasA:        true,
			hasB:        true,
			nameA:       "base-image",
			nameB:       "external-image",
			expectError: true,
			errorMsg:    "cannot specify both base-image and external-image",
		},
		{
			name:        "different parameter names - both missing",
			hasA:        false,
			hasB:        false,
			nameA:       "tag",
			nameB:       "commit",
			expectError: true,
			errorMsg:    "either tag or commit is required",
		},
		{
			name:        "different parameter names - both present",
			hasA:        true,
			hasB:        true,
			nameA:       "tag",
			nameB:       "commit",
			expectError: true,
			errorMsg:    "cannot specify both tag and commit",
		},
		{
			name:        "different parameter names - only A",
			hasA:        true,
			hasB:        false,
			nameA:       "tag",
			nameB:       "commit",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMutuallyExclusiveRequired(tt.hasA, tt.hasB, tt.nameA, tt.nameB)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.errorMsg)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
