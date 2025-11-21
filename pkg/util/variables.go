package util

import (
	"fmt"
	"regexp"
	"strings"
)

// varRefPattern %{VAR_NAME}
var varRefPattern = regexp.MustCompile(`%\{([A-Za-z_][A-Za-z0-9_]*)}`)

func ExtractVariableReferences(s string) []string {
	matches := varRefPattern.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var refs []string
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			refs = append(refs, match[1])
		}
	}
	return refs
}

func ValidateVariableReferences(s string, vars map[string]string, context string) error {
	refs := ExtractVariableReferences(s)
	if len(refs) == 0 {
		return nil
	}

	var undefined []string
	for _, ref := range refs {
		if _, exists := vars[ref]; !exists {
			undefined = append(undefined, "%{"+ref+"}")
		}
	}

	if len(undefined) > 0 {
		return fmt.Errorf("%s: undefined variable(s): %s", context, strings.Join(undefined, ", "))
	}
	return nil
}

func ExpandVars(s string, vars map[string]string) string {
	if vars == nil {
		return s
	}
	result := s
	for key, value := range vars {
		result = strings.ReplaceAll(result, "%{"+key+"}", value)
	}
	return result
}

func ExpandVarsStrict(s string, vars map[string]string, context string) (string, error) {
	if err := ValidateVariableReferences(s, vars, context); err != nil {
		return "", err
	}
	return ExpandVars(s, vars), nil
}
