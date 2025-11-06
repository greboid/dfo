package graph

import (
	"fmt"
	"strings"
)

// Graph represents a dependency graph
type Graph struct {
	nodes        map[string]bool
	dependencies map[string][]string
}

// New creates a new empty graph
func New() *Graph {
	return &Graph{
		nodes:        make(map[string]bool),
		dependencies: make(map[string][]string),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(name string) {
	g.nodes[name] = true
	if g.dependencies[name] == nil {
		g.dependencies[name] = []string{}
	}
}

// AddDependency adds a dependency relationship (from depends on to)
func (g *Graph) AddDependency(from, to string) {
	g.AddNode(from)
	g.AddNode(to)
	g.dependencies[from] = append(g.dependencies[from], to)
}

// TopologicalSort returns nodes in dependency order (dependencies first)
// Returns an error if a cycle is detected
func (g *Graph) TopologicalSort() ([]string, error) {
	// Track visited nodes and nodes in current path (for cycle detection)
	visited := make(map[string]bool)
	inPath := make(map[string]bool)
	path := []string{}
	result := []string{}

	// Visit function for DFS
	var visit func(string) error
	visit = func(node string) error {
		if inPath[node] {
			// Build cycle path for better error message
			cycleStart := -1
			for i, n := range path {
				if n == node {
					cycleStart = i
					break
				}
			}
			cyclePath := append(path[cycleStart:], node)
			return fmt.Errorf("circular dependency detected: %s", strings.Join(cyclePath, " → "))
		}
		if visited[node] {
			return nil
		}

		inPath[node] = true
		visited[node] = true
		path = append(path, node)

		// Visit dependencies first
		for _, dep := range g.dependencies[node] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		inPath[node] = false
		path = path[:len(path)-1]
		result = append(result, node)
		return nil
	}

	// Visit all nodes
	for node := range g.nodes {
		if !visited[node] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// GetDependencies returns all dependencies for a given node in build order
func (g *Graph) GetDependencies(node string) ([]string, error) {
	// Create a subgraph with just this node and its dependencies
	subgraph := New()

	// DFS to find all dependencies
	var findDeps func(string)
	findDeps = func(n string) {
		if !subgraph.nodes[n] {
			subgraph.AddNode(n)
			for _, dep := range g.dependencies[n] {
				subgraph.AddDependency(n, dep)
				findDeps(dep)
			}
		}
	}

	findDeps(node)

	// Sort the subgraph
	return subgraph.TopologicalSort()
}

// String returns a string representation of the graph
func (g *Graph) String() string {
	var sb strings.Builder
	sb.WriteString("Graph:\n")
	for node := range g.nodes {
		deps := g.dependencies[node]
		if len(deps) > 0 {
			sb.WriteString(fmt.Sprintf("  %s -> %v\n", node, deps))
		} else {
			sb.WriteString(fmt.Sprintf("  %s (no dependencies)\n", node))
		}
	}
	return sb.String()
}
