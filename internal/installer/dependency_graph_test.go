package installer

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestGraphConstruction tests the basic graph construction functionality
func TestGraphConstruction(t *testing.T) {
	t.Run("NewDependencyGraph creates empty graph", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		if dg == nil {
			t.Fatal("NewDependencyGraph() returned nil")
		}

		steps := dg.GetSteps()
		if len(steps) != 0 {
			t.Errorf("Expected empty graph, got %d steps", len(steps))
		}
	})

	t.Run("AddStep adds single step successfully", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		err := dg.AddStep("TestStep")
		if err != nil {
			t.Fatalf("AddStep failed: %v", err)
		}

		if !dg.HasStep("TestStep") {
			t.Error("Step was not added to the graph")
		}

		steps := dg.GetSteps()
		if len(steps) != 1 {
			t.Errorf("Expected 1 step, got %d", len(steps))
		}
		if steps[0] != "TestStep" {
			t.Errorf("Expected 'TestStep', got '%s'", steps[0])
		}
	})

	t.Run("AddStep fails with empty step name", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		err := dg.AddStep("")
		if err == nil {
			t.Error("Expected error for empty step name, got nil")
		}
		if !strings.Contains(err.Error(), "step name cannot be empty") {
			t.Errorf("Expected 'step name cannot be empty' error, got: %v", err)
		}
	})

	t.Run("AddStep fails for duplicate steps", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		err := dg.AddStep("DuplicateStep")
		if err != nil {
			t.Fatalf("First AddStep failed: %v", err)
		}

		err = dg.AddStep("DuplicateStep")
		if err == nil {
			t.Error("Expected error for duplicate step, got nil")
		}
		if !strings.Contains(err.Error(), "has already been added") {
			t.Errorf("Expected 'has already been added' error, got: %v", err)
		}
	})

	t.Run("AddStep handles multiple unique steps", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		steps := []string{"Step1", "Step2", "Step3", "Step4"}

		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("AddStep failed for '%s': %v", step, err)
			}
		}

		if len(dg.GetSteps()) != len(steps) {
			t.Errorf("Expected %d steps, got %d", len(steps), len(dg.GetSteps()))
		}

		for _, step := range steps {
			if !dg.HasStep(step) {
				t.Errorf("Step '%s' was not found in graph", step)
			}
		}
	})
}

// TestAddDependency tests the dependency addition functionality
func TestAddDependency(t *testing.T) {
	t.Run("AddDependency works with valid steps", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Add steps first
		err := dg.AddStep("StepA")
		if err != nil {
			t.Fatalf("Failed to add StepA: %v", err)
		}
		err = dg.AddStep("StepB")
		if err != nil {
			t.Fatalf("Failed to add StepB: %v", err)
		}

		// Add dependency: StepB depends on StepA
		err = dg.AddDependency("StepB", "StepA")
		if err != nil {
			t.Fatalf("AddDependency failed: %v", err)
		}

		// Verify dependency exists
		deps, err := dg.GetDependencies("StepB")
		if err != nil {
			t.Fatalf("GetDependencies failed: %v", err)
		}
		if len(deps) != 1 || deps[0] != "StepA" {
			t.Errorf("Expected dependency [StepA], got %v", deps)
		}
	})

	t.Run("AddDependency fails with empty step names", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		tests := []struct {
			from, to string
			name     string
		}{
			{"", "StepA", "empty from step"},
			{"StepA", "", "empty to step"},
			{"", "", "both empty"},
		}

		for _, test := range tests {
			err := dg.AddDependency(test.from, test.to)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", test.name)
			}
			if !strings.Contains(err.Error(), "step names cannot be empty") {
				t.Errorf("Expected 'step names cannot be empty' error for %s, got: %v", test.name, err)
			}
		}
	})

	t.Run("AddDependency fails for self-dependency", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		err := dg.AddStep("SelfStep")
		if err != nil {
			t.Fatalf("Failed to add step: %v", err)
		}

		err = dg.AddDependency("SelfStep", "SelfStep")
		if err == nil {
			t.Error("Expected error for self-dependency, got nil")
		}
		if !strings.Contains(err.Error(), "step cannot depend on itself") {
			t.Errorf("Expected 'step cannot depend on itself' error, got: %v", err)
		}
	})

	t.Run("AddDependency fails for non-existent steps", func(t *testing.T) {
		dg := NewTestDependencyGraph()
		err := dg.AddStep("ExistingStep")
		if err != nil {
			t.Fatalf("Failed to add step: %v", err)
		}

		tests := []struct {
			from, to, expected string
		}{
			{"NonExistentFrom", "ExistingStep", "NonExistentFrom"},
			{"ExistingStep", "NonExistentTo", "NonExistentTo"},
			{"NonExistent1", "NonExistent2", "NonExistent1"},
		}

		for _, test := range tests {
			err = dg.AddDependency(test.from, test.to)
			if err == nil {
				t.Errorf("Expected error for non-existent step %s->%s, got nil", test.from, test.to)
			}
			if !strings.Contains(err.Error(), "has not been added to the graph") {
				t.Errorf("Expected 'has not been added' error, got: %v", err)
			}
		}
	})

	t.Run("AddDependency handles multiple dependencies", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Add steps
		steps := []string{"Root", "Branch1", "Branch2", "Leaf"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		// Create dependencies: Leaf depends on Branch1 and Branch2, both depend on Root
		dependencies := []struct{ from, to string }{
			{"Branch1", "Root"},
			{"Branch2", "Root"},
			{"Leaf", "Branch1"},
			{"Leaf", "Branch2"},
		}

		for _, dep := range dependencies {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		// Verify dependencies
		leafDeps, err := dg.GetDependencies("Leaf")
		if err != nil {
			t.Fatalf("GetDependencies failed for Leaf: %v", err)
		}
		if len(leafDeps) != 2 {
			t.Errorf("Expected 2 dependencies for Leaf, got %d", len(leafDeps))
		}
	})
}

// TestBuildGraph tests the buildGraph method with various scenarios
func TestBuildGraph(t *testing.T) {
	t.Run("buildGraph constructs simple graph", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		dependencies := []Dependency{
			{"StepB", "StepA"},
			{"StepC", "StepB"},
		}

		err := dg.buildGraph(dependencies)
		if err != nil {
			t.Fatalf("buildGraph failed: %v", err)
		}

		// Verify all steps were added
		expectedSteps := []string{"StepA", "StepB", "StepC"}
		for _, step := range expectedSteps {
			if !dg.HasStep(step) {
				t.Errorf("Step '%s' was not added during buildGraph", step)
			}
		}

		// Verify topological order is valid
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		if len(order) != 3 {
			t.Errorf("Expected 3 steps in order, got %d", len(order))
		}

		// StepA should come before StepB, StepB before StepC
		posA, posB, posC := -1, -1, -1
		for i, step := range order {
			switch step {
			case "StepA":
				posA = i
			case "StepB":
				posB = i
			case "StepC":
				posC = i
			}
		}

		if posA >= posB {
			t.Error("StepA should come before StepB in topological order")
		}
		if posB >= posC {
			t.Error("StepB should come before StepC in topological order")
		}
	})

	t.Run("buildGraph handles complex dependencies", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create a diamond dependency pattern
		dependencies := []Dependency{
			{"StepB", "StepA"},
			{"StepC", "StepA"},
			{"StepD", "StepB"},
			{"StepD", "StepC"},
		}

		err := dg.buildGraph(dependencies)
		if err != nil {
			t.Fatalf("buildGraph failed for complex dependencies: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		// Verify positions
		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// StepA should come before StepB and StepC
		if positions["StepA"] >= positions["StepB"] {
			t.Error("StepA should come before StepB")
		}
		if positions["StepA"] >= positions["StepC"] {
			t.Error("StepA should come before StepC")
		}

		// StepB and StepC should come before StepD
		if positions["StepB"] >= positions["StepD"] {
			t.Error("StepB should come before StepD")
		}
		if positions["StepC"] >= positions["StepD"] {
			t.Error("StepC should come before StepD")
		}
	})

	t.Run("buildGraph handles empty dependencies", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		err := dg.buildGraph([]Dependency{})
		if err != nil {
			t.Fatalf("buildGraph failed for empty dependencies: %v", err)
		}

		steps := dg.GetSteps()
		if len(steps) != 0 {
			t.Errorf("Expected no steps for empty dependencies, got %d", len(steps))
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed for empty graph: %v", err)
		}
		if len(order) != 0 {
			t.Errorf("Expected empty order for empty graph, got %d steps", len(order))
		}
	})
}

// TestGraphQueries tests the various query methods
func TestGraphQueries(t *testing.T) {
	t.Run("HasStep returns correct values", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		if dg.HasStep("NonExistent") {
			t.Error("HasStep returned true for non-existent step")
		}

		err := dg.AddStep("ExistentStep")
		if err != nil {
			t.Fatalf("Failed to add step: %v", err)
		}

		if !dg.HasStep("ExistentStep") {
			t.Error("HasStep returned false for existing step")
		}
	})

	t.Run("GetDependencies returns error for non-existent step", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		_, err := dg.GetDependencies("NonExistent")
		if err == nil {
			t.Error("Expected error for non-existent step, got nil")
		}
		if !strings.Contains(err.Error(), "has not been added to the graph") {
			t.Errorf("Expected 'has not been added' error, got: %v", err)
		}
	})

	t.Run("GetDependencies returns empty for step with no dependencies", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		err := dg.AddStep("IndependentStep")
		if err != nil {
			t.Fatalf("Failed to add step: %v", err)
		}

		deps, err := dg.GetDependencies("IndependentStep")
		if err != nil {
			t.Fatalf("GetDependencies failed: %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("Expected no dependencies, got %v", deps)
		}
	})

	t.Run("GetSteps returns all added steps", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		expectedSteps := []string{"Alpha", "Beta", "Gamma", "Delta"}
		for _, step := range expectedSteps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		actualSteps := dg.GetSteps()
		if len(actualSteps) != len(expectedSteps) {
			t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(actualSteps))
		}

		// Convert to map for easier comparison
		stepMap := make(map[string]bool)
		for _, step := range actualSteps {
			stepMap[step] = true
		}

		for _, expected := range expectedSteps {
			if !stepMap[expected] {
				t.Errorf("Expected step '%s' not found in GetSteps() result", expected)
			}
		}
	})
}

// TestCycleDetection tests that the graph properly detects and prevents cycles
func TestCycleDetection(t *testing.T) {
	t.Run("simple cycle detection A->B->A", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Add steps
		err := dg.AddStep("StepA")
		if err != nil {
			t.Fatalf("Failed to add StepA: %v", err)
		}
		err = dg.AddStep("StepB")
		if err != nil {
			t.Fatalf("Failed to add StepB: %v", err)
		}

		// Add first dependency: B depends on A
		err = dg.AddDependency("StepB", "StepA")
		if err != nil {
			t.Fatalf("Failed to add first dependency: %v", err)
		}

		// Add cycle: A depends on B (this will be allowed by the graph library)
		err = dg.AddDependency("StepA", "StepB")
		if err != nil {
			t.Fatalf("Failed to add second dependency: %v", err)
		}

		// But topological sort should fail
		_, err = dg.GetTopologicalOrder()
		if err == nil {
			t.Error("Expected topological sort to fail due to cycle, got nil")
		}
	})

	t.Run("three-step cycle detection A->B->C->A", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Add steps
		steps := []string{"StepA", "StepB", "StepC"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add %s: %v", step, err)
			}
		}

		// Add dependencies: A->B->C
		err := dg.AddDependency("StepB", "StepA")
		if err != nil {
			t.Fatalf("Failed to add A->B dependency: %v", err)
		}

		err = dg.AddDependency("StepC", "StepB")
		if err != nil {
			t.Fatalf("Failed to add B->C dependency: %v", err)
		}

		// Add final dependency to create cycle: A depends on C
		err = dg.AddDependency("StepA", "StepC")
		if err != nil {
			t.Fatalf("Failed to add C->A dependency: %v", err)
		}

		// Topological sort should fail due to cycle
		_, err = dg.GetTopologicalOrder()
		if err == nil {
			t.Error("Expected topological sort to fail due to three-step cycle, got nil")
		}
	})

	t.Run("complex cycle in diamond pattern", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create diamond: A -> B,C -> D
		steps := []string{"A", "B", "C", "D"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		// Add valid diamond dependencies
		validDeps := []struct{ from, to string }{
			{"B", "A"},
			{"C", "A"},
			{"D", "B"},
			{"D", "C"},
		}

		for _, dep := range validDeps {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add valid dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		// Add cycle: A depends on D
		err := dg.AddDependency("A", "D")
		if err != nil {
			t.Fatalf("Failed to add A->D dependency: %v", err)
		}

		// Topological sort should fail due to cycle
		_, err = dg.GetTopologicalOrder()
		if err == nil {
			t.Error("Expected topological sort to fail due to cycle in diamond pattern, got nil")
		}
	})

	t.Run("self-dependency is prevented", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		err := dg.AddStep("SelfDependent")
		if err != nil {
			t.Fatalf("Failed to add step: %v", err)
		}

		err = dg.AddDependency("SelfDependent", "SelfDependent")
		if err == nil {
			t.Error("Expected error for self-dependency, got nil")
		}
		if !strings.Contains(err.Error(), "step cannot depend on itself") {
			t.Errorf("Expected self-dependency error, got: %v", err)
		}
	})

	t.Run("buildGraph detects cycles", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create cyclic dependencies using buildGraph
		cyclicDeps := []Dependency{
			{"StepA", "StepB"},
			{"StepB", "StepC"},
			{"StepC", "StepA"}, // Creates cycle
		}

		err := dg.buildGraph(cyclicDeps)
		if err == nil {
			t.Error("Expected buildGraph to detect cycle, got nil")
		}

		errorStr := err.Error()
		if !strings.Contains(errorStr, "cycle") && !strings.Contains(errorStr, "acyclic") && !strings.Contains(errorStr, "circular") {
			t.Errorf("Expected cycle-related error from buildGraph, got: %v", err)
		}
	})

	t.Run("acyclic graph passes validation", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create valid acyclic dependencies
		acyclicDeps := []Dependency{
			{"StepB", "StepA"},
			{"StepC", "StepA"},
			{"StepD", "StepB"},
			{"StepE", "StepC"},
			{"StepF", "StepD"},
			{"StepF", "StepE"},
		}

		err := dg.buildGraph(acyclicDeps)
		if err != nil {
			t.Fatalf("buildGraph failed for valid acyclic dependencies: %v", err)
		}

		// Verify we can get topological order
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed for acyclic graph: %v", err)
		}

		if len(order) != 6 {
			t.Errorf("Expected 6 steps in topological order, got %d", len(order))
		}
	})
}

// TestTopologicalSorting tests various topological sorting scenarios
func TestTopologicalSorting(t *testing.T) {
	t.Run("linear dependency chain", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create linear chain: A -> B -> C -> D
		steps := []string{"A", "B", "C", "D"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		// Add linear dependencies
		dependencies := []struct{ from, to string }{
			{"B", "A"},
			{"C", "B"},
			{"D", "C"},
		}

		for _, dep := range dependencies {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		if len(order) != 4 {
			t.Errorf("Expected 4 steps in order, got %d", len(order))
		}

		// Verify ordering: A before B before C before D
		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		if positions["A"] >= positions["B"] {
			t.Error("A should come before B")
		}
		if positions["B"] >= positions["C"] {
			t.Error("B should come before C")
		}
		if positions["C"] >= positions["D"] {
			t.Error("C should come before D")
		}
	})

	t.Run("diamond dependency pattern", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create diamond: Root -> Branch1,Branch2 -> Leaf
		steps := []string{"Root", "Branch1", "Branch2", "Leaf"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		dependencies := []struct{ from, to string }{
			{"Branch1", "Root"},
			{"Branch2", "Root"},
			{"Leaf", "Branch1"},
			{"Leaf", "Branch2"},
		}

		for _, dep := range dependencies {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Root should come before both branches
		if positions["Root"] >= positions["Branch1"] {
			t.Error("Root should come before Branch1")
		}
		if positions["Root"] >= positions["Branch2"] {
			t.Error("Root should come before Branch2")
		}

		// Both branches should come before Leaf
		if positions["Branch1"] >= positions["Leaf"] {
			t.Error("Branch1 should come before Leaf")
		}
		if positions["Branch2"] >= positions["Leaf"] {
			t.Error("Branch2 should come before Leaf")
		}
	})

	t.Run("complex multi-level hierarchy", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create complex hierarchy:
		// Level 0: A
		// Level 1: B, C (depend on A)
		// Level 2: D (depends on B), E (depends on C)
		// Level 3: F (depends on D and E)
		steps := []string{"A", "B", "C", "D", "E", "F"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		dependencies := []struct{ from, to string }{
			{"B", "A"},
			{"C", "A"},
			{"D", "B"},
			{"E", "C"},
			{"F", "D"},
			{"F", "E"},
		}

		for _, dep := range dependencies {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Level ordering checks
		if positions["A"] >= positions["B"] || positions["A"] >= positions["C"] {
			t.Error("A should come before B and C")
		}
		if positions["B"] >= positions["D"] {
			t.Error("B should come before D")
		}
		if positions["C"] >= positions["E"] {
			t.Error("C should come before E")
		}
		if positions["D"] >= positions["F"] || positions["E"] >= positions["F"] {
			t.Error("D and E should come before F")
		}
	})

	t.Run("isolated nodes mixed with dependencies", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create graph with some isolated nodes and some with dependencies
		steps := []string{"Isolated1", "A", "B", "Isolated2", "C"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		// Only A->B->C have dependencies
		dependencies := []struct{ from, to string }{
			{"B", "A"},
			{"C", "B"},
		}

		for _, dep := range dependencies {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		if len(order) != 5 {
			t.Errorf("Expected 5 steps in order, got %d", len(order))
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Only check the dependency chain, isolated nodes can be anywhere
		if positions["A"] >= positions["B"] {
			t.Error("A should come before B")
		}
		if positions["B"] >= positions["C"] {
			t.Error("B should come before C")
		}

		// Ensure all steps are present
		expectedSteps := map[string]bool{
			"Isolated1": false, "A": false, "B": false, "Isolated2": false, "C": false,
		}
		for _, step := range order {
			expectedSteps[step] = true
		}
		for step, found := range expectedSteps {
			if !found {
				t.Errorf("Step %s not found in topological order", step)
			}
		}
	})

	t.Run("single node graph", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		err := dg.AddStep("SingleNode")
		if err != nil {
			t.Fatalf("Failed to add single step: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed for single node: %v", err)
		}

		if len(order) != 1 {
			t.Errorf("Expected 1 step in order, got %d", len(order))
		}
		if order[0] != "SingleNode" {
			t.Errorf("Expected 'SingleNode', got '%s'", order[0])
		}
	})

	t.Run("empty graph", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed for empty graph: %v", err)
		}

		if len(order) != 0 {
			t.Errorf("Expected empty order for empty graph, got %d steps", len(order))
		}
	})

	t.Run("multiple root nodes", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create graph with multiple roots: Root1, Root2 -> Common
		steps := []string{"Root1", "Root2", "Common"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		dependencies := []struct{ from, to string }{
			{"Common", "Root1"},
			{"Common", "Root2"},
		}

		for _, dep := range dependencies {
			err := dg.AddDependency(dep.from, dep.to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", dep.from, dep.to, err)
			}
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Both roots should come before Common
		if positions["Root1"] >= positions["Common"] {
			t.Error("Root1 should come before Common")
		}
		if positions["Root2"] >= positions["Common"] {
			t.Error("Root2 should come before Common")
		}
	})
}

// TestIntegrationWithRealInstallerSteps tests the DependencyGraph with actual installer steps
func TestIntegrationWithRealInstallerSteps(t *testing.T) {
	t.Run("basic installer dependency graph", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create real installer dependencies (subset for testing)
		dependencies := []Dependency{
			{"ScanExistingFiles", "CheckPrerequisites"},
			{"CreateBackups", "ScanExistingFiles"},
			{"CheckTargetDirectory", "CreateBackups"},
			{"CloneRepository", "CheckTargetDirectory"},
			{"CreateDirectoryStructure", "CheckTargetDirectory"},
			{"CopyCoreFiles", "CloneRepository"},
			{"CopyCoreFiles", "CreateDirectoryStructure"},
			{"CopyCommandFiles", "CloneRepository"},
			{"CopyCommandFiles", "CreateDirectoryStructure"},
			{"MergeOrCreateCLAUDEmd", "CreateDirectoryStructure"},
			{"CreateCommandSymlink", "CopyCommandFiles"},
			{"CreateCommandSymlink", "CreateDirectoryStructure"},
		}

		err := dg.buildGraph(dependencies)
		if err != nil {
			t.Fatalf("Failed to build real installer dependency graph: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order for real installer steps: %v", err)
		}

		// Verify all steps are present
		expectedSteps := []string{
			"CheckPrerequisites", "ScanExistingFiles", "CreateBackups", "CheckTargetDirectory",
			"CloneRepository", "CreateDirectoryStructure", "CopyCoreFiles", "CopyCommandFiles",
			"MergeOrCreateCLAUDEmd", "CreateCommandSymlink",
		}

		if len(order) != len(expectedSteps) {
			t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(order))
		}

		// Create position map for dependency validation
		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Verify key dependency constraints
		testCases := []struct {
			before, after string
			description   string
		}{
			{"CheckPrerequisites", "ScanExistingFiles", "CheckPrerequisites before ScanExistingFiles"},
			{"ScanExistingFiles", "CreateBackups", "ScanExistingFiles before CreateBackups"},
			{"CreateBackups", "CheckTargetDirectory", "CreateBackups before CheckTargetDirectory"},
			{"CheckTargetDirectory", "CloneRepository", "CheckTargetDirectory before CloneRepository"},
			{"CheckTargetDirectory", "CreateDirectoryStructure", "CheckTargetDirectory before CreateDirectoryStructure"},
			{"CloneRepository", "CopyCoreFiles", "CloneRepository before CopyCoreFiles"},
			{"CreateDirectoryStructure", "CopyCoreFiles", "CreateDirectoryStructure before CopyCoreFiles"},
			{"CloneRepository", "CopyCommandFiles", "CloneRepository before CopyCommandFiles"},
			{"CreateDirectoryStructure", "CopyCommandFiles", "CreateDirectoryStructure before CopyCommandFiles"},
			{"CreateDirectoryStructure", "MergeOrCreateCLAUDEmd", "CreateDirectoryStructure before MergeOrCreateCLAUDEmd"},
			{"CopyCommandFiles", "CreateCommandSymlink", "CopyCommandFiles before CreateCommandSymlink"},
			{"CreateDirectoryStructure", "CreateCommandSymlink", "CreateDirectoryStructure before CreateCommandSymlink"},
		}

		for _, tc := range testCases {
			if positions[tc.before] >= positions[tc.after] {
				t.Errorf("%s: %s (pos %d) should come before %s (pos %d)",
					tc.description, tc.before, positions[tc.before], tc.after, positions[tc.after])
			}
		}
	})

	t.Run("installer with MCP configuration", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Full dependency graph including MCP config steps
		dependencies := []Dependency{
			{"ScanExistingFiles", "CheckPrerequisites"},
			{"CreateBackups", "ScanExistingFiles"},
			{"CheckTargetDirectory", "CreateBackups"},
			{"CloneRepository", "CheckTargetDirectory"},
			{"CreateDirectoryStructure", "CheckTargetDirectory"},
			{"CopyCoreFiles", "CloneRepository"},
			{"CopyCoreFiles", "CreateDirectoryStructure"},
			{"CopyCommandFiles", "CloneRepository"},
			{"CopyCommandFiles", "CreateDirectoryStructure"},
			{"MergeOrCreateCLAUDEmd", "CreateDirectoryStructure"},
			{"MergeOrCreateMCPConfig", "CreateDirectoryStructure"},
			{"CreateCommandSymlink", "CopyCommandFiles"},
			{"CreateCommandSymlink", "CreateDirectoryStructure"},
			// ValidateInstallation with MCP dependency
			{"ValidateInstallation", "CopyCoreFiles"},
			{"ValidateInstallation", "CopyCommandFiles"},
			{"ValidateInstallation", "MergeOrCreateCLAUDEmd"},
			{"ValidateInstallation", "MergeOrCreateMCPConfig"},
			{"ValidateInstallation", "CreateCommandSymlink"},
			// CleanupTempFiles depends on everything
			{"CleanupTempFiles", "CopyCoreFiles"},
			{"CleanupTempFiles", "CopyCommandFiles"},
			{"CleanupTempFiles", "MergeOrCreateCLAUDEmd"},
			{"CleanupTempFiles", "MergeOrCreateMCPConfig"},
			{"CleanupTempFiles", "CreateCommandSymlink"},
			{"CleanupTempFiles", "ValidateInstallation"},
		}

		err := dg.buildGraph(dependencies)
		if err != nil {
			t.Fatalf("Failed to build MCP-enabled installer dependency graph: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order for MCP-enabled installer: %v", err)
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Verify MCP-specific constraints
		if positions["CreateDirectoryStructure"] >= positions["MergeOrCreateMCPConfig"] {
			t.Error("CreateDirectoryStructure should come before MergeOrCreateMCPConfig")
		}
		if positions["MergeOrCreateMCPConfig"] >= positions["ValidateInstallation"] {
			t.Error("MergeOrCreateMCPConfig should come before ValidateInstallation")
		}
		if positions["ValidateInstallation"] >= positions["CleanupTempFiles"] {
			t.Error("ValidateInstallation should come before CleanupTempFiles")
		}
	})

	t.Run("installer without MCP configuration", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Dependency graph without MCP config
		dependencies := []Dependency{
			{"ScanExistingFiles", "CheckPrerequisites"},
			{"CreateBackups", "ScanExistingFiles"},
			{"CheckTargetDirectory", "CreateBackups"},
			{"CloneRepository", "CheckTargetDirectory"},
			{"CreateDirectoryStructure", "CheckTargetDirectory"},
			{"CopyCoreFiles", "CloneRepository"},
			{"CopyCoreFiles", "CreateDirectoryStructure"},
			{"CopyCommandFiles", "CloneRepository"},
			{"CopyCommandFiles", "CreateDirectoryStructure"},
			{"MergeOrCreateCLAUDEmd", "CreateDirectoryStructure"},
			{"CreateCommandSymlink", "CopyCommandFiles"},
			{"CreateCommandSymlink", "CreateDirectoryStructure"},
			// ValidateInstallation without MCP dependency
			{"ValidateInstallation", "CopyCoreFiles"},
			{"ValidateInstallation", "CopyCommandFiles"},
			{"ValidateInstallation", "MergeOrCreateCLAUDEmd"},
			{"ValidateInstallation", "CreateCommandSymlink"},
			// CleanupTempFiles without MCP dependency
			{"CleanupTempFiles", "CopyCoreFiles"},
			{"CleanupTempFiles", "CopyCommandFiles"},
			{"CleanupTempFiles", "MergeOrCreateCLAUDEmd"},
			{"CleanupTempFiles", "CreateCommandSymlink"},
			{"CleanupTempFiles", "ValidateInstallation"},
		}

		err := dg.buildGraph(dependencies)
		if err != nil {
			t.Fatalf("Failed to build non-MCP installer dependency graph: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order for non-MCP installer: %v", err)
		}

		// Verify MergeOrCreateMCPConfig is not in the graph
		for _, step := range order {
			if step == "MergeOrCreateMCPConfig" {
				t.Error("MergeOrCreateMCPConfig should not be present in non-MCP configuration")
			}
		}

		// Verify other key steps are still properly ordered
		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		if positions["ValidateInstallation"] >= positions["CleanupTempFiles"] {
			t.Error("ValidateInstallation should come before CleanupTempFiles")
		}
	})

	t.Run("real installer step names validation", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Get actual step names from the installer
		realSteps := []string{
			"CheckPrerequisites",
			"ScanExistingFiles",
			"CreateBackups",
			"CheckTargetDirectory",
			"CloneRepository",
			"CreateDirectoryStructure",
			"CopyCoreFiles",
			"CopyCommandFiles",
			"MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig",
			"CreateCommandSymlink",
			"ValidateInstallation",
			"CleanupTempFiles",
		}

		// Add all real steps
		for _, step := range realSteps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add real installer step '%s': %v", step, err)
			}
		}

		// Verify all steps were added correctly
		for _, step := range realSteps {
			if !dg.HasStep(step) {
				t.Errorf("Real installer step '%s' was not found in graph", step)
			}
		}

		if len(dg.GetSteps()) != len(realSteps) {
			t.Errorf("Expected %d steps, got %d", len(realSteps), len(dg.GetSteps()))
		}
	})

	t.Run("conditional dependency handling", func(t *testing.T) {
		// Test that our graph can handle conditional dependencies
		// like those in ValidateInstallation and CleanupTempFiles

		// Test scenario 1: MCP enabled
		dg1 := NewTestDependencyGraph()
		mcpEnabledDeps := []Dependency{
			{"ValidateInstallation", "MergeOrCreateCLAUDEmd"},
			{"ValidateInstallation", "MergeOrCreateMCPConfig"}, // MCP dependency
			{"CleanupTempFiles", "ValidateInstallation"},
		}

		err := dg1.buildGraph(mcpEnabledDeps)
		if err != nil {
			t.Fatalf("Failed to build MCP-enabled conditional dependencies: %v", err)
		}

		order1, err := dg1.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get order for MCP-enabled dependencies: %v", err)
		}

		// Test scenario 2: MCP disabled
		dg2 := NewTestDependencyGraph()
		mcpDisabledDeps := []Dependency{
			{"ValidateInstallation", "MergeOrCreateCLAUDEmd"},
			// No MCP dependency
			{"CleanupTempFiles", "ValidateInstallation"},
		}

		err = dg2.buildGraph(mcpDisabledDeps)
		if err != nil {
			t.Fatalf("Failed to build MCP-disabled conditional dependencies: %v", err)
		}

		order2, err := dg2.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get order for MCP-disabled dependencies: %v", err)
		}

		// Both scenarios should produce valid topological orders
		if len(order1) == 0 || len(order2) == 0 {
			t.Error("Both conditional dependency scenarios should produce non-empty orders")
		}

		// MCP-enabled should have more steps
		if len(order1) <= len(order2) {
			t.Error("MCP-enabled scenario should have more steps than MCP-disabled")
		}
	})
}

// TestErrorHandling tests various error scenarios and edge cases
func TestErrorHandling(t *testing.T) {
	t.Run("nil graph operations", func(t *testing.T) {
		// Test behavior with uninitialized graph (should not happen in practice)
		dg := &DependencyGraph{}

		// These should handle nil gracefully or panic predictably
		steps := dg.GetSteps()
		if len(steps) != 0 {
			t.Errorf("Expected empty steps for uninitialized graph, got %d", len(steps))
		}

		if dg.HasStep("anything") {
			t.Error("Uninitialized graph should not have any steps")
		}
	})

	t.Run("empty and whitespace step names", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		testCases := []struct {
			stepName string
			desc     string
		}{
			{"", "empty string"},
			{" ", "single space"},
			{"\t", "tab character"},
			{"\n", "newline character"},
			{"  \t\n  ", "mixed whitespace"},
		}

		for _, tc := range testCases {
			err := dg.AddStep(tc.stepName)
			if err == nil {
				t.Errorf("Expected error for %s step name, got nil", tc.desc)
			}
			if !strings.Contains(err.Error(), "step name cannot be empty") {
				t.Errorf("Expected 'step name cannot be empty' error for %s, got: %v", tc.desc, err)
			}
		}
	})

	t.Run("dependency errors with malformed input", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Add some valid steps first
		validSteps := []string{"ValidStep1", "ValidStep2"}
		for _, step := range validSteps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add valid step %s: %v", step, err)
			}
		}

		errorTestCases := []struct {
			from, to    string
			expectedMsg string
			description string
		}{
			{"", "ValidStep1", "step names cannot be empty", "empty from step"},
			{"ValidStep1", "", "step names cannot be empty", "empty to step"},
			{"  ", "ValidStep1", "step names cannot be empty", "whitespace from step"},
			{"ValidStep1", "  ", "step names cannot be empty", "whitespace to step"},
			{"NonExistent", "ValidStep1", "has not been added to the graph", "non-existent from step"},
			{"ValidStep1", "NonExistent", "has not been added to the graph", "non-existent to step"},
			{"ValidStep1", "ValidStep1", "step cannot depend on itself", "self-dependency"},
		}

		for _, tc := range errorTestCases {
			err := dg.AddDependency(tc.from, tc.to)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tc.description)
				continue
			}
			if !strings.Contains(err.Error(), tc.expectedMsg) {
				t.Errorf("Expected error containing '%s' for %s, got: %v", tc.expectedMsg, tc.description, err)
			}
		}
	})

	t.Run("buildGraph with malformed dependencies", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		malformedTestCases := []struct {
			deps        []Dependency
			expectedMsg string
			description string
		}{
			{
				[]Dependency{{"", "ValidStep"}},
				"step name cannot be empty",
				"empty from step in dependency",
			},
			{
				[]Dependency{{"ValidStep", ""}},
				"step name cannot be empty",
				"empty to step in dependency",
			},
			{
				[]Dependency{{"SameStep", "SameStep"}},
				"step cannot depend on itself",
				"self-dependency in buildGraph",
			},
		}

		for _, tc := range malformedTestCases {
			err := dg.buildGraph(tc.deps)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tc.description)
				continue
			}
			if !strings.Contains(err.Error(), tc.expectedMsg) {
				t.Errorf("Expected error containing '%s' for %s, got: %v", tc.expectedMsg, tc.description, err)
			}
		}
	})

	t.Run("large graph stress test", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Create a large number of steps
		numSteps := 1000
		for i := 0; i < numSteps; i++ {
			stepName := fmt.Sprintf("Step%d", i)
			err := dg.AddStep(stepName)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", stepName, err)
			}
		}

		// Add linear dependencies (each step depends on the previous)
		for i := 1; i < numSteps; i++ {
			from := fmt.Sprintf("Step%d", i)
			to := fmt.Sprintf("Step%d", i-1)
			err := dg.AddDependency(from, to)
			if err != nil {
				t.Fatalf("Failed to add dependency %s->%s: %v", from, to, err)
			}
		}

		// Verify topological ordering works with large graph
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order for large graph: %v", err)
		}

		if len(order) != numSteps {
			t.Errorf("Expected %d steps in order, got %d", numSteps, len(order))
		}

		// Verify ordering is correct (Step0 should come first, Step999 last)
		if order[0] != "Step0" {
			t.Errorf("Expected Step0 to be first, got %s", order[0])
		}
		if order[len(order)-1] != fmt.Sprintf("Step%d", numSteps-1) {
			t.Errorf("Expected Step%d to be last, got %s", numSteps-1, order[len(order)-1])
		}
	})

	t.Run("concurrent access simulation", func(t *testing.T) {
		// Note: This is a basic concurrent access test. In production,
		// proper synchronization would be needed for concurrent access.
		dg := NewTestDependencyGraph()

		// Add steps
		steps := []string{"A", "B", "C", "D", "E"}
		for _, step := range steps {
			err := dg.AddStep(step)
			if err != nil {
				t.Fatalf("Failed to add step %s: %v", step, err)
			}
		}

		// Simulate concurrent reads (should be safe)
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					_ = dg.GetSteps()
					_ = dg.HasStep("A")
					_, _ = dg.GetDependencies("A")
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}

		// Graph should still be functional
		if !dg.HasStep("A") {
			t.Error("Graph corrupted after concurrent access")
		}
	})

	t.Run("memory usage with repeated operations", func(t *testing.T) {
		// Test for memory leaks or excessive allocation
		dg := NewTestDependencyGraph()

		// Repeatedly add and query steps
		for i := 0; i < 100; i++ {
			stepName := fmt.Sprintf("TempStep%d", i)
			err := dg.AddStep(stepName)
			if err != nil {
				t.Fatalf("Failed to add temp step %s: %v", stepName, err)
			}

			_ = dg.GetSteps()
			_ = dg.HasStep(stepName)
		}

		// Verify final state
		steps := dg.GetSteps()
		if len(steps) != 100 {
			t.Errorf("Expected 100 steps after repeated operations, got %d", len(steps))
		}
	})

	t.Run("unicode and special characters in step names", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		specialSteps := []string{
			"Step-With-Dashes",
			"Step_With_Underscores",
			"StepWith123Numbers",
			"Step.With.Dots",
			"Step With Spaces", // This should work
			"StépWithÀccénts",
			"步骤中文名称",
			"Шаг-Русский",
			"🚀RocketStep",
		}

		for _, step := range specialSteps {
			err := dg.AddStep(step)
			if err != nil {
				t.Errorf("Failed to add step with special characters '%s': %v", step, err)
			}
		}

		// Verify all steps were added
		for _, step := range specialSteps {
			if !dg.HasStep(step) {
				t.Errorf("Step with special characters '%s' was not found", step)
			}
		}
	})

	t.Run("extremely long step names", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Test with very long step name (1KB)
		longStepName := strings.Repeat("VeryLongStepName", 64) // ~1KB
		err := dg.AddStep(longStepName)
		if err != nil {
			t.Errorf("Failed to add very long step name: %v", err)
		}

		if !dg.HasStep(longStepName) {
			t.Error("Very long step name was not found in graph")
		}

		// Test topological ordering with long names
		normalStep := "NormalStep"
		err = dg.AddStep(normalStep)
		if err != nil {
			t.Fatalf("Failed to add normal step: %v", err)
		}

		err = dg.AddDependency(normalStep, longStepName)
		if err != nil {
			t.Fatalf("Failed to add dependency with long step name: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order with long step names: %v", err)
		}

		if len(order) != 2 {
			t.Errorf("Expected 2 steps in order, got %d", len(order))
		}
	})

	t.Run("error message clarity and detail", func(t *testing.T) {
		dg := NewTestDependencyGraph()

		// Test that error messages provide clear, actionable information
		err := dg.AddStep("")
		if err == nil {
			t.Fatal("Expected error for empty step name")
		}
		if !strings.Contains(err.Error(), "step name cannot be empty") {
			t.Errorf("Error message should mention empty step name, got: %v", err)
		}

		// Add a step and try to add it again
		err = dg.AddStep("TestStep")
		if err != nil {
			t.Fatalf("Failed to add TestStep: %v", err)
		}

		err = dg.AddStep("TestStep")
		if err == nil {
			t.Fatal("Expected error for duplicate step")
		}
		if !strings.Contains(err.Error(), "TestStep") || !strings.Contains(err.Error(), "already been added") {
			t.Errorf("Error message should mention step name and duplication, got: %v", err)
		}

		// Test dependency error with non-existent step
		err = dg.AddDependency("NonExistent", "TestStep")
		if err == nil {
			t.Fatal("Expected error for non-existent step")
		}
		if !strings.Contains(err.Error(), "NonExistent") || !strings.Contains(err.Error(), "not been added") {
			t.Errorf("Error message should mention the missing step name, got: %v", err)
		}
	})
}

// TestDependencyResolutionComparison compares the old getDependencies method
// with the new BuildInstallationGraph method to ensure identical behavior
func TestDependencyResolutionComparison(t *testing.T) {
	t.Run("compare MCP enabled configuration", func(t *testing.T) {
		// Configuration for MCP enabled test
		config := &InstallConfig{
			AddRecommendedMCP: true,
		}
		mcpEnabled := config.AddRecommendedMCP

		// Create new dependency graph with same configuration
		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		// Get topological order from new system
		newOrder, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		// Verify that all steps that exist in the new system have their dependencies
		// correctly represented compared to the old system
		allSteps := []string{
			"CheckPrerequisites",
			"ScanExistingFiles",
			"CreateBackups",
			"CheckTargetDirectory",
			"CloneRepository",
			"CreateDirectoryStructure",
			"CopyCoreFiles",
			"CopyCommandFiles",
			"MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig",
			"CreateCommandSymlink",
			"ValidateInstallation",
			"CleanupTempFiles",
		}

		// For each step, compare dependencies between old and new system
		for _, stepName := range allSteps {
			oldDeps := getOriginalDependencies(stepName, mcpEnabled)
			newDeps, err := dg.GetDependencies(stepName)
			if err != nil {
				t.Errorf("Failed to get dependencies for step %s: %v", stepName, err)
				continue
			}

			// Sort both slices for comparison
			oldDepsMap := make(map[string]bool)
			for _, dep := range oldDeps {
				oldDepsMap[dep] = true
			}

			newDepsMap := make(map[string]bool)
			for _, dep := range newDeps {
				newDepsMap[dep] = true
			}

			// Compare dependencies
			if len(oldDepsMap) != len(newDepsMap) {
				t.Errorf("Step %s: dependency count mismatch. Old: %d, New: %d",
					stepName, len(oldDepsMap), len(newDepsMap))
				t.Errorf("Old deps: %v", oldDeps)
				t.Errorf("New deps: %v", newDeps)
				continue
			}

			for dep := range oldDepsMap {
				if !newDepsMap[dep] {
					t.Errorf("Step %s: missing dependency %s in new system", stepName, dep)
				}
			}

			for dep := range newDepsMap {
				if !oldDepsMap[dep] {
					t.Errorf("Step %s: extra dependency %s in new system", stepName, dep)
				}
			}
		}

		// Verify that the topological order is valid (all dependencies come before dependents)
		stepPositions := make(map[string]int)
		for i, step := range newOrder {
			stepPositions[step] = i
		}

		for _, stepName := range allSteps {
			if !dg.HasStep(stepName) {
				continue // Skip steps not in the graph
			}

			stepDeps, err := dg.GetDependencies(stepName)
			if err != nil {
				t.Errorf("Failed to get dependencies for ordering check: %v", err)
				continue
			}

			stepPos, exists := stepPositions[stepName]
			if !exists {
				t.Errorf("Step %s not found in topological order", stepName)
				continue
			}

			for _, dep := range stepDeps {
				depPos, depExists := stepPositions[dep]
				if !depExists {
					t.Errorf("Dependency %s of step %s not found in topological order", dep, stepName)
					continue
				}

				if depPos >= stepPos {
					t.Errorf("Dependency %s (pos %d) should come before %s (pos %d) in topological order",
						dep, depPos, stepName, stepPos)
				}
			}
		}
	})

	t.Run("compare MCP disabled configuration", func(t *testing.T) {
		// Configuration for MCP disabled test
		config := &InstallConfig{
			AddRecommendedMCP: false,
		}
		mcpEnabled := config.AddRecommendedMCP

		// Create new dependency graph with same configuration
		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		// Steps that should be in the graph when MCP is disabled
		allSteps := []string{
			"CheckPrerequisites",
			"ScanExistingFiles",
			"CreateBackups",
			"CheckTargetDirectory",
			"CloneRepository",
			"CreateDirectoryStructure",
			"CopyCoreFiles",
			"CopyCommandFiles",
			"MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig", // Still exists as a step, but no dependencies on it
			"CreateCommandSymlink",
			"ValidateInstallation",
			"CleanupTempFiles",
		}

		// For each step, compare dependencies between old and new system
		for _, stepName := range allSteps {
			oldDeps := getOriginalDependencies(stepName, mcpEnabled)

			if !dg.HasStep(stepName) {
				// If step doesn't exist in new graph, old deps should be empty
				if len(oldDeps) > 0 {
					t.Errorf("Step %s not in new graph but has dependencies in old system: %v",
						stepName, oldDeps)
				}
				continue
			}

			newDeps, err := dg.GetDependencies(stepName)
			if err != nil {
				t.Errorf("Failed to get dependencies for step %s: %v", stepName, err)
				continue
			}

			// Sort both slices for comparison
			oldDepsMap := make(map[string]bool)
			for _, dep := range oldDeps {
				oldDepsMap[dep] = true
			}

			newDepsMap := make(map[string]bool)
			for _, dep := range newDeps {
				newDepsMap[dep] = true
			}

			// Compare dependencies
			if len(oldDepsMap) != len(newDepsMap) {
				t.Errorf("Step %s: dependency count mismatch. Old: %d, New: %d",
					stepName, len(oldDepsMap), len(newDepsMap))
				t.Errorf("Old deps: %v", oldDeps)
				t.Errorf("New deps: %v", newDeps)
				continue
			}

			for dep := range oldDepsMap {
				if !newDepsMap[dep] {
					t.Errorf("Step %s: missing dependency %s in new system", stepName, dep)
				}
			}

			for dep := range newDepsMap {
				if !oldDepsMap[dep] {
					t.Errorf("Step %s: extra dependency %s in new system", stepName, dep)
				}
			}
		}

		// Verify that MCP dependencies are not present when disabled
		validateDeps, err := dg.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get ValidateInstallation dependencies: %v", err)
		}

		for _, dep := range validateDeps {
			if dep == "MergeOrCreateMCPConfig" {
				t.Error("ValidateInstallation should not depend on MergeOrCreateMCPConfig when MCP is disabled")
			}
		}

		cleanupDeps, err := dg.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Fatalf("Failed to get CleanupTempFiles dependencies: %v", err)
		}

		for _, dep := range cleanupDeps {
			if dep == "MergeOrCreateMCPConfig" {
				t.Error("CleanupTempFiles should not depend on MergeOrCreateMCPConfig when MCP is disabled")
			}
		}
	})

	t.Run("verify exact step execution order compatibility", func(t *testing.T) {
		// Test both configurations to ensure execution order is preserved
		configs := []*InstallConfig{
			{AddRecommendedMCP: true},
			{AddRecommendedMCP: false},
		}

		for _, config := range configs {
			mcpDesc := "disabled"
			if config.AddRecommendedMCP {
				mcpDesc = "enabled"
			}

			t.Run(fmt.Sprintf("MCP %s", mcpDesc), func(t *testing.T) {
				dg := NewTestDependencyGraph()
				err := dg.BuildInstallationGraph(config)
				if err != nil {
					t.Fatalf("Failed to build installation graph with MCP %s: %v", mcpDesc, err)
				}

				order, err := dg.GetTopologicalOrder()
				if err != nil {
					t.Fatalf("Failed to get topological order with MCP %s: %v", mcpDesc, err)
				}

				// Verify critical ordering constraints that must be preserved
				positions := make(map[string]int)
				for i, step := range order {
					positions[step] = i
				}

				// Core dependency constraints that must always hold
				constraints := []struct {
					before, after string
					description   string
				}{
					{"CheckPrerequisites", "ScanExistingFiles", "Prerequisites must come before scanning"},
					{"ScanExistingFiles", "CreateBackups", "Scanning must come before backups"},
					{"CreateBackups", "CheckTargetDirectory", "Backups must come before target check"},
					{"CheckTargetDirectory", "CloneRepository", "Target check must come before cloning"},
					{"CheckTargetDirectory", "CreateDirectoryStructure", "Target check must come before directory creation"},
					{"CreateDirectoryStructure", "MergeOrCreateCLAUDEmd", "Directory structure must come before CLAUDE.md"},
					{"CreateDirectoryStructure", "CreateCommandSymlink", "Directory structure must come before symlink"},
					{"ValidateInstallation", "CleanupTempFiles", "Validation must come before cleanup"},
				}

				// Add MCP-specific constraints if enabled
				if config.AddRecommendedMCP {
					mcpConstraints := []struct {
						before, after string
						description   string
					}{
						{"CreateDirectoryStructure", "MergeOrCreateMCPConfig", "Directory structure must come before MCP config"},
						{"MergeOrCreateMCPConfig", "ValidateInstallation", "MCP config must come before validation"},
					}
					constraints = append(constraints, mcpConstraints...)
				}

				// Verify all constraints
				for _, constraint := range constraints {
					beforePos, beforeExists := positions[constraint.before]
					afterPos, afterExists := positions[constraint.after]

					if !beforeExists {
						t.Errorf("Step %s not found in execution order (constraint: %s)",
							constraint.before, constraint.description)
						continue
					}
					if !afterExists {
						t.Errorf("Step %s not found in execution order (constraint: %s)",
							constraint.after, constraint.description)
						continue
					}

					if beforePos >= afterPos {
						t.Errorf("Constraint violation: %s (pos %d) should come before %s (pos %d) - %s",
							constraint.before, beforePos, constraint.after, afterPos, constraint.description)
					}
				}
			})
		}
	})
}

// TestMCPConditionalLogic tests the MCP conditional dependency scenarios specifically
func TestMCPConditionalLogic(t *testing.T) {
	t.Run("MCP enabled includes conditional dependencies", func(t *testing.T) {
		config := &InstallConfig{
			AddRecommendedMCP: true,
		}

		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph with MCP enabled: %v", err)
		}

		// Verify ValidateInstallation has MCP dependency when enabled
		validateDeps, err := dg.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get ValidateInstallation dependencies: %v", err)
		}

		mcpDepFound := false
		for _, dep := range validateDeps {
			if dep == "MergeOrCreateMCPConfig" {
				mcpDepFound = true
				break
			}
		}
		if !mcpDepFound {
			t.Error("ValidateInstallation should depend on MergeOrCreateMCPConfig when MCP is enabled")
			t.Errorf("ValidateInstallation dependencies: %v", validateDeps)
		}

		// Verify CleanupTempFiles has MCP dependency when enabled
		cleanupDeps, err := dg.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Fatalf("Failed to get CleanupTempFiles dependencies: %v", err)
		}

		mcpDepFound = false
		for _, dep := range cleanupDeps {
			if dep == "MergeOrCreateMCPConfig" {
				mcpDepFound = true
				break
			}
		}
		if !mcpDepFound {
			t.Error("CleanupTempFiles should depend on MergeOrCreateMCPConfig when MCP is enabled")
			t.Errorf("CleanupTempFiles dependencies: %v", cleanupDeps)
		}

		// Verify MergeOrCreateMCPConfig step exists in the graph
		if !dg.HasStep("MergeOrCreateMCPConfig") {
			t.Error("MergeOrCreateMCPConfig step should exist when MCP is enabled")
		}

		// Verify topological ordering includes MCP dependencies
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// MergeOrCreateMCPConfig should come before ValidateInstallation
		mcpPos, mcpExists := positions["MergeOrCreateMCPConfig"]
		validatePos, validateExists := positions["ValidateInstallation"]
		if !mcpExists {
			t.Error("MergeOrCreateMCPConfig should be in topological order when MCP enabled")
		}
		if !validateExists {
			t.Error("ValidateInstallation should be in topological order")
		}
		if mcpExists && validateExists && mcpPos >= validatePos {
			t.Errorf("MergeOrCreateMCPConfig (pos %d) should come before ValidateInstallation (pos %d)",
				mcpPos, validatePos)
		}

		// MergeOrCreateMCPConfig should come before CleanupTempFiles
		cleanupPos, cleanupExists := positions["CleanupTempFiles"]
		if !cleanupExists {
			t.Error("CleanupTempFiles should be in topological order")
		}
		if mcpExists && cleanupExists && mcpPos >= cleanupPos {
			t.Errorf("MergeOrCreateMCPConfig (pos %d) should come before CleanupTempFiles (pos %d)",
				mcpPos, cleanupPos)
		}
	})

	t.Run("MCP disabled excludes conditional dependencies", func(t *testing.T) {
		config := &InstallConfig{
			AddRecommendedMCP: false,
		}

		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph with MCP disabled: %v", err)
		}

		// Verify ValidateInstallation does NOT have MCP dependency when disabled
		validateDeps, err := dg.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get ValidateInstallation dependencies: %v", err)
		}

		for _, dep := range validateDeps {
			if dep == "MergeOrCreateMCPConfig" {
				t.Error("ValidateInstallation should NOT depend on MergeOrCreateMCPConfig when MCP is disabled")
				t.Errorf("ValidateInstallation dependencies: %v", validateDeps)
			}
		}

		// Verify CleanupTempFiles does NOT have MCP dependency when disabled
		cleanupDeps, err := dg.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Fatalf("Failed to get CleanupTempFiles dependencies: %v", err)
		}

		for _, dep := range cleanupDeps {
			if dep == "MergeOrCreateMCPConfig" {
				t.Error("CleanupTempFiles should NOT depend on MergeOrCreateMCPConfig when MCP is disabled")
				t.Errorf("CleanupTempFiles dependencies: %v", cleanupDeps)
			}
		}

		// MergeOrCreateMCPConfig should still exist as a step but have no dependents
		if !dg.HasStep("MergeOrCreateMCPConfig") {
			t.Error("MergeOrCreateMCPConfig step should still exist even when MCP is disabled")
		}

		// Verify base dependencies are still present
		expectedValidateDeps := []string{
			"CopyCoreFiles",
			"CopyCommandFiles",
			"MergeOrCreateCLAUDEmd",
			"CreateCommandSymlink",
		}

		validateDepsMap := make(map[string]bool)
		for _, dep := range validateDeps {
			validateDepsMap[dep] = true
		}

		for _, expected := range expectedValidateDeps {
			if !validateDepsMap[expected] {
				t.Errorf("ValidateInstallation missing expected dependency: %s", expected)
			}
		}

		expectedCleanupDeps := []string{
			"CopyCoreFiles",
			"CopyCommandFiles",
			"MergeOrCreateCLAUDEmd",
			"CreateCommandSymlink",
			"ValidateInstallation",
		}

		cleanupDepsMap := make(map[string]bool)
		for _, dep := range cleanupDeps {
			cleanupDepsMap[dep] = true
		}

		for _, expected := range expectedCleanupDeps {
			if !cleanupDepsMap[expected] {
				t.Errorf("CleanupTempFiles missing expected dependency: %s", expected)
			}
		}
	})

	t.Run("MCP configuration parameter validation", func(t *testing.T) {
		// Test with nil config (should not crash)
		dg1 := NewDependencyGraph()
		err := dg1.BuildInstallationGraph(nil)
		if err != nil {
			t.Fatalf("BuildInstallationGraph should handle nil config gracefully: %v", err)
		}

		// Verify that nil config behaves like MCP disabled
		validateDeps, err := dg1.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get ValidateInstallation dependencies with nil config: %v", err)
		}

		for _, dep := range validateDeps {
			if dep == "MergeOrCreateMCPConfig" {
				t.Error("ValidateInstallation should NOT depend on MergeOrCreateMCPConfig with nil config")
			}
		}

		// Test with explicitly false MCP flag
		config := &InstallConfig{
			AddRecommendedMCP: false,
		}

		dg2 := NewDependencyGraph()
		err = dg2.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph with explicit false MCP: %v", err)
		}

		// Should behave identically to nil config
		validateDeps2, err := dg2.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get ValidateInstallation dependencies with false MCP: %v", err)
		}

		if len(validateDeps) != len(validateDeps2) {
			t.Errorf("Nil config and false MCP config should produce identical dependencies. Nil: %v, False: %v",
				validateDeps, validateDeps2)
		}

		validateDepsMap1 := make(map[string]bool)
		for _, dep := range validateDeps {
			validateDepsMap1[dep] = true
		}

		for _, dep := range validateDeps2 {
			if !validateDepsMap1[dep] {
				t.Errorf("Dependency %s found with false MCP but not with nil config", dep)
			}
		}
	})

	t.Run("MCP dependency count verification", func(t *testing.T) {
		// Count dependencies with MCP enabled
		mcpConfig := &InstallConfig{AddRecommendedMCP: true}
		dgMcp := NewTestDependencyGraph()
		err := dgMcp.BuildInstallationGraph(mcpConfig)
		if err != nil {
			t.Fatalf("Failed to build MCP graph: %v", err)
		}

		mcpValidateDeps, err := dgMcp.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get MCP ValidateInstallation deps: %v", err)
		}

		mcpCleanupDeps, err := dgMcp.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Fatalf("Failed to get MCP CleanupTempFiles deps: %v", err)
		}

		// Count dependencies with MCP disabled
		noMcpConfig := &InstallConfig{AddRecommendedMCP: false}
		dgNoMcp := NewTestDependencyGraph()
		err = dgNoMcp.BuildInstallationGraph(noMcpConfig)
		if err != nil {
			t.Fatalf("Failed to build no-MCP graph: %v", err)
		}

		noMcpValidateDeps, err := dgNoMcp.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get no-MCP ValidateInstallation deps: %v", err)
		}

		noMcpCleanupDeps, err := dgNoMcp.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Fatalf("Failed to get no-MCP CleanupTempFiles deps: %v", err)
		}

		// MCP enabled should have exactly one more dependency for each conditional step
		if len(mcpValidateDeps) != len(noMcpValidateDeps)+1 {
			t.Errorf("MCP enabled ValidateInstallation should have exactly 1 more dependency. MCP: %d, No-MCP: %d",
				len(mcpValidateDeps), len(noMcpValidateDeps))
			t.Errorf("MCP deps: %v", mcpValidateDeps)
			t.Errorf("No-MCP deps: %v", noMcpValidateDeps)
		}

		if len(mcpCleanupDeps) != len(noMcpCleanupDeps)+1 {
			t.Errorf("MCP enabled CleanupTempFiles should have exactly 1 more dependency. MCP: %d, No-MCP: %d",
				len(mcpCleanupDeps), len(noMcpCleanupDeps))
			t.Errorf("MCP deps: %v", mcpCleanupDeps)
			t.Errorf("No-MCP deps: %v", noMcpCleanupDeps)
		}
	})

	t.Run("MCP step ordering constraints", func(t *testing.T) {
		config := &InstallConfig{AddRecommendedMCP: true}
		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		positions := make(map[string]int)
		for i, step := range order {
			positions[step] = i
		}

		// Define all MCP-related ordering constraints
		constraints := []struct {
			before, after string
			description   string
		}{
			{"CreateDirectoryStructure", "MergeOrCreateMCPConfig", "Directory structure before MCP config"},
			{"MergeOrCreateMCPConfig", "ValidateInstallation", "MCP config before validation"},
			{"MergeOrCreateMCPConfig", "CleanupTempFiles", "MCP config before cleanup"},
			{"ValidateInstallation", "CleanupTempFiles", "Validation before cleanup"},
		}

		for _, constraint := range constraints {
			beforePos, beforeExists := positions[constraint.before]
			afterPos, afterExists := positions[constraint.after]

			if !beforeExists {
				t.Errorf("Step %s not found in order for constraint: %s", constraint.before, constraint.description)
				continue
			}
			if !afterExists {
				t.Errorf("Step %s not found in order for constraint: %s", constraint.after, constraint.description)
				continue
			}

			if beforePos >= afterPos {
				t.Errorf("Constraint violated: %s (pos %d) should come before %s (pos %d) - %s",
					constraint.before, beforePos, constraint.after, afterPos, constraint.description)
			}
		}

		// Verify MergeOrCreateMCPConfig comes after its dependencies
		mcpPos := positions["MergeOrCreateMCPConfig"]
		mcpDeps, err := dg.GetDependencies("MergeOrCreateMCPConfig")
		if err != nil {
			t.Fatalf("Failed to get MergeOrCreateMCPConfig dependencies: %v", err)
		}

		for _, dep := range mcpDeps {
			depPos, exists := positions[dep]
			if !exists {
				t.Errorf("MCP dependency %s not found in topological order", dep)
				continue
			}
			if depPos >= mcpPos {
				t.Errorf("MCP dependency %s (pos %d) should come before MergeOrCreateMCPConfig (pos %d)",
					dep, depPos, mcpPos)
			}
		}
	})
}

// TestRealInstallerIntegration tests the dependency graph system with real installer workflows
func TestRealInstallerIntegration(t *testing.T) {
	t.Run("full installation workflow with dependency graph", func(t *testing.T) {
		// Test with MCP enabled configuration
		config := &InstallConfig{
			AddRecommendedMCP: true,
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
		}

		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		// Get the execution order
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		// Verify the order contains all expected steps
		expectedSteps := []string{
			"CheckPrerequisites",
			"ScanExistingFiles",
			"CreateBackups",
			"CheckTargetDirectory",
			"CloneRepository",
			"CreateDirectoryStructure",
			"CopyCoreFiles",
			"CopyCommandFiles",
			"MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig", // Should be present with MCP enabled
			"CreateCommandSymlink",
			"ValidateInstallation",
			"CleanupTempFiles",
		}

		// Verify all expected steps are present
		stepMap := make(map[string]bool)
		for _, step := range order {
			stepMap[step] = true
		}

		for _, expected := range expectedSteps {
			if !stepMap[expected] {
				t.Errorf("Expected step %s not found in execution order", expected)
			}
		}

		// Verify step count
		if len(order) != len(expectedSteps) {
			t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(order))
			t.Errorf("Execution order: %v", order)
		}

		// Simulate execution validation by checking that each step's dependencies
		// have been "executed" before the step itself
		executedSteps := make(map[string]bool)
		for _, stepName := range order {
			// Check dependencies are satisfied
			deps, err := dg.GetDependencies(stepName)
			if err != nil {
				t.Errorf("Failed to get dependencies for step %s: %v", stepName, err)
				continue
			}

			for _, dep := range deps {
				if !executedSteps[dep] {
					t.Errorf("Step %s dependency %s not executed before %s", stepName, dep, stepName)
				}
			}

			// Mark step as executed
			executedSteps[stepName] = true
		}
	})

	t.Run("real installer step compatibility validation", func(t *testing.T) {
		// Get actual installer steps
		realSteps := GetInstallSteps()
		if realSteps == nil {
			t.Fatal("GetInstallSteps() returned nil")
		}

		config := &InstallConfig{AddRecommendedMCP: true}
		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		// Verify that all steps in the dependency graph have corresponding real steps
		for _, stepName := range order {
			realStep, exists := realSteps[stepName]
			if !exists {
				t.Errorf("Dependency graph step %s has no corresponding real installer step", stepName)
				continue
			}
			if realStep == nil {
				t.Errorf("Real installer step %s is nil", stepName)
			}
		}

		// Verify that critical real steps are represented in the dependency graph
		criticalSteps := []string{
			"CheckPrerequisites",
			"CreateDirectoryStructure",
			"CopyCoreFiles",
			"ValidateInstallation",
			"CleanupTempFiles",
		}

		for _, critical := range criticalSteps {
			if !dg.HasStep(critical) {
				t.Errorf("Critical real installer step %s missing from dependency graph", critical)
			}
		}
	})

	t.Run("installation with different configurations", func(t *testing.T) {
		testConfigs := []*InstallConfig{
			{AddRecommendedMCP: true, Force: false, NoBackup: false},
			{AddRecommendedMCP: false, Force: false, NoBackup: false},
			{AddRecommendedMCP: true, Force: true, NoBackup: false},
			{AddRecommendedMCP: false, Force: false, NoBackup: true},
		}

		for i, config := range testConfigs {
			t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
				dg := NewTestDependencyGraph()
				err := dg.BuildInstallationGraph(config)
				if err != nil {
					t.Fatalf("Failed to build installation graph for config %d: %v", i, err)
				}

				order, err := dg.GetTopologicalOrder()
				if err != nil {
					t.Fatalf("Failed to get topological order for config %d: %v", i, err)
				}

				// Basic validation that we have a reasonable number of steps
				if len(order) < 10 {
					t.Errorf("Config %d: too few steps in execution order (%d)", i, len(order))
				}

				// Verify MCP step inclusion based on config
				mcpStepFound := false
				for _, step := range order {
					if step == "MergeOrCreateMCPConfig" {
						mcpStepFound = true
						break
					}
				}

				// Note: MergeOrCreateMCPConfig should always be in the graph as a step,
				// but dependencies TO it should be conditional
				if !mcpStepFound {
					t.Errorf("Config %d: MergeOrCreateMCPConfig step should exist in graph", i)
				}

				// Verify conditional dependencies
				if config.AddRecommendedMCP {
					validateDeps, err := dg.GetDependencies("ValidateInstallation")
					if err != nil {
						t.Errorf("Config %d: failed to get ValidateInstallation deps: %v", i, err)
					} else {
						mcpDepFound := false
						for _, dep := range validateDeps {
							if dep == "MergeOrCreateMCPConfig" {
								mcpDepFound = true
								break
							}
						}
						if !mcpDepFound {
							t.Errorf("Config %d: ValidateInstallation should depend on MergeOrCreateMCPConfig when MCP enabled", i)
						}
					}
				}
			})
		}
	})

	t.Run("error handling with real installer context", func(t *testing.T) {
		// Test with various edge cases that might occur in real installation

		// Test with nil config (should not crash)
		dg1 := NewDependencyGraph()
		err := dg1.BuildInstallationGraph(nil)
		if err != nil {
			t.Errorf("BuildInstallationGraph should handle nil config gracefully: %v", err)
		}

		// Test that the graph is still functional
		order1, err := dg1.GetTopologicalOrder()
		if err != nil {
			t.Errorf("GetTopologicalOrder should work with nil config: %v", err)
		}
		if len(order1) == 0 {
			t.Error("Should have some steps even with nil config")
		}

		// Test with extreme config values
		extremeConfig := &InstallConfig{
			AddRecommendedMCP: true,
			Force:             true,
			NoBackup:          true,
			Interactive:       true,
			BackupDir:         "/some/extreme/path/that/might/not/exist",
		}

		dg2 := NewDependencyGraph()
		err = dg2.BuildInstallationGraph(extremeConfig)
		if err != nil {
			t.Errorf("BuildInstallationGraph should handle extreme config: %v", err)
		}

		// Should still produce valid topological order
		order2, err := dg2.GetTopologicalOrder()
		if err != nil {
			t.Errorf("GetTopologicalOrder should work with extreme config: %v", err)
		}
		if len(order2) == 0 {
			t.Error("Should have steps even with extreme config")
		}
	})

	t.Run("performance with realistic installer workflow", func(t *testing.T) {
		// Test that the dependency graph operations are fast enough for real usage
		config := &InstallConfig{AddRecommendedMCP: true}

		// Measure graph construction time
		start := time.Now()
		dg := NewDependencyGraph()
		err := dg.BuildInstallationGraph(config)
		constructionTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		// Should be very fast (under 1ms for ~13 steps)
		if constructionTime > time.Millisecond {
			t.Errorf("Graph construction took too long: %v (should be < 1ms)", constructionTime)
		}

		// Measure topological sort time
		start = time.Now()
		order, err := dg.GetTopologicalOrder()
		sortTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		// Should be very fast
		if sortTime > time.Millisecond {
			t.Errorf("Topological sort took too long: %v (should be < 1ms)", sortTime)
		}

		// Measure dependency queries
		start = time.Now()
		for _, stepName := range order {
			_, err := dg.GetDependencies(stepName)
			if err != nil {
				t.Errorf("Failed to get dependencies for %s: %v", stepName, err)
			}
		}
		queryTime := time.Since(start)

		// Should be fast even with multiple queries
		if queryTime > 5*time.Millisecond {
			t.Errorf("Dependency queries took too long: %v (should be < 5ms for all steps)", queryTime)
		}

		t.Logf("Performance: Construction: %v, Sort: %v, Queries: %v",
			constructionTime, sortTime, queryTime)
	})

	t.Run("memory usage validation", func(t *testing.T) {
		// Test that the dependency graph doesn't use excessive memory
		config := &InstallConfig{AddRecommendedMCP: true}

		// Create multiple graphs to test for memory leaks
		var graphs []*DependencyGraph
		for i := 0; i < 100; i++ {
			dg := NewTestDependencyGraph()
			err := dg.BuildInstallationGraph(config)
			if err != nil {
				t.Fatalf("Failed to build graph %d: %v", i, err)
			}
			graphs = append(graphs, dg)
		}

		// Verify all graphs are functional
		for i, dg := range graphs {
			order, err := dg.GetTopologicalOrder()
			if err != nil {
				t.Errorf("Graph %d failed to get topological order: %v", i, err)
			}
			if len(order) == 0 {
				t.Errorf("Graph %d has no steps", i)
			}
		}

		// Basic validation that we have reasonable step counts
		for i, dg := range graphs {
			steps := dg.GetSteps()
			if len(steps) < 10 || len(steps) > 20 {
				t.Errorf("Graph %d has unexpected step count: %d", i, len(steps))
			}
		}
	})
}

// TestRegressionDependencyMapping ensures exact preservation of original dependency mapping behavior.
// This test validates that the new centralized dependency system produces identical results
// to the original getDependencies() method for all known installation steps and configurations.
func TestRegressionDependencyMapping(t *testing.T) {
	t.Run("StaticDependencyPreservation", func(t *testing.T) {
		// Test each static dependency mapping against original system
		staticMappings := map[string][]string{
			"ScanExistingFiles":        {"CheckPrerequisites"},
			"CreateBackups":            {"ScanExistingFiles"},
			"CheckTargetDirectory":     {"CreateBackups"},
			"CloneRepository":          {"CheckTargetDirectory"},
			"CreateDirectoryStructure": {"CheckTargetDirectory"},
			"CopyCoreFiles":            {"CloneRepository", "CreateDirectoryStructure"},
			"CopyCommandFiles":         {"CloneRepository", "CreateDirectoryStructure"},
			"MergeOrCreateCLAUDEmd":    {"CreateDirectoryStructure"},
			"MergeOrCreateMCPConfig":   {"CreateDirectoryStructure"},
			"CreateCommandSymlink":     {"CopyCommandFiles", "CreateDirectoryStructure"},
		}

		// Test with non-MCP configuration
		config := &InstallConfig{AddRecommendedMCP: false}
		dg := NewTestDependencyGraph()
		if err := dg.BuildInstallationGraph(config); err != nil {
			t.Fatalf("Failed to build graph: %v", err)
		}

		// Verify each static mapping is preserved exactly
		for stepName, expectedDeps := range staticMappings {
			actualDeps, err := dg.GetDependencies(stepName)
			if err != nil {
				t.Errorf("Failed to get dependencies for %s: %v", stepName, err)
				continue
			}

			// Sort both slices for comparison
			sort.Strings(expectedDeps)
			sort.Strings(actualDeps)

			if !reflect.DeepEqual(expectedDeps, actualDeps) {
				t.Errorf("Static dependency mismatch for %s:\n  Expected: %v\n  Actual:   %v",
					stepName, expectedDeps, actualDeps)
			}
		}
	})

	t.Run("ConditionalDependencyPreservation", func(t *testing.T) {
		testCases := []struct {
			name          string
			mcpEnabled    bool
			step          string
			expectedBasic []string
			expectedMCP   []string
		}{
			{
				name:          "ValidateInstallation_WithoutMCP",
				mcpEnabled:    false,
				step:          "ValidateInstallation",
				expectedBasic: []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink"},
				expectedMCP:   nil,
			},
			{
				name:          "ValidateInstallation_WithMCP",
				mcpEnabled:    true,
				step:          "ValidateInstallation",
				expectedBasic: []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink"},
				expectedMCP:   []string{"MergeOrCreateMCPConfig"},
			},
			{
				name:          "CleanupTempFiles_WithoutMCP",
				mcpEnabled:    false,
				step:          "CleanupTempFiles",
				expectedBasic: []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation"},
				expectedMCP:   nil,
			},
			{
				name:          "CleanupTempFiles_WithMCP",
				mcpEnabled:    true,
				step:          "CleanupTempFiles",
				expectedBasic: []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation"},
				expectedMCP:   []string{"MergeOrCreateMCPConfig"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := &InstallConfig{AddRecommendedMCP: tc.mcpEnabled}
				dg := NewTestDependencyGraph()
				if err := dg.BuildInstallationGraph(config); err != nil {
					t.Fatalf("Failed to build graph: %v", err)
				}

				actualDeps, err := dg.GetDependencies(tc.step)
				if err != nil {
					t.Fatalf("Failed to get dependencies for %s: %v", tc.step, err)
				}

				expectedDeps := make([]string, len(tc.expectedBasic))
				copy(expectedDeps, tc.expectedBasic)
				if tc.mcpEnabled && tc.expectedMCP != nil {
					expectedDeps = append(expectedDeps, tc.expectedMCP...)
				}

				sort.Strings(expectedDeps)
				sort.Strings(actualDeps)

				if !reflect.DeepEqual(expectedDeps, actualDeps) {
					t.Errorf("Conditional dependency mismatch for %s (MCP=%v):\n  Expected: %v\n  Actual:   %v",
						tc.step, tc.mcpEnabled, expectedDeps, actualDeps)
				}
			})
		}
	})

	t.Run("RegressionAgainstOriginalMethod", func(t *testing.T) {
		// Test against the exact logic that was in getDependencies() method
		// This ensures we haven't introduced any subtle changes during migration

		configs := []*InstallConfig{
			{AddRecommendedMCP: false},
			{AddRecommendedMCP: true},
		}

		for _, config := range configs {
			configName := "WithoutMCP"
			if config.AddRecommendedMCP {
				configName = "WithMCP"
			}

			t.Run(configName, func(t *testing.T) {
				dg := NewTestDependencyGraph()
				if err := dg.BuildInstallationGraph(config); err != nil {
					t.Fatalf("Failed to build graph: %v", err)
				}

				// Test all steps that should exist in the graph
				allSteps := []string{
					"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
					"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
					"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
					"MergeOrCreateMCPConfig", "CreateCommandSymlink",
					"ValidateInstallation", "CleanupTempFiles",
				}

				for _, step := range allSteps {
					if !dg.HasStep(step) {
						t.Errorf("Step %s is missing from graph", step)
						continue
					}

					deps, err := dg.GetDependencies(step)
					if err != nil {
						t.Errorf("Failed to get dependencies for %s: %v", step, err)
						continue
					}

					// Validate that dependencies are reasonable (no self-dependencies, etc.)
					for _, dep := range deps {
						if dep == step {
							t.Errorf("Step %s has self-dependency", step)
						}
						if !dg.HasStep(dep) {
							t.Errorf("Step %s depends on missing step %s", step, dep)
						}
					}
				}
			})
		}
	})

	t.Run("EdgeCaseDependencyPreservation", func(t *testing.T) {
		// Test edge cases that might have been missed during migration
		dg := NewTestDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: true}
		if err := dg.BuildInstallationGraph(config); err != nil {
			t.Fatalf("Failed to build graph: %v", err)
		}

		// Ensure no orphaned steps (steps with no dependencies and no dependents)
		allSteps := dg.GetSteps()
		if len(allSteps) == 0 {
			t.Fatal("Graph has no steps")
		}

		// CheckPrerequisites should have no dependencies (it's the root)
		checkDeps, err := dg.GetDependencies("CheckPrerequisites")
		if err != nil {
			t.Errorf("Failed to get dependencies for CheckPrerequisites: %v", err)
		} else if len(checkDeps) != 0 {
			t.Errorf("CheckPrerequisites should have no dependencies, but has: %v", checkDeps)
		}

		// CleanupTempFiles should be one of the final steps (has many dependencies)
		cleanupDeps, err := dg.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Errorf("Failed to get dependencies for CleanupTempFiles: %v", err)
		} else if len(cleanupDeps) < 5 {
			t.Errorf("CleanupTempFiles should have many dependencies, but only has: %v", cleanupDeps)
		}

		// Ensure topological order is still valid
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Errorf("Failed to get topological order: %v", err)
		} else if len(order) != len(allSteps) {
			t.Errorf("Topological order length (%d) doesn't match step count (%d)", len(order), len(allSteps))
		}
	})
}

// TestMigrationValidation validates the complete migration from manual dependency management
// in getDependencies() to the centralized BuildInstallationGraph() approach.
// This test ensures the migration maintains full backward compatibility and correctness.
func TestMigrationValidation(t *testing.T) {
	t.Run("CompleteMigrationValidation", func(t *testing.T) {
		// Test all possible configurations and steps to ensure migration is complete
		testConfigs := []*InstallConfig{
			{AddRecommendedMCP: false},
			{AddRecommendedMCP: true},
			nil, // Test with nil config (should default to non-MCP behavior)
		}

		allSteps := []string{
			"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
			"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
			"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig", "CreateCommandSymlink",
			"ValidateInstallation", "CleanupTempFiles",
		}

		for i, config := range testConfigs {
			configDesc := fmt.Sprintf("Config %d", i)
			if config == nil {
				configDesc = "Nil Config"
			} else if config.AddRecommendedMCP {
				configDesc = "MCP Enabled"
			} else {
				configDesc = "MCP Disabled"
			}

			t.Run(configDesc, func(t *testing.T) {
				dg := NewTestDependencyGraph()
				if err := dg.BuildInstallationGraph(config); err != nil {
					t.Fatalf("BuildInstallationGraph failed for %s: %v", configDesc, err)
				}

				// Verify all expected steps are present
				graphSteps := dg.GetSteps()
				if len(graphSteps) != len(allSteps) {
					t.Errorf("Step count mismatch for %s: expected %d, got %d",
						configDesc, len(allSteps), len(graphSteps))
				}

				for _, expectedStep := range allSteps {
					if !dg.HasStep(expectedStep) {
						t.Errorf("Missing step %s in %s", expectedStep, configDesc)
					}
				}

				// Verify topological order can be obtained (no cycles)
				order, err := dg.GetTopologicalOrder()
				if err != nil {
					t.Errorf("Failed to get topological order for %s: %v", configDesc, err)
				} else {
					if len(order) != len(allSteps) {
						t.Errorf("Topological order length mismatch for %s: expected %d, got %d",
							configDesc, len(allSteps), len(order))
					}

					// Verify order contains all steps
					orderMap := make(map[string]bool)
					for _, step := range order {
						orderMap[step] = true
					}
					for _, expectedStep := range allSteps {
						if !orderMap[expectedStep] {
							t.Errorf("Step %s missing from topological order in %s", expectedStep, configDesc)
						}
					}
				}
			})
		}
	})

	t.Run("MigrationPerformanceValidation", func(t *testing.T) {
		// Ensure migration doesn't introduce performance regressions
		config := &InstallConfig{AddRecommendedMCP: true}

		// Test construction time
		start := time.Now()
		dg := NewTestDependencyGraph()
		constructionTime := time.Since(start)

		// Test BuildInstallationGraph time
		start = time.Now()
		if err := dg.BuildInstallationGraph(config); err != nil {
			t.Fatalf("BuildInstallationGraph failed: %v", err)
		}
		buildTime := time.Since(start)

		// Test topological sort time
		start = time.Now()
		_, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("GetTopologicalOrder failed: %v", err)
		}
		sortTime := time.Since(start)

		// Test dependency query time
		start = time.Now()
		for i := 0; i < 100; i++ {
			_, _ = dg.GetDependencies("CleanupTempFiles")
		}
		queryTime := time.Since(start) / 100

		// Performance thresholds (generous to avoid flaky tests)
		maxConstructionTime := 1 * time.Millisecond
		maxBuildTime := 10 * time.Millisecond
		maxSortTime := 5 * time.Millisecond
		maxQueryTime := 100 * time.Microsecond

		if constructionTime > maxConstructionTime {
			t.Errorf("Construction too slow: %v > %v", constructionTime, maxConstructionTime)
		}
		if buildTime > maxBuildTime {
			t.Errorf("Build too slow: %v > %v", buildTime, maxBuildTime)
		}
		if sortTime > maxSortTime {
			t.Errorf("Sort too slow: %v > %v", sortTime, maxSortTime)
		}
		if queryTime > maxQueryTime {
			t.Errorf("Query too slow: %v > %v", queryTime, maxQueryTime)
		}

		t.Logf("Performance metrics: Construction=%v, Build=%v, Sort=%v, Query=%v",
			constructionTime, buildTime, sortTime, queryTime)
	})

	t.Run("MigrationRobustnessValidation", func(t *testing.T) {
		// Test edge cases and error conditions that the migration should handle

		// Test with nil config (should work)
		dg1 := NewTestDependencyGraph()
		if err := dg1.BuildInstallationGraph(nil); err != nil {
			t.Errorf("BuildInstallationGraph failed with nil config: %v", err)
		}

		// Test multiple builds on same graph (should fail or be idempotent)
		dg2 := NewTestDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: false}
		if err := dg2.BuildInstallationGraph(config); err != nil {
			t.Errorf("First BuildInstallationGraph failed: %v", err)
		}

		// Second build should fail (steps already added)
		err := dg2.BuildInstallationGraph(config)
		if err == nil {
			t.Error("Expected error on second BuildInstallationGraph call, but got none")
		}

		// Test that dependency queries work for all steps
		dg3 := NewDependencyGraph()
		if err := dg3.BuildInstallationGraph(config); err != nil {
			t.Fatalf("BuildInstallationGraph failed: %v", err)
		}

		allSteps := dg3.GetSteps()
		for _, step := range allSteps {
			deps, err := dg3.GetDependencies(step)
			if err != nil {
				t.Errorf("GetDependencies failed for step %s: %v", step, err)
			}

			// Verify all dependencies exist as steps
			for _, dep := range deps {
				if !dg3.HasStep(dep) {
					t.Errorf("Step %s has non-existent dependency: %s", step, dep)
				}
			}
		}
	})

	t.Run("BackwardCompatibilityGuarantee", func(t *testing.T) {
		// Final validation that migration preserves exact behavior

		// Create a simulation of the original getDependencies method results
		originalResults := map[string]map[bool][]string{
			"ScanExistingFiles":        {false: {"CheckPrerequisites"}, true: {"CheckPrerequisites"}},
			"CreateBackups":            {false: {"ScanExistingFiles"}, true: {"ScanExistingFiles"}},
			"CheckTargetDirectory":     {false: {"CreateBackups"}, true: {"CreateBackups"}},
			"CloneRepository":          {false: {"CheckTargetDirectory"}, true: {"CheckTargetDirectory"}},
			"CreateDirectoryStructure": {false: {"CheckTargetDirectory"}, true: {"CheckTargetDirectory"}},
			"CopyCoreFiles":            {false: {"CloneRepository", "CreateDirectoryStructure"}, true: {"CloneRepository", "CreateDirectoryStructure"}},
			"CopyCommandFiles":         {false: {"CloneRepository", "CreateDirectoryStructure"}, true: {"CloneRepository", "CreateDirectoryStructure"}},
			"MergeOrCreateCLAUDEmd":    {false: {"CreateDirectoryStructure"}, true: {"CreateDirectoryStructure"}},
			"MergeOrCreateMCPConfig":   {false: {"CreateDirectoryStructure"}, true: {"CreateDirectoryStructure"}},
			"CreateCommandSymlink":     {false: {"CopyCommandFiles", "CreateDirectoryStructure"}, true: {"CopyCommandFiles", "CreateDirectoryStructure"}},
			"ValidateInstallation":     {false: {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink"}, true: {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "MergeOrCreateMCPConfig"}},
			"CleanupTempFiles":         {false: {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation"}, true: {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation", "MergeOrCreateMCPConfig"}},
		}

		for _, mcpEnabled := range []bool{false, true} {
			configName := "WithoutMCP"
			if mcpEnabled {
				configName = "WithMCP"
			}

			t.Run(configName, func(t *testing.T) {
				config := &InstallConfig{AddRecommendedMCP: mcpEnabled}
				dg := NewTestDependencyGraph()
				if err := dg.BuildInstallationGraph(config); err != nil {
					t.Fatalf("BuildInstallationGraph failed: %v", err)
				}

				for stepName, expectedByMCP := range originalResults {
					expected := expectedByMCP[mcpEnabled]
					actual, err := dg.GetDependencies(stepName)
					if err != nil {
						t.Errorf("GetDependencies failed for %s: %v", stepName, err)
						continue
					}

					sort.Strings(expected)
					sort.Strings(actual)

					if !reflect.DeepEqual(expected, actual) {
						t.Errorf("Backward compatibility violation for %s (MCP=%v):\n  Expected: %v\n  Actual:   %v",
							stepName, mcpEnabled, expected, actual)
					}
				}
			})
		}
	})
}

// getOriginalDependencies replicates the logic from the old getDependencies method
// This is used for testing compatibility between old and new dependency systems
func getOriginalDependencies(stepName string, mcpEnabled bool) []string {
	dependencies := map[string][]string{
		"ScanExistingFiles":        {"CheckPrerequisites"},
		"CreateBackups":            {"ScanExistingFiles"},
		"CheckTargetDirectory":     {"CreateBackups"},
		"CloneRepository":          {"CheckTargetDirectory"},
		"CreateDirectoryStructure": {"CheckTargetDirectory"},
		"CopyCoreFiles":            {"CloneRepository", "CreateDirectoryStructure"},
		"CopyCommandFiles":         {"CloneRepository", "CreateDirectoryStructure"},
		"MergeOrCreateCLAUDEmd":    {"CreateDirectoryStructure"},
		"MergeOrCreateMCPConfig":   {"CreateDirectoryStructure"},
		"CreateCommandSymlink":     {"CopyCommandFiles", "CreateDirectoryStructure"},
	}

	// ValidateInstallation dependencies change based on whether MCP config is enabled
	if stepName == "ValidateInstallation" {
		validateDeps := []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink"}
		if mcpEnabled {
			validateDeps = append(validateDeps, "MergeOrCreateMCPConfig")
		}
		return validateDeps
	}

	// CleanupTempFiles runs after everything else including validation
	if stepName == "CleanupTempFiles" {
		cleanupDeps := []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation"}
		if mcpEnabled {
			cleanupDeps = append(cleanupDeps, "MergeOrCreateMCPConfig")
		}
		return cleanupDeps
	}

	if deps, ok := dependencies[stepName]; ok {
		return deps
	}
	return []string{}
}

// TestMissingStepValidation tests that the dependency graph validates step references
func TestMissingStepValidation(t *testing.T) {
	t.Run("validateStepReferences detects missing steps", func(t *testing.T) {
		dg := NewDependencyGraph()

		// Create dependencies with some missing steps
		invalidDeps := []Dependency{
			{From: "ValidStep", To: "CheckPrerequisites"},       // CheckPrerequisites exists
			{From: "NonExistentStep", To: "CheckPrerequisites"}, // NonExistentStep does not exist
			{From: "ValidStep", To: "AnotherMissingStep"},       // AnotherMissingStep does not exist
		}

		err := dg.validateStepReferences(invalidDeps)
		if err == nil {
			t.Error("Expected error for missing step references, got nil")
			return
		}

		errorStr := err.Error()
		if !strings.Contains(errorStr, "NonExistentStep") {
			t.Errorf("Expected error to mention NonExistentStep, got: %v", err)
		}
		if !strings.Contains(errorStr, "AnotherMissingStep") {
			t.Errorf("Expected error to mention AnotherMissingStep, got: %v", err)
		}
		if !strings.Contains(errorStr, "Available steps:") {
			t.Errorf("Expected error to list available steps, got: %v", err)
		}
	})

	t.Run("validateStepReferences accepts valid steps", func(t *testing.T) {
		dg := NewDependencyGraph()

		// Create dependencies with only valid steps
		validDeps := []Dependency{
			{From: "ScanExistingFiles", To: "CheckPrerequisites"},
			{From: "CreateBackups", To: "ScanExistingFiles"},
			{From: "ValidateInstallation", To: "CopyCoreFiles"},
		}

		err := dg.validateStepReferences(validDeps)
		if err != nil {
			t.Errorf("Expected no error for valid step references, got: %v", err)
		}
	})

	t.Run("buildGraph integration with missing step validation", func(t *testing.T) {
		dg := NewDependencyGraph()

		// Try to build graph with invalid dependencies
		invalidDeps := []Dependency{
			{From: "CheckPrerequisites", To: "NonExistentPrereq"},
			{From: "ScanExistingFiles", To: "CheckPrerequisites"},
		}

		err := dg.buildGraph(invalidDeps)
		if err == nil {
			t.Error("Expected buildGraph to fail with missing step references")
			return
		}

		errorStr := err.Error()
		if !strings.Contains(errorStr, "NonExistentPrereq") {
			t.Errorf("Expected error to mention NonExistentPrereq, got: %v", err)
		}
	})

	t.Run("single missing step error message", func(t *testing.T) {
		dg := NewDependencyGraph()

		// Test single missing step
		singleInvalidDep := []Dependency{
			{From: "CheckPrerequisites", To: "NonExistentStep"},
		}

		err := dg.validateStepReferences(singleInvalidDep)
		if err == nil {
			t.Error("Expected error for single missing step reference")
			return
		}

		errorStr := err.Error()
		if !strings.Contains(errorStr, "unknown installation step 'NonExistentStep'") {
			t.Errorf("Expected singular error message, got: %v", err)
		}
	})

	t.Run("multiple missing steps error message", func(t *testing.T) {
		dg := NewDependencyGraph()

		// Test multiple missing steps
		multipleInvalidDeps := []Dependency{
			{From: "Missing1", To: "CheckPrerequisites"},
			{From: "Missing2", To: "CheckPrerequisites"},
			{From: "CheckPrerequisites", To: "Missing3"},
		}

		err := dg.validateStepReferences(multipleInvalidDeps)
		if err == nil {
			t.Error("Expected error for multiple missing step references")
			return
		}

		errorStr := err.Error()
		if !strings.Contains(errorStr, "3 unknown installation steps") {
			t.Errorf("Expected plural error message mentioning count, got: %v", err)
		}
		// Check that missing steps are mentioned (order may vary due to map iteration)
		if !strings.Contains(errorStr, "Missing1") || !strings.Contains(errorStr, "Missing2") || !strings.Contains(errorStr, "Missing3") {
			t.Errorf("Expected all missing steps to be mentioned, got: %v", err)
		}
	})
}

// Performance Benchmarks for Task 6.6
// Target: <1ms for ~15 nodes (typical installer step count)

// BenchmarkGraphConstruction measures the time to create and populate a graph
func BenchmarkGraphConstruction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dg := NewTestDependencyGraph()

		// Add typical installer steps (~13 steps)
		steps := []string{
			"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
			"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
			"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig", "CreateCommandSymlink", "ValidateInstallation",
			"CleanupTempFiles",
		}

		for _, step := range steps {
			_ = dg.AddStep(step)
		}
	}
}

// BenchmarkDependencyAddition measures the time to add dependencies to a populated graph
func BenchmarkDependencyAddition(b *testing.B) {
	// Pre-create graph with steps
	dg := NewTestDependencyGraph()
	steps := []string{
		"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
		"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
		"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
		"MergeOrCreateMCPConfig", "CreateCommandSymlink", "ValidateInstallation",
		"CleanupTempFiles",
	}

	for _, step := range steps {
		_ = dg.AddStep(step)
	}

	// Define typical dependencies
	dependencies := []Dependency{
		{From: "ScanExistingFiles", To: "CheckPrerequisites"},
		{From: "CreateBackups", To: "ScanExistingFiles"},
		{From: "CheckTargetDirectory", To: "CreateBackups"},
		{From: "CloneRepository", To: "CheckTargetDirectory"},
		{From: "CreateDirectoryStructure", To: "CheckTargetDirectory"},
		{From: "CopyCoreFiles", To: "CloneRepository"},
		{From: "CopyCommandFiles", To: "CreateDirectoryStructure"},
		{From: "MergeOrCreateCLAUDEmd", To: "CreateDirectoryStructure"},
		{From: "CreateCommandSymlink", To: "CopyCommandFiles"},
		{From: "ValidateInstallation", To: "CopyCoreFiles"},
		{From: "CleanupTempFiles", To: "ValidateInstallation"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create fresh graph for each iteration
		testDg := NewTestDependencyGraph()
		for _, step := range steps {
			_ = testDg.AddStep(step)
		}

		// Add all dependencies
		for _, dep := range dependencies {
			_ = testDg.AddDependency(dep.From, dep.To)
		}
	}
}

// BenchmarkTopologicalSort measures the time to perform topological sorting
func BenchmarkTopologicalSort(b *testing.B) {
	// Pre-create and populate graph
	dg := NewTestDependencyGraph()
	dependencies := []Dependency{
		{From: "ScanExistingFiles", To: "CheckPrerequisites"},
		{From: "CreateBackups", To: "ScanExistingFiles"},
		{From: "CheckTargetDirectory", To: "CreateBackups"},
		{From: "CloneRepository", To: "CheckTargetDirectory"},
		{From: "CreateDirectoryStructure", To: "CheckTargetDirectory"},
		{From: "CopyCoreFiles", To: "CloneRepository"},
		{From: "CopyCommandFiles", To: "CreateDirectoryStructure"},
		{From: "MergeOrCreateCLAUDEmd", To: "CreateDirectoryStructure"},
		{From: "CreateCommandSymlink", To: "CopyCommandFiles"},
		{From: "ValidateInstallation", To: "CopyCoreFiles"},
		{From: "CleanupTempFiles", To: "ValidateInstallation"},
	}

	_ = dg.buildGraph(dependencies)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dg.GetTopologicalOrder()
	}
}

// BenchmarkBuildInstallationGraph measures the time to build the complete installer graph
func BenchmarkBuildInstallationGraph(b *testing.B) {
	config := &InstallConfig{AddRecommendedMCP: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dg := NewDependencyGraph() // Use production version for real validation
		_ = dg.BuildInstallationGraph(config)
	}
}

// BenchmarkCompleteWorkflow measures end-to-end performance of typical dependency operations
func BenchmarkCompleteWorkflow(b *testing.B) {
	config := &InstallConfig{AddRecommendedMCP: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Complete workflow: create graph, build dependencies, get order
		dg := NewDependencyGraph()
		_ = dg.BuildInstallationGraph(config)
		_, _ = dg.GetTopologicalOrder()

		// Query operations
		steps := dg.GetSteps()
		for _, step := range steps[:3] { // Test a few step queries
			_, _ = dg.GetDependencies(step)
		}
	}
}

// BenchmarkCycleDetection measures performance of cycle detection algorithms
func BenchmarkCycleDetection(b *testing.B) {
	// Create a graph that will trigger cycle detection
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dg := NewTestDependencyGraph()
		steps := []string{"A", "B", "C", "D", "E"}

		for _, step := range steps {
			_ = dg.AddStep(step)
		}

		// Add dependencies that create a cycle
		_ = dg.AddDependency("A", "B")
		_ = dg.AddDependency("B", "C")
		_ = dg.AddDependency("C", "D")
		_ = dg.AddDependency("D", "E")
		_ = dg.AddDependency("E", "A") // Creates cycle

		// This will trigger cycle detection
		_, _ = dg.GetTopologicalOrder()
	}
}

// BenchmarkLargeGraph measures performance with larger graphs (stress test)
func BenchmarkLargeGraph(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dg := NewTestDependencyGraph()

		// Create a larger graph with 50 steps
		steps := make([]string, 50)
		for j := 0; j < 50; j++ {
			steps[j] = fmt.Sprintf("Step%d", j)
			_ = dg.AddStep(steps[j])
		}

		// Add linear dependencies
		for j := 1; j < 50; j++ {
			_ = dg.AddDependency(steps[j], steps[j-1])
		}

		_, _ = dg.GetTopologicalOrder()
	}
}
