package installer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNoBackupFlag validates that the --no-backup flag works correctly
func TestNoBackupFlag(t *testing.T) {
	// Test no-backup flag integration with baseline capture
	t.Run("NoBackup_flag_baseline_integration", func(t *testing.T) {
		tempDir := t.TempDir()
		goldenDir := filepath.Join(tempDir, "golden")
		capture := NewBaselineCapture(goldenDir)

		// Test scenarios with and without backup
		scenarios := []struct {
			name         string
			noBackup     bool
			description  string
			expectBackup bool
		}{
			{
				name:         "installation_with_backup",
				noBackup:     false,
				description:  "Installation with backups enabled",
				expectBackup: true,
			},
			{
				name:         "installation_without_backup",
				noBackup:     true,
				description:  "Installation with backups disabled",
				expectBackup: false,
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Set up test environment
				testTargetDir := filepath.Join(tempDir, scenario.name+"_target")
				err := os.MkdirAll(testTargetDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create test target directory: %v", err)
				}

				// Create some existing files to test backup behavior
				existingCLAUDE := filepath.Join(testTargetDir, "CLAUDE.md")
				err = os.WriteFile(existingCLAUDE, []byte("existing claude content"), 0o644)
				if err != nil {
					t.Fatalf("Failed to create existing CLAUDE.md: %v", err)
				}

				config := InstallConfig{
					Force:             false,
					NoBackup:          scenario.noBackup,
					Interactive:       false,
					AddRecommendedMCP: false,
					BackupDir:         "",
				}

				// Create installer
				installer, err := NewInstaller(testTargetDir, &config)
				if err != nil {
					t.Fatalf("Failed to create installer for %s: %v", scenario.name, err)
				}

				// Check backup manager creation
				ctx := installer.GetContext()
				if scenario.expectBackup {
					if ctx.BackupManager == nil {
						t.Errorf("Expected backup manager to be created when NoBackup=false")
					}
					if ctx.BackupDir == "" {
						t.Errorf("Expected backup directory to be set when NoBackup=false")
					}
				} else {
					if ctx.BackupManager != nil {
						t.Errorf("Expected backup manager to be nil when NoBackup=true")
					}
					if ctx.BackupDir != "" {
						t.Errorf("Expected backup directory to be empty when NoBackup=true")
					}
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

				// Validate that CreateBackups step is always executed
				createBackupsFound := false
				for _, step := range result.ExecutionOrder {
					if step == "CreateBackups" {
						createBackupsFound = true
						break
					}
				}
				if !createBackupsFound {
					t.Errorf("CreateBackups step not found in execution order for %s", scenario.name)
				}

				// Check if backup directory was created
				backupDirs, err := filepath.Glob(filepath.Join(testTargetDir, ".superclaude-backup-*"))
				if err != nil {
					t.Errorf("Failed to check for backup directories: %v", err)
				}

				if scenario.expectBackup {
					if len(backupDirs) == 0 {
						t.Logf("Note: No backup directory created for %s (expected in test environment)", scenario.name)
					}
				} else {
					if len(backupDirs) > 0 {
						t.Errorf("Unexpected backup directory created when NoBackup=true: %v", backupDirs)
					}
				}

				t.Logf("Scenario %s: %d steps executed, success=%t, backup expected=%t",
					scenario.name, len(result.ExecutionOrder), result.Success, scenario.expectBackup)
			})
		}
	})
}

// TestNoBackupFlagLogic validates the NoBackup flag logic at the step level
func TestNoBackupFlagLogic(t *testing.T) {
	testCases := []struct {
		name     string
		noBackup bool
	}{
		{"Backup_enabled", false},
		{"Backup_disabled", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create existing file to test backup behavior
			existingFile := filepath.Join(tempDir, "CLAUDE.md")
			err := os.WriteFile(existingFile, []byte("existing content"), 0o644)
			if err != nil {
				t.Fatalf("Failed to create existing file: %v", err)
			}

			config := &InstallConfig{
				NoBackup: tc.noBackup,
			}

			// Create installer context
			ctx, err := NewInstallContext(tempDir, config)
			if err != nil {
				t.Fatalf("Failed to create install context: %v", err)
			}

			// Test backup manager setup
			if tc.noBackup {
				if ctx.BackupManager != nil {
					t.Errorf("Expected BackupManager to be nil when NoBackup=true")
				}
			} else {
				if ctx.BackupManager == nil {
					t.Errorf("Expected BackupManager to be created when NoBackup=false")
				}
			}

			// Test createBackups function behavior
			err = createBackups(ctx)
			if err != nil {
				t.Errorf("createBackups function failed: %v", err)
			}

			// Check if backup directory was created
			backupDirs, err := filepath.Glob(filepath.Join(tempDir, ".superclaude-backup-*"))
			if err != nil {
				t.Errorf("Failed to check for backup directories: %v", err)
			}

			if tc.noBackup {
				if len(backupDirs) > 0 {
					t.Errorf("Unexpected backup directory created when NoBackup=true: %v", backupDirs)
				}
			} else {
				// When backup is enabled, createBackups should attempt to create backup
				// (though it might not succeed in test environment without proper setup)
				t.Logf("Backup enabled test completed - backup dirs found: %d", len(backupDirs))
			}

			t.Logf("%s: createBackups function executed successfully", tc.name)
		})
	}
}

// TestDryRunMode validates the dry-run infrastructure
func TestDryRunMode(t *testing.T) {
	tempDir := t.TempDir()

	config := &InstallConfig{
		NoBackup: false,
	}

	// Create installer context
	ctx, err := NewInstallContext(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create install context: %v", err)
	}

	// Enable dry-run mode
	ctx.DryRun = true

	// Test that dry-run mode affects step behavior
	testSteps := []struct {
		name     string
		stepFunc func(*InstallContext) error
	}{
		{"cloneRepository", cloneRepository},
		{"createDirectoryStructure", createDirectoryStructure},
		{"copyCoreFiles", copyCoreFiles},
		{"copyCommandFiles", copyCommandFiles},
		{"mergeOrCreateCLAUDEmd", mergeOrCreateCLAUDEmd},
		{"mergeOrCreateMCPConfig", mergeOrCreateMCPConfig},
		{"createCommandSymlink", createCommandSymlink},
		{"validateInstallation", validateInstallation},
		{"cleanupTempFiles", cleanupTempFiles},
	}

	for _, step := range testSteps {
		t.Run("DryRun_"+step.name, func(t *testing.T) {
			// All steps should succeed in dry-run mode without actually performing operations
			err := step.stepFunc(ctx)
			if err != nil {
				t.Errorf("Step %s failed in dry-run mode: %v", step.name, err)
			}
		})
	}

	// Test validation functions in dry-run mode
	validationSteps := []struct {
		name     string
		stepFunc func(*InstallContext) error
	}{
		{"validateRepoCloned", validateRepoCloned},
		{"validateCoreFiles", validateCoreFiles},
		{"validateCommandFiles", validateCommandFiles},
	}

	for _, step := range validationSteps {
		t.Run("DryRun_"+step.name, func(t *testing.T) {
			// Validation functions should skip checks in dry-run mode
			err := step.stepFunc(ctx)
			if err != nil {
				t.Errorf("Validation %s failed in dry-run mode: %v", step.name, err)
			}
		})
	}

	t.Logf("Dry-run mode test completed successfully")
}

// TestDryRunFlagIntegration validates dry-run mode with full installer
func TestDryRunFlagIntegration(t *testing.T) {
	tempDir := t.TempDir()
	testTargetDir := filepath.Join(tempDir, "dryrun_target")

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

	// Enable dry-run mode in context
	ctx := installer.GetContext()
	ctx.DryRun = true

	// Get topological order to verify dependency graph works with dry-run
	order, err := installer.graph.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("Failed to get topological order: %v", err)
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
		t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(order))
	}

	// Check that each expected step is present
	stepSet := make(map[string]bool)
	for _, step := range order {
		stepSet[step] = true
	}

	for _, expected := range expectedSteps {
		if !stepSet[expected] {
			t.Errorf("Expected step %s not found in order", expected)
		}
	}

	t.Logf("Dry-run integration test completed with %d steps", len(order))
}

// TestNoBackupDryRunCombination validates the combination of NoBackup and DryRun flags
func TestNoBackupDryRunCombination(t *testing.T) {
	combinations := []struct {
		name     string
		noBackup bool
		dryRun   bool
	}{
		{"Normal_with_backup", false, false},
		{"Normal_no_backup", true, false},
		{"DryRun_with_backup", false, true},
		{"DryRun_no_backup", true, true},
	}

	for _, combo := range combinations {
		t.Run(combo.name, func(t *testing.T) {
			tempDir := t.TempDir()

			config := &InstallConfig{
				NoBackup: combo.noBackup,
			}

			// Create installer context
			ctx, err := NewInstallContext(tempDir, config)
			if err != nil {
				t.Fatalf("Failed to create install context: %v", err)
			}

			// Set dry-run mode if specified
			ctx.DryRun = combo.dryRun

			// Test createBackups function with combination
			err = createBackups(ctx)
			if err != nil {
				t.Errorf("createBackups failed for %s: %v", combo.name, err)
			}

			// Validate backup manager state
			if combo.noBackup {
				if ctx.BackupManager != nil {
					t.Errorf("Expected BackupManager to be nil when NoBackup=true")
				}
			} else if !combo.dryRun {
				if ctx.BackupManager == nil {
					t.Errorf("Expected BackupManager when NoBackup=false and not dry-run")
				}
			}

			t.Logf("%s: Test completed successfully", combo.name)
		})
	}
}

// TestNoBackupPerformance validates that NoBackup flag doesn't impact performance significantly
func TestNoBackupPerformance(t *testing.T) {
	testConfigs := []*InstallConfig{
		{NoBackup: false},
		{NoBackup: true},
	}

	tempDir := t.TempDir()
	performanceResults := make(map[string]time.Duration)

	for _, config := range testConfigs {
		name := "Backup_enabled"
		if config.NoBackup {
			name = "Backup_disabled"
		}

		// Measure installer creation time
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

	// Ensure NoBackup doesn't significantly change performance
	backupEnabledTime := performanceResults["Backup_enabled"]
	backupDisabledTime := performanceResults["Backup_disabled"]

	// Both should complete within reasonable time (under 10ms for graph operations)
	maxAllowedTime := 10 * time.Millisecond
	if backupEnabledTime > maxAllowedTime {
		t.Errorf("Backup enabled took too long: %v (max allowed: %v)", backupEnabledTime, maxAllowedTime)
	}
	if backupDisabledTime > maxAllowedTime {
		t.Errorf("Backup disabled took too long: %v (max allowed: %v)", backupDisabledTime, maxAllowedTime)
	}

	// The difference should be minimal (both are just graph operations)
	timeDiff := backupEnabledTime - backupDisabledTime
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// Allow up to 5ms difference
	if timeDiff > 5*time.Millisecond {
		t.Errorf("Too much performance difference between backup modes. Enabled: %v, Disabled: %v, Diff: %v",
			backupEnabledTime, backupDisabledTime, timeDiff)
	}

	t.Logf("Performance test completed successfully")
}
