package graph

import "sort"

func (g *Graph) TopologicalSort() ([][]string, error) {
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for name := range g.Containers {
		inDegree[name] = 0
	}

	for name, container := range g.Containers {
		for _, dep := range container.Dependencies {
			if _, exists := g.Containers[dep]; exists {
				adjList[dep] = append(adjList[dep], name)
				inDegree[name]++
			}
		}
	}

	var layers [][]string
	processed := make(map[string]bool)

	for len(processed) < len(g.Containers) {
		var currentLayer []string
		for name := range g.Containers {
			if !processed[name] && inDegree[name] == 0 {
				currentLayer = append(currentLayer, name)
			}
		}

		if len(currentLayer) == 0 {
			cycle := g.findCycle(processed)
			return nil, &CircularDependencyError{Chain: cycle}
		}

		sort.Strings(currentLayer)

		for _, name := range currentLayer {
			processed[name] = true

			for _, dependent := range adjList[name] {
				inDegree[dependent]--
			}
		}

		layers = append(layers, currentLayer)
	}

	return layers, nil
}

func (g *Graph) findCycle(processed map[string]bool) []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parent := make(map[string]string)

	var dfs func(node string) string
	dfs = func(node string) string {
		visited[node] = true
		recStack[node] = true

		container := g.Containers[node]
		for _, dep := range container.Dependencies {
			if _, exists := g.Containers[dep]; !exists {
				continue
			}

			if !visited[dep] {
				parent[dep] = node
				if cycleStart := dfs(dep); cycleStart != "" {
					return cycleStart
				}
			} else if recStack[dep] {
				parent[dep] = node
				return dep
			}
		}

		recStack[node] = false
		return ""
	}

	for name := range g.Containers {
		if processed[name] || visited[name] {
			continue
		}

		if cycleStart := dfs(name); cycleStart != "" {
			return g.buildCyclePath(cycleStart, parent)
		}
	}

	return []string{"unknown cycle"}
}

func (g *Graph) buildCyclePath(start string, parent map[string]string) []string {
	path := []string{start}
	current := parent[start]

	for current != start {
		path = append([]string{current}, path...)
		current = parent[current]
	}

	path = append(path, start)
	return path
}
