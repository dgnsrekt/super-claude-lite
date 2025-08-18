# Product Requirements Document: DAG-based Installer Refactoring

## Executive Summary

This document outlines the requirements for refactoring the super-claude-lite installer to use a proper Directed Acyclic Graph (DAG) library instead of the current manual dependency resolution system.

## Problem Statement

The current installer implementation uses a manual dependency resolution algorithm that:
- Failed to detect circular dependencies at build time (discovered in issue #1)
- Uses ad-hoc dependency definitions spread across static maps and conditional logic
- Implements a custom topological sort without proper cycle detection
- Makes debugging dependency issues difficult

### Recent Issue Example
A circular dependency was introduced where `CleanupTempFiles` depended on `ValidateInstallation`, which transitively created a cycle. This was only detected at runtime with a generic error message.

## Proposed Solution

Replace the manual dependency resolution system with the `github.com/dominikbraun/graph` library that provides:
- Build-time cycle detection
- Automatic topological sorting
- Clear dependency declarations
- Better error messages

## Selected Library: github.com/dominikbraun/graph

### Why This Library

- **Zero external dependencies**: Keeps our dependency tree clean
- **Apache-2.0 license**: Permissive and compatible with our project
- **Built-in `Acyclic()` option**: Prevents cycles at construction time
- **Clean, intuitive API**: Easy to understand and maintain
- **Generic support**: Flexible for future enhancements
- **Well-documented**: Good examples and clear documentation

### Example Usage
```go
g := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic())
g.AddVertex("CheckPrerequisites")
g.AddVertex("ScanExistingFiles")
g.AddEdge("ScanExistingFiles", "CheckPrerequisites")
order, _ := graph.TopologicalSort(g)
```

## Technical Requirements

### Functional Requirements

1. **Dependency Declaration**
   - All step dependencies must be declared in a single, centralized location
   - Support for conditional dependencies (e.g., MCP-related steps)
   - Clear separation between step definition and dependency declaration

2. **Cycle Detection**
   - Cycles must be detected at graph construction time
   - Error messages must clearly identify the cycle path
   - Build should fail fast on cycle detection

3. **Execution Order**
   - Automatic topological sorting of steps
   - Support for parallel execution of independent steps (future enhancement)
   - Deterministic ordering when multiple valid orders exist

4. **Error Handling**
   - Clear error messages for missing dependencies
   - Detailed cycle path reporting
   - Validation that all referenced steps exist

### Non-Functional Requirements

1. **Performance**
   - Graph construction should be negligible (<1ms for ~15 nodes)
   - Topological sort should be instant for our graph size

2. **Maintainability**
   - Clear separation of concerns
   - Easy to add/remove/modify steps
   - Unit testable dependency graph

3. **Compatibility**
   - No breaking changes to existing step definitions
   - Preserve existing logging and error reporting
   - Maintain dry-run functionality

## Implementation Plan

### Phase 1: Core Refactoring
1. Add `github.com/dominikbraun/graph` to go.mod
2. Create new `dependency_graph.go` file with DAG builder
3. Migrate existing dependencies to DAG construction
4. Replace manual traversal with topological sort
5. Add comprehensive unit tests

### Phase 2: Enhanced Features
1. Add visualization of dependency graph (optional)
2. Support for parallel step execution
3. Better progress reporting with dependency chains

### Migration Strategy

1. **Backward Compatibility**
   - Keep existing InstallStep structure
   - Maintain current public API
   - Preserve existing configuration options

2. **Testing Strategy**
   - Unit tests for graph construction
   - Tests for cycle detection
   - Integration tests with various configurations
   - Regression tests for reported issues

3. **Rollout Plan**
   - Implement in feature branch
   - Extensive testing including the claude-baseline scenario
   - Beta release as v0.3.0-beta
   - Stable release as v0.3.0

## Success Metrics

1. **Correctness**
   - Zero circular dependency issues in production
   - All existing installation scenarios work correctly

2. **Developer Experience**
   - Clearer error messages for dependency issues
   - Easier to understand and modify dependency structure
   - Reduced debugging time for dependency-related issues

3. **Code Quality**
   - Reduced lines of code in installer.go
   - Higher test coverage for dependency logic
   - Cleaner separation of concerns

## Example Implementation

```go
// dependency_graph.go
package installer

import (
    "fmt"
    "github.com/dominikbraun/graph"
)

type DependencyGraph struct {
    g graph.Graph[string, string]
    config *InstallConfig
}

func NewDependencyGraph(config *InstallConfig) (*DependencyGraph, error) {
    g := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic())
    
    dg := &DependencyGraph{
        g: g,
        config: config,
    }
    
    if err := dg.buildGraph(); err != nil {
        return nil, err
    }
    
    return dg, nil
}

func (dg *DependencyGraph) buildGraph() error {
    // Add all vertices
    steps := []string{
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
    
    for _, step := range steps {
        if err := dg.g.AddVertex(step); err != nil {
            return fmt.Errorf("failed to add vertex %s: %w", step, err)
        }
    }
    
    // Add static dependencies
    dependencies := []struct{from, to string}{
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
        {"ValidateInstallation", "CopyCoreFiles"},
        {"ValidateInstallation", "CopyCommandFiles"},
        {"ValidateInstallation", "MergeOrCreateCLAUDEmd"},
        {"ValidateInstallation", "CreateCommandSymlink"},
        {"CleanupTempFiles", "ValidateInstallation"},
    }
    
    for _, dep := range dependencies {
        if err := dg.g.AddEdge(dep.from, dep.to); err != nil {
            return fmt.Errorf("circular dependency detected: %s -> %s: %w", 
                dep.from, dep.to, err)
        }
    }
    
    // Add conditional dependencies
    if dg.config.AddRecommendedMCP {
        if err := dg.g.AddEdge("ValidateInstallation", "MergeOrCreateMCPConfig"); err != nil {
            return fmt.Errorf("failed to add MCP dependency: %w", err)
        }
        if err := dg.g.AddEdge("CleanupTempFiles", "MergeOrCreateMCPConfig"); err != nil {
            return fmt.Errorf("failed to add MCP cleanup dependency: %w", err)
        }
    }
    
    return nil
}

func (dg *DependencyGraph) GetExecutionOrder() ([]string, error) {
    order, err := graph.TopologicalSort(dg.g)
    if err != nil {
        return nil, fmt.Errorf("failed to determine execution order: %w", err)
    }
    return order, nil
}

func (dg *DependencyGraph) ValidateNoCycles() error {
    // The Acyclic() option prevents cycles at construction,
    // but we can add explicit validation if needed
    _, err := graph.TopologicalSort(dg.g)
    if err != nil {
        return fmt.Errorf("dependency graph contains cycles: %w", err)
    }
    return nil
}
```

## Risk Analysis

### Risks
1. **API Stability**: Library is currently v0.x and may have breaking changes
   - Mitigation: Pin to specific version, consider vendoring if needed

2. **Unknown Edge Cases**: Current manual system may handle cases not documented
   - Mitigation: Extensive testing, beta period, maintain test coverage

3. **Learning Curve**: Team needs to understand new library
   - Mitigation: Good documentation, clear examples in code

## Timeline

- Week 1: Library integration and core refactoring
- Week 2: Testing and bug fixes
- Week 3: Beta release and user testing
- Week 4: Stable release

## Conclusion

Refactoring to use `github.com/dominikbraun/graph` will:
- Prevent future circular dependency issues through compile-time detection
- Improve code maintainability with cleaner dependency declarations
- Provide better error messages with clear cycle identification
- Enable future enhancements like parallel execution
- Reduce manual dependency resolution code by ~60%

The investment in this refactoring will pay dividends in reduced debugging time and increased confidence in the installer's correctness.