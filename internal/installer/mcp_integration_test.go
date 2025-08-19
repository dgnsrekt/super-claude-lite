package installer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMCPFlagIntegration validates that the --add-mcp flag works correctly in full installation scenarios
func TestMCPFlagIntegration(t *testing.T) {
	// Test MCP flag integration with baseline capture
	t.Run("MCP_flag_baseline_integration", func(t *testing.T) {
		tempDir := t.TempDir()
		goldenDir := filepath.Join(tempDir, "golden")
		capture := NewBaselineCapture(goldenDir)

		// Test scenarios with and without MCP
		scenarios := []struct {
			name        string
			mcpEnabled  bool
			description string
		}{
			{
				name:        "installation_without_mcp",
				mcpEnabled:  false,
				description: "Installation with MCP recommendations disabled",
			},
			{
				name:        "installation_with_mcp",
				mcpEnabled:  true,
				description: "Installation with MCP recommendations enabled",
			},
		}

		var results []BaselineTestResult

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Set up test environment
				testTargetDir := filepath.Join(tempDir, scenario.name+"_target")
				err := os.MkdirAll(testTargetDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create test target directory: %v", err)
				}
				config := InstallConfig{
					Force:             false,
					NoBackup:          false,
					Interactive:       false,
					AddRecommendedMCP: scenario.mcpEnabled,
					BackupDir:         "",
				}

				// Create installer
				installer, err := NewInstaller(testTargetDir, &config)
				if err != nil {
					t.Fatalf("Failed to create installer for %s: %v", scenario.name, err)
				}

				// Create test scenario structure
				testScenario := TestScenario{
					Name:        scenario.name,
					Config:      config,
					Description: scenario.description,
				}

				// Capture execution behavior
				result, err := capture.CaptureExecutionOrder(installer, &testScenario)
				if err != nil {
					t.Logf("Installation failed for %s (expected in test environment): %v", scenario.name, err)
				}

				// Store result for comparison
				results = append(results, result)

				// Validate execution order contains all expected steps
				if len(result.ExecutionOrder) == 0 {
					t.Errorf("No steps executed for scenario %s", scenario.name)
					return
				}

				// Check that MergeOrCreateMCPConfig is always present
				mcpConfigFound := false
				for _, step := range result.ExecutionOrder {
					if step == "MergeOrCreateMCPConfig" {
						mcpConfigFound = true
						break
					}
				}
				if !mcpConfigFound {
					t.Errorf("MergeOrCreateMCPConfig step not found in execution order for %s", scenario.name)
				}

				t.Logf("Scenario %s: %d steps executed, success=%t",
					scenario.name, len(result.ExecutionOrder), result.Success)
			})
		}

		// Compare results between MCP enabled and disabled
		if len(results) == 2 {
			mcpDisabled := results[0]
			mcpEnabled := results[1]

			// Both should execute the same number of steps (MCP affects ordering, not step count)
			if len(mcpDisabled.ExecutionOrder) != len(mcpEnabled.ExecutionOrder) {
				t.Errorf("Expected same number of steps. MCP disabled: %d, MCP enabled: %d",
					len(mcpDisabled.ExecutionOrder), len(mcpEnabled.ExecutionOrder))
			}

			// Check for ordering differences
			mcpConfigIndexEnabled := -1
			validateIndexEnabled := -1

			for i, step := range mcpEnabled.ExecutionOrder {
				if step == "MergeOrCreateMCPConfig" {
					mcpConfigIndexEnabled = i
				}
				if step == "ValidateInstallation" {
					validateIndexEnabled = i
				}
			}

			// With MCP enabled, MergeOrCreateMCPConfig should execute before ValidateInstallation
			if mcpEnabled.Config.AddRecommendedMCP {
				if mcpConfigIndexEnabled >= validateIndexEnabled {
					t.Errorf("When MCP is enabled, MergeOrCreateMCPConfig should execute before ValidateInstallation. "+
						"MCP config index: %d, Validate index: %d", mcpConfigIndexEnabled, validateIndexEnabled)
				}
			}

			t.Logf("MCP flag integration test completed successfully")
		}
	})
}

// TestMCPFlagDependencyValidation validates MCP conditional dependencies at runtime
func TestMCPFlagDependencyValidation(t *testing.T) {
	testCases := []struct {
		name       string
		mcpEnabled bool
	}{
		{"MCP_disabled", false},
		{"MCP_enabled", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &InstallConfig{
				AddRecommendedMCP: tc.mcpEnabled,
			}

			// Create installer (which builds the dependency graph)
			tempDir := t.TempDir()
			installer, err := NewInstaller(tempDir, config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			// Access the internal dependency graph
			graph := installer.graph
			if graph == nil {
				t.Fatal("Installer dependency graph is nil")
			}

			// Get topological order
			order, err := graph.GetTopologicalOrder()
			if err != nil {
				t.Fatalf("Failed to get topological order: %v", err)
			}

			// Validate the order contains all expected steps
			expectedSteps := []string{
				"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
				"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
				"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd",
				"MergeOrCreateMCPConfig", "CreateCommandSymlink", "ValidateInstallation",
				"CleanupTempFiles",
			}

			if len(order) != len(expectedSteps) {
				t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(order))
			}

			// Check that all expected steps are present
			stepSet := make(map[string]bool)
			for _, step := range order {
				stepSet[step] = true
			}

			for _, expected := range expectedSteps {
				if !stepSet[expected] {
					t.Errorf("Expected step %s not found in order", expected)
				}
			}

			// Validate MCP-specific ordering when enabled
			if tc.mcpEnabled {
				mcpConfigIndex := -1
				validateIndex := -1
				cleanupIndex := -1

				for i, step := range order {
					switch step {
					case "MergeOrCreateMCPConfig":
						mcpConfigIndex = i
					case "ValidateInstallation":
						validateIndex = i
					case "CleanupTempFiles":
						cleanupIndex = i
					}
				}

				if mcpConfigIndex == -1 || validateIndex == -1 || cleanupIndex == -1 {
					t.Fatal("Required steps not found in execution order")
				}

				// MCP config must execute before validation and cleanup when MCP is enabled
				if mcpConfigIndex >= validateIndex {
					t.Errorf("MergeOrCreateMCPConfig (index %d) should execute before ValidateInstallation (index %d) when MCP enabled",
						mcpConfigIndex, validateIndex)
				}
				if mcpConfigIndex >= cleanupIndex {
					t.Errorf("MergeOrCreateMCPConfig (index %d) should execute before CleanupTempFiles (index %d) when MCP enabled",
						mcpConfigIndex, cleanupIndex)
				}
			}

			t.Logf("%s: Validated execution order with %d steps", tc.name, len(order))
		})
	}
}

// TestMCPFlagPerformanceImpact validates that MCP flag doesn't significantly impact performance
func TestMCPFlagPerformanceImpact(t *testing.T) {
	testConfigs := []*InstallConfig{
		{AddRecommendedMCP: false},
		{AddRecommendedMCP: true},
	}

	tempDir := t.TempDir()
	performanceResults := make(map[string]time.Duration)

	for _, config := range testConfigs {
		name := "MCP_disabled"
		if config.AddRecommendedMCP {
			name = "MCP_enabled"
		}

		// Measure installer creation time (includes dependency graph building)
		start := time.Now()
		installer, err := NewInstaller(tempDir, config)
		creationTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to create installer for %s: %v", name, err)
		}

		// Measure topological order calculation time
		start = time.Now()
		_, err = installer.graph.GetTopologicalOrder()
		sortTime := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to get topological order for %s: %v", name, err)
		}

		totalTime := creationTime + sortTime
		performanceResults[name] = totalTime

		t.Logf("%s: Creation time: %v, Sort time: %v, Total: %v",
			name, creationTime, sortTime, totalTime)
	}

	// Ensure MCP enabled doesn't significantly slow down the system
	mcpDisabledTime := performanceResults["MCP_disabled"]
	mcpEnabledTime := performanceResults["MCP_enabled"]

	// Allow up to 2x slower for MCP enabled (very generous threshold)
	if mcpEnabledTime > mcpDisabledTime*2 {
		t.Errorf("MCP enabled took too long compared to disabled. Disabled: %v, Enabled: %v",
			mcpDisabledTime, mcpEnabledTime)
	}

	// Both should complete within reasonable time (under 10ms for graph operations)
	maxAllowedTime := 10 * time.Millisecond
	if mcpDisabledTime > maxAllowedTime {
		t.Errorf("MCP disabled took too long: %v (max allowed: %v)", mcpDisabledTime, maxAllowedTime)
	}
	if mcpEnabledTime > maxAllowedTime {
		t.Errorf("MCP enabled took too long: %v (max allowed: %v)", mcpEnabledTime, maxAllowedTime)
	}

	t.Logf("Performance test completed successfully")
}
