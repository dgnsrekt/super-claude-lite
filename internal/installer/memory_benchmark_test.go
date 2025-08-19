package installer

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// MemoryMetrics captures memory usage statistics
type MemoryMetrics struct {
	AllocBytes      uint64        // Bytes allocated during operation
	TotalAllocBytes uint64        // Total bytes allocated throughout operation
	SysBytes        uint64        // Bytes obtained from OS
	NumGC           uint32        // Number of GC cycles
	PauseTotalNs    uint64        // Total GC pause time
	ExecutionTime   time.Duration // Time to complete operation
}

// measureMemoryUsage captures memory metrics around a function execution
func measureMemoryUsage(fn func() error) (*MemoryMetrics, error) {
	// Force GC and get initial memory stats
	runtime.GC()
	runtime.GC() // Call twice to ensure clean state
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	// Measure execution time
	start := time.Now()
	err := fn()
	executionTime := time.Since(start)
	
	// Get final memory stats
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	metrics := &MemoryMetrics{
		AllocBytes:      memAfter.Alloc - memBefore.Alloc,
		TotalAllocBytes: memAfter.TotalAlloc - memBefore.TotalAlloc,
		SysBytes:        memAfter.Sys - memBefore.Sys,
		NumGC:           memAfter.NumGC - memBefore.NumGC,
		PauseTotalNs:    memAfter.PauseTotalNs - memBefore.PauseTotalNs,
		ExecutionTime:   executionTime,
	}
	
	return metrics, err
}

// TestMemoryUsageGraphConstruction tests memory consumption during graph construction
func TestMemoryUsageGraphConstruction(t *testing.T) {
	scenarios := []struct {
		name   string
		config *InstallConfig
	}{
		{
			name: "Minimal_12Steps",
			config: &InstallConfig{
				Force:             false,
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
			},
		},
		{
			name: "Full_13Steps",
			config: &InstallConfig{
				Force:             false,
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			metrics, err := measureMemoryUsage(func() error {
				dg := NewDependencyGraph()
				return dg.BuildInstallationGraph(scenario.config)
			})

			if err != nil {
				t.Fatalf("Failed to build installation graph: %v", err)
			}

			t.Logf("Memory metrics for %s:", scenario.name)
			t.Logf("  Execution time: %v", metrics.ExecutionTime)
			t.Logf("  Allocated: %d bytes", metrics.AllocBytes)
			t.Logf("  Total allocated: %d bytes", metrics.TotalAllocBytes)
			t.Logf("  System memory: %d bytes", metrics.SysBytes)
			t.Logf("  GC cycles: %d", metrics.NumGC)
			if metrics.PauseTotalNs > 0 {
				t.Logf("  GC pause time: %v", time.Duration(metrics.PauseTotalNs))
			}

			// Verify reasonable memory usage (should be well under 1MB for small graphs)
			const maxExpectedMemory = 1024 * 1024 // 1MB
			if metrics.TotalAllocBytes > maxExpectedMemory {
				t.Errorf("Memory usage too high: %d bytes > %d bytes", metrics.TotalAllocBytes, maxExpectedMemory)
			}

			// Verify execution time is reasonable (should be well under 1ms)
			const maxExpectedTime = time.Millisecond
			if metrics.ExecutionTime > maxExpectedTime {
				t.Errorf("Execution time too slow: %v > %v", metrics.ExecutionTime, maxExpectedTime)
			}
		})
	}
}

// TestMemoryUsageTopologicalSort tests memory consumption during topological sort
func TestMemoryUsageTopologicalSort(t *testing.T) {
	// Pre-build graphs of different sizes
	graphs := []struct {
		name string
		dg   *DependencyGraph
	}{
		{
			name: "InstallationGraph_13Steps",
			dg: func() *DependencyGraph {
				dg := NewDependencyGraph()
				config := &InstallConfig{AddRecommendedMCP: true}
				_ = dg.BuildInstallationGraph(config)
				return dg
			}(),
		},
		{
			name: "LinearChain_15Steps",
			dg: func() *DependencyGraph {
				dg := NewTestDependencyGraph()
				for i := 0; i < 15; i++ {
					_ = dg.AddStep(fmt.Sprintf("Step_%d", i))
				}
				for i := 1; i < 15; i++ {
					_ = dg.AddDependency(fmt.Sprintf("Step_%d", i), fmt.Sprintf("Step_%d", i-1))
				}
				return dg
			}(),
		},
		{
			name: "TreeStructure_15Steps",
			dg: func() *DependencyGraph {
				dg := NewTestDependencyGraph()
				_ = dg.AddStep("Root")
				for i := 0; i < 4; i++ {
					stepName := fmt.Sprintf("L1_Step_%d", i)
					_ = dg.AddStep(stepName)
					_ = dg.AddDependency(stepName, "Root")
				}
				stepIndex := 0
				for i := 0; i < 4; i++ {
					numChildren := 2
					if i < 2 {
						numChildren = 3
					}
					for j := 0; j < numChildren; j++ {
						stepName := fmt.Sprintf("L2_Step_%d", stepIndex)
						_ = dg.AddStep(stepName)
						_ = dg.AddDependency(stepName, fmt.Sprintf("L1_Step_%d", i))
						stepIndex++
					}
				}
				return dg
			}(),
		},
	}

	for _, graph := range graphs {
		t.Run(graph.name, func(t *testing.T) {
			metrics, err := measureMemoryUsage(func() error {
				_, err := graph.dg.GetTopologicalOrder()
				return err
			})

			if err != nil {
				t.Fatalf("Failed to get topological order: %v", err)
			}

			t.Logf("Memory metrics for %s:", graph.name)
			t.Logf("  Execution time: %v", metrics.ExecutionTime)
			t.Logf("  Allocated: %d bytes", metrics.AllocBytes)
			t.Logf("  Total allocated: %d bytes", metrics.TotalAllocBytes)
			t.Logf("  System memory: %d bytes", metrics.SysBytes)
			t.Logf("  GC cycles: %d", metrics.NumGC)
			if metrics.PauseTotalNs > 0 {
				t.Logf("  GC pause time: %v", time.Duration(metrics.PauseTotalNs))
			}

			// Verify reasonable memory usage for topological sort
			const maxExpectedMemory = 512 * 1024 // 512KB should be plenty for sort
			if metrics.TotalAllocBytes > maxExpectedMemory {
				t.Errorf("Memory usage too high: %d bytes > %d bytes", metrics.TotalAllocBytes, maxExpectedMemory)
			}

			// Verify execution time meets performance target
			const maxExpectedTime = time.Millisecond
			if metrics.ExecutionTime > maxExpectedTime {
				t.Errorf("Execution time too slow: %v > %v", metrics.ExecutionTime, maxExpectedTime)
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("GraphConstruction_MemAllocs", func(b *testing.B) {
		config := &InstallConfig{AddRecommendedMCP: true}
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			dg := NewDependencyGraph()
			err := dg.BuildInstallationGraph(config)
			if err != nil {
				b.Fatalf("Failed to build graph: %v", err)
			}
		}
	})

	b.Run("TopologicalSort_MemAllocs", func(b *testing.B) {
		// Pre-build graph
		dg := NewDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: true}
		_ = dg.BuildInstallationGraph(config)
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_, err := dg.GetTopologicalOrder()
			if err != nil {
				b.Fatalf("Failed to get topological order: %v", err)
			}
		}
	})
}

// TestPerformanceRegression validates performance hasn't regressed
func TestPerformanceRegression(t *testing.T) {
	// Performance baselines established from benchmarks
	baselines := map[string]struct {
		maxExecutionTime time.Duration
		maxMemoryBytes   uint64
	}{
		"GraphConstruction": {
			maxExecutionTime: 200 * time.Microsecond, // 200µs baseline + margin
			maxMemoryBytes:   100 * 1024,             // 100KB
		},
		"TopologicalSort": {
			maxExecutionTime: 100 * time.Microsecond, // 100µs baseline + margin
			maxMemoryBytes:   50 * 1024,              // 50KB
		},
		"CombinedWorkflow": {
			maxExecutionTime: 300 * time.Microsecond, // 300µs baseline + margin
			maxMemoryBytes:   150 * 1024,             // 150KB
		},
	}

	t.Run("GraphConstruction_Regression", func(t *testing.T) {
		baseline := baselines["GraphConstruction"]
		
		// Test multiple iterations to ensure consistency
		for i := 0; i < 10; i++ {
			metrics, err := measureMemoryUsage(func() error {
				dg := NewDependencyGraph()
				config := &InstallConfig{AddRecommendedMCP: true}
				return dg.BuildInstallationGraph(config)
			})

			if err != nil {
				t.Fatalf("Iteration %d failed: %v", i, err)
			}

			if metrics.ExecutionTime > baseline.maxExecutionTime {
				t.Errorf("Performance regression in iteration %d: %v > %v", 
					i, metrics.ExecutionTime, baseline.maxExecutionTime)
			}

			if metrics.TotalAllocBytes > baseline.maxMemoryBytes {
				t.Errorf("Memory regression in iteration %d: %d bytes > %d bytes", 
					i, metrics.TotalAllocBytes, baseline.maxMemoryBytes)
			}
		}
	})

	t.Run("TopologicalSort_Regression", func(t *testing.T) {
		baseline := baselines["TopologicalSort"]
		
		// Pre-build graph
		dg := NewDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: true}
		_ = dg.BuildInstallationGraph(config)
		
		// Test multiple iterations
		for i := 0; i < 10; i++ {
			metrics, err := measureMemoryUsage(func() error {
				_, err := dg.GetTopologicalOrder()
				return err
			})

			if err != nil {
				t.Fatalf("Iteration %d failed: %v", i, err)
			}

			if metrics.ExecutionTime > baseline.maxExecutionTime {
				t.Errorf("Performance regression in iteration %d: %v > %v", 
					i, metrics.ExecutionTime, baseline.maxExecutionTime)
			}

			if metrics.TotalAllocBytes > baseline.maxMemoryBytes {
				t.Errorf("Memory regression in iteration %d: %d bytes > %d bytes", 
					i, metrics.TotalAllocBytes, baseline.maxMemoryBytes)
			}
		}
	})

	t.Run("CombinedWorkflow_Regression", func(t *testing.T) {
		baseline := baselines["CombinedWorkflow"]
		
		// Test multiple iterations of the complete workflow
		for i := 0; i < 10; i++ {
			metrics, err := measureMemoryUsage(func() error {
				dg := NewDependencyGraph()
				config := &InstallConfig{AddRecommendedMCP: true}
				
				// Build graph
				if err := dg.BuildInstallationGraph(config); err != nil {
					return err
				}
				
				// Get topological order
				_, err := dg.GetTopologicalOrder()
				return err
			})

			if err != nil {
				t.Fatalf("Iteration %d failed: %v", i, err)
			}

			if metrics.ExecutionTime > baseline.maxExecutionTime {
				t.Errorf("Performance regression in iteration %d: %v > %v", 
					i, metrics.ExecutionTime, baseline.maxExecutionTime)
			}

			if metrics.TotalAllocBytes > baseline.maxMemoryBytes {
				t.Errorf("Memory regression in iteration %d: %d bytes > %d bytes", 
					i, metrics.TotalAllocBytes, baseline.maxMemoryBytes)
			}
		}
	})
}

// TestConcurrentPerformance tests performance under concurrent load
func TestConcurrentPerformance(t *testing.T) {
	t.Run("ConcurrentGraphConstruction", func(t *testing.T) {
		const numGoroutines = 10
		const iterations = 5
		
		results := make(chan time.Duration, numGoroutines*iterations)
		
		config := &InstallConfig{AddRecommendedMCP: true}
		
		// Start multiple goroutines
		for g := 0; g < numGoroutines; g++ {
			go func() {
				for i := 0; i < iterations; i++ {
					start := time.Now()
					
					dg := NewDependencyGraph()
					err := dg.BuildInstallationGraph(config)
					
					duration := time.Since(start)
					
					if err != nil {
						t.Errorf("Concurrent graph construction failed: %v", err)
						return
					}
					
					results <- duration
				}
			}()
		}
		
		// Collect results
		var totalDuration time.Duration
		var maxDuration time.Duration
		
		for i := 0; i < numGoroutines*iterations; i++ {
			duration := <-results
			totalDuration += duration
			if duration > maxDuration {
				maxDuration = duration
			}
		}
		
		averageDuration := totalDuration / time.Duration(numGoroutines*iterations)
		
		t.Logf("Concurrent graph construction performance:")
		t.Logf("  Average time: %v", averageDuration)
		t.Logf("  Max time: %v", maxDuration)
		t.Logf("  Goroutines: %d", numGoroutines)
		t.Logf("  Iterations per goroutine: %d", iterations)
		
		// Verify performance under concurrent load
		const maxExpectedTime = 500 * time.Microsecond // Allow more time under load
		if averageDuration > maxExpectedTime {
			t.Errorf("Concurrent performance degraded: average %v > %v", averageDuration, maxExpectedTime)
		}
		
		if maxDuration > 2*maxExpectedTime {
			t.Errorf("Concurrent max time too high: %v > %v", maxDuration, 2*maxExpectedTime)
		}
	})

	t.Run("ConcurrentTopologicalSort", func(t *testing.T) {
		// Pre-build graph that can be shared across goroutines
		dg := NewDependencyGraph()
		config := &InstallConfig{AddRecommendedMCP: true}
		_ = dg.BuildInstallationGraph(config)
		
		const numGoroutines = 10
		const iterations = 10
		
		results := make(chan time.Duration, numGoroutines*iterations)
		
		// Start multiple goroutines
		for g := 0; g < numGoroutines; g++ {
			go func() {
				for i := 0; i < iterations; i++ {
					start := time.Now()
					
					_, err := dg.GetTopologicalOrder()
					
					duration := time.Since(start)
					
					if err != nil {
						t.Errorf("Concurrent topological sort failed: %v", err)
						return
					}
					
					results <- duration
				}
			}()
		}
		
		// Collect results
		var totalDuration time.Duration
		var maxDuration time.Duration
		
		for i := 0; i < numGoroutines*iterations; i++ {
			duration := <-results
			totalDuration += duration
			if duration > maxDuration {
				maxDuration = duration
			}
		}
		
		averageDuration := totalDuration / time.Duration(numGoroutines*iterations)
		
		t.Logf("Concurrent topological sort performance:")
		t.Logf("  Average time: %v", averageDuration)
		t.Logf("  Max time: %v", maxDuration)
		t.Logf("  Goroutines: %d", numGoroutines)
		t.Logf("  Iterations per goroutine: %d", iterations)
		
		// Verify performance under concurrent load
		const maxExpectedTime = 200 * time.Microsecond // Allow some overhead for concurrent access
		if averageDuration > maxExpectedTime {
			t.Errorf("Concurrent performance degraded: average %v > %v", averageDuration, maxExpectedTime)
		}
		
		if maxDuration > 5*maxExpectedTime {
			t.Errorf("Concurrent max time too high: %v > %v", maxDuration, 5*maxExpectedTime)
		}
	})
}