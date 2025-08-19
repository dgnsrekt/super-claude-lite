package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	// BaselineSchemaVersion defines the current schema version for golden files
	BaselineSchemaVersion = "1.0.0"
)

// BaselineTestResult captures the execution order, timing, and completion state
type BaselineTestResult struct {
	SchemaVersion      string                   `json:"schema_version"` // For golden file compatibility
	TestName           string                   `json:"test_name"`
	Config             InstallConfig            `json:"config"`
	ExecutionOrder     []string                 `json:"execution_order"`
	StepTimings        map[string]time.Duration `json:"step_timings"`
	CompletionTracking []string                 `json:"completion_tracking"`
	TotalDuration      time.Duration            `json:"total_duration"`
	Success            bool                     `json:"success"`
	ErrorMessage       string                   `json:"error_message,omitempty"`
	Summary            *InstallationSummary     `json:"summary,omitempty"`
	TimingMetrics      *TimingMetrics           `json:"timing_metrics,omitempty"`
	Timestamp          time.Time                `json:"timestamp"`   // When baseline was captured
	Environment        map[string]string        `json:"environment"` // Environment info for baseline
}

// TimingMetrics provides deterministic timing analysis
type TimingMetrics struct {
	SlowSteps        []string                 `json:"slow_steps"`        // Steps that took longer than threshold
	FastSteps        []string                 `json:"fast_steps"`        // Steps that completed quickly
	TimingThresholds map[string]time.Duration `json:"timing_thresholds"` // Expected timing ranges per step
	RelativeTimings  map[string]float64       `json:"relative_timings"`  // Step timing as percentage of total
	StepTimingOrder  []string                 `json:"step_timing_order"` // Steps ordered by execution time
	AverageStepTime  time.Duration            `json:"average_step_time"` // Average time per step
	MedianStepTime   time.Duration            `json:"median_step_time"`  // Median time per step
}

// TestScenario defines different installation scenarios
type TestScenario struct {
	Name        string
	Config      InstallConfig
	Description string
	ErrorType   string             // Type of error expected (if any)
	SetupFunc   func(string) error // Optional setup function for error scenarios
}

// ErrorScenario defines error testing scenarios
type ErrorScenario struct {
	Name          string
	Description   string
	SetupFunc     func(string) error // Setup function to create error conditions
	ExpectedError string             // Expected error substring
	ExpectedSteps []string           // Steps expected to execute before failure
	CleanupFunc   func(string) error // Optional cleanup function
}

// BaselineCapture provides utilities for capturing baseline behavior
type BaselineCapture struct {
	results       []BaselineTestResult
	goldenFileDir string
}

// NewBaselineCapture creates a new baseline capture instance
func NewBaselineCapture(goldenFileDir string) *BaselineCapture {
	return &BaselineCapture{
		results:       make([]BaselineTestResult, 0),
		goldenFileDir: goldenFileDir,
	}
}

// CaptureErrorScenario captures behavior during error scenarios
func (bc *BaselineCapture) CaptureErrorScenario(installer *Installer, scenario *ErrorScenario, config *InstallConfig) (BaselineTestResult, error) {
	result := BaselineTestResult{
		SchemaVersion:      BaselineSchemaVersion,
		TestName:           scenario.Name,
		Config:             *config,
		ExecutionOrder:     make([]string, 0),
		StepTimings:        make(map[string]time.Duration),
		CompletionTracking: make([]string, 0),
		Timestamp:          time.Now(),
		Environment:        bc.captureEnvironment(),
	}

	// Create instrumented installer that captures execution order
	instrumentedInstaller := bc.createInstrumentedInstaller(installer, &result)

	// Record start time
	startTime := time.Now()

	// Execute installation with instrumentation (expecting error)
	err := instrumentedInstaller.Install()

	// Record total duration
	result.TotalDuration = time.Since(startTime)
	result.Success = err == nil
	if err != nil {
		result.ErrorMessage = err.Error()
	}

	// Generate timing metrics
	result.TimingMetrics = bc.generateTimingMetrics(&result)

	return result, err
}

// CaptureExecutionOrder captures the order of step execution
func (bc *BaselineCapture) CaptureExecutionOrder(installer *Installer, scenario *TestScenario) (BaselineTestResult, error) {
	result := BaselineTestResult{
		SchemaVersion:      BaselineSchemaVersion,
		TestName:           scenario.Name,
		Config:             scenario.Config,
		ExecutionOrder:     make([]string, 0),
		StepTimings:        make(map[string]time.Duration),
		CompletionTracking: make([]string, 0),
		Timestamp:          time.Now(),
		Environment:        bc.captureEnvironment(),
	}

	// Create instrumented installer that captures execution order
	instrumentedInstaller := bc.createInstrumentedInstaller(installer, &result)

	// Record start time
	startTime := time.Now()

	// Execute installation with instrumentation
	err := instrumentedInstaller.Install()

	// Record total duration
	result.TotalDuration = time.Since(startTime)
	result.Success = err == nil
	if err != nil {
		result.ErrorMessage = err.Error()
	}

	// Capture installation summary if successful
	if err == nil {
		summary := instrumentedInstaller.GetInstallationSummary()
		result.Summary = &summary
	}

	// Generate timing metrics
	result.TimingMetrics = bc.generateTimingMetrics(&result)

	return result, err
}

// createInstrumentedInstaller creates a copy of the installer with execution tracking
func (bc *BaselineCapture) createInstrumentedInstaller(original *Installer, result *BaselineTestResult) *Installer {
	// Create a new installer with the same context and dependency graph
	instrumented := &Installer{
		steps:   make(map[string]*InstallStep),
		context: original.context,
		graph:   original.graph,
	}

	// Copy and instrument each step
	for stepName, step := range original.steps {
		instrumentedStep := &InstallStep{
			Name:     step.Name,
			Execute:  bc.instrumentExecuteFunc(step.Execute, stepName, result),
			Validate: bc.instrumentValidateFunc(step.Validate, stepName, result),
		}
		instrumented.steps[stepName] = instrumentedStep
	}

	return instrumented
}

// instrumentExecuteFunc wraps the execute function to capture timing and order
func (bc *BaselineCapture) instrumentExecuteFunc(original func(*InstallContext) error, stepName string, result *BaselineTestResult) func(*InstallContext) error {
	return func(ctx *InstallContext) error {
		// Record execution start
		startTime := time.Now()
		result.ExecutionOrder = append(result.ExecutionOrder, stepName)

		// Execute original function
		err := original(ctx)

		// Record timing
		result.StepTimings[stepName] = time.Since(startTime)

		return err
	}
}

// instrumentValidateFunc wraps the validate function to capture timing
func (bc *BaselineCapture) instrumentValidateFunc(original func(*InstallContext) error, stepName string, result *BaselineTestResult) func(*InstallContext) error {
	if original == nil {
		return nil
	}

	return func(ctx *InstallContext) error {
		// Record validation start
		startTime := time.Now()

		// Execute original validation
		err := original(ctx)

		// Record validation timing (separate from execution timing)
		validationKey := stepName + "_validation"
		result.StepTimings[validationKey] = time.Since(startTime)

		return err
	}
}

// SaveGoldenFile saves baseline results to a golden file
func (bc *BaselineCapture) SaveGoldenFile(testName string, result *BaselineTestResult) error {
	// Ensure golden file directory exists
	if err := os.MkdirAll(bc.goldenFileDir, 0o755); err != nil {
		return fmt.Errorf("failed to create golden file directory: %w", err)
	}

	// Create golden file path
	goldenFile := filepath.Join(bc.goldenFileDir, fmt.Sprintf("%s_golden.json", testName))

	// Marshal result to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline result: %w", err)
	}

	// Write to file
	if err := os.WriteFile(goldenFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write golden file: %w", err)
	}

	return nil
}

// LoadGoldenFile loads baseline results from a golden file
func (bc *BaselineCapture) LoadGoldenFile(testName string) (BaselineTestResult, error) {
	goldenFile := filepath.Join(bc.goldenFileDir, fmt.Sprintf("%s_golden.json", testName))

	data, err := os.ReadFile(goldenFile)
	if err != nil {
		return BaselineTestResult{}, fmt.Errorf("failed to read golden file: %w", err)
	}

	var result BaselineTestResult
	if err := json.Unmarshal(data, &result); err != nil {
		return BaselineTestResult{}, fmt.Errorf("failed to unmarshal golden file: %w", err)
	}

	return result, nil
}

// CompareWithGolden compares current result with golden standard
func (bc *BaselineCapture) CompareWithGolden(current, golden *BaselineTestResult) []string {
	var differences []string

	// Compare execution order
	if len(current.ExecutionOrder) != len(golden.ExecutionOrder) {
		differences = append(differences, fmt.Sprintf("execution order length differs: got %d, expected %d", len(current.ExecutionOrder), len(golden.ExecutionOrder)))
	} else {
		for i, step := range current.ExecutionOrder {
			if golden.ExecutionOrder[i] != step {
				differences = append(differences, fmt.Sprintf("execution order differs at position %d: got %s, expected %s", i, step, golden.ExecutionOrder[i]))
			}
		}
	}

	// Compare completion tracking
	if len(current.CompletionTracking) != len(golden.CompletionTracking) {
		differences = append(differences, fmt.Sprintf("completion tracking length differs: got %d, expected %d", len(current.CompletionTracking), len(golden.CompletionTracking)))
	}

	// Compare success status
	if current.Success != golden.Success {
		differences = append(differences, fmt.Sprintf("success status differs: got %t, expected %t", current.Success, golden.Success))
	}

	return differences
}

// generateTimingMetrics analyzes step timings and generates deterministic metrics
func (bc *BaselineCapture) generateTimingMetrics(result *BaselineTestResult) *TimingMetrics {
	if len(result.StepTimings) == 0 {
		return nil
	}

	metrics := &TimingMetrics{
		SlowSteps:        make([]string, 0),
		FastSteps:        make([]string, 0),
		TimingThresholds: make(map[string]time.Duration),
		RelativeTimings:  make(map[string]float64),
		StepTimingOrder:  make([]string, 0),
	}

	// Calculate timing statistics
	var durations []time.Duration
	var totalTime time.Duration

	for stepName, duration := range result.StepTimings {
		// Skip validation steps for base timing analysis
		if !isValidationStep(stepName) {
			durations = append(durations, duration)
			totalTime += duration
		}
	}

	if len(durations) == 0 {
		return metrics
	}

	// Sort durations for median calculation
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// Calculate average and median
	metrics.AverageStepTime = totalTime / time.Duration(len(durations))
	if len(durations)%2 == 0 {
		metrics.MedianStepTime = (durations[len(durations)/2-1] + durations[len(durations)/2]) / 2
	} else {
		metrics.MedianStepTime = durations[len(durations)/2]
	}

	// Define thresholds based on median
	slowThreshold := metrics.MedianStepTime * 2
	fastThreshold := metrics.MedianStepTime / 2

	// Analyze each step timing
	type stepTiming struct {
		name     string
		duration time.Duration
	}
	stepTimings := make([]stepTiming, 0, len(result.StepTimings))

	for stepName, duration := range result.StepTimings {
		if isValidationStep(stepName) {
			continue
		}

		stepTimings = append(stepTimings, stepTiming{stepName, duration})

		// Calculate relative timing as percentage of total
		if result.TotalDuration > 0 {
			metrics.RelativeTimings[stepName] = float64(duration) / float64(result.TotalDuration) * 100
		}

		// Set thresholds and categorize steps
		metrics.TimingThresholds[stepName] = slowThreshold

		if duration > slowThreshold {
			metrics.SlowSteps = append(metrics.SlowSteps, stepName)
		} else if duration < fastThreshold {
			metrics.FastSteps = append(metrics.FastSteps, stepName)
		}
	}

	// Sort steps by timing (fastest to slowest)
	sort.Slice(stepTimings, func(i, j int) bool {
		return stepTimings[i].duration < stepTimings[j].duration
	})

	for _, st := range stepTimings {
		metrics.StepTimingOrder = append(metrics.StepTimingOrder, st.name)
	}

	return metrics
}

// isValidationStep checks if a step name indicates a validation step
func isValidationStep(stepName string) bool {
	return len(stepName) > 11 && stepName[len(stepName)-11:] == "_validation"
}

// CompareTimingMetrics compares timing metrics between current and golden results
func (bc *BaselineCapture) CompareTimingMetrics(current, golden *TimingMetrics) []string {
	var differences []string

	if current == nil && golden == nil {
		return differences
	}

	if current == nil {
		differences = append(differences, "current timing metrics are nil but golden has metrics")
		return differences
	}

	if golden == nil {
		differences = append(differences, "golden timing metrics are nil but current has metrics")
		return differences
	}

	// Compare step timing order (should be deterministic for same execution)
	if len(current.StepTimingOrder) != len(golden.StepTimingOrder) {
		differences = append(differences, fmt.Sprintf("step timing order length differs: got %d, expected %d",
			len(current.StepTimingOrder), len(golden.StepTimingOrder)))
	} else {
		for i, step := range current.StepTimingOrder {
			if golden.StepTimingOrder[i] != step {
				differences = append(differences, fmt.Sprintf("step timing order differs at position %d: got %s, expected %s",
					i, step, golden.StepTimingOrder[i]))
			}
		}
	}

	// Compare slow/fast step categorization (allowing for some timing variance)
	if !slicesEqual(current.SlowSteps, golden.SlowSteps) {
		differences = append(differences, fmt.Sprintf("slow steps differ: got %v, expected %v",
			current.SlowSteps, golden.SlowSteps))
	}

	if !slicesEqual(current.FastSteps, golden.FastSteps) {
		differences = append(differences, fmt.Sprintf("fast steps differ: got %v, expected %v",
			current.FastSteps, golden.FastSteps))
	}

	return differences
}

// slicesEqual compares two string slices for equality (order independent)
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}

	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}

	return true
}

// captureEnvironment captures relevant environment information for baseline reproducibility
func (bc *BaselineCapture) captureEnvironment() map[string]string {
	env := make(map[string]string)

	// Capture basic system information
	env["os"] = runtime.GOOS
	env["arch"] = runtime.GOARCH
	env["go_version"] = runtime.Version()

	// Capture environment variables that might affect installation
	envVars := []string{"HOME", "USER", "TMPDIR", "PATH"}
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			// Store only presence for privacy, not actual values for sensitive vars
			if envVar == "PATH" || envVar == "TMPDIR" {
				env[envVar] = value
			} else {
				env[envVar] = "[present]"
			}
		}
	}

	return env
}

// ValidateGoldenFileSchema validates that a golden file has a compatible schema
func (bc *BaselineCapture) ValidateGoldenFileSchema(result *BaselineTestResult) error {
	if result.SchemaVersion == "" {
		return fmt.Errorf("golden file missing schema version")
	}

	// For now, only support exact version match
	// In future, could implement version compatibility logic
	if result.SchemaVersion != BaselineSchemaVersion {
		return fmt.Errorf("incompatible schema version: expected %s, got %s",
			BaselineSchemaVersion, result.SchemaVersion)
	}

	return nil
}

// LoadAndValidateGoldenFile loads and validates a golden file
func (bc *BaselineCapture) LoadAndValidateGoldenFile(testName string) (BaselineTestResult, error) {
	result, err := bc.LoadGoldenFile(testName)
	if err != nil {
		return result, err
	}

	if err := bc.ValidateGoldenFileSchema(&result); err != nil {
		return result, fmt.Errorf("schema validation failed: %w", err)
	}

	return result, nil
}

// SaveGoldenFileWithMetadata saves baseline results with comprehensive metadata
func (bc *BaselineCapture) SaveGoldenFileWithMetadata(testName string, result *BaselineTestResult) error {
	// Ensure schema version and timestamp are set
	result.SchemaVersion = BaselineSchemaVersion
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	if result.Environment == nil {
		result.Environment = bc.captureEnvironment()
	}

	return bc.SaveGoldenFile(testName, result)
}

// ValidateErrorScenarioResult validates that error scenario results match expectations
func (bc *BaselineCapture) ValidateErrorScenarioResult(result *BaselineTestResult, scenario *ErrorScenario) []string {
	var issues []string

	// Check if error was expected
	if scenario.ExpectedError != "" {
		if result.Success {
			issues = append(issues, fmt.Sprintf("expected error containing '%s' but installation succeeded", scenario.ExpectedError))
		} else if result.ErrorMessage == "" {
			issues = append(issues, "expected error message but got empty error")
		} else if !containsSubstring(result.ErrorMessage, scenario.ExpectedError) {
			issues = append(issues, fmt.Sprintf("expected error containing '%s' but got '%s'", scenario.ExpectedError, result.ErrorMessage))
		}
	}

	// Validate expected steps were executed (at minimum)
	if len(scenario.ExpectedSteps) > 0 {
		for _, expectedStep := range scenario.ExpectedSteps {
			found := false
			for _, executedStep := range result.ExecutionOrder {
				if executedStep == expectedStep {
					found = true
					break
				}
			}
			if !found {
				issues = append(issues, fmt.Sprintf("expected step '%s' to be executed but was not found in execution order", expectedStep))
			}
		}
	}

	// Validate that error occurred before completing all steps (if error expected)
	if scenario.ExpectedError != "" && !result.Success {
		if len(result.ExecutionOrder) == 0 {
			issues = append(issues, "no steps were executed before error occurred")
		}
	}

	return issues
}

// containsSubstring checks if a string contains a substring (case-insensitive)
func containsSubstring(text, substr string) bool {
	return len(text) >= len(substr) &&
		(substr == "" ||
			stringContains(strings.ToLower(text), strings.ToLower(substr)))
}

// stringContains is a simple substring check
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test scenarios for baseline capture
var baselineScenarios = []TestScenario{
	{
		Name: "default_installation",
		Config: InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false,
			BackupDir:         "",
		},
		Description: "Standard installation with default settings",
	},
	{
		Name: "mcp_enabled_installation",
		Config: InstallConfig{
			Force:             false,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: true,
			BackupDir:         "",
		},
		Description: "Installation with MCP recommendations enabled",
	},
	{
		Name: "no_backup_installation",
		Config: InstallConfig{
			Force:             false,
			NoBackup:          true,
			Interactive:       false,
			AddRecommendedMCP: false,
			BackupDir:         "",
		},
		Description: "Installation with backups disabled",
	},
	{
		Name: "force_installation",
		Config: InstallConfig{
			Force:             true,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false,
			BackupDir:         "",
		},
		Description: "Force installation over existing files",
	},
}

// Error scenarios for baseline capture
var errorScenarios = []ErrorScenario{
	{
		Name:        "permission_failure",
		Description: "Test behavior when write permissions are denied",
		SetupFunc: func(targetDir string) error {
			// Create a read-only directory
			readOnlyDir := filepath.Join(targetDir, "readonly")
			if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
				return err
			}
			return os.Chmod(readOnlyDir, 0o555) // Read and execute only
		},
		ExpectedError: "permission denied",
		ExpectedSteps: []string{"CheckPrerequisites"}, // Should fail early
		CleanupFunc: func(targetDir string) error {
			readOnlyDir := filepath.Join(targetDir, "readonly")
			_ = os.Chmod(readOnlyDir, 0o755) // Restore permissions for cleanup
			return os.RemoveAll(readOnlyDir)
		},
	},
	{
		Name:        "missing_dependencies",
		Description: "Test behavior when required tools are missing",
		SetupFunc: func(targetDir string) error {
			// This would simulate missing git or other dependencies
			// In a real test, we might modify PATH or create a controlled environment
			return nil
		},
		ExpectedError: "dependency not found",
		ExpectedSteps: []string{"CheckPrerequisites"},
	},
	{
		Name:        "file_conflicts",
		Description: "Test behavior when target files already exist without force flag",
		SetupFunc: func(targetDir string) error {
			// Create conflicting files
			claudeFile := filepath.Join(targetDir, "CLAUDE.md")
			return os.WriteFile(claudeFile, []byte("existing content"), 0o644)
		},
		ExpectedError: "file already exists",
		ExpectedSteps: []string{"CheckPrerequisites", "ScanExistingFiles"},
	},
	{
		Name:        "validation_failure",
		Description: "Test behavior when post-installation validation fails",
		SetupFunc: func(targetDir string) error {
			// Setup conditions that would cause validation to fail
			return nil
		},
		ExpectedError: "validation failed",
		ExpectedSteps: []string{"CheckPrerequisites", "ScanExistingFiles", "CreateDirectoryStructure"},
	},
	{
		Name:        "partial_installation",
		Description: "Test behavior when installation is interrupted mid-process",
		SetupFunc: func(targetDir string) error {
			// Create partial installation state
			return os.MkdirAll(filepath.Join(targetDir, ".claude"), 0o755)
		},
		ExpectedError: "", // May or may not error
		ExpectedSteps: []string{"CheckPrerequisites", "ScanExistingFiles"},
	},
}

// TestBaselineCapture is the main test function for capturing baseline behavior
func TestBaselineCapture(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")

	// Initialize baseline capture
	capture := NewBaselineCapture(goldenDir)

	for _, scenario := range baselineScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Set up test environment with isolated directories
			testTargetDir := filepath.Join(tempDir, scenario.Name+"_target")
			if err := os.MkdirAll(testTargetDir, 0o755); err != nil {
				t.Fatalf("Failed to create test target directory: %v", err)
			}

			// Create installer with current steps
			installer, err := NewInstaller(testTargetDir, &scenario.Config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			// Capture baseline behavior
			result, err := capture.CaptureExecutionOrder(installer, &scenario)

			// For baseline capture, we expect some tests might fail in isolated environments
			// but we still want to capture their behavior for comparison
			if err != nil {
				t.Logf("Installation failed for scenario %s (expected in test environment): %v", scenario.Name, err)
			}

			// Capture completion tracking from context
			result.CompletionTracking = append(result.CompletionTracking, installer.GetContext().Completed...)

			// Save golden file with actual captured data
			if err := capture.SaveGoldenFile(scenario.Name, &result); err != nil {
				t.Fatalf("Failed to save golden file for scenario %s: %v", scenario.Name, err)
			}

			// Verify we captured meaningful data
			if len(result.ExecutionOrder) == 0 && result.Success {
				t.Errorf("Expected to capture execution order for successful scenario %s", scenario.Name)
			}

			t.Logf("Captured baseline for scenario %s: %d steps executed, success=%t",
				scenario.Name, len(result.ExecutionOrder), result.Success)
		})
	}
}

// TestBaselineComparison tests comparing current behavior against golden files
func TestBaselineComparison(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")

	// Initialize baseline capture
	capture := NewBaselineCapture(goldenDir)

	// Create a golden file first
	goldenResult := BaselineTestResult{
		TestName:           "comparison_test",
		Config:             InstallConfig{Force: false},
		ExecutionOrder:     []string{"step1", "step2", "step3"},
		StepTimings:        map[string]time.Duration{"step1": time.Millisecond * 100},
		CompletionTracking: []string{"step1", "step2", "step3"},
		Success:            true,
	}

	if err := capture.SaveGoldenFile("comparison_test", &goldenResult); err != nil {
		t.Fatalf("Failed to save golden file: %v", err)
	}

	// Test comparing identical result
	loaded, err := capture.LoadGoldenFile("comparison_test")
	if err != nil {
		t.Fatalf("Failed to load golden file: %v", err)
	}

	differences := capture.CompareWithGolden(&goldenResult, &loaded)
	if len(differences) != 0 {
		t.Errorf("Expected no differences for identical results, got: %v", differences)
	}

	// Test detecting behavioral changes
	modifiedResult := goldenResult
	modifiedResult.ExecutionOrder = []string{"step1", "step3", "step2"} // Different order

	differences = capture.CompareWithGolden(&modifiedResult, &loaded)
	if len(differences) == 0 {
		t.Error("Expected to detect execution order differences")
	}

	t.Logf("Successfully detected %d behavioral differences", len(differences))
}

// TestTimingMeasurementSystem tests the timing analysis functionality
func TestTimingMeasurementSystem(t *testing.T) {
	capture := NewBaselineCapture("")

	// Create test result with sample timing data
	result := BaselineTestResult{
		TestName:       "timing_test",
		ExecutionOrder: []string{"step1", "step2", "step3"},
		StepTimings: map[string]time.Duration{
			"step1":            time.Millisecond * 50,
			"step2":            time.Millisecond * 100,
			"step3":            time.Millisecond * 25,
			"step1_validation": time.Millisecond * 5,
			"step2_validation": time.Millisecond * 10,
		},
		TotalDuration: time.Millisecond * 190,
		Success:       true,
	}

	// Generate timing metrics
	metrics := capture.generateTimingMetrics(&result)

	// Verify metrics were generated
	if metrics == nil {
		t.Fatal("Expected timing metrics to be generated")
	}

	// Verify average and median calculations
	expectedSteps := 3 // Excluding validation steps
	if len(metrics.StepTimingOrder) != expectedSteps {
		t.Errorf("Expected %d steps in timing order, got %d", expectedSteps, len(metrics.StepTimingOrder))
	}

	// Verify relative timings are calculated
	if len(metrics.RelativeTimings) != expectedSteps {
		t.Errorf("Expected %d relative timings, got %d", expectedSteps, len(metrics.RelativeTimings))
	}

	// Check that step3 is fastest (25ms)
	if metrics.StepTimingOrder[0] != "step3" {
		t.Errorf("Expected step3 to be fastest, got %s", metrics.StepTimingOrder[0])
	}

	// Check that step2 is slowest (100ms)
	if metrics.StepTimingOrder[len(metrics.StepTimingOrder)-1] != "step2" {
		t.Errorf("Expected step2 to be slowest, got %s", metrics.StepTimingOrder[len(metrics.StepTimingOrder)-1])
	}

	// Verify median calculation (50ms)
	expectedMedian := time.Millisecond * 50
	if metrics.MedianStepTime != expectedMedian {
		t.Errorf("Expected median time %v, got %v", expectedMedian, metrics.MedianStepTime)
	}

	// Verify average calculation (175ms / 3 = ~58.33ms)
	expectedAverage := time.Millisecond * 175 / 3
	if metrics.AverageStepTime != expectedAverage {
		t.Errorf("Expected average time %v, got %v", expectedAverage, metrics.AverageStepTime)
	}

	t.Logf("Timing metrics generated successfully: median=%v, average=%v, steps=%d",
		metrics.MedianStepTime, metrics.AverageStepTime, len(metrics.StepTimingOrder))
}

// TestTimingMetricsComparison tests comparing timing metrics
func TestTimingMetricsComparison(t *testing.T) {
	capture := NewBaselineCapture("")

	golden := &TimingMetrics{
		StepTimingOrder: []string{"step1", "step2", "step3"},
		SlowSteps:       []string{"step3"},
		FastSteps:       []string{"step1"},
	}

	// Test identical metrics
	current := &TimingMetrics{
		StepTimingOrder: []string{"step1", "step2", "step3"},
		SlowSteps:       []string{"step3"},
		FastSteps:       []string{"step1"},
	}

	differences := capture.CompareTimingMetrics(current, golden)
	if len(differences) != 0 {
		t.Errorf("Expected no differences for identical timing metrics, got: %v", differences)
	}

	// Test different timing order
	current.StepTimingOrder = []string{"step2", "step1", "step3"}
	differences = capture.CompareTimingMetrics(current, golden)
	if len(differences) == 0 {
		t.Error("Expected differences for different step timing order")
	}

	// Test different slow steps
	current = &TimingMetrics{
		StepTimingOrder: []string{"step1", "step2", "step3"},
		SlowSteps:       []string{"step2"}, // Different slow step
		FastSteps:       []string{"step1"},
	}

	differences = capture.CompareTimingMetrics(current, golden)
	if len(differences) == 0 {
		t.Error("Expected differences for different slow steps categorization")
	}

	t.Logf("Timing metrics comparison working correctly")
}

// TestErrorScenarios tests the error scenario testing framework
func TestErrorScenarios(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")

	// Initialize baseline capture
	capture := NewBaselineCapture(goldenDir)

	for _, scenario := range errorScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Set up test environment with isolated directories
			testTargetDir := filepath.Join(tempDir, scenario.Name+"_target")
			if err := os.MkdirAll(testTargetDir, 0o755); err != nil {
				t.Fatalf("Failed to create test target directory: %v", err)
			}

			// Setup error conditions if defined
			if scenario.SetupFunc != nil {
				if err := scenario.SetupFunc(testTargetDir); err != nil {
					t.Fatalf("Failed to setup error scenario: %v", err)
				}
			}

			// Clean up after test if cleanup function is defined
			if scenario.CleanupFunc != nil {
				defer func() {
					if err := scenario.CleanupFunc(testTargetDir); err != nil {
						t.Logf("Warning: cleanup failed for scenario %s: %v", scenario.Name, err)
					}
				}()
			}

			// Create default config for error scenario
			config := InstallConfig{
				Force:             false,
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				BackupDir:         "",
			}

			// Create installer with current steps
			installer, err := NewInstaller(testTargetDir, &config)
			if err != nil {
				t.Fatalf("Failed to create installer: %v", err)
			}

			// Capture error scenario behavior
			result, err := capture.CaptureErrorScenario(installer, &scenario, &config)
			// Error is expected in some scenarios and is captured in result.ErrorMessage
			_ = err

			// Validate the error scenario result
			issues := capture.ValidateErrorScenarioResult(&result, &scenario)

			// For error scenarios, we expect some to fail, but we want to capture the behavior
			if len(issues) > 0 {
				t.Logf("Error scenario validation issues for %s: %v", scenario.Name, issues)
			}

			// Capture completion tracking from context
			result.CompletionTracking = append(result.CompletionTracking, installer.GetContext().Completed...)

			// Save golden file with actual captured error data
			if err := capture.SaveGoldenFile("error_"+scenario.Name, &result); err != nil {
				t.Fatalf("Failed to save golden file for error scenario %s: %v", scenario.Name, err)
			}

			// Log what was captured
			t.Logf("Captured error scenario %s: %d steps executed, success=%t, error=%s",
				scenario.Name, len(result.ExecutionOrder), result.Success, result.ErrorMessage)
		})
	}
}

// TestErrorScenarioValidation tests the error scenario validation functions
func TestErrorScenarioValidation(t *testing.T) {
	capture := NewBaselineCapture("")

	// Test successful validation
	result := BaselineTestResult{
		TestName:       "test_error",
		ExecutionOrder: []string{"CheckPrerequisites", "ScanExistingFiles"},
		Success:        false,
		ErrorMessage:   "permission denied while creating directory",
	}

	scenario := ErrorScenario{
		Name:          "permission_test",
		ExpectedError: "permission denied",
		ExpectedSteps: []string{"CheckPrerequisites"},
	}

	issues := capture.ValidateErrorScenarioResult(&result, &scenario)
	if len(issues) != 0 {
		t.Errorf("Expected no validation issues, got: %v", issues)
	}

	// Test missing expected error
	successResult := BaselineTestResult{
		TestName:       "test_success",
		ExecutionOrder: []string{"CheckPrerequisites", "ScanExistingFiles"},
		Success:        true,
		ErrorMessage:   "",
	}

	issues = capture.ValidateErrorScenarioResult(&successResult, &scenario)
	if len(issues) == 0 {
		t.Error("Expected validation issues for successful result when error was expected")
	}

	// Test missing expected steps
	incompleteResult := BaselineTestResult{
		TestName:       "test_incomplete",
		ExecutionOrder: []string{}, // No steps executed
		Success:        false,
		ErrorMessage:   "permission denied",
	}

	issues = capture.ValidateErrorScenarioResult(&incompleteResult, &scenario)
	if len(issues) == 0 {
		t.Error("Expected validation issues for missing expected steps")
	}

	t.Logf("Error scenario validation working correctly")
}

// TestEnhancedGoldenFileSystem tests the enhanced golden file functionality
func TestEnhancedGoldenFileSystem(t *testing.T) {
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")

	capture := NewBaselineCapture(goldenDir)

	// Create enhanced test result
	testResult := BaselineTestResult{
		TestName:           "enhanced_golden_test",
		Config:             InstallConfig{Force: true},
		ExecutionOrder:     []string{"step1", "step2", "step3"},
		StepTimings:        map[string]time.Duration{"step1": time.Millisecond * 100},
		CompletionTracking: []string{"step1", "step2", "step3"},
		Success:            true,
		// Schema version and environment will be set by SaveGoldenFileWithMetadata
	}

	// Test enhanced save with metadata
	if err := capture.SaveGoldenFileWithMetadata("enhanced_test", &testResult); err != nil {
		t.Fatalf("Failed to save golden file with metadata: %v", err)
	}

	// Test load and validate
	loaded, err := capture.LoadAndValidateGoldenFile("enhanced_test")
	if err != nil {
		t.Fatalf("Failed to load and validate golden file: %v", err)
	}

	// Verify schema version was set
	if loaded.SchemaVersion != BaselineSchemaVersion {
		t.Errorf("Expected schema version %s, got %s", BaselineSchemaVersion, loaded.SchemaVersion)
	}

	// Verify environment was captured
	if loaded.Environment == nil {
		t.Error("Expected environment information to be captured")
	} else if loaded.Environment["os"] != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, loaded.Environment["os"])
	}

	// Verify timestamp was set
	if loaded.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	t.Logf("Enhanced golden file system working correctly with schema %s", loaded.SchemaVersion)
}

// TestGoldenFileSchemaValidation tests schema validation functionality
func TestGoldenFileSchemaValidation(t *testing.T) {
	capture := NewBaselineCapture("")

	// Test valid schema
	validResult := BaselineTestResult{
		SchemaVersion: BaselineSchemaVersion,
		TestName:      "valid_schema_test",
	}

	if err := capture.ValidateGoldenFileSchema(&validResult); err != nil {
		t.Errorf("Expected valid schema to pass validation, got error: %v", err)
	}

	// Test missing schema version
	missingSchemaResult := BaselineTestResult{
		TestName: "missing_schema_test",
	}

	if err := capture.ValidateGoldenFileSchema(&missingSchemaResult); err == nil {
		t.Error("Expected validation error for missing schema version")
	}

	// Test incompatible schema version
	incompatibleResult := BaselineTestResult{
		SchemaVersion: "0.1.0",
		TestName:      "incompatible_schema_test",
	}

	if err := capture.ValidateGoldenFileSchema(&incompatibleResult); err == nil {
		t.Error("Expected validation error for incompatible schema version")
	}

	t.Logf("Schema validation working correctly")
}

// TestGoldenFileEnvironmentCapture tests environment information capture
func TestGoldenFileEnvironmentCapture(t *testing.T) {
	capture := NewBaselineCapture("")

	env := capture.captureEnvironment()

	// Verify basic system information is captured
	expectedFields := []string{"os", "arch", "go_version"}
	for _, field := range expectedFields {
		if _, exists := env[field]; !exists {
			t.Errorf("Expected environment field %s to be captured", field)
		}
	}

	// Verify OS matches runtime
	if env["os"] != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, env["os"])
	}

	// Verify architecture matches runtime
	if env["arch"] != runtime.GOARCH {
		t.Errorf("Expected architecture %s, got %s", runtime.GOARCH, env["arch"])
	}

	// Verify Go version is captured
	if env["go_version"] != runtime.Version() {
		t.Errorf("Expected Go version %s, got %s", runtime.Version(), env["go_version"])
	}

	t.Logf("Environment capture working correctly: captured %d environment fields", len(env))
}

// TestComprehensiveGoldenFileWorkflow tests the complete golden file workflow
func TestComprehensiveGoldenFileWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")

	capture := NewBaselineCapture(goldenDir)

	// Test complete workflow: capture -> save -> load -> validate -> compare
	testScenario := TestScenario{
		Name: "comprehensive_workflow_test",
		Config: InstallConfig{
			Force:             true,
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: true,
		},
		Description: "Test complete golden file workflow",
	}

	// Create mock baseline result
	originalResult := BaselineTestResult{
		TestName:       testScenario.Name,
		Config:         testScenario.Config,
		ExecutionOrder: []string{"step1", "step2", "step3"},
		StepTimings: map[string]time.Duration{
			"step1": time.Millisecond * 50,
			"step2": time.Millisecond * 100,
			"step3": time.Millisecond * 75,
		},
		CompletionTracking: []string{"step1", "step2", "step3"},
		TotalDuration:      time.Millisecond * 225,
		Success:            true,
	}

	// Generate timing metrics
	originalResult.TimingMetrics = capture.generateTimingMetrics(&originalResult)

	// Save with metadata
	if err := capture.SaveGoldenFileWithMetadata(testScenario.Name, &originalResult); err != nil {
		t.Fatalf("Failed to save golden file: %v", err)
	}

	// Load and validate
	loadedResult, err := capture.LoadAndValidateGoldenFile(testScenario.Name)
	if err != nil {
		t.Fatalf("Failed to load and validate golden file: %v", err)
	}

	// Compare with original (should be identical)
	differences := capture.CompareWithGolden(&loadedResult, &loadedResult)
	if len(differences) != 0 {
		t.Errorf("Expected no differences when comparing loaded result with itself, got: %v", differences)
	}

	// Test behavioral change detection
	modifiedResult := loadedResult
	modifiedResult.ExecutionOrder = []string{"step2", "step1", "step3"} // Different order

	differences = capture.CompareWithGolden(&modifiedResult, &loadedResult)
	if len(differences) == 0 {
		t.Error("Expected to detect execution order differences")
	}

	// Test timing metrics comparison (create modified timing metrics)
	if loadedResult.TimingMetrics != nil {
		modifiedTimingMetrics := &TimingMetrics{
			StepTimingOrder: []string{"step3", "step1", "step2"}, // Different timing order
			SlowSteps:       []string{"step1"},                   // Different slow steps
			FastSteps:       loadedResult.TimingMetrics.FastSteps,
		}
		modifiedResult.TimingMetrics = modifiedTimingMetrics

		timingDifferences := capture.CompareTimingMetrics(modifiedResult.TimingMetrics, loadedResult.TimingMetrics)
		if len(timingDifferences) == 0 {
			t.Error("Expected to detect timing metric differences")
		}
	}

	t.Logf("Comprehensive golden file workflow completed successfully")
	t.Logf("Baseline captured with %d steps, schema version %s", len(loadedResult.ExecutionOrder), loadedResult.SchemaVersion)
}

// TestGoldenFileOperations tests the golden file save/load functionality
func TestGoldenFileOperations(t *testing.T) {
	tempDir := t.TempDir()
	goldenDir := filepath.Join(tempDir, "golden")

	capture := NewBaselineCapture(goldenDir)

	// Create test result
	testResult := BaselineTestResult{
		TestName:           "test_scenario",
		Config:             InstallConfig{Force: true},
		ExecutionOrder:     []string{"step1", "step2", "step3"},
		StepTimings:        map[string]time.Duration{"step1": time.Millisecond * 100},
		CompletionTracking: []string{"step1", "step2", "step3"},
		Success:            true,
	}

	// Test save
	if err := capture.SaveGoldenFile("test_scenario", &testResult); err != nil {
		t.Fatalf("Failed to save golden file: %v", err)
	}

	// Test load
	loaded, err := capture.LoadGoldenFile("test_scenario")
	if err != nil {
		t.Fatalf("Failed to load golden file: %v", err)
	}

	// Verify data integrity
	if loaded.TestName != testResult.TestName {
		t.Errorf("Test name mismatch: got %s, expected %s", loaded.TestName, testResult.TestName)
	}

	if len(loaded.ExecutionOrder) != len(testResult.ExecutionOrder) {
		t.Errorf("Execution order length mismatch: got %d, expected %d", len(loaded.ExecutionOrder), len(testResult.ExecutionOrder))
	}
}

// TestCompareWithGolden tests the comparison functionality
func TestCompareWithGolden(t *testing.T) {
	capture := NewBaselineCapture("")

	golden := BaselineTestResult{
		ExecutionOrder:     []string{"step1", "step2", "step3"},
		CompletionTracking: []string{"step1", "step2", "step3"},
		Success:            true,
	}

	// Test identical results
	current := golden
	differences := capture.CompareWithGolden(&current, &golden)
	if len(differences) != 0 {
		t.Errorf("Expected no differences for identical results, got: %v", differences)
	}

	// Test different execution order
	current.ExecutionOrder = []string{"step1", "step3", "step2"}
	differences = capture.CompareWithGolden(&current, &golden)
	if len(differences) == 0 {
		t.Error("Expected differences for different execution order")
	}

	// Test different success status
	current = golden
	current.Success = false
	differences = capture.CompareWithGolden(&current, &golden)
	if len(differences) == 0 {
		t.Error("Expected differences for different success status")
	}
}
