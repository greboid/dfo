package graph

import (
	"errors"
	"testing"

	"github.com/greboid/dfo/pkg/config"
)

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name       string
		graph      *Graph
		wantLayers int
		wantErr    bool
	}{
		{
			name: "single node",
			graph: &Graph{
				Containers: map[string]*Container{
					"app": {
						Name:         "app",
						Config:       &config.BuildConfig{Package: config.Package{Name: "app"}},
						Dependencies: nil,
					},
				},
			},
			wantLayers: 1,
			wantErr:    false,
		},
		{
			name: "linear dependencies",
			graph: &Graph{
				Containers: map[string]*Container{
					"base": {
						Name:         "base",
						Dependencies: nil,
					},
					"app": {
						Name:         "app",
						Dependencies: []string{"base"},
					},
					"final": {
						Name:         "final",
						Dependencies: []string{"app"},
					},
				},
			},
			wantLayers: 3,
			wantErr:    false,
		},
		{
			name: "independent nodes",
			graph: &Graph{
				Containers: map[string]*Container{
					"app1": {
						Name:         "app1",
						Dependencies: nil,
					},
					"app2": {
						Name:         "app2",
						Dependencies: nil,
					},
				},
			},
			wantLayers: 1,
			wantErr:    false,
		},
		{
			name: "diamond dependency",
			graph: &Graph{
				Containers: map[string]*Container{
					"base": {
						Name:         "base",
						Dependencies: nil,
					},
					"app1": {
						Name:         "app1",
						Dependencies: []string{"base"},
					},
					"app2": {
						Name:         "app2",
						Dependencies: []string{"base"},
					},
					"final": {
						Name:         "final",
						Dependencies: []string{"app1", "app2"},
					},
				},
			},
			wantLayers: 3,
			wantErr:    false,
		},
		{
			name: "self-cycle",
			graph: &Graph{
				Containers: map[string]*Container{
					"app": {
						Name:         "app",
						Dependencies: []string{"app"},
					},
				},
			},
			wantLayers: 0,
			wantErr:    true,
		},
		{
			name: "two-node cycle",
			graph: &Graph{
				Containers: map[string]*Container{
					"app1": {
						Name:         "app1",
						Dependencies: []string{"app2"},
					},
					"app2": {
						Name:         "app2",
						Dependencies: []string{"app1"},
					},
				},
			},
			wantLayers: 0,
			wantErr:    true,
		},
		{
			name: "three-node cycle",
			graph: &Graph{
				Containers: map[string]*Container{
					"app1": {
						Name:         "app1",
						Dependencies: []string{"app2"},
					},
					"app2": {
						Name:         "app2",
						Dependencies: []string{"app3"},
					},
					"app3": {
						Name:         "app3",
						Dependencies: []string{"app1"},
					},
				},
			},
			wantLayers: 0,
			wantErr:    true,
		},
		{
			name: "partial cycle with independent nodes",
			graph: &Graph{
				Containers: map[string]*Container{
					"standalone": {
						Name:         "standalone",
						Dependencies: nil,
					},
					"app1": {
						Name:         "app1",
						Dependencies: []string{"app2"},
					},
					"app2": {
						Name:         "app2",
						Dependencies: []string{"app1"},
					},
				},
			},
			wantLayers: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layers, err := tt.graph.TopologicalSort()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var circularDependencyError *CircularDependencyError
				if !errors.As(err, &circularDependencyError) {
					t.Errorf("error type = %T, want CircularDependencyError", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(layers) != tt.wantLayers {
				t.Errorf("got %d layers, want %d", len(layers), tt.wantLayers)
			}

			totalNodes := 0
			for _, layer := range layers {
				totalNodes += len(layer)
			}
			if totalNodes != len(tt.graph.Containers) {
				t.Errorf("sorted %d nodes, want %d", totalNodes, len(tt.graph.Containers))
			}
		})
	}
}

func TestTopologicalSort_Ordering(t *testing.T) {
	graph := &Graph{
		Containers: map[string]*Container{
			"base": {
				Name:         "base",
				Dependencies: nil,
			},
			"app1": {
				Name:         "app1",
				Dependencies: []string{"base"},
			},
			"app2": {
				Name:         "app2",
				Dependencies: []string{"base"},
			},
		},
	}

	layers, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(layers) != 2 {
		t.Fatalf("got %d layers, want 2", len(layers))
	}

	if len(layers[0]) != 1 {
		t.Errorf("layer 0 has %d nodes, want 1 (base)", len(layers[0]))
	}
	if layers[0][0] != "base" {
		t.Errorf("layer 0[0] = %q, want 'base'", layers[0][0])
	}

	if len(layers[1]) != 2 {
		t.Errorf("layer 1 has %d nodes, want 2 (app1, app2)", len(layers[1]))
	}
}

func TestTopologicalSort_LayerOrdering(t *testing.T) {
	tests := []struct {
		name          string
		graph         *Graph
		expectedOrder []string
	}{
		{
			name: "linear chain",
			graph: &Graph{
				Containers: map[string]*Container{
					"a": {Name: "a", Dependencies: nil},
					"b": {Name: "b", Dependencies: []string{"a"}},
					"c": {Name: "c", Dependencies: []string{"b"}},
				},
			},
			expectedOrder: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layers, err := tt.graph.TopologicalSort()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			flattened := []string{}
			for _, layer := range layers {
				flattened = append(flattened, layer...)
			}

			for i, expected := range tt.expectedOrder {
				if flattened[i] != expected {
					t.Errorf("order[%d] = %q, want %q", i, flattened[i], expected)
				}
			}
		})
	}
}

func TestFindCycle(t *testing.T) {
	tests := []struct {
		name      string
		graph     *Graph
		processed map[string]bool
		wantNil   bool
		wantCycle []string
	}{
		{
			name: "no cycle",
			graph: &Graph{
				Containers: map[string]*Container{
					"base": {Name: "base", Dependencies: nil},
					"app":  {Name: "app", Dependencies: []string{"base"}},
				},
			},
			processed: map[string]bool{},
			wantNil:   false,
			wantCycle: []string{"unknown cycle"},
		},
		{
			name: "self-cycle",
			graph: &Graph{
				Containers: map[string]*Container{
					"app": {Name: "app", Dependencies: []string{"app"}},
				},
			},
			processed: map[string]bool{},
			wantNil:   false,
			wantCycle: []string{"app", "app"},
		},
		{
			name: "two-node cycle",
			graph: &Graph{
				Containers: map[string]*Container{
					"app1": {Name: "app1", Dependencies: []string{"app2"}},
					"app2": {Name: "app2", Dependencies: []string{"app1"}},
				},
			},
			processed: map[string]bool{},
			wantNil:   false,
			wantCycle: []string{"app1", "app2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycle := tt.graph.findCycle(tt.processed)
			if tt.wantNil {
				if len(cycle) != 1 || cycle[0] != "unknown cycle" {
					t.Errorf("findCycle() = %v, want [\"unknown cycle\"]", cycle)
				}
				return
			}
			if cycle == nil {
				t.Fatal("findCycle() = nil, want a cycle")
			}
			if len(cycle) == 0 {
				t.Fatal("findCycle() returned empty cycle")
			}

			expectedNodes := make(map[string]bool)
			for _, node := range tt.wantCycle {
				if node != "unknown cycle" {
					expectedNodes[node] = true
				}
			}

			for _, node := range cycle {
				if expectedNodes[node] {
					expectedNodes[node] = false
				}
			}

			for node, shouldExist := range expectedNodes {
				if shouldExist {
					t.Errorf("findCycle() missing expected node %q in cycle %v", node, cycle)
				}
			}
		})
	}
}

func TestCircularDependencyError(t *testing.T) {
	err := &CircularDependencyError{
		Chain: []string{"app1", "app2", "app3"},
	}

	wantMsg := "circular dependency detected: app1 -> app2 -> app3"
	if err.Error() != wantMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), wantMsg)
	}
}
