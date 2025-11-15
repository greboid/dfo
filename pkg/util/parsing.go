package util

import "fmt"

func ParseArrayParam[T any](data any, itemName string, parseItem func(map[string]any, int) (T, error)) ([]T, error) {
	items, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", itemName)
	}

	var result []T
	for i, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s at index %d must be a map", itemName, i)
		}

		parsed, err := parseItem(itemMap, i)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}

	return result, nil
}

func ExtractRequiredString(m map[string]any, key string, context string) (string, error) {
	value, ok := m[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("%s: %s is required and must be a non-empty string", context, key)
	}
	return value, nil
}

func ExtractOptionalString(m map[string]any, key string) string {
	value, ok := m[key].(string)
	if !ok {
		return ""
	}
	return value
}

func ExtractRequiredInt(m map[string]any, key string, context string) (int, error) {

	if floatVal, ok := m[key].(float64); ok {
		return int(floatVal), nil
	}
	if intVal, ok := m[key].(int); ok {
		return intVal, nil
	}
	return 0, fmt.Errorf("%s: %s is required and must be a number", context, key)
}

func ExtractOptionalInt(m map[string]any, key string, defaultValue int) int {
	if floatVal, ok := m[key].(float64); ok {
		return int(floatVal)
	}
	if intVal, ok := m[key].(int); ok {
		return intVal
	}
	return defaultValue
}

func ExtractStringSlice(m map[string]any, key string) []string {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case string:
			return []string{v}
		case []string:
			return v
		case []any:
			var result []string
			for _, item := range v {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}
