package util

import "fmt"

func ValidateStringParam(params map[string]any, key string) (string, error) {
	val, exists := params[key]
	if !exists {
		return "", fmt.Errorf("%s is required", key)
	}
	value, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string, got %T", key, val)
	}
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", key)
	}
	return value, nil
}

func ValidateIntParam(params map[string]any, key string) (int, error) {
	val, exists := params[key]
	if !exists {
		return 0, fmt.Errorf("%s is required", key)
	}
	if floatVal, ok := val.(float64); ok {
		return int(floatVal), nil
	}
	if intVal, ok := val.(int); ok {
		return intVal, nil
	}
	return 0, fmt.Errorf("%s must be an integer, got %T", key, val)
}

func ValidateOptionalStringParam(params map[string]any, key string, defaultVal string) string {
	value, ok := params[key].(string)
	if !ok || value == "" {
		return defaultVal
	}
	return value
}

func ValidateOptionalStringParamStrict(params map[string]any, key string, defaultVal string) (string, error) {
	val, exists := params[key]
	if !exists || val == nil {
		return defaultVal, nil
	}
	value, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string, got %T", key, val)
	}
	if value == "" {
		return defaultVal, nil
	}
	return value, nil
}

func ValidateOptionalBoolParam(params map[string]any, key string, defaultVal bool) (bool, error) {
	val, exists := params[key]
	if !exists || val == nil {
		return defaultVal, nil
	}
	value, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean, got %T", key, val)
	}
	return value, nil
}

func ValidateOptionalIntParam(params map[string]any, key string, defaultVal int) (int, error) {
	val, exists := params[key]
	if !exists || val == nil {
		return defaultVal, nil
	}
	if floatVal, ok := val.(float64); ok {
		return int(floatVal), nil
	}
	if intVal, ok := val.(int); ok {
		return intVal, nil
	}
	return 0, fmt.Errorf("%s must be an integer, got %T", key, val)
}

func ValidateMutuallyExclusiveRequired(hasA, hasB bool, nameA, nameB string) error {
	if !hasA && !hasB {
		return fmt.Errorf("either %s or %s is required", nameA, nameB)
	}
	if hasA && hasB {
		return fmt.Errorf("cannot specify both %s and %s", nameA, nameB)
	}
	return nil
}
