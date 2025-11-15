package util

import (
	"reflect"
	"testing"
)

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name: "single key",
			input: map[string]string{
				"key1": "value1",
			},
			expected: []string{"key1"},
		},
		{
			name: "multiple keys sorted alphabetically",
			input: map[string]string{
				"zebra":  "z",
				"alpha":  "a",
				"middle": "m",
			},
			expected: []string{"alpha", "middle", "zebra"},
		},
		{
			name:     "empty map returns empty slice",
			input:    map[string]string{},
			expected: []string{},
		},
		{
			name:     "nil map returns empty slice",
			input:    nil,
			expected: []string{},
		},
		{
			name: "keys with numbers",
			input: map[string]string{
				"key10": "value",
				"key2":  "value",
				"key1":  "value",
			},
			expected: []string{"key1", "key10", "key2"},
		},
		{
			name: "keys with special characters",
			input: map[string]string{
				"key_underscore": "value",
				"key-dash":       "value",
				"key.dot":        "value",
			},
			expected: []string{"key-dash", "key.dot", "key_underscore"},
		},
		{
			name: "keys with mixed case (case-sensitive sort)",
			input: map[string]string{
				"Zebra": "value",
				"alpha": "value",
				"Apple": "value",
			},
			expected: []string{"Apple", "Zebra", "alpha"},
		},
		{
			name: "many keys",
			input: map[string]string{
				"e": "value",
				"d": "value",
				"c": "value",
				"b": "value",
				"a": "value",
			},
			expected: []string{"a", "b", "c", "d", "e"},
		},
		{
			name: "keys with prefixes",
			input: map[string]string{
				"prefix_zebra": "value",
				"prefix_alpha": "value",
				"prefix_beta":  "value",
			},
			expected: []string{"prefix_alpha", "prefix_beta", "prefix_zebra"},
		},
		{
			name: "empty string key",
			input: map[string]string{
				"":     "value",
				"key1": "value",
				"key2": "value",
			},
			expected: []string{"", "key1", "key2"},
		},
		{
			name: "unicode keys (byte-wise sort)",
			input: map[string]string{
				"zürich":   "value",
				"apple":    "value",
				"ångström": "value",
			},
			expected: []string{"apple", "zürich", "ångström"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortedKeys(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}

			if len(result) > 1 {
				for i := 0; i < len(result)-1; i++ {
					if result[i] > result[i+1] {
						t.Errorf("result is not sorted: %v", result)
						break
					}
				}
			}

			if len(result) != len(tt.input) {
				t.Errorf("expected %d keys, got %d", len(tt.input), len(result))
			}

			for _, key := range result {
				if _, exists := tt.input[key]; !exists && tt.input != nil {
					t.Errorf("key %q in result but not in input map", key)
				}
			}
		})
	}
}
