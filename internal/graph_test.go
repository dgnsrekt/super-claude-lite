package internal

import (
	"testing"

	"github.com/dominikbraun/graph"
)

// TestGraphLibraryImport verifies that the github.com/dominikbraun/graph
// library can be imported and basic functionality works as expected.
func TestGraphLibraryImport(t *testing.T) {
	// Create a new directed graph with string vertices
	g := graph.New(graph.StringHash, graph.Directed())

	// Test adding vertices
	err := g.AddVertex("A")
	if err != nil {
		t.Fatalf("Failed to add vertex A: %v", err)
	}

	err = g.AddVertex("B")
	if err != nil {
		t.Fatalf("Failed to add vertex B: %v", err)
	}

	err = g.AddVertex("C")
	if err != nil {
		t.Fatalf("Failed to add vertex C: %v", err)
	}

	// Test adding edges
	err = g.AddEdge("A", "B")
	if err != nil {
		t.Fatalf("Failed to add edge A->B: %v", err)
	}

	err = g.AddEdge("B", "C")
	if err != nil {
		t.Fatalf("Failed to add edge B->C: %v", err)
	}

	// Test retrieving adjacency map to verify graph structure
	adjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		t.Fatalf("Failed to get adjacency map: %v", err)
	}

	// Verify that vertex A has an edge to B
	if _, exists := adjacencyMap["A"]["B"]; !exists {
		t.Error("Expected edge A->B to exist")
	}

	// Verify that vertex B has an edge to C
	if _, exists := adjacencyMap["B"]["C"]; !exists {
		t.Error("Expected edge B->C to exist")
	}

	// Verify vertex count
	expectedVertices := 3
	if len(adjacencyMap) != expectedVertices {
		t.Errorf("Expected %d vertices, got %d", expectedVertices, len(adjacencyMap))
	}
}

// TestGraphCycleDetection verifies that the library's cycle detection works
func TestGraphCycleDetection(t *testing.T) {
	// Create a directed graph
	g := graph.New(graph.StringHash, graph.Directed())

	// Add vertices
	vertices := []string{"A", "B", "C"}
	for _, v := range vertices {
		if err := g.AddVertex(v); err != nil {
			t.Fatalf("Failed to add vertex %s: %v", v, err)
		}
	}

	// Add edges to create a cycle: A -> B -> C -> A
	edges := [][2]string{
		{"A", "B"},
		{"B", "C"},
		{"C", "A"},
	}

	for _, edge := range edges {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatalf("Failed to add edge %s->%s: %v", edge[0], edge[1], err)
		}
	}

	// Test cycle detection
	cycle, err := graph.TopologicalSort(g)
	if err == nil {
		t.Error("Expected topological sort to fail due to cycle, but it succeeded")
	}

	// The cycle should be empty when there's a cycle
	if len(cycle) != 0 {
		t.Errorf("Expected empty cycle result due to cycle, got %v", cycle)
	}
}

// TestGraphTopologicalSort verifies topological sorting on an acyclic graph
func TestGraphTopologicalSort(t *testing.T) {
	// Create a directed acyclic graph
	g := graph.New(graph.StringHash, graph.Directed())

	// Add vertices
	vertices := []string{"A", "B", "C", "D"}
	for _, v := range vertices {
		if err := g.AddVertex(v); err != nil {
			t.Fatalf("Failed to add vertex %s: %v", v, err)
		}
	}

	// Add edges: A -> B, A -> C, B -> D, C -> D
	edges := [][2]string{
		{"A", "B"},
		{"A", "C"},
		{"B", "D"},
		{"C", "D"},
	}

	for _, edge := range edges {
		if err := g.AddEdge(edge[0], edge[1]); err != nil {
			t.Fatalf("Failed to add edge %s->%s: %v", edge[0], edge[1], err)
		}
	}

	// Test topological sort
	sorted, err := graph.TopologicalSort(g)
	if err != nil {
		t.Fatalf("Topological sort failed: %v", err)
	}

	// Verify the sort order is valid (A should come before B and C, D should be last)
	if len(sorted) != 4 {
		t.Errorf("Expected 4 vertices in sorted order, got %d", len(sorted))
	}

	// Find positions
	positions := make(map[string]int)
	for i, vertex := range sorted {
		positions[vertex] = i
	}

	// Verify ordering constraints
	if positions["A"] >= positions["B"] {
		t.Error("A should come before B in topological order")
	}
	if positions["A"] >= positions["C"] {
		t.Error("A should come before C in topological order")
	}
	if positions["B"] >= positions["D"] {
		t.Error("B should come before D in topological order")
	}
	if positions["C"] >= positions["D"] {
		t.Error("C should come before D in topological order")
	}
}
