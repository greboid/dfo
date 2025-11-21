package util

import (
	"testing"
)

func TestParseArrayParam(t *testing.T) {
	stringParser := func(m map[string]any, index int) (string, error) {
		return ExtractRequiredString(m, "value", "test item")
	}

	tests := []struct {
		name        string
		data        any
		itemName    string
		expected    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid array with single item",
			data:     []any{map[string]any{"value": "test1"}},
			itemName: "items",
			expected: []string{"test1"},
		},
		{
			name: "valid array with multiple items",
			data: []any{
				map[string]any{"value": "test1"},
				map[string]any{"value": "test2"},
				map[string]any{"value": "test3"},
			},
			itemName: "items",
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "empty array",
			data:     []any{},
			itemName: "items",
			expected: []string{},
		},
		{
			name:        "not an array - string",
			data:        "not an array",
			itemName:    "items",
			expectError: true,
			errorMsg:    "items must be an array",
		},
		{
			name:        "not an array - map",
			data:        map[string]any{"key": "value"},
			itemName:    "items",
			expectError: true,
			errorMsg:    "items must be an array",
		},
		{
			name:        "array item is not a map",
			data:        []any{"string", "another"},
			itemName:    "items",
			expectError: true,
			errorMsg:    "items at index 0 must be a map",
		},
		{
			name: "array with invalid item at index 1",
			data: []any{
				map[string]any{"value": "test1"},
				"not a map",
			},
			itemName:    "items",
			expectError: true,
			errorMsg:    "items at index 1 must be a map",
		},
		{
			name: "parser error propagates",
			data: []any{
				map[string]any{"value": "test1"},
				map[string]any{"wrong": "key"},
			},
			itemName:    "items",
			expectError: true,
			errorMsg:    "test item: value is required and must be a non-empty string",
		},
		{
			name:        "nil data",
			data:        nil,
			itemName:    "items",
			expectError: true,
			errorMsg:    "items must be an array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseArrayParam(tt.data, tt.itemName, stringParser)

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
				if len(result) != len(tt.expected) {
					t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
					return
				}
				for i, exp := range tt.expected {
					if result[i] != exp {
						t.Errorf("at index %d: expected %q, got %q", i, exp, result[i])
					}
				}
			}
		})
	}
}

func TestExtractRequiredString(t *testing.T) {
	tests := []struct {
		name        string
		m           map[string]any
		key         string
		context     string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid string",
			m:        map[string]any{"name": "test"},
			key:      "name",
			context:  "user",
			expected: "test",
		},
		{
			name:        "missing key",
			m:           map[string]any{"other": "value"},
			key:         "name",
			context:     "user",
			expectError: true,
			errorMsg:    "user: name is required and must be a non-empty string",
		},
		{
			name:        "empty string",
			m:           map[string]any{"name": ""},
			key:         "name",
			context:     "user",
			expectError: true,
			errorMsg:    "user: name is required and must be a non-empty string",
		},
		{
			name:        "wrong type - integer",
			m:           map[string]any{"name": 123},
			key:         "name",
			context:     "user",
			expectError: true,
			errorMsg:    "user: name is required and must be a non-empty string",
		},
		{
			name:        "wrong type - boolean",
			m:           map[string]any{"name": true},
			key:         "name",
			context:     "user",
			expectError: true,
			errorMsg:    "user: name is required and must be a non-empty string",
		},
		{
			name:        "nil map",
			m:           nil,
			key:         "name",
			context:     "user",
			expectError: true,
			errorMsg:    "user: name is required and must be a non-empty string",
		},
		{
			name:     "whitespace string is valid",
			m:        map[string]any{"name": "   "},
			key:      "name",
			context:  "user",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractRequiredString(tt.m, tt.key, tt.context)

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

func TestExtractOptionalString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{
			name:     "valid string",
			m:        map[string]any{"name": "test"},
			key:      "name",
			expected: "test",
		},
		{
			name:     "missing key returns empty",
			m:        map[string]any{"other": "value"},
			key:      "name",
			expected: "",
		},
		{
			name:     "empty string",
			m:        map[string]any{"name": ""},
			key:      "name",
			expected: "",
		},
		{
			name:     "wrong type - integer",
			m:        map[string]any{"name": 123},
			key:      "name",
			expected: "",
		},
		{
			name:     "wrong type - boolean",
			m:        map[string]any{"name": true},
			key:      "name",
			expected: "",
		},
		{
			name:     "wrong type - array",
			m:        map[string]any{"name": []string{"test"}},
			key:      "name",
			expected: "",
		},
		{
			name:     "nil map",
			m:        nil,
			key:      "name",
			expected: "",
		},
		{
			name:     "whitespace string",
			m:        map[string]any{"name": "   "},
			key:      "name",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOptionalString(tt.m, tt.key)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractRequiredInt(t *testing.T) {
	tests := []struct {
		name        string
		m           map[string]any
		key         string
		context     string
		expected    int
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid int",
			m:        map[string]any{"port": 8080},
			key:      "port",
			context:  "server",
			expected: 8080,
		},
		{
			name:     "valid float64",
			m:        map[string]any{"port": 8080.0},
			key:      "port",
			context:  "server",
			expected: 8080,
		},
		{
			name:     "float64 with decimal truncates",
			m:        map[string]any{"port": 8080.7},
			key:      "port",
			context:  "server",
			expected: 8080,
		},
		{
			name:     "zero value",
			m:        map[string]any{"port": 0},
			key:      "port",
			context:  "server",
			expected: 0,
		},
		{
			name:     "negative value",
			m:        map[string]any{"port": -1},
			key:      "port",
			context:  "server",
			expected: -1,
		},
		{
			name:        "missing key",
			m:           map[string]any{"other": 123},
			key:         "port",
			context:     "server",
			expectError: true,
			errorMsg:    "server: port is required and must be a number",
		},
		{
			name:        "wrong type - string",
			m:           map[string]any{"port": "8080"},
			key:         "port",
			context:     "server",
			expectError: true,
			errorMsg:    "server: port is required and must be a number",
		},
		{
			name:        "wrong type - boolean",
			m:           map[string]any{"port": true},
			key:         "port",
			context:     "server",
			expectError: true,
			errorMsg:    "server: port is required and must be a number",
		},
		{
			name:        "nil map",
			m:           nil,
			key:         "port",
			context:     "server",
			expectError: true,
			errorMsg:    "server: port is required and must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractRequiredInt(tt.m, tt.key, tt.context)

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

func TestExtractOptionalInt(t *testing.T) {
	tests := []struct {
		name         string
		m            map[string]any
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid int",
			m:            map[string]any{"port": 8080},
			key:          "port",
			defaultValue: 3000,
			expected:     8080,
		},
		{
			name:         "valid float64",
			m:            map[string]any{"port": 8080.0},
			key:          "port",
			defaultValue: 3000,
			expected:     8080,
		},
		{
			name:         "float64 with decimal truncates",
			m:            map[string]any{"port": 8080.7},
			key:          "port",
			defaultValue: 3000,
			expected:     8080,
		},
		{
			name:         "zero value",
			m:            map[string]any{"port": 0},
			key:          "port",
			defaultValue: 3000,
			expected:     0,
		},
		{
			name:         "negative value",
			m:            map[string]any{"port": -1},
			key:          "port",
			defaultValue: 3000,
			expected:     -1,
		},
		{
			name:         "missing key returns default",
			m:            map[string]any{"other": 123},
			key:          "port",
			defaultValue: 3000,
			expected:     3000,
		},
		{
			name:         "wrong type - string returns default",
			m:            map[string]any{"port": "8080"},
			key:          "port",
			defaultValue: 3000,
			expected:     3000,
		},
		{
			name:         "wrong type - boolean returns default",
			m:            map[string]any{"port": true},
			key:          "port",
			defaultValue: 3000,
			expected:     3000,
		},
		{
			name:         "nil map returns default",
			m:            nil,
			key:          "port",
			defaultValue: 3000,
			expected:     3000,
		},
		{
			name:         "zero default value",
			m:            map[string]any{"other": 123},
			key:          "port",
			defaultValue: 0,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOptionalInt(tt.m, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestExtractStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected []string
	}{
		{
			name:     "single string converted to slice",
			m:        map[string]any{"tags": "latest"},
			key:      "tags",
			expected: []string{"latest"},
		},
		{
			name:     "string slice",
			m:        map[string]any{"tags": []string{"v1", "v2", "latest"}},
			key:      "tags",
			expected: []string{"v1", "v2", "latest"},
		},
		{
			name:     "[]any with strings",
			m:        map[string]any{"tags": []any{"v1", "v2", "latest"}},
			key:      "tags",
			expected: []string{"v1", "v2", "latest"},
		},
		{
			name:     "[]any with mixed types filters non-strings",
			m:        map[string]any{"tags": []any{"v1", 123, "v2", true, "latest"}},
			key:      "tags",
			expected: []string{"v1", "v2", "latest"},
		},
		{
			name:     "[]any with no strings",
			m:        map[string]any{"tags": []any{123, true, 45.6}},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "missing key returns nil",
			m:        map[string]any{"other": "value"},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "empty string slice",
			m:        map[string]any{"tags": []string{}},
			key:      "tags",
			expected: []string{},
		},
		{
			name:     "empty []any",
			m:        map[string]any{"tags": []any{}},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "wrong type - int returns nil",
			m:        map[string]any{"tags": 123},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "wrong type - bool returns nil",
			m:        map[string]any{"tags": true},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "wrong type - map returns nil",
			m:        map[string]any{"tags": map[string]string{"key": "value"}},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "nil map",
			m:        nil,
			key:      "tags",
			expected: nil,
		},
		{
			name:     "empty string as single value",
			m:        map[string]any{"tags": ""},
			key:      "tags",
			expected: []string{""},
		},
		{
			name:     "empty strings in slice",
			m:        map[string]any{"tags": []string{"", "v1", ""}},
			key:      "tags",
			expected: []string{"", "v1", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractStringSlice(tt.m, tt.key)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("at index %d: expected %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}
