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
	graph graph.Graph[string, string]
	steps map[string]bool // Track added steps for validation
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
		graph: graph.New(graph.StringHash, graph.Directed(), graph.Acyclic()),
		steps: make(map[string]bool),
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
// Returns an error if the graph contains cycles (which should be prevented by graph.Acyclic()).
func (dg *DependencyGraph) GetTopologicalOrder() ([]string, error) {
	order, err := graph.TopologicalSort(dg.graph)
	if err != nil {
		return nil, fmt.Errorf("failed to get topological order: %w", err)
	}

	return order, nil
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

// buildGraph constructs the complete dependency graph using the provided dependencies.
// It adds all steps and their relationships, returning an error if any issues occur
// during graph construction or validation.
func (dg *DependencyGraph) buildGraph(dependencies []Dependency) error {
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
