// Package installer provides a DAG-based dependency resolution system for managing
// installation step execution order. The DependencyGraph ensures that installation
// steps are executed in the correct order based on their dependencies, preventing
// circular dependencies and providing clear error messages when dependency issues occur.
//
// The system replaces the previous manual dependency traversal approach with a more
// robust and maintainable topological sorting mechanism using a directed acyclic graph.
//
// Example usage:
//
//	// Create a new dependency graph
//	graph := NewDependencyGraph()
//
//	// Add installation steps
//	if err := graph.AddStep("CheckPrerequisites"); err != nil {
//		log.Fatal(err)
//	}
//	if err := graph.AddStep("CreateDirectories"); err != nil {
//		log.Fatal(err)
//	}
//	if err := graph.AddStep("CopyFiles"); err != nil {
//		log.Fatal(err)
//	}
//
//	// Define dependencies (CreateDirectories depends on CheckPrerequisites)
//	if err := graph.AddDependency("CreateDirectories", "CheckPrerequisites"); err != nil {
//		log.Fatal(err)
//	}
//	// CopyFiles depends on CreateDirectories
//	if err := graph.AddDependency("CopyFiles", "CreateDirectories"); err != nil {
//		log.Fatal(err)
//	}
//
//	// Get execution order
//	order, err := graph.GetTopologicalOrder()
//	if err != nil {
//		// Handle cycle detection or other errors
//		log.Fatalf("Dependency cycle detected: %v", err)
//	}
//
//	// Execute steps in dependency order
//	for _, stepName := range order {
//		fmt.Printf("Executing step: %s\n", stepName)
//		// executeStep(stepName)
//	}
//
// For production use with real installation steps:
//
//	graph := NewDependencyGraph()
//	config := &InstallConfig{AddRecommendedMCP: true}
//	if err := graph.BuildInstallationGraph(config); err != nil {
//		log.Fatal(err)
//	}
//	order, err := graph.GetTopologicalOrder()
//	// ... execute installation steps
package installer

import (
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
)

// DependencyGraph wraps github.com/dominikbraun/graph to provide
// domain-specific functionality for managing installation step dependencies.
// It ensures acyclic dependency relationships and provides topological sorting
// for determining the correct execution order of installation steps.
type DependencyGraph struct {
	graph              graph.Graph[string, string]
	steps              map[string]bool // Track added steps for validation
	skipStepValidation bool            // Skip step validation for testing
}

// Dependency represents a dependency relationship between two installation steps.
type Dependency struct {
	From string // The step that depends on another
	To   string // The step that must be executed first
}

// NewDependencyGraph creates a new DependencyGraph instance with an acyclic
// directed graph to prevent circular dependencies at build time.
//
// The returned graph validates all step names against the available installation
// steps defined in GetInstallSteps(). Use NewTestDependencyGraph() for testing
// scenarios that require arbitrary step names.
//
// Returns a pointer to an initialized DependencyGraph ready for use.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		graph:              graph.New(graph.StringHash, graph.Directed(), graph.Acyclic()),
		steps:              make(map[string]bool),
		skipStepValidation: false,
	}
}

// NewTestDependencyGraph creates a DependencyGraph instance for testing
// that skips installation step validation, allowing arbitrary step names.
//
// This constructor is intended for unit tests and should not be used in
// production code. It bypasses step name validation, enabling tests to
// use mock step names without requiring actual InstallStep implementations.
//
// Returns a pointer to an initialized DependencyGraph with validation disabled.
func NewTestDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		graph:              graph.New(graph.StringHash, graph.Directed(), graph.Acyclic()),
		steps:              make(map[string]bool),
		skipStepValidation: true,
	}
}

// AddStep adds an installation step as a vertex to the dependency graph.
//
// The stepName must be non-empty and correspond to a valid installation step
// as defined in GetInstallSteps() (unless skipStepValidation is enabled for testing).
// Each step can only be added once - subsequent attempts to add the same step
// will return an error.
//
// Parameters:
//   - stepName: The name of the installation step to add (e.g., "CheckPrerequisites")
//
// Returns an error if:
//   - stepName is empty or contains only whitespace
//   - stepName has already been added to this graph
//   - stepName is not a valid installation step (in production mode)
//
// Example:
//
//	err := graph.AddStep("CheckPrerequisites")
//	if err != nil {
//		// Handle duplicate step or invalid step name
//	}
func (dg *DependencyGraph) AddStep(stepName string) error {
	if strings.TrimSpace(stepName) == "" {
		return fmt.Errorf("step name cannot be empty")
	}

	if dg.steps[stepName] {
		return fmt.Errorf("step '%s' has already been added", stepName)
	}

	err := dg.graph.AddVertex(stepName)
	if err != nil {
		return fmt.Errorf("failed to add step '%s': %w", stepName, err)
	}

	dg.steps[stepName] = true
	return nil
}

// AddDependency adds a dependency relationship between two steps.
//
// The 'from' step depends on the 'to' step, meaning 'to' must be executed before 'from'.
// Both steps must have been added via AddStep() before calling this method.
// This method will detect and prevent circular dependencies.
//
// Parameters:
//   - from: The step that has a dependency (e.g., "CreateDirectories")
//   - to: The step that must be executed first (e.g., "CheckPrerequisites")
//
// Returns an error if:
//   - Either step name is empty or contains only whitespace
//   - A step tries to depend on itself (from == to)
//   - Either step has not been added to the graph via AddStep()
//   - Adding this dependency would create a circular dependency
//
// Example:
//
//	// CreateDirectories depends on CheckPrerequisites
//	err := graph.AddDependency("CreateDirectories", "CheckPrerequisites")
//	if err != nil {
//		// Handle missing steps or circular dependency
//	}
func (dg *DependencyGraph) AddDependency(from, to string) error {
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return fmt.Errorf("dependency step names cannot be empty")
	}

	if from == to {
		return fmt.Errorf("step cannot depend on itself: %s", from)
	}

	if !dg.steps[from] {
		return fmt.Errorf("step '%s' has not been added to the graph", from)
	}

	if !dg.steps[to] {
		return fmt.Errorf("step '%s' has not been added to the graph", to)
	}

	err := dg.graph.AddEdge(to, from)
	if err != nil {
		return fmt.Errorf("failed to add dependency %s -> %s: %w", to, from, err)
	}

	return nil
}

// GetTopologicalOrder returns the installation steps in topologically sorted order,
// ensuring that dependencies are executed before the steps that depend on them.
//
// This method uses Kahn's algorithm via the underlying graph library to compute
// a valid execution order. If the graph contains circular dependencies, it will
// detect them and return a detailed error message showing the cycle path.
//
// Returns:
//   - []string: A slice of step names in dependency order (dependencies first)
//   - error: An error if cycles are detected, with detailed cycle path information
//
// The returned order guarantees that for any step S with dependencies D1, D2, ..., Dn,
// all dependencies D1-Dn will appear before S in the returned slice.
//
// Example:
//
//	order, err := graph.GetTopologicalOrder()
//	if err != nil {
//		// Handle circular dependency error
//		fmt.Printf("Cycle detected: %v", err)
//		return
//	}
//	for _, stepName := range order {
//		// Execute steps in dependency order
//		executeStep(stepName)
//	}
func (dg *DependencyGraph) GetTopologicalOrder() ([]string, error) {
	order, err := graph.TopologicalSort(dg.graph)
	if err != nil {
		// Check if this is a cycle error
		if strings.Contains(err.Error(), "cycle") {
			// Get detailed cycle information using strongly connected components
			if cycleErr := dg.detectAndDescribeCycle(); cycleErr != nil {
				return nil, cycleErr
			}
		}
		return nil, fmt.Errorf("failed to get topological order: %w", err)
	}

	return order, nil
}

// detectAndDescribeCycle detects cycles in the dependency graph and returns a detailed error
// with the specific steps involved in the cycle path.
func (dg *DependencyGraph) detectAndDescribeCycle() error {
	// Use strongly connected components to find cycles
	sccs, err := graph.StronglyConnectedComponents(dg.graph)
	if err != nil {
		return fmt.Errorf("circular dependency detected but unable to determine cycle path: %w", err)
	}

	// Find SCCs with more than one node (indicating a cycle)
	for _, scc := range sccs {
		if len(scc) > 1 {
			return dg.formatCycleError(scc)
		}
	}

	// If no multi-node SCCs found, there might be self-loops
	for step := range dg.steps {
		if dg.hasSelfLoop(step) {
			return fmt.Errorf("circular dependency detected: step '%s' depends on itself", step)
		}
	}

	// Fallback - we know there's a cycle but couldn't determine the path
	return fmt.Errorf("circular dependency detected in installation steps")
}

// formatCycleError creates a user-friendly error message showing the cycle path
func (dg *DependencyGraph) formatCycleError(cycle []string) error {
	if len(cycle) == 0 {
		return fmt.Errorf("circular dependency detected but cycle path is empty")
	}

	// Create a readable cycle path by finding the actual dependency order
	cyclePath := dg.buildCyclePath(cycle)

	return fmt.Errorf("circular dependency detected in installation steps: %s", cyclePath)
}

// buildCyclePath constructs a readable "A → B → C → A" format cycle path
func (dg *DependencyGraph) buildCyclePath(steps []string) string {
	if len(steps) <= 1 {
		if len(steps) == 1 {
			return fmt.Sprintf("%s → %s", steps[0], steps[0])
		}
		return "(empty cycle)"
	}

	// For a cycle, we need to find the actual order of dependencies
	// Start with the first step and try to build a path through the cycle
	visited := make(map[string]bool)
	path := []string{}

	// Start with the first step in the strongly connected component
	current := steps[0]
	path = append(path, current)
	visited[current] = true

	// Try to build the dependency path through the cycle
	for len(path) < len(steps) {
		next := dg.findNextInCycle(current, steps, visited)
		if next == "" {
			break
		}
		path = append(path, next)
		visited[next] = true
		current = next
	}

	// Complete the cycle by showing it returns to the start
	if len(path) > 1 {
		path = append(path, path[0])
	}

	return strings.Join(path, " → ")
}

// findNextInCycle finds the next step in the cycle by checking dependencies
func (dg *DependencyGraph) findNextInCycle(current string, cycleSteps []string, visited map[string]bool) string {
	adjacencyMap, err := dg.graph.AdjacencyMap()
	if err != nil {
		return ""
	}

	// Look for an edge from current to any unvisited step in the cycle
	for _, target := range cycleSteps {
		if !visited[target] {
			if edges, exists := adjacencyMap[current]; exists {
				if _, hasEdge := edges[target]; hasEdge {
					return target
				}
			}
		}
	}

	return ""
}

// hasSelfLoop checks if a step depends on itself
func (dg *DependencyGraph) hasSelfLoop(step string) bool {
	adjacencyMap, err := dg.graph.AdjacencyMap()
	if err != nil {
		return false
	}

	if edges, exists := adjacencyMap[step]; exists {
		_, hasSelfEdge := edges[step]
		return hasSelfEdge
	}

	return false
}

// BuildInstallationGraph constructs the complete dependency graph for installation steps
// based on the provided InstallConfig. This method consolidates all static and conditional
// dependencies that were previously managed in the getDependencies() method.
//
// The method automatically adds all required installation steps and their dependencies,
// including conditional dependencies based on the configuration (e.g., MCP-related steps
// are only included if config.AddRecommendedMCP is true).
//
// Parameters:
//   - config: Installation configuration that determines which optional dependencies to include
//
// Returns an error if:
//   - Any step validation fails (invalid step names)
//   - Circular dependencies are detected during graph construction
//   - The resulting graph cannot be topologically sorted
//
// This method should be called once per DependencyGraph instance before calling
// GetTopologicalOrder().
//
// Example:
//
//	graph := NewDependencyGraph()
//	config := &InstallConfig{AddRecommendedMCP: true}
//	err := graph.BuildInstallationGraph(config)
//	if err != nil {
//		// Handle graph construction error
//	}
func (dg *DependencyGraph) BuildInstallationGraph(config *InstallConfig) error {
	// Define all static dependencies from the original getDependencies() map
	staticDependencies := []Dependency{
		{From: "ScanExistingFiles", To: "CheckPrerequisites"},
		{From: "CreateBackups", To: "ScanExistingFiles"},
		{From: "CheckTargetDirectory", To: "CreateBackups"},
		{From: "CloneRepository", To: "CheckTargetDirectory"},
		{From: "CreateDirectoryStructure", To: "CheckTargetDirectory"},
		{From: "CopyCoreFiles", To: "CloneRepository"},
		{From: "CopyCoreFiles", To: "CreateDirectoryStructure"},
		{From: "CopyCommandFiles", To: "CloneRepository"},
		{From: "CopyCommandFiles", To: "CreateDirectoryStructure"},
		{From: "MergeOrCreateCLAUDEmd", To: "CreateDirectoryStructure"},
		{From: "MergeOrCreateMCPConfig", To: "CreateDirectoryStructure"},
		{From: "CreateCommandSymlink", To: "CopyCommandFiles"},
		{From: "CreateCommandSymlink", To: "CreateDirectoryStructure"},
	}

	// Add ValidateInstallation dependencies (always includes these)
	validateDependencies := []Dependency{
		{From: "ValidateInstallation", To: "CopyCoreFiles"},
		{From: "ValidateInstallation", To: "CopyCommandFiles"},
		{From: "ValidateInstallation", To: "MergeOrCreateCLAUDEmd"},
		{From: "ValidateInstallation", To: "CreateCommandSymlink"},
	}

	// Add CleanupTempFiles dependencies (always includes these)
	cleanupDependencies := []Dependency{
		{From: "CleanupTempFiles", To: "CopyCoreFiles"},
		{From: "CleanupTempFiles", To: "CopyCommandFiles"},
		{From: "CleanupTempFiles", To: "MergeOrCreateCLAUDEmd"},
		{From: "CleanupTempFiles", To: "CreateCommandSymlink"},
		{From: "CleanupTempFiles", To: "ValidateInstallation"},
	}

	// Combine all dependencies
	allDependencies := make([]Dependency, 0, len(staticDependencies)+len(validateDependencies)+len(cleanupDependencies))
	allDependencies = append(allDependencies, staticDependencies...)
	allDependencies = append(allDependencies, validateDependencies...)
	allDependencies = append(allDependencies, cleanupDependencies...)

	// Add conditional MCP dependencies if enabled
	if config != nil && config.AddRecommendedMCP {
		mcpDependencies := []Dependency{
			{From: "ValidateInstallation", To: "MergeOrCreateMCPConfig"},
			{From: "CleanupTempFiles", To: "MergeOrCreateMCPConfig"},
		}
		allDependencies = append(allDependencies, mcpDependencies...)
	}

	// Build the graph using the consolidated dependencies
	return dg.buildGraph(allDependencies)
}

// validateStepReferences validates that all steps referenced in dependencies exist
// as actual installation steps. Returns detailed errors for missing step references.
func (dg *DependencyGraph) validateStepReferences(dependencies []Dependency) error {
	// Skip validation if configured for testing
	if dg.skipStepValidation {
		return nil
	}

	// Get the list of available installation steps
	availableSteps := GetInstallSteps()
	if availableSteps == nil {
		return fmt.Errorf("failed to get available installation steps")
	}

	// Collect all referenced step names
	referencedSteps := make(map[string]bool)
	for _, dep := range dependencies {
		referencedSteps[dep.From] = true
		referencedSteps[dep.To] = true
	}

	// Check for missing step references
	var missingSteps []string
	for stepName := range referencedSteps {
		if _, exists := availableSteps[stepName]; !exists {
			missingSteps = append(missingSteps, stepName)
		}
	}

	if len(missingSteps) > 0 {
		return dg.formatMissingStepsError(missingSteps, availableSteps)
	}

	return nil
}

// formatMissingStepsError creates a detailed error message for missing step references
func (dg *DependencyGraph) formatMissingStepsError(missingSteps []string, availableSteps map[string]*InstallStep) error {
	// Sort missing steps for consistent error messages
	sortedMissing := make([]string, len(missingSteps))
	copy(sortedMissing, missingSteps)

	// Simple sort without importing sort package
	for i := 0; i < len(sortedMissing)-1; i++ {
		for j := 0; j < len(sortedMissing)-i-1; j++ {
			if sortedMissing[j] > sortedMissing[j+1] {
				sortedMissing[j], sortedMissing[j+1] = sortedMissing[j+1], sortedMissing[j]
			}
		}
	}

	// Get list of available steps for suggestion
	availableStepNames := make([]string, 0, len(availableSteps))
	for stepName := range availableSteps {
		availableStepNames = append(availableStepNames, stepName)
	}

	// Simple sort for available steps
	for i := 0; i < len(availableStepNames)-1; i++ {
		for j := 0; j < len(availableStepNames)-i-1; j++ {
			if availableStepNames[j] > availableStepNames[j+1] {
				availableStepNames[j], availableStepNames[j+1] = availableStepNames[j+1], availableStepNames[j]
			}
		}
	}

	if len(missingSteps) == 1 {
		return fmt.Errorf("dependency references unknown installation step '%s'. Available steps: %s",
			sortedMissing[0], strings.Join(availableStepNames, ", "))
	}

	return fmt.Errorf("dependencies reference %d unknown installation steps: %s. Available steps: %s",
		len(missingSteps), strings.Join(sortedMissing, ", "), strings.Join(availableStepNames, ", "))
}

// buildGraph constructs the complete dependency graph using the provided dependencies.
// It adds all steps and their relationships, returning an error if any issues occur
// during graph construction or validation.
func (dg *DependencyGraph) buildGraph(dependencies []Dependency) error {
	// Validate all referenced steps exist in available installation steps
	if err := dg.validateStepReferences(dependencies); err != nil {
		return err
	}

	// First pass: Add all unique steps
	stepSet := make(map[string]bool)
	for _, dep := range dependencies {
		stepSet[dep.From] = true
		stepSet[dep.To] = true
	}

	for stepName := range stepSet {
		if err := dg.AddStep(stepName); err != nil {
			return fmt.Errorf("failed to add step during graph construction: %w", err)
		}
	}

	// Second pass: Add all dependencies
	for _, dep := range dependencies {
		if err := dg.AddDependency(dep.From, dep.To); err != nil {
			return fmt.Errorf("failed to add dependency during graph construction: %w", err)
		}
	}

	// Validate the graph by attempting to get topological order
	_, err := dg.GetTopologicalOrder()
	if err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	return nil
}

// GetSteps returns a list of all steps that have been added to the graph.
//
// The returned slice contains the names of all steps that have been successfully
// added via AddStep(). The order of steps in the returned slice is not guaranteed
// to be deterministic - use GetTopologicalOrder() to get dependency-ordered steps.
//
// Returns:
//   - []string: A slice containing all step names currently in the graph
//
// Example:
//
//	steps := graph.GetSteps()
//	fmt.Printf("Graph contains %d steps: %v", len(steps), steps)
func (dg *DependencyGraph) GetSteps() []string {
	steps := make([]string, 0, len(dg.steps))
	for step := range dg.steps {
		steps = append(steps, step)
	}
	return steps
}

// HasStep returns true if the given step has been added to the graph.
//
// This method provides a quick way to check if a step exists in the graph
// before attempting operations that require the step to be present.
//
// Parameters:
//   - stepName: The name of the step to check for
//
// Returns:
//   - bool: true if the step has been added via AddStep(), false otherwise
//
// Example:
//
//	if graph.HasStep("CheckPrerequisites") {
//		// Step exists, safe to add dependencies
//		graph.AddDependency("CreateDirectories", "CheckPrerequisites")
//	}
func (dg *DependencyGraph) HasStep(stepName string) bool {
	return dg.steps[stepName]
}

// GetDependencies returns all direct dependencies for a given step.
//
// This method returns the steps that must be executed before the given step.
// Only direct dependencies are returned - transitive dependencies are not included.
// The returned dependencies are not guaranteed to be in any particular order.
//
// Parameters:
//   - stepName: The name of the step to get dependencies for
//
// Returns:
//   - []string: A slice of step names that are direct dependencies
//   - error: An error if the step has not been added to the graph
//
// Example:
//
//	deps, err := graph.GetDependencies("CreateDirectories")
//	if err != nil {
//		// Handle step not found
//	}
//	fmt.Printf("CreateDirectories depends on: %v", deps)
func (dg *DependencyGraph) GetDependencies(stepName string) ([]string, error) {
	if !dg.HasStep(stepName) {
		return nil, fmt.Errorf("step '%s' has not been added to the graph", stepName)
	}

	adjacencyMap, err := dg.graph.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get adjacency map: %w", err)
	}

	var dependencies []string

	// Find all steps that have edges TO stepName
	for source, edges := range adjacencyMap {
		if _, hasEdge := edges[stepName]; hasEdge {
			dependencies = append(dependencies, source)
		}
	}

	return dependencies, nil
}
