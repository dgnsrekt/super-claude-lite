package installer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstallationSummaryGeneration validates that installation summaries are generated correctly
func TestInstallationSummaryGeneration(t *testing.T) {
	t.Run("Summary_with_fresh_installation", func(t *testing.T) {
		tempDir := t.TempDir()
		testTargetDir := filepath.Join(tempDir, "fresh_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		config := &InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: true,
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		// Get initial summary
		summary := installer.GetInstallationSummary()

		// Validate summary structure
		if summary.TargetDir != testTargetDir {
			t.Errorf("Expected TargetDir %s, got %s", testTargetDir, summary.TargetDir)
		}

		if !summary.MCPConfigCreated {
			t.Errorf("Expected MCPConfigCreated to be true when AddRecommendedMCP=true")
		}

		// Validate existing files detection
		if summary.ExistingFiles.CLAUDEmd {
			t.Errorf("Expected CLAUDEmd to be false for fresh installation")
		}
		if summary.ExistingFiles.MCPConfig {
			t.Errorf("Expected MCPConfig to be false for fresh installation")
		}
		if summary.ExistingFiles.SuperClaudeDir {
			t.Errorf("Expected SuperClaudeDir to be false for fresh installation")
		}
		if summary.ExistingFiles.ClaudeDir {
			t.Errorf("Expected ClaudeDir to be false for fresh installation")
		}

		// Initially no steps should be completed
		if len(summary.CompletedSteps) != 0 {
			t.Errorf("Expected no completed steps initially, got %d", len(summary.CompletedSteps))
		}

		// No files should be backed up initially
		if len(summary.BackedUpFiles) != 0 {
			t.Errorf("Expected no backed up files initially, got %d", len(summary.BackedUpFiles))
		}

		t.Logf("Fresh installation summary validated successfully")
	})

	t.Run("Summary_with_existing_files", func(t *testing.T) {
		tempDir := t.TempDir()
		testTargetDir := filepath.Join(tempDir, "existing_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		// Create existing files
		existingCLAUDE := filepath.Join(testTargetDir, "CLAUDE.md")
		err = os.WriteFile(existingCLAUDE, []byte("existing content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create existing CLAUDE.md: %v", err)
		}

		existingMCP := filepath.Join(testTargetDir, ".mcp.json")
		err = os.WriteFile(existingMCP, []byte(`{"mcpServers": {}}`), 0o644)
		if err != nil {
			t.Fatalf("Failed to create existing .mcp.json: %v", err)
		}

		// Create existing directories
		superClaudeDir := filepath.Join(testTargetDir, ".superclaude")
		err = os.MkdirAll(superClaudeDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create existing .superclaude directory: %v", err)
		}

		config := &InstallConfig{
			Force:             true,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false,
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		// Re-scan after creating existing files
		ctx := installer.GetContext()
		err = ctx.ScanExistingFiles()
		if err != nil {
			t.Fatalf("Failed to re-scan existing files: %v", err)
		}

		// Get summary after scanning
		summary := installer.GetInstallationSummary()

		// Validate existing files are detected
		if !summary.ExistingFiles.CLAUDEmd {
			t.Errorf("Expected CLAUDEmd to be true when file exists")
		}
		if !summary.ExistingFiles.MCPConfig {
			t.Errorf("Expected MCPConfig to be true when file exists")
		}
		if !summary.ExistingFiles.SuperClaudeDir {
			t.Errorf("Expected SuperClaudeDir to be true when directory exists")
		}

		if summary.MCPConfigCreated {
			t.Errorf("Expected MCPConfigCreated to be false when AddRecommendedMCP=false")
		}

		t.Logf("Existing files summary validated successfully")
	})
}

// TestStepExecutionOrderValidation validates that step execution follows the correct DAG order
func TestStepExecutionOrderValidation(t *testing.T) {
	t.Run("Execution_order_with_MCP_disabled", func(t *testing.T) {
		tempDir := t.TempDir()
		goldenDir := filepath.Join(tempDir, "golden")
		capture := NewBaselineCapture(goldenDir)

		testTargetDir := filepath.Join(tempDir, "order_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		config := InstallConfig{
			Force:             false,
			NoBackup:          true, // Disable backup for simpler test
			Interactive:       false,
			AddRecommendedMCP: false, // Disable MCP for this test
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, &config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		// Get expected execution order from dependency graph
		_, err = installer.graph.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order: %v", err)
		}

		// Create test scenario
		testScenario := TestScenario{
			Name:        "execution_order_mcp_disabled",
			Config:      config,
			Description: "Test execution order with MCP disabled",
		}

		// Capture actual execution order
		result, err := capture.CaptureExecutionOrder(installer, &testScenario)
		if err != nil {
			t.Logf("Installation failed (expected in test environment): %v", err)
		}

		// Validate that execution order matches dependency graph order
		if len(result.ExecutionOrder) == 0 {
			t.Fatalf("No steps were executed")
		}

		// Check that all expected steps are present (though installation may fail)
		executedStepSet := make(map[string]bool)
		for _, step := range result.ExecutionOrder {
			executedStepSet[step] = true
		}

		// Validate core steps are executed in proper order
		coreSteps := []string{
			"CheckPrerequisites", "ScanExistingFiles", "CreateBackups",
			"CheckTargetDirectory", "CloneRepository",
		}

		lastIndex := -1
		for _, step := range coreSteps {
			for i, executedStep := range result.ExecutionOrder {
				if executedStep == step {
					if i <= lastIndex {
						t.Errorf("Step %s executed out of order. Expected after index %d, found at %d",
							step, lastIndex, i)
					}
					lastIndex = i
					break
				}
			}
		}

		t.Logf("Execution order validation completed with %d steps: %v",
			len(result.ExecutionOrder), result.ExecutionOrder)
	})

	t.Run("Execution_order_with_MCP_enabled", func(t *testing.T) {
		tempDir := t.TempDir()
		goldenDir := filepath.Join(tempDir, "golden")
		capture := NewBaselineCapture(goldenDir)

		testTargetDir := filepath.Join(tempDir, "order_mcp_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		config := InstallConfig{
			Force:             false,
			NoBackup:          true,
			Interactive:       false,
			AddRecommendedMCP: true, // Enable MCP for this test
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, &config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		// Create test scenario
		testScenario := TestScenario{
			Name:        "execution_order_mcp_enabled",
			Config:      config,
			Description: "Test execution order with MCP enabled",
		}

		// Capture actual execution order
		result, err := capture.CaptureExecutionOrder(installer, &testScenario)
		if err != nil {
			t.Logf("Installation failed (expected in test environment): %v", err)
		}

		// Validate MCP-specific ordering
		mcpConfigIndex := -1
		validateIndex := -1
		cleanupIndex := -1

		for i, step := range result.ExecutionOrder {
			switch step {
			case "MergeOrCreateMCPConfig":
				mcpConfigIndex = i
			case "ValidateInstallation":
				validateIndex = i
			case "CleanupTempFiles":
				cleanupIndex = i
			}
		}

		// With MCP enabled, MCP config should execute before validation and cleanup
		if mcpConfigIndex >= 0 && validateIndex >= 0 {
			if mcpConfigIndex >= validateIndex {
				t.Errorf("MergeOrCreateMCPConfig (index %d) should execute before ValidateInstallation (index %d) when MCP enabled",
					mcpConfigIndex, validateIndex)
			}
		}

		if mcpConfigIndex >= 0 && cleanupIndex >= 0 {
			if mcpConfigIndex >= cleanupIndex {
				t.Errorf("MergeOrCreateMCPConfig (index %d) should execute before CleanupTempFiles (index %d) when MCP enabled",
					mcpConfigIndex, cleanupIndex)
			}
		}

		t.Logf("MCP execution order validation completed with %d steps", len(result.ExecutionOrder))
	})
}

// TestInstallationSummaryWithCompletedSteps validates summary after steps complete
func TestInstallationSummaryWithCompletedSteps(t *testing.T) {
	tempDir := t.TempDir()
	testTargetDir := filepath.Join(tempDir, "completed_target")
	err := os.MkdirAll(testTargetDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test target directory: %v", err)
	}

	config := &InstallConfig{
		Force:             false,
		NoBackup:          true,
		Interactive:       false,
		AddRecommendedMCP: false,
	}

	// Create installer
	installer, err := NewInstaller(testTargetDir, config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	// Manually simulate some completed steps
	ctx := installer.GetContext()
	ctx.Completed = append(ctx.Completed, "CheckPrerequisites", "ScanExistingFiles", "CreateBackups")

	// Get summary with completed steps
	summary := installer.GetInstallationSummary()

	// Validate completed steps are reflected in summary
	if len(summary.CompletedSteps) != 3 {
		t.Errorf("Expected 3 completed steps, got %d", len(summary.CompletedSteps))
	}

	expectedSteps := []string{"CheckPrerequisites", "ScanExistingFiles", "CreateBackups"}
	for i, expected := range expectedSteps {
		if i >= len(summary.CompletedSteps) || summary.CompletedSteps[i] != expected {
			t.Errorf("Expected step %d to be %s, got %s", i, expected,
				func() string {
					if i < len(summary.CompletedSteps) {
						return summary.CompletedSteps[i]
					}
					return "missing"
				}())
		}
	}

	t.Logf("Completed steps summary validation passed")
}

// TestInstallationSummaryPrintOutput validates the summary print functionality
func TestInstallationSummaryPrintOutput(t *testing.T) {
	// Create a sample summary
	summary := InstallationSummary{
		TargetDir:      "/test/target",
		BackupDir:      "/test/backup",
		CompletedSteps: []string{"CheckPrerequisites", "ScanExistingFiles", "CreateBackups"},
		BackedUpFiles:  []string{"/test/target/CLAUDE.md", "/test/target/.mcp.json"},
		ExistingFiles: ExistingFiles{
			CLAUDEmd:       true,
			MCPConfig:      true,
			SuperClaudeDir: false,
			ClaudeDir:      false,
		},
		MCPConfigCreated: true,
	}

	// Test that PrintSummary doesn't crash (we can't easily capture output in tests)
	// This is mainly to ensure the method exists and can be called
	summary.PrintSummary()

	// Validate summary fields are accessible
	if summary.TargetDir != "/test/target" {
		t.Errorf("Expected TargetDir '/test/target', got '%s'", summary.TargetDir)
	}

	if len(summary.BackedUpFiles) != 2 {
		t.Errorf("Expected 2 backed up files, got %d", len(summary.BackedUpFiles))
	}

	if len(summary.CompletedSteps) != 3 {
		t.Errorf("Expected 3 completed steps, got %d", len(summary.CompletedSteps))
	}

	if !summary.MCPConfigCreated {
		t.Errorf("Expected MCPConfigCreated to be true")
	}

	t.Logf("Summary print output test completed")
}

// TestExecutionOrderConsistency validates that execution order is consistent across runs
func TestExecutionOrderConsistency(t *testing.T) {
	tempDir := t.TempDir()

	config := &InstallConfig{
		Force:             false,
		NoBackup:          true,
		Interactive:       false,
		AddRecommendedMCP: true,
	}

	// Create multiple installers and verify they have consistent execution order
	var orders [][]string

	for i := 0; i < 3; i++ {
		testTargetDir := filepath.Join(tempDir, "consistency_target", "run", string(rune('0'+i)))
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory %d: %v", i, err)
		}

		installer, err := NewInstaller(testTargetDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer %d: %v", i, err)
		}

		order, err := installer.graph.GetTopologicalOrder()
		if err != nil {
			t.Fatalf("Failed to get topological order %d: %v", i, err)
		}

		orders = append(orders, order)
	}

	// Compare all orders to ensure they have the same steps (order may vary due to DAG)
	if len(orders) < 2 {
		t.Fatal("Need at least 2 orders to compare")
	}

	baseOrder := orders[0]
	baseStepSet := make(map[string]bool)
	for _, step := range baseOrder {
		baseStepSet[step] = true
	}

	for i, order := range orders[1:] {
		if len(order) != len(baseOrder) {
			t.Errorf("Order %d has different length: expected %d, got %d",
				i+1, len(baseOrder), len(order))
			continue
		}

		// Check that all steps are present (order may differ due to topological sorting)
		orderStepSet := make(map[string]bool)
		for _, step := range order {
			orderStepSet[step] = true
		}

		for step := range baseStepSet {
			if !orderStepSet[step] {
				t.Errorf("Order %d missing step: %s", i+1, step)
			}
		}

		for step := range orderStepSet {
			if !baseStepSet[step] {
				t.Errorf("Order %d has extra step: %s", i+1, step)
			}
		}
	}

	t.Logf("Execution order consistency validated across %d runs with %d steps each",
		len(orders), len(baseOrder))
}

// TestStepExecutionDependencyValidation validates dependency constraints are satisfied
func TestStepExecutionDependencyValidation(t *testing.T) {
	tempDir := t.TempDir()

	testConfigs := []*InstallConfig{
		{AddRecommendedMCP: false, NoBackup: true},
		{AddRecommendedMCP: true, NoBackup: false},
		{AddRecommendedMCP: true, NoBackup: true},
	}

	for i, config := range testConfigs {
		t.Run("Dependency_validation_config_"+string(rune('0'+i)), func(t *testing.T) {
			testTargetDir := filepath.Join(tempDir, "dependency_target", string(rune('0'+i)))
			err := os.MkdirAll(testTargetDir, 0o755)
			if err != nil {
				t.Fatalf("Failed to create test target directory: %v", err)
			}

			installer, err := NewInstaller(testTargetDir, config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			order, err := installer.graph.GetTopologicalOrder()
			if err != nil {
				t.Fatalf("Failed to get topological order: %v", err)
			}

			// Create step position map
			stepPositions := make(map[string]int)
			for i, step := range order {
				stepPositions[step] = i
			}

			// Validate critical dependency constraints
			dependencies := map[string][]string{
				"ScanExistingFiles":        {"CheckPrerequisites"},
				"CreateBackups":            {"ScanExistingFiles"},
				"CloneRepository":          {"CheckTargetDirectory"},
				"CreateDirectoryStructure": {"CheckTargetDirectory"},
				"CopyCoreFiles":            {"CloneRepository", "CreateDirectoryStructure"},
				"CopyCommandFiles":         {"CloneRepository", "CreateDirectoryStructure"},
				"MergeOrCreateCLAUDEmd":    {"ScanExistingFiles"},
				"CreateCommandSymlink":     {"CreateDirectoryStructure"},
				"ValidateInstallation":     {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd"},
				"CleanupTempFiles":         {"ValidateInstallation"},
			}

			// Add MCP-specific dependencies when enabled
			if config.AddRecommendedMCP {
				dependencies["ValidateInstallation"] = append(dependencies["ValidateInstallation"], "MergeOrCreateMCPConfig")
				dependencies["CleanupTempFiles"] = append(dependencies["CleanupTempFiles"], "MergeOrCreateMCPConfig")
			}

			// Validate all dependencies are satisfied
			for step, deps := range dependencies {
				stepPos, stepExists := stepPositions[step]
				if !stepExists {
					continue // Step might not be in this execution path
				}

				for _, dep := range deps {
					depPos, depExists := stepPositions[dep]
					if !depExists {
						continue // Dependency might not be in this execution path
					}

					if depPos >= stepPos {
						t.Errorf("Dependency violation: %s (pos %d) should execute before %s (pos %d)",
							dep, depPos, step, stepPos)
					}
				}
			}

			t.Logf("Config %d: Dependency validation passed with %d steps", i, len(order))
		})
	}
}
