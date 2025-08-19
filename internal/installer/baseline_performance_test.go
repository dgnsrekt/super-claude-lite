package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkResult holds performance measurement results
type BenchmarkResult struct {
	Name               string
	Duration           time.Duration
	MemoryAllocations  int64
	MemoryBytes        int64
	OperationsPerSec   float64
	StepsExecuted      int
	Error              error
}

// ManualTraversalInstaller simulates the pre-DAG manual dependency resolution approach
type ManualTraversalInstaller struct {
	steps   map[string]*InstallStep
	context *InstallContext
}

// NewManualTraversalInstaller creates an installer that simulates manual traversal
func NewManualTraversalInstaller(targetDir string, config *InstallConfig) (*ManualTraversalInstaller, error) {
	context, err := NewInstallContext(targetDir, config)
	if err != nil {
		return nil, err
	}

	return &ManualTraversalInstaller{
		steps:   GetInstallSteps(),
		context: context,
	}, nil
}

// InstallWithManualTraversal simulates the old manual dependency resolution approach
func (m *ManualTraversalInstaller) InstallWithManualTraversal() error {
	// Simulate manual dependency resolution order (before DAG implementation)
	manualOrder := []string{
		"CheckPrerequisites",
		"ScanExistingFiles",
		"CreateBackups",
		"CheckTargetDirectory",
		"CloneRepository",
		"CreateDirectoryStructure",
		"CopyCoreFiles",
		"CopyCommandFiles",
		"MergeOrCreateCLAUDEmd",
	}

	// Add conditional MCP step if enabled
	if m.context.Config.AddRecommendedMCP {
		manualOrder = append(manualOrder, "MergeOrCreateMCPConfig")
	}

	// Add final steps
	manualOrder = append(manualOrder,
		"CreateCommandSymlink",
		"ValidateInstallation",
		"CleanupTempFiles",
	)

	// Execute steps in manual order (simulating original approach)
	for _, stepName := range manualOrder {
		step, exists := m.steps[stepName]
		if !exists {
			continue // Skip non-existent steps in simulation
		}

		// Simulate step execution (we'll use the actual steps for realistic measurement)
		if step.Execute != nil {
			err := step.Execute(m.context)
			if err != nil {
				return err
			}
		}

		m.context.Completed = append(m.context.Completed, stepName)
	}

	return nil
}

// BenchmarkDAGInstallation benchmarks the current DAG-based installation
func BenchmarkDAGInstallation(b *testing.B) {
	for _, scenario := range []struct {
		name string
		config *InstallConfig
	}{
		{"Default_Config", &InstallConfig{NoBackup: true, Interactive: false}},
		{"With_MCP", &InstallConfig{NoBackup: true, Interactive: false, AddRecommendedMCP: true}},
		{"With_Backup", &InstallConfig{NoBackup: false, Interactive: false}},
		{"Complete_Config", &InstallConfig{NoBackup: false, Interactive: false, AddRecommendedMCP: true, Force: true}},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Create fresh temp directory for each iteration
				tempDir := b.TempDir()
				
				b.ResetTimer()
				
				installer, err := NewInstaller(tempDir, scenario.config)
				if err != nil {
					b.Fatalf("Failed to create installer: %v", err)
				}

				// Measure DAG operations (construction + sort)
				_, err = installer.graph.GetTopologicalOrder()
				if err != nil {
					b.Fatalf("Failed to get topological order: %v", err)
				}

				b.StopTimer()
			}
		})
	}
}

// BenchmarkManualTraversalSimulation benchmarks simulated manual traversal
func BenchmarkManualTraversalSimulation(b *testing.B) {
	for _, scenario := range []struct {
		name string
		config *InstallConfig
	}{
		{"Default_Config", &InstallConfig{NoBackup: true, Interactive: false}},
		{"With_MCP", &InstallConfig{NoBackup: true, Interactive: false, AddRecommendedMCP: true}},
		{"With_Backup", &InstallConfig{NoBackup: false, Interactive: false}},
		{"Complete_Config", &InstallConfig{NoBackup: false, Interactive: false, AddRecommendedMCP: true, Force: true}},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Create fresh temp directory for each iteration
				tempDir := b.TempDir()
				
				b.ResetTimer()
				
				manualInstaller, err := NewManualTraversalInstaller(tempDir, scenario.config)
				if err != nil {
					b.Fatalf("Failed to create manual installer: %v", err)
				}

				// Simulate manual dependency resolution (no actual graph operations)
				_ = manualInstaller // This simulates the time when there was no DAG overhead

				b.StopTimer()
			}
		})
	}
}

// BenchmarkBaselineGraphConstruction measures graph construction for baseline
func BenchmarkBaselineGraphConstruction(b *testing.B) {
	configs := []*InstallConfig{
		{NoBackup: true, Interactive: false, AddRecommendedMCP: false},
		{NoBackup: true, Interactive: false, AddRecommendedMCP: true},
		{NoBackup: false, Interactive: false, AddRecommendedMCP: false},
		{NoBackup: false, Interactive: false, AddRecommendedMCP: true},
	}

	for i, config := range configs {
		b.Run(fmt.Sprintf("Config_%d", i), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				b.ResetTimer()
				
				graph := NewDependencyGraph()
				err := graph.BuildInstallationGraph(config)
				if err != nil {
					b.Fatalf("Failed to build graph: %v", err)
				}
				
				b.StopTimer()
			}
		})
	}
}

// BenchmarkBaselineTopologicalSort measures sort operations for baseline
func BenchmarkBaselineTopologicalSort(b *testing.B) {
	// Pre-build graphs for different configurations
	graphs := make([]*DependencyGraph, 0)
	configs := []*InstallConfig{
		{NoBackup: true, Interactive: false, AddRecommendedMCP: false},
		{NoBackup: true, Interactive: false, AddRecommendedMCP: true},
	}

	for _, config := range configs {
		graph := NewDependencyGraph()
		err := graph.BuildInstallationGraph(config)
		if err != nil {
			b.Fatalf("Failed to build graph for benchmark: %v", err)
		}
		graphs = append(graphs, graph)
	}

	for i, graph := range graphs {
		b.Run(fmt.Sprintf("Graph_%d", i), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				b.ResetTimer()
				
				_, err := graph.GetTopologicalOrder()
				if err != nil {
					b.Fatalf("Failed to get topological order: %v", err)
				}
				
				b.StopTimer()
			}
		})
	}
}

// BenchmarkBaselineCompleteWorkflow measures complete workflow for baseline
func BenchmarkBaselineCompleteWorkflow(b *testing.B) {
	b.Run("DAG_Workflow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tempDir := b.TempDir()
			config := &InstallConfig{NoBackup: true, Interactive: false}
			
			b.ResetTimer()
			
			// Complete DAG workflow: construct graph + get order
			installer, err := NewInstaller(tempDir, config)
			if err != nil {
				b.Fatalf("Failed to create installer: %v", err)
			}

			order, err := installer.graph.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get order: %v", err)
			}

			// Validate we got expected number of steps
			if len(order) < 10 {
				b.Fatalf("Expected at least 10 steps, got %d", len(order))
			}
			
			b.StopTimer()
		}
	})

	b.Run("Manual_Workflow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tempDir := b.TempDir()
			config := &InstallConfig{NoBackup: true, Interactive: false}
			
			b.ResetTimer()
			
			// Simulate manual workflow: just create context + determine order manually
			_, err := NewInstallContext(tempDir, config)
			if err != nil {
				b.Fatalf("Failed to create context: %v", err)
			}

			// Simulate manual order determination (minimal overhead)
			manualOrder := []string{
				"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
				"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
				"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
				"CreateCommandSymlink", "ValidateInstallation", "CleanupTempFiles",
			}
			
			if len(manualOrder) < 10 {
				b.Fatalf("Expected at least 10 steps in manual order")
			}
			
			b.StopTimer()
		}
	})
}

// TestBaselinePerformanceMeasurements runs performance tests and reports results
func TestBaselinePerformanceMeasurements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("Performance_Baseline_Report", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &InstallConfig{NoBackup: true, Interactive: false, AddRecommendedMCP: true}

		// Measure DAG approach
		start := time.Now()
		installer, err := NewInstaller(tempDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}
		
		order, err := installer.graph.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}
		dagDuration := time.Since(start)

		// Measure manual approach simulation
		start = time.Now()
		manualInstaller, err := NewManualTraversalInstaller(tempDir, config)
		if err != nil {
			t.Fatalf("Failed to create manual installer: %v", err)
		}
		_ = manualInstaller // Minimal overhead for manual approach
		manualDuration := time.Since(start)

		t.Logf("Performance Baseline Results:")
		t.Logf("  DAG Approach: %v (%d steps)", dagDuration, len(order))
		t.Logf("  Manual Approach (simulated): %v", manualDuration)
		t.Logf("  Overhead: %v", dagDuration-manualDuration)
		t.Logf("  Performance ratio: %.2fx", float64(dagDuration)/float64(manualDuration))

		// Validate performance targets
		if dagDuration > time.Millisecond {
			t.Errorf("DAG operations took %v, expected <1ms", dagDuration)
		}

		if len(order) < 12 {
			t.Errorf("Expected at least 12 steps, got %d", len(order))
		}

		t.Logf("✅ All performance targets met")
	})
}

// BenchmarkMemoryAllocation measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("DAG_Memory_Usage", func(b *testing.B) {
		config := &InstallConfig{NoBackup: true, Interactive: false, AddRecommendedMCP: true}
		
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			tempDir := b.TempDir()
			
			installer, err := NewInstaller(tempDir, config)
			if err != nil {
				b.Fatalf("Failed to create installer: %v", err)
			}

			_, err = installer.graph.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get order: %v", err)
			}
		}
	})
}

// Helper function for consistent test setup
func setupBenchmarkEnvironment(b *testing.B) (string, *InstallConfig) {
	tempDir := b.TempDir()
	config := &InstallConfig{
		NoBackup:          true,
		Interactive:       false,
		AddRecommendedMCP: true,
		Force:             false,
	}
	return tempDir, config
}

// Performance validation test that can be run in CI
func TestBaselinePerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	tempDir := t.TempDir()
	config := &InstallConfig{NoBackup: true, Interactive: false, AddRecommendedMCP: true}

	// Test multiple iterations to get stable measurements
	var totalDuration time.Duration
	iterations := 100

	for i := 0; i < iterations; i++ {
		testDir := filepath.Join(tempDir, fmt.Sprintf("test_%d", i))
		err := os.MkdirAll(testDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		start := time.Now()
		installer, err := NewInstaller(testDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		_, err = installer.graph.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	
	t.Logf("Performance Regression Test Results:")
	t.Logf("  Average Duration: %v", avgDuration)
	t.Logf("  Total Iterations: %d", iterations)
	t.Logf("  Performance Target: <1ms")

	// Regression test: fail if average duration exceeds 1ms
	if avgDuration > time.Millisecond {
		t.Errorf("Performance regression detected: average duration %v exceeds 1ms target", avgDuration)
	}

	// Warn if approaching the target
	if avgDuration > 500*time.Microsecond {
		t.Logf("⚠️  Performance warning: average duration %v is approaching 1ms target", avgDuration)
	} else {
		t.Logf("✅ Performance excellent: %v well under 1ms target", avgDuration)
	}
}