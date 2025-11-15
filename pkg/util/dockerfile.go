package util

import (
	"fmt"
	"strings"
)

func FormatDockerfileArray(directive string, values []string) string {
	if len(values) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(directive)
	b.WriteString(" [")
	for i, val := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%q", val))
	}
	b.WriteString("]\n\n")
	return b.String()
}

func FormatMapDirectives(directive string, values map[string]string) string {
	if len(values) == 0 {
		return ""
	}

	var b strings.Builder
	for _, key := range SortedKeys(values) {
		b.WriteString(fmt.Sprintf("%s %s=%q\n", directive, key, values[key]))
	}
	b.WriteString("\n")
	return b.String()
}

func WrapRun(command string) string {
	if command == "" {
		return ""
	}
	return fmt.Sprintf("RUN %s\n", command)
}

func BuildPackageList(packages []string) string {
	if len(packages) == 0 {
		return ""
	}

	var pkgArgs []string
	for _, pkg := range packages {
		pkgArgs = append(pkgArgs, fmt.Sprintf("%q", pkg))
	}
	return strings.Join(pkgArgs, " ")
}
