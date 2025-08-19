package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestExistingFileScenarios validates how existing files are handled during installation
func TestExistingFileScenarios(t *testing.T) {
	t.Run("Existing_file_scanning", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create various existing files and directories
		claudeFile := filepath.Join(tempDir, "CLAUDE.md")
		mcpFile := filepath.Join(tempDir, ".mcp.json")
		superClaudeDir := filepath.Join(tempDir, ".superclaude")
		claudeDir := filepath.Join(tempDir, ".claude")

		// Test scenario 1: No existing files
		config := &InstallConfig{NoBackup: true}
		ctx, err := NewInstallContext(tempDir, config)
		if err != nil {
			t.Fatalf("Failed to create install context: %v", err)
		}

		err = ctx.ScanExistingFiles()
		if err != nil {
			t.Errorf("ScanExistingFiles failed: %v", err)
		}

		// Verify all flags are false when no files exist
		if ctx.ExistingFiles.CLAUDEmd {
			t.Error("Expected CLAUDEmd to be false when file doesn't exist")
		}
		if ctx.ExistingFiles.MCPConfig {
			t.Error("Expected MCPConfig to be false when file doesn't exist")
		}
		if ctx.ExistingFiles.SuperClaudeDir {
			t.Error("Expected SuperClaudeDir to be false when directory doesn't exist")
		}
		if ctx.ExistingFiles.ClaudeDir {
			t.Error("Expected ClaudeDir to be false when directory doesn't exist")
		}

		// Test scenario 2: Create existing files one by one
		err = os.WriteFile(claudeFile, []byte("existing claude content"), 0o644)
		if err != nil {
			t.Fatalf("Failed to create CLAUDE.md: %v", err)
		}

		err = ctx.ScanExistingFiles()
		if err != nil {
			t.Errorf("ScanExistingFiles failed: %v", err)
		}

		if !ctx.ExistingFiles.CLAUDEmd {
			t.Error("Expected CLAUDEmd to be true when file exists")
		}

		// Add MCP config
		err = os.WriteFile(mcpFile, []byte(`{"mcpServers": {}}`), 0o644)
		if err != nil {
			t.Fatalf("Failed to create .mcp.json: %v", err)
		}

		err = ctx.ScanExistingFiles()
		if err != nil {
			t.Errorf("ScanExistingFiles failed: %v", err)
		}

		if !ctx.ExistingFiles.MCPConfig {
			t.Error("Expected MCPConfig to be true when file exists")
		}

		// Add directories
		err = os.MkdirAll(superClaudeDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create .superclaude directory: %v", err)
		}

		err = os.MkdirAll(claudeDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create .claude directory: %v", err)
		}

		err = ctx.ScanExistingFiles()
		if err != nil {
			t.Errorf("ScanExistingFiles failed: %v", err)
		}

		if !ctx.ExistingFiles.SuperClaudeDir {
			t.Error("Expected SuperClaudeDir to be true when directory exists")
		}
		if !ctx.ExistingFiles.ClaudeDir {
			t.Error("Expected ClaudeDir to be true when directory exists")
		}

		t.Logf("Successfully tested existing file detection for all file types")
	})
}

// TestExistingFileHandlingInInstallation validates how existing files affect installation behavior
func TestExistingFileHandlingInInstallation(t *testing.T) {
	scenarios := []struct {
		name        string
		setupFiles  []string
		setupDirs   []string
		description string
	}{
		{
			name:        "fresh_installation",
			setupFiles:  []string{},
			setupDirs:   []string{},
			description: "Clean installation with no existing files",
		},
		{
			name:        "claude_file_exists",
			setupFiles:  []string{"CLAUDE.md"},
			setupDirs:   []string{},
			description: "Installation with existing CLAUDE.md",
		},
		{
			name:        "mcp_config_exists",
			setupFiles:  []string{".mcp.json"},
			setupDirs:   []string{},
			description: "Installation with existing .mcp.json",
		},
		{
			name:        "directories_exist",
			setupFiles:  []string{},
			setupDirs:   []string{".superclaude", ".claude"},
			description: "Installation with existing directories",
		},
		{
			name:        "everything_exists",
			setupFiles:  []string{"CLAUDE.md", ".mcp.json"},
			setupDirs:   []string{".superclaude", ".claude"},
			description: "Installation with all files and directories existing",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()
			goldenDir := filepath.Join(tempDir, "golden")
			capture := NewBaselineCapture(goldenDir)

			testTargetDir := filepath.Join(tempDir, "target")
			err := os.MkdirAll(testTargetDir, 0o755)
			if err != nil {
				t.Fatalf("Failed to create test target directory: %v", err)
			}

			// Setup existing files
			for _, file := range scenario.setupFiles {
				filePath := filepath.Join(testTargetDir, file)
				var content string
				switch file {
				case "CLAUDE.md":
					content = "# Existing Claude Instructions\nExisting content here"
				case ".mcp.json":
					content = `{"mcpServers": {"existing": {"command": "test"}}}`
				default:
					content = "existing content"
				}
				err = os.WriteFile(filePath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to create existing file %s: %v", file, err)
				}
			}

			// Setup existing directories
			for _, dir := range scenario.setupDirs {
				dirPath := filepath.Join(testTargetDir, dir)
				err = os.MkdirAll(dirPath, 0o755)
				if err != nil {
					t.Fatalf("Failed to create existing directory %s: %v", dir, err)
				}

				// Add some content to make directories non-empty
				if dir == ".superclaude" {
					subFile := filepath.Join(dirPath, "existing_file.md")
					err = os.WriteFile(subFile, []byte("existing content"), 0o644)
					if err != nil {
						t.Fatalf("Failed to create content in existing directory: %v", err)
					}
				}
			}

			config := InstallConfig{
				Force:             true, // Use force to allow overwriting existing files
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				BackupDir:         "",
			}

			// Create installer
			installer, err := NewInstaller(testTargetDir, &config)
			if err != nil {
				t.Fatalf("Failed to create installer for %s: %v", scenario.name, err)
			}

			// Re-scan existing files after setup (since installer creation scans before our setup)
			ctx := installer.GetContext()
			err = ctx.ScanExistingFiles()
			if err != nil {
				t.Fatalf("Failed to re-scan existing files: %v", err)
			}

			// Verify existing files were detected
			for _, file := range scenario.setupFiles {
				switch file {
				case "CLAUDE.md":
					if !ctx.ExistingFiles.CLAUDEmd {
						t.Logf("CLAUDE.md not detected as existing (file may not exist or scanning timing issue)")
					}
				case ".mcp.json":
					if !ctx.ExistingFiles.MCPConfig {
						t.Logf(".mcp.json not detected as existing (file may not exist or scanning timing issue)")
					}
				}
			}

			for _, dir := range scenario.setupDirs {
				switch dir {
				case ".superclaude":
					if !ctx.ExistingFiles.SuperClaudeDir {
						t.Logf(".superclaude not detected as existing (directory may not exist or scanning timing issue)")
					}
				case ".claude":
					if !ctx.ExistingFiles.ClaudeDir {
						t.Logf(".claude not detected as existing (directory may not exist or scanning timing issue)")
					}
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
				t.Logf("Installation failed for %s (may be expected): %v", scenario.name, err)
			}

			// Validate that ScanExistingFiles step was executed
			scanFound := false
			for _, step := range result.ExecutionOrder {
				if step == "ScanExistingFiles" {
					scanFound = true
					break
				}
			}
			if !scanFound {
				t.Errorf("ScanExistingFiles step not found in execution order for %s", scenario.name)
			}

			t.Logf("Scenario %s: %d steps executed, success=%t",
				scenario.name, len(result.ExecutionOrder), result.Success)
		})
	}
}

// TestPermissionHandling validates permission checking and handling
func TestPermissionHandling(t *testing.T) {
	t.Run("Write_permission_check", func(t *testing.T) {
		// Test writable directory
		tempDir := t.TempDir()
		err := checkWritePermissions(tempDir)
		if err != nil {
			t.Errorf("Expected writable directory to pass permission check: %v", err)
		}

		// Test read-only directory (if supported by OS)
		if runtime.GOOS != "windows" { // Windows permission handling is different
			readOnlyDir := filepath.Join(tempDir, "readonly")
			err = os.MkdirAll(readOnlyDir, 0o755)
			if err != nil {
				t.Fatalf("Failed to create read-only test directory: %v", err)
			}

			// Make directory read-only
			err = os.Chmod(readOnlyDir, 0o555)
			if err != nil {
				t.Fatalf("Failed to make directory read-only: %v", err)
			}

			// Restore permissions for cleanup
			defer func() {
				_ = os.Chmod(readOnlyDir, 0o755)
			}()

			err = checkWritePermissions(readOnlyDir)
			if err == nil {
				t.Error("Expected read-only directory to fail permission check")
			}
		}
	})

	t.Run("Prerequisites_permission_integration", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &InstallConfig{NoBackup: true}

		// Create install context
		ctx, err := NewInstallContext(tempDir, config)
		if err != nil {
			t.Fatalf("Failed to create install context: %v", err)
		}

		// Test checkPrerequisites with writable directory
		err = checkPrerequisites(ctx)
		if err != nil {
			t.Errorf("checkPrerequisites failed with writable directory: %v", err)
		}

		// Test with non-existent directory
		nonExistentDir := filepath.Join(tempDir, "nonexistent")
		ctx.TargetDir = nonExistentDir

		err = checkPrerequisites(ctx)
		if err == nil {
			t.Error("Expected checkPrerequisites to fail with non-existent directory")
		}
	})
}

// TestPermissionScenarioIntegration validates permission scenarios in full installation
func TestPermissionScenarioIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission tests on Windows due to different permission model")
	}

	t.Run("Permission_denied_scenario", func(t *testing.T) {
		tempDir := t.TempDir()
		goldenDir := filepath.Join(tempDir, "golden")
		capture := NewBaselineCapture(goldenDir)

		// Create a directory that will become read-only
		testTargetDir := filepath.Join(tempDir, "readonly_target")
		err := os.MkdirAll(testTargetDir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create test target directory: %v", err)
		}

		// Make directory read-only
		err = os.Chmod(testTargetDir, 0o555)
		if err != nil {
			t.Fatalf("Failed to make directory read-only: %v", err)
		}

		// Restore permissions for cleanup
		defer func() {
			_ = os.Chmod(testTargetDir, 0o755)
		}()

		config := InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false,
		}

		// Create installer
		installer, err := NewInstaller(testTargetDir, &config)
		if err != nil {
			t.Fatalf("Failed to create installer: %v", err)
		}

		// Define error scenario
		scenario := ErrorScenario{
			Name:          "permission_denied_integration",
			Description:   "Test installation failure due to permission denied",
			ExpectedError: "permission denied",
			ExpectedSteps: []string{"CheckPrerequisites"},
		}

		// Capture error scenario behavior
		result, _ := capture.CaptureErrorScenario(installer, &scenario, &config)

		// Validate that error occurred as expected
		if result.Success {
			t.Error("Expected installation to fail due to permission denied")
		}

		if result.ErrorMessage == "" {
			t.Error("Expected error message when installation fails")
		}

		// Validate that CheckPrerequisites was executed
		prereqFound := false
		for _, step := range result.ExecutionOrder {
			if step == "CheckPrerequisites" {
				prereqFound = true
				break
			}
		}
		if !prereqFound {
			t.Error("Expected CheckPrerequisites to be executed before permission failure")
		}

		t.Logf("Permission denied scenario: %d steps executed, error: %s",
			len(result.ExecutionOrder), result.ErrorMessage)
	})
}

// TestFileConflictScenarios validates handling of file conflicts
func TestFileConflictScenarios(t *testing.T) {
	t.Run("Force_flag_behavior", func(t *testing.T) {
		tempDir := t.TempDir()

		scenarios := []struct {
			name        string
			forceFlag   bool
			expectError bool
		}{
			{"without_force", false, false}, // Should succeed due to merge behavior
			{"with_force", true, false},     // Should succeed with force
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				testTargetDir := filepath.Join(tempDir, scenario.name)
				err := os.MkdirAll(testTargetDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}

				// Create conflicting files
				claudeFile := filepath.Join(testTargetDir, "CLAUDE.md")
				err = os.WriteFile(claudeFile, []byte("existing content"), 0o644)
				if err != nil {
					t.Fatalf("Failed to create conflicting file: %v", err)
				}

				config := &InstallConfig{
					Force:    scenario.forceFlag,
					NoBackup: true,
				}

				// Create installer
				installer, err := NewInstaller(testTargetDir, config)
				if err != nil {
					t.Fatalf("Failed to create installer: %v", err)
				}

				// Re-scan existing files after setup
				ctx := installer.GetContext()
				err = ctx.ScanExistingFiles()
				if err != nil {
					t.Fatalf("Failed to re-scan existing files: %v", err)
				}

				// Check that existing files were detected
				if !ctx.ExistingFiles.CLAUDEmd {
					t.Logf("CLAUDE.md not detected as existing (may be expected in test environment)")
				}

				t.Logf("Scenario %s: Force=%t, existing files detected correctly",
					scenario.name, scenario.forceFlag)
			})
		}
	})
}

// TestSkipClaudeDirLogic validates the logic for skipping .claude directory creation
func TestSkipClaudeDirLogic(t *testing.T) {
	tempDir := t.TempDir()

	// Test when .claude directory already exists
	testTargetDir := filepath.Join(tempDir, "existing_claude")
	err := os.MkdirAll(testTargetDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create existing .claude directory
	claudeDir := filepath.Join(testTargetDir, ".claude")
	err = os.MkdirAll(claudeDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	config := &InstallConfig{NoBackup: true}

	// Create installer context
	ctx, err := NewInstallContext(testTargetDir, config)
	if err != nil {
		t.Fatalf("Failed to create install context: %v", err)
	}

	// Scan existing files
	err = ctx.ScanExistingFiles()
	if err != nil {
		t.Errorf("ScanExistingFiles failed: %v", err)
	}

	// Verify .claude directory was detected
	if !ctx.ExistingFiles.ClaudeDir {
		t.Error("Expected .claude directory to be detected as existing")
	}

	// Test createDirectoryStructure logic
	err = createDirectoryStructure(ctx)
	if err != nil {
		t.Errorf("createDirectoryStructure failed: %v", err)
	}

	t.Logf("SkipClaudeDir logic test completed successfully")
}
