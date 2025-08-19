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
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		graph:              graph.New(graph.StringHash, graph.Directed(), graph.Acyclic()),
		steps:              make(map[string]bool),
		skipStepValidation: false,
	}
}

// NewTestDependencyGraph creates a DependencyGraph instance for testing
// that skips installation step validation, allowing arbitrary step names.
func NewTestDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		graph:              graph.New(graph.StringHash, graph.Directed(), graph.Acyclic()),
		steps:              make(map[string]bool),
		skipStepValidation: true,
	}
}

// AddStep adds an installation step as a vertex to the dependency graph.
// Returns an error if the step has already been added.
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
// The 'from' step depends on the 'to' step, meaning 'to' must be executed before 'from'.
// Both steps must have been added via AddStep() before calling this method.
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
// Returns an error with detailed cycle information if the graph contains cycles.
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
func (dg *DependencyGraph) GetSteps() []string {
	steps := make([]string, 0, len(dg.steps))
	for step := range dg.steps {
		steps = append(steps, step)
	}
	return steps
}

// HasStep returns true if the given step has been added to the graph.
func (dg *DependencyGraph) HasStep(stepName string) bool {
	return dg.steps[stepName]
}

// GetDependencies returns all direct dependencies for a given step.
// Returns an error if the step has not been added to the graph.
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
