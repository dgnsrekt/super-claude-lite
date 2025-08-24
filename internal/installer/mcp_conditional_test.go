package installer

import (
	"testing"
)

// TestMCPConditionalDependencies validates that MCP flag affects dependency graph correctly
func TestMCPConditionalDependencies(t *testing.T) {
	// Test with MCP disabled
	t.Run("MCP_disabled", func(t *testing.T) {
		dg := NewDependencyGraph()
		config := &InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false, // MCP disabled
		}

		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build dependency graph with MCP disabled: %v", err)
		}

		// Get topological order
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order with MCP disabled: %v", err)
		}

		// In non-MCP mode, ValidateInstallation should NOT depend on MergeOrCreateMCPConfig
		// This means MergeOrCreateMCPConfig can execute after ValidateInstallation
		validateIndex := -1
		mcpConfigIndex := -1

		for i, step := range order {
			if step == "ValidateInstallation" {
				validateIndex = i
			}
			if step == "MergeOrCreateMCPConfig" {
				mcpConfigIndex = i
			}
		}

		if validateIndex == -1 {
			t.Fatal("ValidateInstallation not found in execution order")
		}
		if mcpConfigIndex == -1 {
			t.Fatal("MergeOrCreateMCPConfig not found in execution order")
		}

		// With MCP disabled, there should be no strict ordering requirement between these steps
		// (other than their shared dependencies on CreateDirectoryStructure)
		t.Logf("MCP disabled - ValidateInstallation at index %d, MergeOrCreateMCPConfig at index %d",
			validateIndex, mcpConfigIndex)
	})

	// Test with MCP enabled
	t.Run("MCP_enabled", func(t *testing.T) {
		dg := NewDependencyGraph()
		config := &InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: true, // MCP enabled
		}

		err := dg.BuildInstallationGraph(config)
		if err != nil {
			t.Fatalf("Failed to build dependency graph with MCP enabled: %v", err)
		}

		// Get topological order
		order, err := dg.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order with MCP enabled: %v", err)
		}

		// In MCP mode, ValidateInstallation MUST depend on MergeOrCreateMCPConfig
		// This means MergeOrCreateMCPConfig must execute before ValidateInstallation
		validateIndex := -1
		mcpConfigIndex := -1
		cleanupIndex := -1

		for i, step := range order {
			if step == "ValidateInstallation" {
				validateIndex = i
			}
			if step == "MergeOrCreateMCPConfig" {
				mcpConfigIndex = i
			}
			if step == "CleanupTempFiles" {
				cleanupIndex = i
			}
		}

		if validateIndex == -1 {
			t.Fatal("ValidateInstallation not found in execution order")
		}
		if mcpConfigIndex == -1 {
			t.Fatal("MergeOrCreateMCPConfig not found in execution order")
		}
		if cleanupIndex == -1 {
			t.Fatal("CleanupTempFiles not found in execution order")
		}

		// With MCP enabled, MergeOrCreateMCPConfig must execute before ValidateInstallation
		if mcpConfigIndex >= validateIndex {
			t.Errorf("Expected MergeOrCreateMCPConfig (index %d) to execute before ValidateInstallation (index %d) when MCP is enabled",
				mcpConfigIndex, validateIndex)
		}

		// With MCP enabled, MergeOrCreateMCPConfig must execute before CleanupTempFiles
		if mcpConfigIndex >= cleanupIndex {
			t.Errorf("Expected MergeOrCreateMCPConfig (index %d) to execute before CleanupTempFiles (index %d) when MCP is enabled",
				mcpConfigIndex, cleanupIndex)
		}

		t.Logf("MCP enabled - MergeOrCreateMCPConfig at index %d, ValidateInstallation at index %d, CleanupTempFiles at index %d",
			mcpConfigIndex, validateIndex, cleanupIndex)
	})

	// Test dependency graph structure directly
	t.Run("Dependency_structure_verification", func(t *testing.T) {
		// Test MCP disabled
		dgDisabled := NewDependencyGraph()
		configDisabled := &InstallConfig{AddRecommendedMCP: false}
		err := dgDisabled.BuildInstallationGraph(configDisabled)
		if err != nil {
			t.Fatalf("Failed to build graph with MCP disabled: %v", err)
		}

		// Test MCP enabled
		dgEnabled := NewDependencyGraph()
		configEnabled := &InstallConfig{AddRecommendedMCP: true}
		err = dgEnabled.BuildInstallationGraph(configEnabled)
		if err != nil {
			t.Fatalf("Failed to build graph with MCP enabled: %v", err)
		}

		// Check that MCP enabled has more dependencies
		disabledDeps, err := dgDisabled.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get dependencies for ValidateInstallation (MCP disabled): %v", err)
		}
		enabledDeps, err := dgEnabled.GetDependencies("ValidateInstallation")
		if err != nil {
			t.Fatalf("Failed to get dependencies for ValidateInstallation (MCP enabled): %v", err)
		}

		if len(enabledDeps) <= len(disabledDeps) {
			t.Errorf("Expected MCP enabled graph to have more dependencies than disabled. Disabled: %d, Enabled: %d",
				len(disabledDeps), len(enabledDeps))
		}

		// Check for specific MCP dependency on ValidateInstallation
		mcpDepFound := false
		for _, dep := range enabledDeps {
			if dep == "MergeOrCreateMCPConfig" {
				mcpDepFound = true
				break
			}
		}

		if !mcpDepFound {
			t.Errorf("Expected ValidateInstallation to depend on MergeOrCreateMCPConfig when MCP is enabled")
		}

		// Also check CleanupTempFiles dependencies
		cleanupDeps, err := dgEnabled.GetDependencies("CleanupTempFiles")
		if err != nil {
			t.Fatalf("Failed to get dependencies for CleanupTempFiles: %v", err)
		}
		cleanupMcpDepFound := false
		for _, dep := range cleanupDeps {
			if dep == "MergeOrCreateMCPConfig" {
				cleanupMcpDepFound = true
				break
			}
		}

		if !cleanupMcpDepFound {
			t.Errorf("Expected CleanupTempFiles to depend on MergeOrCreateMCPConfig when MCP is enabled")
		}

		t.Logf("Dependency count - MCP disabled: %d, MCP enabled: %d", len(disabledDeps), len(enabledDeps))
	})
}

// TestMCPDependencyGraphPerformance validates performance with conditional dependencies
func TestMCPDependencyGraphPerformance(t *testing.T) {
	testConfigs := []*InstallConfig{
		{AddRecommendedMCP: false},
		{AddRecommendedMCP: true},
	}

	for _, config := range testConfigs {
		name := "MCP_disabled"
		if config.AddRecommendedMCP {
			name = "MCP_enabled"
		}

		t.Run(name, func(t *testing.T) {
			dg := NewDependencyGraph()

			err := dg.BuildInstallationGraph(config)
			if err != nil {
				t.Fatalf("Failed to build dependency graph for %s: %v", name, err)
			}

			// Test topological order calculation
			order, err := dg.GetTopologicalOrder()
			if err != nil {
				t.Fatalf("Failed to get topological order for %s: %v", name, err)
			}

			// Validate that all expected steps are present
			expectedSteps := []string{
				"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
				"CheckTargetDirectory", "CloneRepository", "CreateDirectoryStructure",
				"CopyCoreFiles", "CopyCommandFiles", "CopyAgentFiles", "CopyModeFiles",
				"CopyMCPFiles", "MergeOrCreateCLAUDEmd", "MergeOrCreateMCPConfig",
				"CreateCommandSymlink", "CreateAgentSymlink", "ValidateInstallation",
				"CleanupTempFiles",
			}

			if len(order) != len(expectedSteps) {
				t.Errorf("Expected %d steps, got %d for %s", len(expectedSteps), len(order), name)
			}

			// Check that each expected step is present
			stepSet := make(map[string]bool)
			for _, step := range order {
				stepSet[step] = true
			}

			for _, expected := range expectedSteps {
				if !stepSet[expected] {
					t.Errorf("Expected step %s not found in order for %s", expected, name)
				}
			}

			t.Logf("%s completed successfully with %d steps", name, len(order))
		})
	}
}
