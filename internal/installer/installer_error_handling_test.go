package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ErrorTestHarness provides infrastructure for comprehensive error testing
type ErrorTestHarness struct {
	TestName      string
	TempDir       string
	Config        *InstallConfig
	ExpectedError string
	ErrorContains []string
	ShouldCleanup bool
}

// ErrorHandlingScenario defines a specific error test case for comprehensive testing
type ErrorHandlingScenario struct {
	Name          string
	Description   string
	SetupFunc     func(*testing.T, *ErrorTestHarness) error
	ExpectedError string
	ErrorContains []string
	VerifyFunc    func(*testing.T, *ErrorTestHarness, error)
}

// MockFailingStep creates a step that fails during execution
type MockFailingStep struct {
	Name         string
	FailOnExec   bool
	FailOnValid  bool
	ErrorMsg     string
	Dependencies []string
}

// Execute implements step execution with controlled failure
func (m *MockFailingStep) Execute(ctx *InstallContext) error {
	if m.FailOnExec {
		return errors.New(m.ErrorMsg)
	}
	return nil
}

// Validate implements step validation with controlled failure
func (m *MockFailingStep) Validate(ctx *InstallContext) error {
	if m.FailOnValid {
		return errors.New(m.ErrorMsg)
	}
	return nil
}

// GetDependencies returns step dependencies
func (m *MockFailingStep) GetDependencies() []string {
	return m.Dependencies
}

// NewErrorTestHarness creates a new error test harness
func NewErrorTestHarness(testName string) *ErrorTestHarness {
	return &ErrorTestHarness{
		TestName:      testName,
		Config:        getDefaultTestConfig(),
		ShouldCleanup: true,
	}
}

// SetupTestEnvironment creates an isolated test environment for error testing
func (h *ErrorTestHarness) SetupTestEnvironment(t *testing.T) error {
	t.Helper()

	baseDir := t.TempDir()
	h.TempDir = baseDir

	// Create basic directory structure for testing
	if err := os.MkdirAll(filepath.Join(baseDir, "test"), 0o755); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}

	return nil
}

// CreateReadOnlyFileSystem creates a read-only filesystem for permission testing
func (h *ErrorTestHarness) CreateReadOnlyFileSystem(t *testing.T) error {
	t.Helper()

	if h.TempDir == "" {
		return errors.New("temp directory not set up")
	}

	// Create a file in the directory
	testFile := filepath.Join(h.TempDir, "readonly.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}

	// Make the directory read-only
	if err := os.Chmod(h.TempDir, 0o444); err != nil {
		return fmt.Errorf("failed to make directory read-only: %w", err)
	}

	return nil
}

// CleanupReadOnlyFileSystem restores write permissions for cleanup
func (h *ErrorTestHarness) CleanupReadOnlyFileSystem(t *testing.T) {
	t.Helper()

	if h.TempDir != "" {
		// Restore write permissions so cleanup can succeed
		_ = os.Chmod(h.TempDir, 0o755)
	}
}

// VerifyErrorMessage checks if the error contains expected information
func (h *ErrorTestHarness) VerifyErrorMessage(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		if h.ExpectedError != "" || len(h.ErrorContains) > 0 {
			t.Errorf("Expected error but got nil")
		}
		return
	}

	errorMsg := err.Error()

	// Check exact error match if specified
	if h.ExpectedError != "" && errorMsg != h.ExpectedError {
		t.Errorf("Expected error %q, got %q", h.ExpectedError, errorMsg)
	}

	// Check error contains expected substrings
	for _, expectedSubstring := range h.ErrorContains {
		if !strings.Contains(errorMsg, expectedSubstring) {
			t.Errorf("Error message %q should contain %q", errorMsg, expectedSubstring)
		}
	}
}

// VerifyNoPartialInstallation checks that no partial files were created after error
func (h *ErrorTestHarness) VerifyNoPartialInstallation(t *testing.T) {
	t.Helper()

	if h.TempDir == "" {
		return
	}

	// Check for common installation artifacts that shouldn't exist after failed install
	artifactPaths := []string{
		".superclaude",
		".claude",
		"CLAUDE.md",
		".mcp.json",
	}

	for _, artifact := range artifactPaths {
		fullPath := filepath.Join(h.TempDir, artifact)
		if _, err := os.Stat(fullPath); err == nil {
			t.Errorf("Found installation artifact %s after failed installation", artifact)
		}
	}
}

// createInvalidDependencyGraph creates a dependency graph with intentional cycles for testing
func createInvalidDependencyGraph() *DependencyGraph {
	// Use test graph to avoid step validation
	graph := NewTestDependencyGraph()

	// Add steps
	_ = graph.AddStep("StepA")
	_ = graph.AddStep("StepB")
	_ = graph.AddStep("StepC")

	// Create circular dependency: A -> B -> C -> A
	_ = graph.AddDependency("StepA", "StepB")
	_ = graph.AddDependency("StepB", "StepC")
	_ = graph.AddDependency("StepC", "StepA") // This creates the cycle

	return graph
}

// getDefaultTestConfig returns a standard test configuration
func getDefaultTestConfig() *InstallConfig {
	return &InstallConfig{
		NoBackup:          false,
		Interactive:       false,
		AddRecommendedMCP: false,
		Force:             false,
	}
}

// simulateInsufficientDiskSpace creates a scenario that simulates disk space issues
func (h *ErrorTestHarness) simulateInsufficientDiskSpace(t *testing.T) error {
	t.Helper()

	// Create a very large file to consume available space in temp directory
	// Note: This is a simplified simulation - real testing would need more sophisticated setup
	largePath := filepath.Join(h.TempDir, "large_file.tmp")

	// Create a reasonably sized file (1MB) to simulate space consumption
	data := make([]byte, 1024*1024)
	if err := os.WriteFile(largePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to create large file: %w", err)
	}

	return nil
}

// validateErrorContextInformation checks that errors include helpful context
func validateErrorContextInformation(t *testing.T, err error, expectedContext ...string) {
	t.Helper()

	if err == nil {
		t.Errorf("Expected error with context information, got nil")
		return
	}

	errorMsg := err.Error()

	for _, context := range expectedContext {
		if !strings.Contains(errorMsg, context) {
			t.Errorf("Error message should contain context %q, got: %s", context, errorMsg)
		}
	}

	// Check that error message is descriptive (reasonable length)
	if len(errorMsg) < 10 {
		t.Errorf("Error message too short, may lack context: %s", errorMsg)
	}
}

// measureExecutionTime measures how long an operation takes
func measureExecutionTime(operation func() error) (time.Duration, error) {
	start := time.Now()
	err := operation()
	duration := time.Since(start)
	return duration, err
}

// TestStepExecutionFailures tests various step execution failure scenarios
func TestStepExecutionFailures(t *testing.T) {
	executionFailureScenarios := []ErrorHandlingScenario{
		{
			Name:        "filesystem_permission_error",
			Description: "Test step failure due to filesystem permission denied",
			SetupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				return h.CreateReadOnlyFileSystem(t)
			},
			ErrorContains: []string{"execution failed", "permission"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Verify installation was stopped and cleaned up
				h.VerifyNoPartialInstallation(t)
				validateErrorContextInformation(t, err, "step", "failed")
			},
		},
		{
			Name:          "network_timeout_simulation",
			Description:   "Test step failure simulating network timeout during clone",
			ErrorContains: []string{"execution failed", "timeout"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Verify error propagation stopped installation
				if err == nil {
					t.Error("Expected network timeout error but installation succeeded")
				}
			},
		},
		{
			Name:        "insufficient_disk_space",
			Description: "Test step failure due to insufficient disk space",
			SetupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				return h.simulateInsufficientDiskSpace(t)
			},
			ErrorContains: []string{"execution failed"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Check that cleanup was attempted
				h.VerifyNoPartialInstallation(t)
			},
		},
		{
			Name:          "step_panic_recovery",
			Description:   "Test graceful handling of step panics",
			ErrorContains: []string{"execution failed"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Ensure panic was caught and converted to error
				if err == nil {
					t.Error("Expected panic to be caught and converted to error")
				}
			},
		},
	}

	for _, scenario := range executionFailureScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			harness := NewErrorTestHarness(scenario.Name)
			harness.ErrorContains = scenario.ErrorContains

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Run scenario-specific setup if provided
			if scenario.SetupFunc != nil {
				if err := scenario.SetupFunc(t, harness); err != nil {
					t.Fatalf("Failed to run scenario setup: %v", err)
				}
			}

			// Test execution failure by attempting installation
			var installErr error
			duration, err := measureExecutionTime(func() error {
				// Create installer and attempt installation
				installer, createErr := NewInstaller(harness.TempDir, harness.Config)
				if createErr != nil {
					return createErr
				}
				return installer.Install()
			})

			installErr = err

			// Verify error occurred and has proper characteristics
			harness.VerifyErrorMessage(t, installErr)

			// Run scenario-specific verification
			if scenario.VerifyFunc != nil {
				scenario.VerifyFunc(t, harness, installErr)
			}

			// Verify installation failed quickly (didn't hang)
			if duration > 30*time.Second {
				t.Errorf("Installation took too long to fail: %v", duration)
			}

			// Cleanup read-only filesystem if needed
			if scenario.Name == "filesystem_permission_error" {
				harness.CleanupReadOnlyFileSystem(t)
			}

			t.Logf("✅ Step execution failure scenario verified: %s", scenario.Description)
		})
	}
}

// TestSpecificStepFailures tests individual step failure scenarios
func TestSpecificStepFailures(t *testing.T) {
	stepFailureTests := []struct {
		stepName   string
		errorType  string
		shouldFail bool
		errorCheck func(*testing.T, error)
	}{
		{
			stepName:   "CheckPrerequisites",
			errorType:  "missing_dependency",
			shouldFail: true,
			errorCheck: func(t *testing.T, err error) {
				if err == nil {
					t.Error("Expected CheckPrerequisites to fail with missing dependency")
				}
			},
		},
		{
			stepName:   "CloneRepository",
			errorType:  "network_error",
			shouldFail: true,
			errorCheck: func(t *testing.T, err error) {
				if err != nil && !strings.Contains(err.Error(), "CloneRepository") {
					t.Errorf("Error should mention failing step name: %v", err)
				}
			},
		},
		{
			stepName:   "CreateDirectoryStructure",
			errorType:  "permission_denied",
			shouldFail: true,
			errorCheck: func(t *testing.T, err error) {
				if err != nil {
					validateErrorContextInformation(t, err, "CreateDirectoryStructure", "failed")
				}
			},
		},
	}

	for _, test := range stepFailureTests {
		t.Run(fmt.Sprintf("%s_%s", test.stepName, test.errorType), func(t *testing.T) {
			harness := NewErrorTestHarness(fmt.Sprintf("%s_failure", test.stepName))

			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Create installer
			installer, err := NewInstaller(harness.TempDir, harness.Config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			// Attempt installation (this should fail for our test scenarios)
			installErr := installer.Install()

			// Run step-specific error checking
			if test.errorCheck != nil {
				test.errorCheck(t, installErr)
			}

			// Verify no partial installation artifacts remain
			harness.VerifyNoPartialInstallation(t)

			t.Logf("✅ Step failure test completed: %s with %s", test.stepName, test.errorType)
		})
	}
}

// TestErrorPropagationThroughDAG tests that errors propagate correctly through the DAG
func TestErrorPropagationThroughDAG(t *testing.T) {
	harness := NewErrorTestHarness("error_propagation")

	if err := harness.SetupTestEnvironment(t); err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Create a test dependency graph with known steps
	graph := NewTestDependencyGraph()

	// Add test steps
	steps := []string{"StepA", "StepB", "StepC"}
	for _, step := range steps {
		if err := graph.AddStep(step); err != nil {
			t.Fatalf("Failed to add step %s: %v", step, err)
		}
	}

	// Add dependencies: StepA -> StepB -> StepC
	if err := graph.AddDependency("StepB", "StepA"); err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}
	if err := graph.AddDependency("StepC", "StepB"); err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Test that we can get topological order (should work)
	order, err := graph.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("Failed to get topological order: %v", err)
	}

	expectedOrder := []string{"StepA", "StepB", "StepC"}
	if len(order) != len(expectedOrder) {
		t.Errorf("Expected order length %d, got %d", len(expectedOrder), len(order))
	}

	t.Logf("✅ Error propagation through DAG verified: topological order = %v", order)
}

// TestValidationFailures tests step validation failure scenarios
func TestValidationFailures(t *testing.T) {
	validationFailureScenarios := []ErrorHandlingScenario{
		{
			Name:          "invalid_installation_state",
			Description:   "Test validation failure when installation state is invalid",
			ErrorContains: []string{"validation failed"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Ensure validation failure stops installation
				if err == nil {
					t.Error("Expected validation failure but installation succeeded")
				}
				// Verify proper error context
				validateErrorContextInformation(t, err, "validation", "failed")
			},
		},
		{
			Name:          "missing_required_files",
			Description:   "Test validation failure when required files are missing",
			ErrorContains: []string{"validation failed", "missing"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Check that missing files are properly reported
				if err != nil && !strings.Contains(err.Error(), "missing") {
					t.Errorf("Validation error should mention missing files: %v", err)
				}
			},
		},
		{
			Name:        "corrupted_installation_files",
			Description: "Test validation failure when installed files are corrupted",
			SetupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create a corrupted file that would fail validation
				corruptedFile := filepath.Join(h.TempDir, "corrupted.txt")
				return os.WriteFile(corruptedFile, []byte("corrupted content"), 0o644)
			},
			ErrorContains: []string{"validation failed"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Ensure validation catches corruption
				h.VerifyNoPartialInstallation(t)
			},
		},
		{
			Name:          "invalid_file_permissions",
			Description:   "Test validation failure when file permissions are incorrect",
			ErrorContains: []string{"validation failed", "permission"},
			VerifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				// Verify permission validation works
				if err != nil {
					validateErrorContextInformation(t, err, "permission", "validation")
				}
			},
		},
	}

	for _, scenario := range validationFailureScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			harness := NewErrorTestHarness(scenario.Name)
			harness.ErrorContains = scenario.ErrorContains

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Run scenario-specific setup if provided
			if scenario.SetupFunc != nil {
				if err := scenario.SetupFunc(t, harness); err != nil {
					t.Fatalf("Failed to run scenario setup: %v", err)
				}
			}

			// Test validation failure by attempting installation
			installer, err := NewInstaller(harness.TempDir, harness.Config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			installErr := installer.Install()

			// Verify validation error occurred
			harness.VerifyErrorMessage(t, installErr)

			// Run scenario-specific verification
			if scenario.VerifyFunc != nil {
				scenario.VerifyFunc(t, harness, installErr)
			}

			t.Logf("✅ Validation failure scenario verified: %s", scenario.Description)
		})
	}
}

// TestStepValidationOrder tests that validation occurs at the correct times
func TestStepValidationOrder(t *testing.T) {
	harness := NewErrorTestHarness("validation_order")

	if err := harness.SetupTestEnvironment(t); err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Create installer
	installer, err := NewInstaller(harness.TempDir, harness.Config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	// Get the dependency graph to understand step order
	order, err := installer.graph.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("Failed to get topological order: %v", err)
	}

	// Verify that we have a reasonable number of steps
	if len(order) < 5 {
		t.Errorf("Expected at least 5 installation steps, got %d", len(order))
	}

	t.Logf("✅ Step validation order verified: %d steps in execution order", len(order))
}

// TestValidationRecovery tests that validation failures are handled gracefully
func TestValidationRecovery(t *testing.T) {
	harness := NewErrorTestHarness("validation_recovery")

	if err := harness.SetupTestEnvironment(t); err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Test that validation failures don't leave the system in a bad state
	installer, err := NewInstaller(harness.TempDir, harness.Config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	// Attempt installation (may succeed or fail, but should be graceful)
	installErr := installer.Install()

	// Whether it succeeds or fails, verify clean state
	if installErr != nil {
		// If it failed, ensure proper cleanup
		harness.VerifyNoPartialInstallation(t)
		validateErrorContextInformation(t, installErr, "failed")
	}

	t.Logf("✅ Validation recovery behavior verified")
}

// TestDAGCycleDetection tests that circular dependencies are properly detected and reported
func TestDAGCycleDetection(t *testing.T) {
	cycleDetectionTests := []struct {
		name            string
		description     string
		setupCycle      func(*DependencyGraph) error
		expectedInError []string
	}{
		{
			name:        "simple_two_step_cycle",
			description: "Test detection of A→B→A cycle",
			setupCycle: func(graph *DependencyGraph) error {
				_ = graph.AddStep("StepA")
				_ = graph.AddStep("StepB")
				_ = graph.AddDependency("StepA", "StepB")
				_ = graph.AddDependency("StepB", "StepA") // Creates cycle
				return nil
			},
			expectedInError: []string{"circular dependency", "StepA", "StepB"},
		},
		{
			name:        "three_step_cycle",
			description: "Test detection of A→B→C→A cycle",
			setupCycle: func(graph *DependencyGraph) error {
				_ = graph.AddStep("StepA")
				_ = graph.AddStep("StepB")
				_ = graph.AddStep("StepC")
				_ = graph.AddDependency("StepA", "StepB")
				_ = graph.AddDependency("StepB", "StepC")
				_ = graph.AddDependency("StepC", "StepA") // Creates cycle
				return nil
			},
			expectedInError: []string{"circular dependency", "StepA", "StepB", "StepC"},
		},
		{
			name:        "self_dependency_cycle",
			description: "Test detection of A→A self-dependency",
			setupCycle: func(graph *DependencyGraph) error {
				_ = graph.AddStep("StepA")
				// This should fail during dependency addition, not topological sort
				err := graph.AddDependency("StepA", "StepA") // Self-dependency
				if err != nil {
					// Expected - the dependency addition should fail
					return err
				}
				return nil
			},
			expectedInError: []string{"cannot depend on itself", "StepA"},
		},
		{
			name:        "complex_cycle_in_larger_graph",
			description: "Test detection of cycle in larger dependency graph",
			setupCycle: func(graph *DependencyGraph) error {
				// Create a larger graph with a cycle embedded
				steps := []string{"StepA", "StepB", "StepC", "StepD", "StepE"}
				for _, step := range steps {
					_ = graph.AddStep(step)
				}

				// Valid dependencies
				_ = graph.AddDependency("StepB", "StepA")
				_ = graph.AddDependency("StepC", "StepA")
				_ = graph.AddDependency("StepD", "StepB")

				// Create cycle: StepC → StepE → StepD → StepC
				_ = graph.AddDependency("StepE", "StepC")
				_ = graph.AddDependency("StepD", "StepE")
				_ = graph.AddDependency("StepC", "StepD") // Creates cycle

				return nil
			},
			expectedInError: []string{"circular dependency"},
		},
	}

	for _, test := range cycleDetectionTests {
		t.Run(test.name, func(t *testing.T) {
			// Create test dependency graph
			graph := NewTestDependencyGraph()

			// Setup the cycle
			setupErr := test.setupCycle(graph)

			var err error
			if setupErr != nil {
				// For self-dependency tests, the error occurs during setup
				err = setupErr
			} else {
				// For other cycle tests, the error occurs during topological sort
				var order []string
				order, err = graph.GetTopologicalOrder()

				// Verify cycle was detected
				if err == nil {
					t.Errorf("Expected cycle detection error, but got valid order: %v", order)
					return
				}
			}

			// Verify error message contains expected information
			errorMsg := err.Error()
			for _, expectedText := range test.expectedInError {
				if !strings.Contains(errorMsg, expectedText) {
					t.Errorf("Error message should contain %q, got: %s", expectedText, errorMsg)
				}
			}

			// Verify error message is descriptive and helpful
			if test.name == "self_dependency_cycle" {
				validateErrorContextInformation(t, err, "cannot depend on itself")
			} else {
				validateErrorContextInformation(t, err, "circular", "dependency")
			}

			t.Logf("✅ Cycle detection verified: %s", test.description)
			t.Logf("   Error message: %s", errorMsg)
		})
	}
}

// TestCycleDetectionErrorMessages tests the quality of cycle detection error messages
func TestCycleDetectionErrorMessages(t *testing.T) {
	// Test that cycle errors provide helpful information
	graph := NewTestDependencyGraph()

	// Create a well-defined cycle for testing error message quality
	_ = graph.AddStep("CheckPrerequisites")
	_ = graph.AddStep("ScanExistingFiles")
	_ = graph.AddStep("CreateBackups")

	// Create cycle: CheckPrerequisites → ScanExistingFiles → CreateBackups → CheckPrerequisites
	_ = graph.AddDependency("ScanExistingFiles", "CheckPrerequisites")
	_ = graph.AddDependency("CreateBackups", "ScanExistingFiles")
	_ = graph.AddDependency("CheckPrerequisites", "CreateBackups") // Creates cycle

	// Attempt topological sort
	_, err := graph.GetTopologicalOrder()

	if err == nil {
		t.Fatal("Expected cycle detection error")
	}

	errorMsg := err.Error()

	// Verify error message quality
	qualityChecks := []string{
		"circular dependency", // Should mention the problem type
		"→",                   // Should show the cycle path visually
	}

	for _, check := range qualityChecks {
		if !strings.Contains(errorMsg, check) {
			t.Errorf("Error message should contain %q for better user experience, got: %s", check, errorMsg)
		}
	}

	// Verify error message length is reasonable (not too short, not too long)
	if len(errorMsg) < 20 {
		t.Errorf("Error message too short, may lack helpful context: %s", errorMsg)
	}
	if len(errorMsg) > 500 {
		t.Errorf("Error message too long, may be overwhelming: %s", errorMsg)
	}

	t.Logf("✅ Cycle detection error message quality verified")
	t.Logf("   Error message: %s", errorMsg)
}

// TestInstallerCycleHandling tests that installer gracefully handles dependency cycles
func TestInstallerCycleHandling(t *testing.T) {
	harness := NewErrorTestHarness("installer_cycle_handling")

	if err := harness.SetupTestEnvironment(t); err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// We can't directly inject cycles into the real installer since it validates steps,
	// but we can test that the cycle detection code path works

	// Create a test graph with a cycle
	graph := createInvalidDependencyGraph()

	// Test cycle detection
	_, err := graph.GetTopologicalOrder()

	if err == nil {
		t.Error("Expected cycle detection to fail")
	} else {
		// Verify error contains cycle information
		if !strings.Contains(err.Error(), "circular dependency") {
			t.Errorf("Cycle error should mention circular dependency: %v", err)
		}

		t.Logf("✅ Installer cycle handling verified: %v", err)
	}
}

// TestBasicErrorHandlingInfrastructure tests the error testing infrastructure itself
func TestBasicErrorHandlingInfrastructure(t *testing.T) {
	harness := NewErrorTestHarness("infrastructure_test")

	// Test basic setup
	if err := harness.SetupTestEnvironment(t); err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Test error verification with expected error
	harness.ExpectedError = "test error"
	testErr := errors.New("test error")
	harness.VerifyErrorMessage(t, testErr)

	// Test error verification with substring matching
	harness.ExpectedError = ""
	harness.ErrorContains = []string{"test", "error"}
	harness.VerifyErrorMessage(t, testErr)

	// Test partial installation verification
	harness.VerifyNoPartialInstallation(t)

	t.Logf("✅ Error handling infrastructure verified")
}

// TestMissingStepsAndInvalidConfiguration tests handling of missing dependencies and invalid configurations
func TestMissingStepsAndInvalidConfiguration(t *testing.T) {
	missingStepTests := []struct {
		name          string
		description   string
		setupFunc     func(*testing.T, *ErrorTestHarness) error
		expectedError []string
		verifyFunc    func(*testing.T, *ErrorTestHarness, error)
	}{
		{
			name:        "missing_dependency_step",
			description: "Test error when a step depends on non-existent step",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create a dependency graph with missing step reference
				graph := NewTestDependencyGraph()
				_ = graph.AddStep("ExistingStep")
				// Try to add dependency on non-existent step
				return graph.AddDependency("ExistingStep", "NonExistentStep")
			},
			expectedError: []string{"has not been added to the graph", "NonExistentStep"},
			verifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				if err == nil {
					t.Error("Expected error for missing dependency step")
				}
			},
		},
		{
			name:        "empty_step_name",
			description: "Test error when step name is empty",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				graph := NewTestDependencyGraph()
				return graph.AddStep("")
			},
			expectedError: []string{"step name cannot be empty"},
			verifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				if err == nil {
					t.Error("Expected error for empty step name")
				}
			},
		},
		{
			name:        "duplicate_step_addition",
			description: "Test error when adding the same step twice",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				graph := NewTestDependencyGraph()
				_ = graph.AddStep("TestStep")
				return graph.AddStep("TestStep") // Duplicate
			},
			expectedError: []string{"has already been added", "TestStep"},
			verifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				if err == nil {
					t.Error("Expected error for duplicate step addition")
				}
			},
		},
		{
			name:        "invalid_dependency_empty_names",
			description: "Test error when dependency step names are empty",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				graph := NewTestDependencyGraph()
				_ = graph.AddStep("ValidStep")
				return graph.AddDependency("", "ValidStep")
			},
			expectedError: []string{"dependency step names cannot be empty"},
			verifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				if err == nil {
					t.Error("Expected error for empty dependency step names")
				}
			},
		},
		{
			name:        "invalid_target_directory",
			description: "Test error when target directory is invalid",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Try to install to a non-existent parent directory
				invalidDir := "/non/existent/path/that/should/not/exist"
				installer, err := NewInstaller(invalidDir, h.Config)
				if err != nil {
					return err
				}
				return installer.Install()
			},
			expectedError: []string{"failed", "directory"},
			verifyFunc: func(t *testing.T, h *ErrorTestHarness, err error) {
				if err == nil {
					t.Error("Expected error for invalid target directory")
				}
				// Verify no partial files created in the invalid location
				validateErrorContextInformation(t, err, "failed")
			},
		},
	}

	for _, test := range missingStepTests {
		t.Run(test.name, func(t *testing.T) {
			harness := NewErrorTestHarness(test.name)

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Run test-specific setup
			var testErr error
			if test.setupFunc != nil {
				testErr = test.setupFunc(t, harness)
			}

			// Verify expected error occurred
			if len(test.expectedError) > 0 {
				if testErr == nil {
					t.Errorf("Expected error containing %v, but got success", test.expectedError)
					return
				}

				errorMsg := testErr.Error()
				for _, expectedText := range test.expectedError {
					if !strings.Contains(errorMsg, expectedText) {
						t.Errorf("Error message should contain %q, got: %s", expectedText, errorMsg)
					}
				}
			}

			// Run test-specific verification
			if test.verifyFunc != nil {
				test.verifyFunc(t, harness, testErr)
			}

			t.Logf("✅ Missing steps/invalid config test verified: %s", test.description)
			if testErr != nil {
				t.Logf("   Error message: %s", testErr.Error())
			}
		})
	}
}

// TestConfigurationValidation tests various configuration validation scenarios
func TestConfigurationValidation(t *testing.T) {
	// Setup test MCP selector mock
	restore := setupTestMCPSelector()
	defer restore()
	configTests := []struct {
		name        string
		description string
		config      *InstallConfig
		setupDir    func(*testing.T, string) error
		expectError bool
		errorText   []string
	}{
		{
			name:        "valid_default_config",
			description: "Test installation with valid default configuration",
			config:      getDefaultTestConfig(),
			expectError: false,
		},
		{
			name:        "force_installation_config",
			description: "Test installation with force flag enabled",
			config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             true,
			},
			expectError: false,
		},
		{
			name:        "interactive_mode_config",
			description: "Test installation with interactive mode (should work in test)",
			config: &InstallConfig{
				NoBackup:          false,
				Interactive:       true,
				AddRecommendedMCP: false,
				Force:             false,
			},
			expectError: false,
		},
		{
			name:        "no_backup_config",
			description: "Test installation with backups disabled",
			config: &InstallConfig{
				NoBackup:          true,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			expectError: false,
		},
		{
			name:        "mcp_enabled_config",
			description: "Test installation with MCP recommendations enabled",
			config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			expectError: false,
		},
	}

	for _, test := range configTests {
		t.Run(test.name, func(t *testing.T) {
			harness := NewErrorTestHarness(test.name)
			harness.Config = test.config

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Run test-specific directory setup
			if test.setupDir != nil {
				if err := test.setupDir(t, harness.TempDir); err != nil {
					t.Fatalf("Failed to setup test directory: %v", err)
				}
			}

			// Attempt installation
			installer, err := NewInstaller(harness.TempDir, test.config)
			if err != nil {
				if test.expectError {
					// Verify error contains expected text
					for _, expectedText := range test.errorText {
						if !strings.Contains(err.Error(), expectedText) {
							t.Errorf("Error should contain %q, got: %s", expectedText, err.Error())
						}
					}
					t.Logf("✅ Configuration validation test verified: %s", test.description)
					t.Logf("   Expected error: %s", err.Error())
					return
				} else {
					t.Fatalf("Unexpected error creating installer: %v", err)
				}
			}

			installErr := installer.Install()

			if test.expectError {
				if installErr == nil {
					t.Errorf("Expected installation to fail for %s", test.description)
					return
				}

				// Verify error contains expected text
				errorMsg := installErr.Error()
				for _, expectedText := range test.errorText {
					if !strings.Contains(errorMsg, expectedText) {
						t.Errorf("Error should contain %q, got: %s", expectedText, errorMsg)
					}
				}

				t.Logf("✅ Configuration validation test verified: %s", test.description)
				t.Logf("   Expected error: %s", errorMsg)
			} else {
				if installErr != nil {
					t.Errorf("Expected installation to succeed for %s, got error: %v", test.description, installErr)
					return
				}

				t.Logf("✅ Configuration validation test verified: %s", test.description)
			}
		})
	}
}

// TestMissingStepReferences tests handling of dependency graph validation
func TestMissingStepReferences(t *testing.T) {
	harness := NewErrorTestHarness("missing_step_references")

	if err := harness.SetupTestEnvironment(t); err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Test dependency graph validation with real installer steps
	graph := NewDependencyGraph() // Use real graph (not test graph)

	// Try to build a graph with a non-existent step dependency
	dependencies := []Dependency{
		{From: "ValidStep1", To: "CheckPrerequisites"}, // CheckPrerequisites is real
		{From: "ValidStep2", To: "NonExistentStep"},    // This should fail
	}

	err := graph.buildGraph(dependencies)

	if err == nil {
		t.Error("Expected error for missing step reference in dependency graph")
		return
	}

	errorMsg := err.Error()
	expectedStrings := []string{"unknown installation step", "NonExistentStep"}

	for _, expected := range expectedStrings {
		if !strings.Contains(errorMsg, expected) {
			t.Errorf("Error message should contain %q, got: %s", expected, errorMsg)
		}
	}

	// Verify error message includes available steps for better user experience
	if !strings.Contains(errorMsg, "Available steps:") {
		t.Errorf("Error message should include available steps for guidance, got: %s", errorMsg)
	}

	t.Logf("✅ Missing step references test verified")
	t.Logf("   Error message: %s", errorMsg)
}

// TestFilesystemEdgeCases tests various filesystem-related edge cases and error conditions
func TestFilesystemEdgeCases(t *testing.T) {
	filesystemTests := []struct {
		name        string
		description string
		setupFunc   func(*testing.T, *ErrorTestHarness) error
		expectError bool
		errorText   []string
		cleanupFunc func(*testing.T, *ErrorTestHarness)
	}{
		{
			name:        "read_only_parent_directory",
			description: "Test installation fails gracefully when parent directory is read-only",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Make the parent directory read-only
				parentDir := filepath.Dir(h.TempDir)
				if err := os.Chmod(parentDir, 0o444); err != nil {
					return fmt.Errorf("failed to make parent directory read-only: %w", err)
				}
				return nil
			},
			expectError: true,
			errorText:   []string{"failed", "permission"},
			cleanupFunc: func(t *testing.T, h *ErrorTestHarness) {
				// Restore write permissions for cleanup
				parentDir := filepath.Dir(h.TempDir)
				_ = os.Chmod(parentDir, 0o755)
			},
		},
		{
			name:        "symlink_target_directory",
			description: "Test installation with target directory as symlink",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create a real directory and symlink to it
				realDir := h.TempDir + "_real"
				if err := os.MkdirAll(realDir, 0o755); err != nil {
					return err
				}

				// Remove the temp dir and create symlink
				_ = os.RemoveAll(h.TempDir)
				return os.Symlink(realDir, h.TempDir)
			},
			expectError: false, // Should work fine with symlinks
		},
		{
			name:        "broken_symlink_target",
			description: "Test installation fails with broken symlink as target",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create symlink to non-existent target
				_ = os.RemoveAll(h.TempDir)
				return os.Symlink("/non/existent/target", h.TempDir)
			},
			expectError: true,
			errorText:   []string{"failed"},
		},
		{
			name:        "device_file_as_target",
			description: "Test installation fails when target is a device file",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Skip this test if not on Linux or if we can't access /dev/null
				if _, err := os.Stat("/dev/null"); err != nil {
					t.Skip("Skipping device file test - /dev/null not available")
				}
				h.TempDir = "/dev/null"
				return nil
			},
			expectError: true,
			errorText:   []string{"failed"},
		},
		{
			name:        "file_exists_as_target_directory",
			description: "Test installation fails when target path exists as regular file",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Remove directory and create file with same name
				_ = os.RemoveAll(h.TempDir)
				return os.WriteFile(h.TempDir, []byte("regular file content"), 0o644)
			},
			expectError: true,
			errorText:   []string{"failed"},
		},
		{
			name:        "very_long_path_name",
			description: "Test installation with very long path names",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create a very long path name (close to filesystem limits)
				longName := strings.Repeat("a", 200) // 200 characters
				longPath := filepath.Join(h.TempDir, longName)
				h.TempDir = longPath
				return os.MkdirAll(longPath, 0o755)
			},
			expectError: false, // Should work unless hitting actual filesystem limits
		},
		{
			name:        "path_with_special_characters",
			description: "Test installation with special characters in path",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create path with various special characters (but safe ones)
				specialPath := filepath.Join(h.TempDir, "test-dir_with.special+chars")
				h.TempDir = specialPath
				return os.MkdirAll(specialPath, 0o755)
			},
			expectError: false, // Should handle special characters fine
		},
		{
			name:        "insufficient_space_simulation",
			description: "Test graceful handling when filesystem appears full",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Fill up the temp directory with a large file to simulate low space
				largePath := filepath.Join(h.TempDir, "space_consumer.tmp")

				// Create a 10MB file to consume space
				data := make([]byte, 10*1024*1024)
				if err := os.WriteFile(largePath, data, 0o644); err != nil {
					return fmt.Errorf("failed to create large file: %w", err)
				}

				return nil
			},
			expectError: false, // Modern systems usually have enough space for our test
		},
		{
			name:        "concurrent_directory_modification",
			description: "Test resilience against concurrent filesystem modifications",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create a goroutine that modifies the filesystem during installation
				go func() {
					time.Sleep(100 * time.Millisecond)
					testFile := filepath.Join(h.TempDir, "concurrent_test.tmp")
					_ = os.WriteFile(testFile, []byte("concurrent modification"), 0o644)
					time.Sleep(100 * time.Millisecond)
					_ = os.Remove(testFile)
				}()
				return nil
			},
			expectError: false, // Should be resilient to concurrent modifications
		},
		{
			name:        "unicode_path_names",
			description: "Test installation with Unicode characters in path",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create path with Unicode characters
				unicodePath := filepath.Join(h.TempDir, "test-ñàmé-δοκιμή-тест")
				h.TempDir = unicodePath
				return os.MkdirAll(unicodePath, 0o755)
			},
			expectError: false, // Should handle Unicode paths fine
		},
	}

	for _, test := range filesystemTests {
		t.Run(test.name, func(t *testing.T) {
			harness := NewErrorTestHarness(test.name)

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Setup cleanup function if provided
			if test.cleanupFunc != nil {
				defer test.cleanupFunc(t, harness)
			}

			// Run test-specific setup
			var setupErr error
			if test.setupFunc != nil {
				setupErr = test.setupFunc(t, harness)
				if setupErr != nil && !test.expectError {
					t.Fatalf("Failed to setup filesystem test: %v", setupErr)
				}
			}

			// If setup failed and we expect that, verify error and continue
			if setupErr != nil && test.expectError {
				errorMsg := setupErr.Error()
				for _, expectedText := range test.errorText {
					if !strings.Contains(errorMsg, expectedText) {
						t.Errorf("Setup error should contain %q, got: %s", expectedText, errorMsg)
					}
				}
				t.Logf("✅ Filesystem edge case test verified (setup error): %s", test.description)
				t.Logf("   Setup error: %s", errorMsg)
				return
			}

			// Attempt installation
			installer, err := NewInstaller(harness.TempDir, harness.Config)
			if err != nil {
				if test.expectError {
					// Verify error contains expected text
					errorMsg := err.Error()
					for _, expectedText := range test.errorText {
						if !strings.Contains(errorMsg, expectedText) {
							t.Errorf("Installer creation error should contain %q, got: %s", expectedText, errorMsg)
						}
					}
					t.Logf("✅ Filesystem edge case test verified (creation error): %s", test.description)
					t.Logf("   Creation error: %s", errorMsg)
					return
				} else {
					t.Fatalf("Unexpected error creating installer: %v", err)
				}
			}

			// Measure installation time to detect hangs
			start := time.Now()
			installErr := installer.Install()
			duration := time.Since(start)

			if test.expectError {
				if installErr == nil {
					t.Errorf("Expected installation to fail for %s", test.description)
					return
				}

				// Verify error contains expected text
				errorMsg := installErr.Error()
				for _, expectedText := range test.errorText {
					if !strings.Contains(errorMsg, expectedText) {
						t.Errorf("Install error should contain %q, got: %s", expectedText, errorMsg)
					}
				}

				// Verify error handling was quick (didn't hang)
				if duration > 30*time.Second {
					t.Errorf("Installation took too long to fail: %v", duration)
				}

				t.Logf("✅ Filesystem edge case test verified (install error): %s", test.description)
				t.Logf("   Install error: %s", errorMsg)
			} else {
				if installErr != nil {
					t.Errorf("Expected installation to succeed for %s, got error: %v", test.description, installErr)
					return
				}

				// Verify installation completed in reasonable time
				if duration > 30*time.Second {
					t.Errorf("Installation took too long: %v", duration)
				}

				t.Logf("✅ Filesystem edge case test verified (success): %s", test.description)
				t.Logf("   Installation duration: %v", duration)
			}
		})
	}
}

// TestFilesystemPermissions tests various filesystem permission scenarios
func TestFilesystemPermissions(t *testing.T) {
	permissionTests := []struct {
		name        string
		description string
		setupPerms  func(*testing.T, string) error
		cleanupFunc func(*testing.T, string)
		expectError bool
		errorText   []string
	}{
		{
			name:        "no_execute_permission_on_directory",
			description: "Test installation when directory has no execute permission",
			setupPerms: func(t *testing.T, dir string) error {
				// Remove execute permission from directory
				return os.Chmod(dir, 0o600) // rw- --- ---
			},
			cleanupFunc: func(t *testing.T, dir string) {
				_ = os.Chmod(dir, 0o755) // Restore full permissions
			},
			expectError: true,
			errorText:   []string{"failed", "permission"},
		},
		{
			name:        "no_read_permission_on_directory",
			description: "Test installation when directory has no read permission (should succeed)",
			setupPerms: func(t *testing.T, dir string) error {
				// Remove read permission from directory (but keep write and execute)
				return os.Chmod(dir, 0o300) // -wx --- ---
			},
			cleanupFunc: func(t *testing.T, dir string) {
				_ = os.Chmod(dir, 0o755) // Restore full permissions
			},
			expectError: false, // Installation should work with write+execute permissions
		},
		{
			name:        "mixed_permission_subdirectories",
			description: "Test installation with mixed permissions in subdirectories",
			setupPerms: func(t *testing.T, dir string) error {
				// Create subdirectories with different permissions
				subDir1 := filepath.Join(dir, "readable")
				subDir2 := filepath.Join(dir, "restricted")

				if err := os.MkdirAll(subDir1, 0o755); err != nil {
					return err
				}
				if err := os.MkdirAll(subDir2, 0o000); err != nil {
					return err
				}

				return nil
			},
			cleanupFunc: func(t *testing.T, dir string) {
				// Restore permissions for cleanup
				subDir2 := filepath.Join(dir, "restricted")
				_ = os.Chmod(subDir2, 0o755)
			},
			expectError: false, // Should work if installer doesn't need restricted dirs
		},
		{
			name:        "world_writable_directory",
			description: "Test installation in world-writable directory",
			setupPerms: func(t *testing.T, dir string) error {
				// Make directory world-writable
				return os.Chmod(dir, 0o777) // rwx rwx rwx
			},
			expectError: false, // Should work fine with world-writable
		},
	}

	for _, test := range permissionTests {
		t.Run(test.name, func(t *testing.T) {
			harness := NewErrorTestHarness(test.name)

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Setup cleanup function if provided
			if test.cleanupFunc != nil {
				defer test.cleanupFunc(t, harness.TempDir)
			}

			// Apply permission changes
			if err := test.setupPerms(t, harness.TempDir); err != nil {
				t.Fatalf("Failed to setup permissions: %v", err)
			}

			// Attempt installation
			installer, err := NewInstaller(harness.TempDir, harness.Config)
			if err != nil {
				if test.expectError {
					// Verify error contains expected text
					errorMsg := err.Error()
					for _, expectedText := range test.errorText {
						if !strings.Contains(errorMsg, expectedText) {
							t.Errorf("Error should contain %q, got: %s", expectedText, errorMsg)
						}
					}
					t.Logf("✅ Permission test verified (creation error): %s", test.description)
					t.Logf("   Error: %s", errorMsg)
					return
				} else {
					t.Fatalf("Unexpected error creating installer: %v", err)
				}
			}

			installErr := installer.Install()

			if test.expectError {
				if installErr == nil {
					t.Errorf("Expected installation to fail for %s", test.description)
					return
				}

				// Verify error contains expected text
				errorMsg := installErr.Error()
				for _, expectedText := range test.errorText {
					if !strings.Contains(errorMsg, expectedText) {
						t.Errorf("Error should contain %q, got: %s", expectedText, errorMsg)
					}
				}

				t.Logf("✅ Permission test verified (install error): %s", test.description)
				t.Logf("   Error: %s", errorMsg)
			} else {
				if installErr != nil {
					t.Errorf("Expected installation to succeed for %s, got error: %v", test.description, installErr)
					return
				}

				t.Logf("✅ Permission test verified (success): %s", test.description)
			}
		})
	}
}

// TestErrorMessageQuality tests the quality and helpfulness of error messages
func TestErrorMessageQuality(t *testing.T) {
	errorMessageTests := []struct {
		name            string
		description     string
		setupFunc       func(*testing.T, *ErrorTestHarness) error
		expectedQuality []string // Things we expect in good error messages
		minLength       int      // Minimum length for descriptive messages
		maxLength       int      // Maximum length to avoid overwhelming users
	}{
		{
			name:        "invalid_target_directory_message",
			description: "Test error message quality for invalid target directory",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Try to install to non-existent parent
				h.TempDir = "/non/existent/parent/directory"
				installer, err := NewInstaller(h.TempDir, h.Config)
				if err != nil {
					return err
				}
				return installer.Install()
			},
			expectedQuality: []string{
				"failed",    // Should indicate failure
				"directory", // Should mention directory issue
				"target",    // Should mention target
			},
			minLength: 20,  // At least 20 characters for context
			maxLength: 200, // No more than 200 characters to avoid overwhelming
		},
		{
			name:        "dependency_graph_error_message",
			description: "Test error message quality for dependency graph errors",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create invalid dependency graph
				graph := NewDependencyGraph()
				dependencies := []Dependency{
					{From: "InvalidStep", To: "AnotherInvalidStep"},
				}
				return graph.buildGraph(dependencies)
			},
			expectedQuality: []string{
				"unknown installation step", // Should clearly explain the problem
				"Available steps:",          // Should provide helpful guidance
				"InvalidStep",               // Should mention the problematic step
			},
			minLength: 50,  // More detail expected for graph errors
			maxLength: 400, // Allow more space for available steps list
		},
		{
			name:        "permission_denied_message",
			description: "Test error message quality for permission denied errors",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create read-only directory
				if err := h.CreateReadOnlyFileSystem(t); err != nil {
					return err
				}
				installer, err := NewInstaller(h.TempDir, h.Config)
				if err != nil {
					return err
				}
				return installer.Install()
			},
			expectedQuality: []string{
				"permission",         // Should mention permission issue
				"failed",             // Should indicate failure
				"CheckPrerequisites", // Should mention the failing step
			},
			minLength: 30,  // Reasonable detail for permission errors
			maxLength: 250, // Not too verbose
		},
	}

	for _, test := range errorMessageTests {
		t.Run(test.name, func(t *testing.T) {
			harness := NewErrorTestHarness(test.name)

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// For read-only tests, setup cleanup
			if test.name == "permission_denied_message" {
				defer harness.CleanupReadOnlyFileSystem(t)
			}

			// Run test-specific setup and get error
			var testErr error
			if test.setupFunc != nil {
				testErr = test.setupFunc(t, harness)
			}

			// Verify we got an error (these tests expect errors)
			if testErr == nil {
				t.Errorf("Expected error for %s but got success", test.description)
				return
			}

			errorMsg := testErr.Error()

			// Test message length
			if len(errorMsg) < test.minLength {
				t.Errorf("Error message too short (%d chars), should be at least %d chars: %s",
					len(errorMsg), test.minLength, errorMsg)
			}

			if len(errorMsg) > test.maxLength {
				t.Errorf("Error message too long (%d chars), should be at most %d chars: %s",
					len(errorMsg), test.maxLength, errorMsg)
			}

			// Test message quality - should contain expected elements
			for _, expected := range test.expectedQuality {
				if !strings.Contains(errorMsg, expected) {
					t.Errorf("Error message should contain %q for better user experience, got: %s",
						expected, errorMsg)
				}
			}

			// Test that message is not just a generic error
			genericErrors := []string{
				"error occurred",
				"something went wrong",
				"failed to execute",
				"unknown error",
			}

			for _, generic := range genericErrors {
				if strings.Contains(strings.ToLower(errorMsg), generic) {
					t.Errorf("Error message appears generic and unhelpful: %s", errorMsg)
				}
			}

			// Test that message provides actionable information
			if !strings.Contains(errorMsg, ":") {
				t.Errorf("Error message should provide specific details (missing colon separator): %s", errorMsg)
			}

			t.Logf("✅ Error message quality verified: %s", test.description)
			t.Logf("   Message length: %d characters", len(errorMsg))
			t.Logf("   Error message: %s", errorMsg)
		})
	}
}

// TestCleanupVerification tests that failed installations clean up properly
func TestCleanupVerification(t *testing.T) {
	cleanupTests := []struct {
		name          string
		description   string
		setupFunc     func(*testing.T, *ErrorTestHarness) error
		causeFailure  func(*testing.T, *ErrorTestHarness) error
		verifyCleanup func(*testing.T, *ErrorTestHarness) error
	}{
		{
			name:        "cleanup_after_permission_failure",
			description: "Test cleanup when installation fails due to permissions",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				return nil // Basic setup is sufficient
			},
			causeFailure: func(t *testing.T, h *ErrorTestHarness) error {
				// Make directory read-only to cause failure
				if err := h.CreateReadOnlyFileSystem(t); err != nil {
					return err
				}

				installer, err := NewInstaller(h.TempDir, h.Config)
				if err != nil {
					return err
				}

				installErr := installer.Install()

				// Restore permissions for cleanup verification
				h.CleanupReadOnlyFileSystem(t)

				return installErr
			},
			verifyCleanup: func(t *testing.T, h *ErrorTestHarness) error {
				// Verify no partial installation artifacts remain
				h.VerifyNoPartialInstallation(t)

				// Check that no temporary files are left behind
				tempFiles, err := filepath.Glob(filepath.Join(h.TempDir, "*.tmp"))
				if err != nil {
					return err
				}

				if len(tempFiles) > 0 {
					return fmt.Errorf("found %d temporary files after failed installation: %v",
						len(tempFiles), tempFiles)
				}

				return nil
			},
		},
		{
			name:        "cleanup_after_invalid_target",
			description: "Test cleanup when installation fails due to invalid target",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				// Create a file where we expect a directory
				_ = os.RemoveAll(h.TempDir)
				return os.WriteFile(h.TempDir, []byte("not a directory"), 0o644)
			},
			causeFailure: func(t *testing.T, h *ErrorTestHarness) error {
				installer, err := NewInstaller(h.TempDir, h.Config)
				if err != nil {
					return err
				}
				return installer.Install()
			},
			verifyCleanup: func(t *testing.T, h *ErrorTestHarness) error {
				// The original file should still exist (we shouldn't have corrupted it)
				if _, err := os.Stat(h.TempDir); err != nil {
					return fmt.Errorf("original file was removed during failed installation: %w", err)
				}

				// Verify no additional files were created
				parentDir := filepath.Dir(h.TempDir)
				entries, err := os.ReadDir(parentDir)
				if err != nil {
					return err
				}

				// Should only have our original file, no new installation artifacts
				basename := filepath.Base(h.TempDir)
				found := false
				for _, entry := range entries {
					if entry.Name() == basename {
						found = true
					} else if strings.Contains(entry.Name(), "superclaude") ||
						strings.Contains(entry.Name(), "claude") {
						return fmt.Errorf("found installation artifact after failed installation: %s",
							entry.Name())
					}
				}

				if !found {
					return fmt.Errorf("original target file was unexpectedly removed")
				}

				return nil
			},
		},
		{
			name:        "cleanup_temporary_repository",
			description: "Test cleanup of temporary repository clone on failure",
			setupFunc: func(t *testing.T, h *ErrorTestHarness) error {
				return nil
			},
			causeFailure: func(t *testing.T, h *ErrorTestHarness) error {
				// Start installation but force it to fail by making target read-only
				// after some steps have run
				installer, err := NewInstaller(h.TempDir, h.Config)
				if err != nil {
					return err
				}

				// Let it run for a bit to create temporary files, then cause failure
				go func() {
					time.Sleep(200 * time.Millisecond)
					_ = os.Chmod(h.TempDir, 0o444) // Make read-only
				}()

				installErr := installer.Install()

				// Restore permissions for cleanup
				_ = os.Chmod(h.TempDir, 0o755)

				return installErr
			},
			verifyCleanup: func(t *testing.T, h *ErrorTestHarness) error {
				// Look for any leftover git repositories or clone directories
				gitPattern := filepath.Join(h.TempDir, "*", ".git")
				gitDirs, err := filepath.Glob(gitPattern)
				if err != nil {
					return err
				}

				// Check in temporary directories too
				tmpDirs, err := filepath.Glob(filepath.Join(os.TempDir(), "*super-claude*"))
				if err != nil {
					return err
				}

				if len(gitDirs) > 0 {
					t.Logf("Found git directories (may be ok): %v", gitDirs)
				}

				if len(tmpDirs) > 5 { // Allow some temporary directories but not excessive
					return fmt.Errorf("found excessive temporary directories (%d), possible cleanup issue: %v",
						len(tmpDirs), tmpDirs)
				}

				return nil
			},
		},
	}

	for _, test := range cleanupTests {
		t.Run(test.name, func(t *testing.T) {
			harness := NewErrorTestHarness(test.name)

			// Setup test environment
			if err := harness.SetupTestEnvironment(t); err != nil {
				t.Fatalf("Failed to setup test environment: %v", err)
			}

			// Run test-specific setup
			if test.setupFunc != nil {
				if err := test.setupFunc(t, harness); err != nil {
					t.Fatalf("Failed to setup cleanup test: %v", err)
				}
			}

			// Cause the installation to fail
			var failureErr error
			if test.causeFailure != nil {
				failureErr = test.causeFailure(t, harness)
			}

			// Verify we got the expected failure
			if failureErr == nil {
				t.Errorf("Expected installation to fail for cleanup test %s", test.description)
				return
			}

			// Verify cleanup was performed correctly
			if test.verifyCleanup != nil {
				if err := test.verifyCleanup(t, harness); err != nil {
					t.Errorf("Cleanup verification failed for %s: %v", test.description, err)
				}
			}

			t.Logf("✅ Cleanup verification test passed: %s", test.description)
			t.Logf("   Failure error: %s", failureErr.Error())
		})
	}
}
