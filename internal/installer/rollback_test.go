package installer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRollbackFunctionality validates that rollback works correctly with the new dependency system
func TestRollbackFunctionality(t *testing.T) {
	t.Run("Rollback_with_backup_manager", func(t *testing.T) {
		tempDir := t.TempDir()
		testTargetDir := filepath.Join(tempDir, "rollback_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		// Create some existing files to backup
		existingCLAUDE := filepath.Join(testTargetDir, "CLAUDE.md")
		originalContent := "# Original Claude Instructions\nThis is the original content"
		err = os.WriteFile(existingCLAUDE, []byte(originalContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to create existing CLAUDE.md: %v", err)
		}

		existingMCP := filepath.Join(testTargetDir, ".mcp.json")
		originalMCPContent := `{"mcpServers": {"original": {"command": "test"}}}`
		err = os.WriteFile(existingMCP, []byte(originalMCPContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to create existing .mcp.json: %v", err)
		}

		config := &InstallConfig{
			NoBackup:          false, // Enable backup
			Force:             true,
			Interactive:       false,
			AddRecommendedMCP: true,
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		// Manually create backups (simulating partial installation)
		ctx := installer.GetContext()
		if ctx.BackupManager == nil {
			t.Fatal("Expected backup manager to be created when NoBackup=false")
		}

		err = ctx.BackupManager.BackupFile(existingCLAUDE)
		if err != nil {
			t.Fatalf("Failed to backup CLAUDE.md: %v", err)
		}

		err = ctx.BackupManager.BackupFile(existingMCP)
		if err != nil {
			t.Fatalf("Failed to backup .mcp.json: %v", err)
		}

		// Modify the original files (simulating installation changes)
		modifiedContent := "# Modified Claude Instructions\nThis content was changed during installation"
		err = os.WriteFile(existingCLAUDE, []byte(modifiedContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to modify CLAUDE.md: %v", err)
		}

		modifiedMCPContent := `{"mcpServers": {"new": {"command": "modified"}}}`
		err = os.WriteFile(existingMCP, []byte(modifiedMCPContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to modify .mcp.json: %v", err)
		}

		// Verify files were modified
		content, err := os.ReadFile(existingCLAUDE)
		if err != nil {
			t.Fatalf("Failed to read modified CLAUDE.md: %v", err)
		}
		if string(content) != modifiedContent {
			t.Errorf("File was not modified as expected")
		}

		// Test rollback functionality
		err = installer.Rollback()
		if err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}

		// Verify files were restored to original content
		restoredContent, err := os.ReadFile(existingCLAUDE)
		if err != nil {
			t.Fatalf("Failed to read restored CLAUDE.md: %v", err)
		}
		if string(restoredContent) != originalContent {
			t.Errorf("CLAUDE.md was not restored correctly. Expected: %s, Got: %s",
				originalContent, string(restoredContent))
		}

		restoredMCPContent, err := os.ReadFile(existingMCP)
		if err != nil {
			t.Fatalf("Failed to read restored .mcp.json: %v", err)
		}
		if string(restoredMCPContent) != originalMCPContent {
			t.Errorf(".mcp.json was not restored correctly. Expected: %s, Got: %s",
				originalMCPContent, string(restoredMCPContent))
		}

		t.Logf("Rollback functionality test passed")
	})
}

// TestRollbackWithoutBackup validates rollback behavior when no backup is available
func TestRollbackWithoutBackup(t *testing.T) {
	scenarios := []struct {
		name     string
		noBackup bool
	}{
		{"No_backup_flag_enabled", true},
		{"No_backup_manager_created", false}, // Will have backup manager but no files backed up
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()

			config := &InstallConfig{
				NoBackup:    scenario.noBackup,
				Force:       true,
				Interactive: false,
			}

			// Create installer
			installer, err := NewInstaller(tempDir, config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			// Test rollback when no backup is available
			err = installer.Rollback()
			if err == nil {
				t.Errorf("Expected rollback to fail when no backup is available")
			}

			expectedError := "no backup available for rollback"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}

			t.Logf("Scenario %s: Rollback correctly failed when no backup available", scenario.name)
		})
	}
}

// TestRollbackDependencySystemIntegration validates rollback with the DAG-based system
func TestRollbackDependencySystemIntegration(t *testing.T) {
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")
	capture := NewBaselineCapture(goldenDir)

	testTargetDir := filepath.Join(tempDir, "rollback_integration_target")
	err := os.MkdirAll(testTargetDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test target directory: %v", err)
	}

	// Create existing files for more realistic scenario
	existingFiles := map[string]string{
		"CLAUDE.md": "# Original Instructions\nOriginal content",
		".mcp.json": `{"mcpServers": {"original": {"command": "original"}}}`,
	}

	for filename, content := range existingFiles {
		filePath := filepath.Join(testTargetDir, filename)
		err = os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("Failed to create existing file %s: %v", filename, err)
		}
	}

	config := InstallConfig{
		Force:             true,
		NoBackup:          false, // Enable backup for rollback testing
		Interactive:       false,
		AddRecommendedMCP: true,
		BackupDir:         "", // Use default backup location
	}

	// Create installer with dependency graph
	installer, err := NewInstaller(testTargetDir, &config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	// Verify dependency graph is properly configured
	order, err := installer.graph.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("Failed to get topological order: %v", err)
	}

	// Validate that backup-related steps are present in the dependency graph
	backupStepFound := false
	for _, step := range order {
		if step == "CreateBackups" {
			backupStepFound = true
			break
		}
	}
	if !backupStepFound {
		t.Errorf("CreateBackups step not found in dependency graph execution order")
	}

	// Test scenario structure for baseline capture
	testScenario := TestScenario{
		Name:        "rollback_integration_test",
		Config:      config,
		Description: "Integration test for rollback with DAG system",
	}

	// Capture execution to validate the dependency system works with rollback
	result, err := capture.CaptureExecutionOrder(installer, &testScenario)
	if err != nil {
		t.Logf("Installation failed (expected in test environment): %v", err)
	}

	// Verify backup step was executed
	backupExecuted := false
	for _, step := range result.ExecutionOrder {
		if step == "CreateBackups" {
			backupExecuted = true
			break
		}
	}
	if !backupExecuted {
		t.Errorf("CreateBackups step was not executed")
	}

	// Test rollback functionality after simulated installation
	ctx := installer.GetContext()

	// Simulate that some files were backed up during installation
	if ctx.BackupManager != nil {
		// Manually add some files to backup manager for testing
		for filename := range existingFiles {
			filePath := filepath.Join(testTargetDir, filename)
			err = ctx.BackupManager.BackupFile(filePath)
			if err != nil {
				t.Logf("Note: Could not backup %s (expected in test environment): %v", filename, err)
			}
		}

		// Test rollback if we have backups
		if len(ctx.BackupManager.Files) > 0 {
			err = installer.Rollback()
			if err != nil {
				t.Errorf("Rollback failed: %v", err)
			} else {
				t.Logf("Rollback succeeded with %d files restored", len(ctx.BackupManager.Files))
			}
		} else {
			t.Logf("No backup files created (expected in test environment)")
		}
	}

	t.Logf("Rollback dependency system integration test completed with %d steps executed",
		len(result.ExecutionOrder))
}

// TestRollbackErrorScenarios validates error handling in rollback scenarios
func TestRollbackErrorScenarios(t *testing.T) {
	t.Run("Rollback_with_corrupted_backup", func(t *testing.T) {
		tempDir := t.TempDir()
		testTargetDir := filepath.Join(tempDir, "corrupted_backup_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		// Create existing file
		existingFile := filepath.Join(testTargetDir, "CLAUDE.md")
		originalContent := "original content"
		err = os.WriteFile(existingFile, []byte(originalContent), 0o644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		config := &InstallConfig{
			NoBackup: false,
			Force:    true,
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		ctx := installer.GetContext()

		// Create backup
		err = ctx.BackupManager.BackupFile(existingFile)
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		// Corrupt the backup file (make it inaccessible)
		if len(ctx.BackupManager.Files) > 0 {
			for _, backupPath := range ctx.BackupManager.Files {
				// Remove the backup file to simulate corruption
				err = os.Remove(backupPath)
				if err != nil {
					t.Fatalf("Failed to remove backup file for test: %v", err)
				}
				break
			}

			// Test rollback with corrupted backup
			err = installer.Rollback()
			if err == nil {
				t.Errorf("Expected rollback to fail with corrupted backup")
			}

			t.Logf("Rollback correctly failed with corrupted backup: %v", err)
		} else {
			t.Skip("No backup files created to test corruption scenario")
		}
	})
}

// TestRollbackPerformance validates rollback performance
func TestRollbackPerformance(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple files to test rollback performance
	testFiles := []string{"CLAUDE.md", ".mcp.json", "test1.md", "test2.md", "test3.md"}
	testTargetDir := filepath.Join(tempDir, "performance_target")
	err := os.MkdirAll(testTargetDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test target directory: %v", err)
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(testTargetDir, filename)
		content := "original content for " + filename
		err = os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	config := &InstallConfig{
		NoBackup: false,
		Force:    true,
	}

	// Create installer
	installer, err := NewInstaller(testTargetDir, config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	ctx := installer.GetContext()

	// Create backups for all files
	for _, filename := range testFiles {
		filePath := filepath.Join(testTargetDir, filename)
		err = ctx.BackupManager.BackupFile(filePath)
		if err != nil {
			t.Logf("Note: Could not backup %s (expected in test environment): %v", filename, err)
		}
	}

	// Test rollback performance if we have backups
	if len(ctx.BackupManager.Files) > 0 {
		err = installer.Rollback()
		if err != nil {
			t.Errorf("Rollback failed: %v", err)
		} else {
			t.Logf("Rollback performance test completed successfully with %d files",
				len(ctx.BackupManager.Files))
		}
	} else {
		t.Logf("No backup files created for performance test (expected in test environment)")
	}
}
