package graph

import (
	"fmt"
	"strings"

	"github.com/greboid/dfo/pkg/config"
)

type Container struct {
	Name         string
	ConfigPath   string
	Config       *config.BuildConfig
	Dependencies []string
}

type Graph struct {
	Containers map[string]*Container
}

type CircularDependencyError struct {
	Chain []string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s",
		strings.Join(e.Chain, " -> "))
}
