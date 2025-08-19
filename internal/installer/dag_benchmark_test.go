package installer

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkDAGGraphConstruction benchmarks dependency graph construction performance
func BenchmarkDAGGraphConstruction(b *testing.B) {
	// Test configuration that covers typical installation scenarios
	config := &InstallConfig{
		Force:             false,
		NoBackup:          false,
		Interactive:       false,
		AddRecommendedMCP: true, // Full scenario with MCP
		BackupDir:         "",
	}

	b.Run("Construction_Full_13Steps", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dg := NewDependencyGraph()
			err := dg.BuildInstallationGraph(config)
			if err != nil {
				b.Fatalf("Failed to build installation graph: %v", err)
			}
		}
	})

	b.Run("Construction_Minimal_12Steps", func(b *testing.B) {
		minimalConfig := &InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false, // Minimal scenario without MCP
			BackupDir:         "",
		}

		for i := 0; i < b.N; i++ {
			dg := NewDependencyGraph()
			err := dg.BuildInstallationGraph(minimalConfig)
			if err != nil {
				b.Fatalf("Failed to build installation graph: %v", err)
			}
		}
	})
}

// BenchmarkGraphOperations benchmarks individual graph operations
func BenchmarkGraphOperations(b *testing.B) {
	b.Run("AddStep_Individual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			dg := NewTestDependencyGraph()
			b.StartTimer()

			err := dg.AddStep(fmt.Sprintf("Step_%d", i))
			if err != nil {
				b.Fatalf("Failed to add step: %v", err)
			}
		}
	})

	b.Run("AddStep_Batch_15Steps", func(b *testing.B) {
		stepNames := []string{
			"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
			"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
			"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
			"MergeOrCreateMCPConfig", "CreateCommandSymlink", "ValidateInstallation",
			"CleanupTempFiles", "ExtraStep1", "ExtraStep2",
		}

		for i := 0; i < b.N; i++ {
			dg := NewTestDependencyGraph()

			for _, stepName := range stepNames {
				err := dg.AddStep(stepName)
				if err != nil {
					b.Fatalf("Failed to add step %s: %v", stepName, err)
				}
			}
		}
	})

	b.Run("AddDependency_Individual", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			dg := NewTestDependencyGraph()
			_ = dg.AddStep("StepA")
			_ = dg.AddStep("StepB")
			b.StartTimer()

			err := dg.AddDependency("StepB", "StepA")
			if err != nil {
				b.Fatalf("Failed to add dependency: %v", err)
			}
		}
	})

	b.Run("AddDependency_Batch_InstallationGraph", func(b *testing.B) {
		// Create test dependencies similar to real installation graph
		testDependencies := []Dependency{
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
			{From: "ValidateInstallation", To: "CopyCoreFiles"},
			{From: "ValidateInstallation", To: "CopyCommandFiles"},
			{From: "ValidateInstallation", To: "MergeOrCreateCLAUDEmd"},
			{From: "ValidateInstallation", To: "MergeOrCreateMCPConfig"},
			{From: "ValidateInstallation", To: "CreateCommandSymlink"},
			{From: "CleanupTempFiles", To: "ValidateInstallation"},
		}

		for i := 0; i < b.N; i++ {
			dg := NewTestDependencyGraph()

			// Add all steps first
			stepSet := make(map[string]bool)
			for _, dep := range testDependencies {
				stepSet[dep.From] = true
				stepSet[dep.To] = true
			}

			for stepName := range stepSet {
				_ = dg.AddStep(stepName)
			}

			// Add all dependencies
			for _, dep := range testDependencies {
				err := dg.AddDependency(dep.From, dep.To)
				if err != nil {
					b.Fatalf("Failed to add dependency %s -> %s: %v", dep.From, dep.To, err)
				}
			}
		}
	})
}

// BenchmarkDAGTopologicalSort benchmarks topological sort performance
func BenchmarkDAGTopologicalSort(b *testing.B) {
	b.Run("TopologicalSort_InstallationGraph_13Steps", func(b *testing.B) {
		// Pre-build the graph once
		dg := NewDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: true}
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			b.Fatalf("Failed to build installation graph: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dg.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get topological order: %v", err)
			}
		}
	})

	b.Run("TopologicalSort_MinimalGraph_12Steps", func(b *testing.B) {
		// Pre-build the minimal graph
		dg := NewDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: false}
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			b.Fatalf("Failed to build installation graph: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dg.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get topological order: %v", err)
			}
		}
	})

	b.Run("TopologicalSort_LinearChain_15Steps", func(b *testing.B) {
		// Create a linear dependency chain for comparison
		dg := NewTestDependencyGraph()
		stepCount := 15

		// Add steps
		for i := 0; i < stepCount; i++ {
			_ = dg.AddStep(fmt.Sprintf("Step_%d", i))
		}

		// Add linear dependencies (Step_1 -> Step_0, Step_2 -> Step_1, etc.)
		for i := 1; i < stepCount; i++ {
			_ = dg.AddDependency(fmt.Sprintf("Step_%d", i), fmt.Sprintf("Step_%d", i-1))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dg.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get topological order: %v", err)
			}
		}
	})

	b.Run("TopologicalSort_TreeStructure_15Steps", func(b *testing.B) {
		// Create a tree-like dependency structure
		dg := NewTestDependencyGraph()

		// Add root
		_ = dg.AddStep("Root")

		// Add level 1 (4 steps depending on root)
		for i := 0; i < 4; i++ {
			stepName := fmt.Sprintf("L1_Step_%d", i)
			_ = dg.AddStep(stepName)
			_ = dg.AddDependency(stepName, "Root")
		}

		// Add level 2 (10 steps, 2-3 depending on each L1 step)
		stepIndex := 0
		for i := 0; i < 4; i++ {
			numChildren := 2
			if i < 2 {
				numChildren = 3 // First two L1 steps have 3 children each
			}

			for j := 0; j < numChildren; j++ {
				stepName := fmt.Sprintf("L2_Step_%d", stepIndex)
				_ = dg.AddStep(stepName)
				_ = dg.AddDependency(stepName, fmt.Sprintf("L1_Step_%d", i))
				stepIndex++
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dg.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get topological order: %v", err)
			}
		}
	})
}

// BenchmarkCompleteDAGWorkflow benchmarks the full DAG workflow
func BenchmarkCompleteDAGWorkflow(b *testing.B) {
	b.Run("FullWorkflow_Build_And_Sort", func(b *testing.B) {
		config := &InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: true,
			BackupDir:         "",
		}

		for i := 0; i < b.N; i++ {
			// Build graph
			dg := NewDependencyGraph()
			err := dg.BuildInstallationGraph(config)
			if err != nil {
				b.Fatalf("Failed to build installation graph: %v", err)
			}

			// Get topological order
			_, err = dg.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get topological order: %v", err)
			}
		}
	})
}

// TestPerformanceTarget validates that performance targets are met
func TestPerformanceTarget(t *testing.T) {
	t.Run("TopologicalSort_PerformanceTarget_1ms", func(t *testing.T) {
		// Build full installation graph
		dg := NewDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: true}
		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build installation graph: %v", err)
		}

		// Warm up
		for i := 0; i < 10; i++ {
			_, _ = dg.GetTopologicalOrder()
		}

		// Measure multiple runs to get reliable timing
		const iterations = 100
		totalTime := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			_, err := dg.GetTopologicalOrder()
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Failed to get topological order: %v", err)
			}

			totalTime += duration
		}

		averageTime := totalTime / iterations
		targetTime := time.Millisecond // 1ms target

		t.Logf("Topological sort performance:")
		t.Logf("  Average time: %v", averageTime)
		t.Logf("  Target time: %v", targetTime)
		t.Logf("  Graph size: %d steps", len(dg.GetSteps()))

		if averageTime > targetTime {
			t.Errorf("Performance target not met: average %v > target %v", averageTime, targetTime)
		} else {
			t.Logf("✓ Performance target met: %v < %v", averageTime, targetTime)
		}
	})

	t.Run("GraphConstruction_PerformanceTarget_1ms", func(t *testing.T) {
		config := &InstallConfig{AddRecommendedMCP: true}

		// Warm up
		for i := 0; i < 10; i++ {
			dg := NewDependencyGraph()
			_ = dg.BuildInstallationGraph(config)
		}

		// Measure construction time
		const iterations = 100
		totalTime := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			dg := NewDependencyGraph()
			err := dg.BuildInstallationGraph(config)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Failed to build installation graph: %v", err)
			}

			totalTime += duration
		}

		averageTime := totalTime / iterations
		targetTime := time.Millisecond // 1ms target for construction

		t.Logf("Graph construction performance:")
		t.Logf("  Average time: %v", averageTime)
		t.Logf("  Target time: %v", targetTime)

		if averageTime > targetTime {
			t.Errorf("Performance target not met: average %v > target %v", averageTime, targetTime)
		} else {
			t.Logf("✓ Performance target met: %v < %v", averageTime, targetTime)
		}
	})

	t.Run("CombinedWorkflow_PerformanceTarget_2ms", func(t *testing.T) {
		config := &InstallConfig{AddRecommendedMCP: true}

		// Warm up
		for i := 0; i < 10; i++ {
			dg := NewDependencyGraph()
			_ = dg.BuildInstallationGraph(config)
			_, _ = dg.GetTopologicalOrder()
		}

		// Measure combined construction + sort time
		const iterations = 100
		totalTime := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()

			// Build graph
			dg := NewDependencyGraph()
			err := dg.BuildInstallationGraph(config)
			if err != nil {
				t.Fatalf("Failed to build installation graph: %v", err)
			}

			// Get topological order
			_, err = dg.GetTopologicalOrder()
			if err != nil {
				t.Fatalf("Failed to get topological order: %v", err)
			}

			duration := time.Since(start)
			totalTime += duration
		}

		averageTime := totalTime / iterations
		targetTime := 2 * time.Millisecond // 2ms target for combined workflow

		t.Logf("Combined workflow performance:")
		t.Logf("  Average time: %v", averageTime)
		t.Logf("  Target time: %v", targetTime)

		if averageTime > targetTime {
			t.Errorf("Performance target not met: average %v > target %v", averageTime, targetTime)
		} else {
			t.Logf("✓ Performance target met: %v < %v", averageTime, targetTime)
		}
	})
}

// TestScalabilityCharacteristics tests how the DAG operations scale with graph size
func TestScalabilityCharacteristics(t *testing.T) {
	graphSizes := []int{5, 10, 15, 20, 30, 50}

	for _, size := range graphSizes {
		t.Run(fmt.Sprintf("Scalability_LinearChain_%dSteps", size), func(t *testing.T) {
			// Create linear dependency chain
			dg := NewTestDependencyGraph()

			// Add steps
			for i := 0; i < size; i++ {
				_ = dg.AddStep(fmt.Sprintf("Step_%d", i))
			}

			// Add linear dependencies
			for i := 1; i < size; i++ {
				_ = dg.AddDependency(fmt.Sprintf("Step_%d", i), fmt.Sprintf("Step_%d", i-1))
			}

			// Measure topological sort time
			const iterations = 50
			totalTime := time.Duration(0)

			for i := 0; i < iterations; i++ {
				start := time.Now()
				_, err := dg.GetTopologicalOrder()
				duration := time.Since(start)

				if err != nil {
					t.Fatalf("Failed to get topological order: %v", err)
				}

				totalTime += duration
			}

			averageTime := totalTime / iterations
			t.Logf("Graph size %d: average sort time %v", size, averageTime)

			// For linear chains, topological sort should remain very fast even at larger sizes
			if size <= 20 && averageTime > time.Millisecond {
				t.Errorf("Sort time too slow for size %d: %v > 1ms", size, averageTime)
			}
		})
	}
}
