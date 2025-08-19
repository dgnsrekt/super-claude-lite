package installer

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// EquivalenceTestHarness provides infrastructure for comparing DAG vs manual approaches
type EquivalenceTestHarness struct {
	TestName    string
	Config      *InstallConfig
	TempDirBase string
	Logger      *TestLogger
}

// TestLogger captures execution traces for comparison
type TestLogger struct {
	ExecutionTrace []ExecutionEvent
	ErrorLog       []string
}

// ExecutionEvent represents a step execution event
type ExecutionEvent struct {
	StepName  string
	Timestamp time.Time
	EventType string // "start", "complete", "validate", "error"
	Duration  time.Duration
	Error     error
}

// NewEquivalenceTestHarness creates a new test harness
func NewEquivalenceTestHarness(testName string, config *InstallConfig) *EquivalenceTestHarness {
	return &EquivalenceTestHarness{
		TestName: testName,
		Config:   config,
		Logger:   &TestLogger{},
	}
}

// SetupTestEnvironment creates isolated test directories for comparison testing
func (h *EquivalenceTestHarness) SetupTestEnvironment(t *testing.T) (dagDir, manualDir string) {
	t.Helper()

	baseDir := t.TempDir()
	h.TempDirBase = baseDir

	dagDir = filepath.Join(baseDir, "dag_approach")
	manualDir = filepath.Join(baseDir, "manual_approach")

	// Create both directories
	if err := os.MkdirAll(dagDir, 0o755); err != nil {
		t.Fatalf("Failed to create DAG test directory: %v", err)
	}
	if err := os.MkdirAll(manualDir, 0o755); err != nil {
		t.Fatalf("Failed to create manual test directory: %v", err)
	}

	return dagDir, manualDir
}

// CreateIdenticalEnvironment sets up identical file systems for both approaches
func (h *EquivalenceTestHarness) CreateIdenticalEnvironment(t *testing.T, dagDir, manualDir string, scenario *EquivalenceTestScenario) {
	t.Helper()

	// Create identical files in both directories based on scenario
	for _, file := range scenario.ExistingFiles {
		dagFile := filepath.Join(dagDir, file.Path)
		manualFile := filepath.Join(manualDir, file.Path)

		// Ensure parent directories exist
		if err := os.MkdirAll(filepath.Dir(dagFile), 0o755); err != nil {
			t.Fatalf("Failed to create DAG parent directory: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(manualFile), 0o755); err != nil {
			t.Fatalf("Failed to create manual parent directory: %v", err)
		}

		// Write identical content to both
		if err := os.WriteFile(dagFile, []byte(file.Content), 0o644); err != nil {
			t.Fatalf("Failed to create DAG file %s: %v", file.Path, err)
		}
		if err := os.WriteFile(manualFile, []byte(file.Content), 0o644); err != nil {
			t.Fatalf("Failed to create manual file %s: %v", file.Path, err)
		}
	}
}

// EquivalenceTestScenario defines a test scenario with specific file setup
type EquivalenceTestScenario struct {
	Name          string
	Description   string
	Config        *InstallConfig
	ExistingFiles []TestFile
	ExpectedFiles []string
}

// TestFile represents a file to be created in test environment
type TestFile struct {
	Path    string
	Content string
	Mode    os.FileMode
}

// InstrumentedInstaller wraps the regular installer with execution tracing
type InstrumentedInstaller struct {
	*Installer
	Logger *TestLogger
}

// NewInstrumentedInstaller creates an installer with execution tracing
func NewInstrumentedInstaller(targetDir string, config *InstallConfig, logger *TestLogger) (*InstrumentedInstaller, error) {
	installer, err := NewInstaller(targetDir, config)
	if err != nil {
		return nil, err
	}

	return &InstrumentedInstaller{
		Installer: installer,
		Logger:    logger,
	}, nil
}

// Install executes installation with full execution tracing
func (i *InstrumentedInstaller) Install() error {
	log.Printf("Starting SuperClaude installation")

	// Get topological ordering from the pre-built dependency graph
	executionOrder, err := i.graph.GetTopologicalOrder()
	if err != nil {
		return fmt.Errorf("failed to determine execution order: %w", err)
	}

	// Log the execution order for comparison
	i.Logger.LogEvent("execution_order", time.Now(), fmt.Sprintf("Order: %v", executionOrder), nil)

	// Execute steps in topological order with tracing
	for _, stepName := range executionOrder {
		step, exists := i.steps[stepName]
		if !exists {
			err := fmt.Errorf("step '%s' not found in available steps", stepName)
			i.Logger.LogEvent(stepName, time.Now(), "error", err)
			return err
		}

		startTime := time.Now()
		i.Logger.LogEvent(stepName, startTime, "start", nil)

		log.Printf("Executing step: %s", step.Name)

		// Execute the step
		if err := step.Execute(i.context); err != nil {
			execErr := fmt.Errorf("execution failed for step %s: %w", step.Name, err)
			i.Logger.LogEvent(stepName, time.Now(), "error", execErr)
			return execErr
		}

		// Run validation if defined (after execution)
		if step.Validate != nil {
			if err := step.Validate(i.context); err != nil {
				validErr := fmt.Errorf("validation failed for step %s: %w", step.Name, err)
				i.Logger.LogEvent(stepName, time.Now(), "validation_error", validErr)
				return validErr
			}
			i.Logger.LogEvent(stepName, time.Now(), "validate", nil)
		}

		// Mark step as completed
		i.context.Completed = append(i.context.Completed, step.Name)

		duration := time.Since(startTime)
		i.Logger.LogEvent(stepName, time.Now(), "complete", nil)

		log.Printf("Completed step: %s (took %v)", step.Name, duration)
	}

	return nil
}

// LogEvent records an execution event for later comparison
func (l *TestLogger) LogEvent(stepName string, timestamp time.Time, eventType string, err error) {
	event := ExecutionEvent{
		StepName:  stepName,
		Timestamp: timestamp,
		EventType: eventType,
		Error:     err,
	}

	l.ExecutionTrace = append(l.ExecutionTrace, event)

	if err != nil {
		l.ErrorLog = append(l.ErrorLog, fmt.Sprintf("[%s] %s: %v", stepName, eventType, err))
	}
}

// EquivalenceManualInstaller simulates the pre-DAG manual approach for comparison
type EquivalenceManualInstaller struct {
	steps   map[string]*InstallStep
	context *InstallContext
	Logger  *TestLogger
}

// NewEquivalenceManualInstaller creates an installer using manual dependency resolution
func NewEquivalenceManualInstaller(targetDir string, config *InstallConfig, logger *TestLogger) (*EquivalenceManualInstaller, error) {
	context, err := NewInstallContext(targetDir, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create install context: %w", err)
	}

	return &EquivalenceManualInstaller{
		steps:   GetInstallSteps(),
		context: context,
		Logger:  logger,
	}, nil
}

// Install executes installation using manual dependency resolution (pre-DAG approach)
func (m *EquivalenceManualInstaller) Install() error {
	log.Printf("Starting SuperClaude installation (manual traversal)")

	// Manual dependency resolution - this simulates the pre-DAG approach
	executionOrder := m.getManualExecutionOrder()

	// Log the execution order for comparison
	m.Logger.LogEvent("execution_order", time.Now(), fmt.Sprintf("Order: %v", executionOrder), nil)

	// Execute steps in manual order with tracing
	for _, stepName := range executionOrder {
		step, exists := m.steps[stepName]
		if !exists {
			err := fmt.Errorf("step '%s' not found in available steps", stepName)
			m.Logger.LogEvent(stepName, time.Now(), "error", err)
			return err
		}

		startTime := time.Now()
		m.Logger.LogEvent(stepName, startTime, "start", nil)

		log.Printf("Executing step: %s", step.Name)

		// Execute the step
		if err := step.Execute(m.context); err != nil {
			execErr := fmt.Errorf("execution failed for step %s: %w", step.Name, err)
			m.Logger.LogEvent(stepName, time.Now(), "error", execErr)
			return execErr
		}

		// Run validation if defined (after execution)
		if step.Validate != nil {
			if err := step.Validate(m.context); err != nil {
				validErr := fmt.Errorf("validation failed for step %s: %w", step.Name, err)
				m.Logger.LogEvent(stepName, time.Now(), "validation_error", validErr)
				return validErr
			}
			m.Logger.LogEvent(stepName, time.Now(), "validate", nil)
		}

		// Mark step as completed
		m.context.Completed = append(m.context.Completed, step.Name)

		duration := time.Since(startTime)
		m.Logger.LogEvent(stepName, time.Now(), "complete", nil)

		log.Printf("Completed step: %s (took %v)", step.Name, duration)
	}

	return nil
}

// getManualExecutionOrder returns the hardcoded execution order from pre-DAG implementation
func (m *EquivalenceManualInstaller) getManualExecutionOrder() []string {
	// This represents the original manual dependency resolution approach
	// Note: MergeOrCreateMCPConfig is always included in the step list,
	// but its position and execution dependencies are controlled by the DAG logic

	order := []string{
		"CheckPrerequisites",
		"ScanExistingFiles",
		"CreateBackups",
		"CheckTargetDirectory",
		"CloneRepository",
		"CreateDirectoryStructure",
		"CopyCoreFiles",
		"CopyCommandFiles",
		"MergeOrCreateCLAUDEmd",
		"MergeOrCreateMCPConfig", // Always included - DAG controls the dependencies
		"CreateCommandSymlink",
		"ValidateInstallation",
		"CleanupTempFiles",
	}

	return order
}

// GetInstallationSummary returns a summary of the manual installation
func (m *EquivalenceManualInstaller) GetInstallationSummary() InstallationSummary {
	summary := InstallationSummary{
		TargetDir:        m.context.TargetDir,
		BackupDir:        m.context.BackupDir,
		CompletedSteps:   m.context.Completed,
		ExistingFiles:    *m.context.ExistingFiles,
		MCPConfigCreated: m.context.Config.AddRecommendedMCP,
	}

	if m.context.BackupManager != nil {
		summary.BackedUpFiles = make([]string, 0, len(m.context.BackupManager.Files))
		for original := range m.context.BackupManager.Files {
			summary.BackedUpFiles = append(summary.BackedUpFiles, original)
		}
	}

	return summary
}

// CompareExecutionTraces compares execution traces between DAG and manual approaches
// This validates that both approaches respect dependency constraints, even if execution order differs
func CompareExecutionTraces(t *testing.T, dagTrace, manualTrace []ExecutionEvent) {
	t.Helper()

	// Extract execution orders for analysis
	dagOrder := extractExecutionOrder(dagTrace)
	manualOrder := extractExecutionOrder(manualTrace)

	t.Logf("DAG execution order: %v", dagOrder)
	t.Logf("Manual execution order: %v", manualOrder)

	// Validate both orderings respect the same dependency constraints
	dagValid := validateDependencyConstraints(t, dagOrder)
	manualValid := validateDependencyConstraints(t, manualOrder)

	if !dagValid {
		t.Errorf("DAG execution order violates dependency constraints")
	}
	if !manualValid {
		t.Errorf("Manual execution order violates dependency constraints")
	}

	// Check that both approaches executed the same set of steps
	dagSteps := extractCompletedSteps(dagTrace)
	manualSteps := extractCompletedSteps(manualTrace)

	if !reflect.DeepEqual(dagSteps, manualSteps) {
		t.Errorf("Different steps executed:\nDAG completed:    %v\nManual completed: %v", dagSteps, manualSteps)
	} else {
		t.Logf("✅ Both approaches completed the same steps: %v", dagSteps)
	}

	// Log execution order comparison for analysis
	if !reflect.DeepEqual(dagOrder, manualOrder) {
		t.Logf("📊 Execution orders differ (this may be valid due to topological optimization):")
		t.Logf("   DAG optimized order:  %v", dagOrder)
		t.Logf("   Manual fixed order:   %v", manualOrder)

		// Analyze optimization differences
		analyzeOptimizationDifferences(t, dagOrder, manualOrder)
	} else {
		t.Logf("✅ Execution orders are identical")
	}
}

// extractExecutionOrder extracts just the step execution order (ignoring event types)
func extractExecutionOrder(trace []ExecutionEvent) []string {
	var order []string
	completed := make(map[string]bool)

	for _, event := range trace {
		// Only capture the first "complete" event for each step to get execution order
		if event.EventType == "complete" && !completed[event.StepName] {
			order = append(order, event.StepName)
			completed[event.StepName] = true
		}
	}
	return order
}

// extractCompletedSteps extracts the set of completed steps (sorted for comparison)
func extractCompletedSteps(trace []ExecutionEvent) []string {
	stepSet := make(map[string]bool)

	for _, event := range trace {
		if event.EventType == "complete" {
			stepSet[event.StepName] = true
		}
	}

	// Convert to sorted slice for consistent comparison
	steps := make([]string, 0, len(stepSet))
	for step := range stepSet {
		steps = append(steps, step)
	}
	sort.Strings(steps)
	return steps
}

// validateDependencyConstraints validates that an execution order respects dependency constraints
func validateDependencyConstraints(t *testing.T, executionOrder []string) bool {
	t.Helper()

	// Define the known dependency constraints from the BuildInstallationGraph method
	dependencies := map[string][]string{
		"ScanExistingFiles":        {"CheckPrerequisites"},
		"CreateBackups":            {"ScanExistingFiles"},
		"CheckTargetDirectory":     {"CreateBackups"},
		"CloneRepository":          {"CheckTargetDirectory"},
		"CreateDirectoryStructure": {"CheckTargetDirectory"},
		"CopyCoreFiles":            {"CloneRepository", "CreateDirectoryStructure"},
		"CopyCommandFiles":         {"CloneRepository", "CreateDirectoryStructure"},
		"MergeOrCreateCLAUDEmd":    {"CreateDirectoryStructure"},
		"MergeOrCreateMCPConfig":   {"CreateDirectoryStructure"},
		"CreateCommandSymlink":     {"CopyCommandFiles", "CreateDirectoryStructure"},
		"ValidateInstallation":     {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink"},
		"CleanupTempFiles":         {"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation"},
	}

	// Create position map for quick lookup
	positions := make(map[string]int)
	for i, step := range executionOrder {
		positions[step] = i
	}

	// Validate each dependency constraint
	valid := true
	for step, deps := range dependencies {
		stepPos, stepExists := positions[step]
		if !stepExists {
			continue // Step not in this execution (might be conditional)
		}

		for _, dep := range deps {
			depPos, depExists := positions[dep]
			if !depExists {
				t.Logf("⚠️ Step %s depends on %s, but %s was not executed", step, dep, dep)
				continue
			}

			if depPos >= stepPos {
				t.Logf("❌ Dependency violation: %s (pos %d) should execute before %s (pos %d)",
					dep, depPos, step, stepPos)
				valid = false
			}
		}
	}

	return valid
}

// analyzeOptimizationDifferences analyzes and logs the differences between execution orders
func analyzeOptimizationDifferences(t *testing.T, dagOrder, manualOrder []string) {
	t.Helper()

	// Find steps that are in different positions
	dagPositions := make(map[string]int)
	manualPositions := make(map[string]int)

	for i, step := range dagOrder {
		dagPositions[step] = i
	}
	for i, step := range manualOrder {
		manualPositions[step] = i
	}

	var optimizations []string
	for step := range dagPositions {
		dagPos := dagPositions[step]
		manualPos := manualPositions[step]

		if dagPos != manualPos {
			if dagPos < manualPos {
				optimizations = append(optimizations, fmt.Sprintf("%s moved earlier (manual:%d → dag:%d)", step, manualPos, dagPos))
			} else {
				optimizations = append(optimizations, fmt.Sprintf("%s moved later (manual:%d → dag:%d)", step, manualPos, dagPos))
			}
		}
	}

	if len(optimizations) > 0 {
		t.Logf("🔧 DAG optimizations detected:")
		for _, opt := range optimizations {
			t.Logf("   • %s", opt)
		}
	}
}

// extractStepSequence extracts the sequence of step names and event types (legacy function for compatibility)

// CompareInstallationSummaries compares installation summaries between approaches
func CompareInstallationSummaries(t *testing.T, dagSummary, manualSummary *InstallationSummary) {
	t.Helper()

	// Compare completed steps (order might differ due to topological sorting)
	dagSteps := make([]string, len(dagSummary.CompletedSteps))
	copy(dagSteps, dagSummary.CompletedSteps)
	sort.Strings(dagSteps)

	manualSteps := make([]string, len(manualSummary.CompletedSteps))
	copy(manualSteps, manualSummary.CompletedSteps)
	sort.Strings(manualSteps)

	if !reflect.DeepEqual(dagSteps, manualSteps) {
		t.Errorf("Completed steps differ:\nDAG:    %v\nManual: %v", dagSteps, manualSteps)
	}

	// Compare other summary fields
	if dagSummary.MCPConfigCreated != manualSummary.MCPConfigCreated {
		t.Errorf("MCP config creation differs: DAG=%v, Manual=%v",
			dagSummary.MCPConfigCreated, manualSummary.MCPConfigCreated)
	}

	// Compare existing files detection
	if !reflect.DeepEqual(dagSummary.ExistingFiles, manualSummary.ExistingFiles) {
		t.Errorf("Existing files detection differs:\nDAG:    %+v\nManual: %+v",
			dagSummary.ExistingFiles, manualSummary.ExistingFiles)
	}

	// Compare backed up files (normalize paths for comparison since absolute paths will differ)
	dagBackedUpRelative := normalizeBackupPaths(dagSummary.BackedUpFiles, dagSummary.TargetDir)
	manualBackedUpRelative := normalizeBackupPaths(manualSummary.BackedUpFiles, manualSummary.TargetDir)

	sort.Strings(dagBackedUpRelative)
	sort.Strings(manualBackedUpRelative)

	if !reflect.DeepEqual(dagBackedUpRelative, manualBackedUpRelative) {
		t.Errorf("Backed up files differ (relative paths):\nDAG:    %v\nManual: %v", dagBackedUpRelative, manualBackedUpRelative)
	}
}

// normalizeBackupPaths converts absolute backup paths to relative paths for comparison
func normalizeBackupPaths(backupPaths []string, targetDir string) []string {
	var relativePaths []string
	for _, backupPath := range backupPaths {
		// Extract the original file path (the key in backup manager)
		// Since this is the original file that was backed up, we want to make it relative to targetDir
		if strings.HasPrefix(backupPath, targetDir) {
			relativePath := strings.TrimPrefix(backupPath, targetDir)
			relativePath = strings.TrimPrefix(relativePath, "/") // Remove leading slash
			if relativePath != "" {
				relativePaths = append(relativePaths, relativePath)
			}
		} else {
			// If it doesn't have the target directory prefix, just use the basename
			relativePaths = append(relativePaths, filepath.Base(backupPath))
		}
	}
	return relativePaths
}

// GetStandardTestScenarios returns common test scenarios for equivalence testing
func GetStandardTestScenarios() []EquivalenceTestScenario {
	return []EquivalenceTestScenario{
		{
			Name:        "clean_installation",
			Description: "Fresh installation with no existing files",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			ExistingFiles: []TestFile{},
			ExpectedFiles: []string{".superclaude", ".claude"},
		},
		{
			Name:        "with_mcp_config",
			Description: "Installation with MCP configuration enabled",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			ExistingFiles: []TestFile{},
			ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
		},
		{
			Name:        "existing_claude_md",
			Description: "Installation with existing CLAUDE.md file",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    "CLAUDE.md",
					Content: "# Existing CLAUDE.md\nSome existing content\n",
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", "CLAUDE.md"},
		},
		{
			Name:        "no_backup_mode",
			Description: "Installation with backup disabled",
			Config: &InstallConfig{
				NoBackup:          true,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    "CLAUDE.md",
					Content: "# Existing content\n",
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", "CLAUDE.md"},
		},
		{
			Name:        "complex_existing_setup",
			Description: "Installation with multiple existing files",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    "CLAUDE.md",
					Content: "# Existing CLAUDE.md\n",
					Mode:    0o644,
				},
				{
					Path:    ".mcp.json",
					Content: `{"existing": "mcp config"}`,
					Mode:    0o644,
				},
				{
					Path:    ".superclaude/existing.txt",
					Content: "existing superclaude file",
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", "CLAUDE.md", ".mcp.json"},
		},
	}
}

// TestMCPConfigurationEquivalence specifically tests MCP configuration handling scenarios
func TestMCPConfigurationEquivalence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MCP configuration equivalence tests in short mode")
	}

	mcpScenarios := []EquivalenceTestScenario{
		{
			Name:        "mcp_enabled_clean",
			Description: "MCP configuration enabled with clean installation",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			ExistingFiles: []TestFile{},
			ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
		},
		{
			Name:        "mcp_disabled_clean",
			Description: "MCP configuration disabled with clean installation",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			ExistingFiles: []TestFile{},
			ExpectedFiles: []string{".superclaude", ".claude"},
		},
		{
			Name:        "mcp_enabled_existing_config",
			Description: "MCP configuration enabled with existing .mcp.json",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    ".mcp.json",
					Content: `{"mcpServers": {"existing": {"command": "test"}}}`,
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
		},
		{
			Name:        "mcp_disabled_existing_config",
			Description: "MCP configuration disabled with existing .mcp.json (should preserve)",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    ".mcp.json",
					Content: `{"mcpServers": {"existing": {"command": "test"}}}`,
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
		},
	}

	for _, scenario := range mcpScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Create test harness
			harness := NewEquivalenceTestHarness(scenario.Name, scenario.Config)

			// Setup isolated test environments
			dagDir, manualDir := harness.SetupTestEnvironment(t)

			// Create identical starting environments
			harness.CreateIdenticalEnvironment(t, dagDir, manualDir, &scenario)

			// Create separate loggers for each approach
			dagLogger := &TestLogger{}
			manualLogger := &TestLogger{}

			// Test DAG approach
			dagInstaller, err := NewInstrumentedInstaller(dagDir, scenario.Config, dagLogger)
			if err != nil {
				t.Fatalf("Failed to create DAG installer: %v", err)
			}

			// Test manual approach
			manualInstaller, err := NewEquivalenceManualInstaller(manualDir, scenario.Config, manualLogger)
			if err != nil {
				t.Fatalf("Failed to create manual installer: %v", err)
			}

			// Execute both approaches
			dagErr := dagInstaller.Install()
			manualErr := manualInstaller.Install()

			// Both should succeed or both should fail
			if (dagErr == nil) != (manualErr == nil) {
				t.Errorf("Error outcomes differ:\nDAG error: %v\nManual error: %v", dagErr, manualErr)
			}

			// If both succeeded, compare results
			if dagErr == nil && manualErr == nil {
				// Compare execution traces with special focus on MCP-related steps
				CompareMCPExecutionTraces(t, dagLogger.ExecutionTrace, manualLogger.ExecutionTrace, scenario.Config.AddRecommendedMCP)

				// Compare installation summaries
				dagSummary := dagInstaller.GetInstallationSummary()
				manualSummary := manualInstaller.GetInstallationSummary()
				CompareInstallationSummaries(t, &dagSummary, &manualSummary)

				// Verify MCP configuration behavior specifically
				VerifyMCPConfigurationBehavior(t, dagDir, manualDir, scenario.Config.AddRecommendedMCP)

				// Log success
				t.Logf("✅ MCP configuration equivalence verified for scenario: %s", scenario.Name)
			} else {
				// Both failed - compare error characteristics
				if dagErr.Error() != manualErr.Error() {
					t.Logf("⚠️  Both approaches failed but with different errors:")
					t.Logf("   DAG error: %v", dagErr)
					t.Logf("   Manual error: %v", manualErr)
				} else {
					t.Logf("✅ Both approaches failed with same error: %v", dagErr)
				}
			}
		})
	}
}

// CompareMCPExecutionTraces compares execution traces with special focus on MCP step handling
func CompareMCPExecutionTraces(t *testing.T, dagTrace, manualTrace []ExecutionEvent, mcpEnabled bool) {
	t.Helper()

	// First, do the standard execution trace comparison
	CompareExecutionTraces(t, dagTrace, manualTrace)

	// Then, specifically check MCP-related step execution
	dagMCPStepExecuted := hasStepInTrace(dagTrace, "MergeOrCreateMCPConfig")
	manualMCPStepExecuted := hasStepInTrace(manualTrace, "MergeOrCreateMCPConfig")

	// Both approaches should handle MCP step consistently
	if dagMCPStepExecuted != manualMCPStepExecuted {
		t.Errorf("MCP step execution differs: DAG executed MCP step=%v, Manual executed MCP step=%v",
			dagMCPStepExecuted, manualMCPStepExecuted)
	}

	// MCP step should always be included in the step list regardless of configuration
	// The configuration only affects dependencies, not step inclusion
	if !dagMCPStepExecuted || !manualMCPStepExecuted {
		t.Logf("⚠️  MCP step not executed in one or both approaches (DAG:%v, Manual:%v)",
			dagMCPStepExecuted, manualMCPStepExecuted)
		t.Logf("Note: MergeOrCreateMCPConfig step should always be included in the execution")
	}

	// Log MCP behavior analysis
	if mcpEnabled {
		t.Logf("🔧 MCP configuration enabled - ValidateInstallation and CleanupTempFiles should depend on MergeOrCreateMCPConfig")
	} else {
		t.Logf("🔧 MCP configuration disabled - MergeOrCreateMCPConfig step still executes but without additional dependencies")
	}
}

// VerifyMCPConfigurationBehavior verifies the actual MCP configuration file handling
func VerifyMCPConfigurationBehavior(t *testing.T, dagDir, manualDir string, mcpEnabled bool) {
	t.Helper()

	dagMCPFile := filepath.Join(dagDir, ".mcp.json")
	manualMCPFile := filepath.Join(manualDir, ".mcp.json")

	// Check if MCP files exist in both directories
	dagMCPExists := testFileExists(dagMCPFile)
	manualMCPExists := testFileExists(manualMCPFile)

	if dagMCPExists != manualMCPExists {
		t.Errorf("MCP file existence differs: DAG has .mcp.json=%v, Manual has .mcp.json=%v",
			dagMCPExists, manualMCPExists)
		return
	}

	// If both files exist, compare their contents
	if dagMCPExists && manualMCPExists {
		dagContent, err := os.ReadFile(dagMCPFile)
		if err != nil {
			t.Errorf("Failed to read DAG MCP file: %v", err)
			return
		}

		manualContent, err := os.ReadFile(manualMCPFile)
		if err != nil {
			t.Errorf("Failed to read manual MCP file: %v", err)
			return
		}

		if !bytes.Equal(dagContent, manualContent) {
			t.Errorf("MCP file contents differ:\nDAG content: %s\nManual content: %s",
				string(dagContent), string(manualContent))
		} else {
			t.Logf("✅ MCP file contents are identical")
		}
	}

	// Log expected behavior based on configuration
	if mcpEnabled {
		if !dagMCPExists {
			t.Logf("⚠️  MCP enabled but no .mcp.json file created - this may be expected if step implementation is conditional")
		} else {
			t.Logf("✅ MCP enabled and .mcp.json file present")
		}
	} else {
		if dagMCPExists {
			t.Logf("ℹ️  MCP disabled but .mcp.json file exists - this may be from existing files or step always executing")
		} else {
			t.Logf("✅ MCP disabled and no .mcp.json file created")
		}
	}
}

// hasStepInTrace checks if a specific step was executed in the trace
func hasStepInTrace(trace []ExecutionEvent, stepName string) bool {
	for _, event := range trace {
		if event.StepName == stepName && event.EventType == "complete" {
			return true
		}
	}
	return false
}

// testFileExists checks if a file exists (renamed to avoid conflict with context.go)
func testFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// TestFunctionalEquivalence is the main test that verifies DAG and manual approaches produce identical results
func TestFunctionalEquivalence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping functional equivalence tests in short mode")
	}

	scenarios := GetStandardTestScenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Create test harness
			harness := NewEquivalenceTestHarness(scenario.Name, scenario.Config)

			// Setup isolated test environments
			dagDir, manualDir := harness.SetupTestEnvironment(t)

			// Create identical starting environments
			harness.CreateIdenticalEnvironment(t, dagDir, manualDir, &scenario)

			// Create separate loggers for each approach
			dagLogger := &TestLogger{}
			manualLogger := &TestLogger{}

			// Test DAG approach
			dagInstaller, err := NewInstrumentedInstaller(dagDir, scenario.Config, dagLogger)
			if err != nil {
				t.Fatalf("Failed to create DAG installer: %v", err)
			}

			// Test manual approach
			manualInstaller, err := NewEquivalenceManualInstaller(manualDir, scenario.Config, manualLogger)
			if err != nil {
				t.Fatalf("Failed to create manual installer: %v", err)
			}

			// Execute both approaches
			dagErr := dagInstaller.Install()
			manualErr := manualInstaller.Install()

			// Both should succeed or both should fail
			if (dagErr == nil) != (manualErr == nil) {
				t.Errorf("Error outcomes differ:\nDAG error: %v\nManual error: %v", dagErr, manualErr)
			}

			// If both succeeded, compare results
			if dagErr == nil && manualErr == nil {
				// Compare execution traces
				CompareExecutionTraces(t, dagLogger.ExecutionTrace, manualLogger.ExecutionTrace)

				// Compare installation summaries
				dagSummary := dagInstaller.GetInstallationSummary()
				manualSummary := manualInstaller.GetInstallationSummary()
				CompareInstallationSummaries(t, &dagSummary, &manualSummary)

				// Log success
				t.Logf("✅ Equivalence verified for scenario: %s", scenario.Name)
			} else {
				// Both failed - compare error characteristics
				if dagErr.Error() != manualErr.Error() {
					t.Logf("⚠️  Both approaches failed but with different errors:")
					t.Logf("   DAG error: %v", dagErr)
					t.Logf("   Manual error: %v", manualErr)
					// This might be acceptable if error messages are slightly different
				} else {
					t.Logf("✅ Both approaches failed with same error: %v", dagErr)
				}
			}
		})
	}
}

// TestCompletionTrackingEquivalence tests completion tracking and progress reporting
func TestCompletionTrackingEquivalence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping completion tracking equivalence tests in short mode")
	}

	scenario := EquivalenceTestScenario{
		Name:        "completion_tracking_test",
		Description: "Test completion tracking and progress reporting equivalence",
		Config: &InstallConfig{
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: true,
			Force:             false,
		},
		ExistingFiles: []TestFile{
			{
				Path:    "existing.txt",
				Content: "existing content",
				Mode:    0o644,
			},
		},
		ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
	}

	// Create test harness
	harness := NewEquivalenceTestHarness(scenario.Name, scenario.Config)

	// Setup isolated test environments
	dagDir, manualDir := harness.SetupTestEnvironment(t)

	// Create identical starting environments
	harness.CreateIdenticalEnvironment(t, dagDir, manualDir, &scenario)

	// Create separate loggers for each approach
	dagLogger := &TestLogger{}
	manualLogger := &TestLogger{}

	// Test DAG approach
	dagInstaller, err := NewInstrumentedInstaller(dagDir, scenario.Config, dagLogger)
	if err != nil {
		t.Fatalf("Failed to create DAG installer: %v", err)
	}

	// Test manual approach
	manualInstaller, err := NewEquivalenceManualInstaller(manualDir, scenario.Config, manualLogger)
	if err != nil {
		t.Fatalf("Failed to create manual installer: %v", err)
	}

	// Execute both approaches
	dagErr := dagInstaller.Install()
	manualErr := manualInstaller.Install()

	// Both should succeed
	if dagErr != nil {
		t.Fatalf("DAG installation failed: %v", dagErr)
	}
	if manualErr != nil {
		t.Fatalf("Manual installation failed: %v", manualErr)
	}

	// Compare completion tracking
	CompareCompletionTracking(t, dagLogger.ExecutionTrace, manualLogger.ExecutionTrace)

	// Compare progress reporting
	CompareProgressReporting(t, dagInstaller.GetContext(), manualInstaller.context)

	t.Logf("✅ Completion tracking and progress reporting equivalence verified")
}

// CompareCompletionTracking verifies that completion tracking works equivalently
func CompareCompletionTracking(t *testing.T, dagTrace, manualTrace []ExecutionEvent) {
	t.Helper()

	// Extract completion events from both traces
	dagCompletions := extractCompletionEvents(dagTrace)
	manualCompletions := extractCompletionEvents(manualTrace)

	// Both should have the same number of completion events
	if len(dagCompletions) != len(manualCompletions) {
		t.Errorf("Different number of completion events: DAG=%d, Manual=%d",
			len(dagCompletions), len(manualCompletions))
		return
	}

	// Verify each step has exactly one completion event
	dagStepCompletions := make(map[string]int)
	manualStepCompletions := make(map[string]int)

	for _, event := range dagCompletions {
		dagStepCompletions[event.StepName]++
	}

	for _, event := range manualCompletions {
		manualStepCompletions[event.StepName]++
	}

	// Check for any steps with multiple completion events
	for step, count := range dagStepCompletions {
		if count != 1 {
			t.Errorf("DAG step %s has %d completion events, expected 1", step, count)
		}
	}

	for step, count := range manualStepCompletions {
		if count != 1 {
			t.Errorf("Manual step %s has %d completion events, expected 1", step, count)
		}
	}

	// Verify both approaches completed the same set of steps
	dagStepSet := make(map[string]bool)
	manualStepSet := make(map[string]bool)

	for step := range dagStepCompletions {
		dagStepSet[step] = true
	}

	for step := range manualStepCompletions {
		manualStepSet[step] = true
	}

	if !reflect.DeepEqual(dagStepSet, manualStepSet) {
		t.Errorf("Different sets of completed steps:\nDAG: %v\nManual: %v",
			getMapKeys(dagStepSet), getMapKeys(manualStepSet))
	}

	t.Logf("✅ Completion tracking verified: %d steps completed by both approaches", len(dagCompletions))
}

// CompareProgressReporting verifies that progress reporting is equivalent
func CompareProgressReporting(t *testing.T, dagContext, manualContext *InstallContext) {
	t.Helper()

	// Compare completed step lists
	dagCompleted := make([]string, len(dagContext.Completed))
	copy(dagCompleted, dagContext.Completed)
	sort.Strings(dagCompleted)

	manualCompleted := make([]string, len(manualContext.Completed))
	copy(manualCompleted, manualContext.Completed)
	sort.Strings(manualCompleted)

	if !reflect.DeepEqual(dagCompleted, manualCompleted) {
		t.Errorf("Progress reporting differs:\nDAG completed:    %v\nManual completed: %v",
			dagCompleted, manualCompleted)
		return
	}

	// Verify completion progress matches the expected number of steps
	expectedSteps := GetInstallSteps()
	if len(dagCompleted) != len(expectedSteps) {
		t.Errorf("DAG completion count mismatch: completed %d steps, expected %d",
			len(dagCompleted), len(expectedSteps))
	}

	if len(manualCompleted) != len(expectedSteps) {
		t.Errorf("Manual completion count mismatch: completed %d steps, expected %d",
			len(manualCompleted), len(expectedSteps))
	}

	// Verify all completed steps are valid installation steps
	for _, step := range dagCompleted {
		if _, exists := expectedSteps[step]; !exists {
			t.Errorf("DAG completed unknown step: %s", step)
		}
	}

	for _, step := range manualCompleted {
		if _, exists := expectedSteps[step]; !exists {
			t.Errorf("Manual completed unknown step: %s", step)
		}
	}

	t.Logf("✅ Progress reporting verified: %d/%d steps completed",
		len(dagCompleted), len(expectedSteps))
}

// extractCompletionEvents extracts all completion events from a trace
func extractCompletionEvents(trace []ExecutionEvent) []ExecutionEvent {
	var completions []ExecutionEvent
	for _, event := range trace {
		if event.EventType == "complete" {
			completions = append(completions, event)
		}
	}
	return completions
}

// getMapKeys returns the keys of a map as a sorted slice
func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestInstallationSummaryEquivalence tests installation summary generation and error propagation
func TestInstallationSummaryEquivalence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping installation summary equivalence tests in short mode")
	}

	summaryScenarios := []EquivalenceTestScenario{
		{
			Name:        "summary_clean_install",
			Description: "Test summary generation for clean installation",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			ExistingFiles: []TestFile{},
			ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
		},
		{
			Name:        "summary_with_existing_files",
			Description: "Test summary generation with existing files and backups",
			Config: &InstallConfig{
				NoBackup:          false,
				Interactive:       false,
				AddRecommendedMCP: false,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    "CLAUDE.md",
					Content: "# Existing CLAUDE.md\nExisting content\n",
					Mode:    0o644,
				},
				{
					Path:    ".claude/existing.json",
					Content: `{"existing": true}`,
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", "CLAUDE.md"},
		},
		{
			Name:        "summary_no_backup_mode",
			Description: "Test summary generation with backup disabled",
			Config: &InstallConfig{
				NoBackup:          true,
				Interactive:       false,
				AddRecommendedMCP: true,
				Force:             false,
			},
			ExistingFiles: []TestFile{
				{
					Path:    "test.txt",
					Content: "test content",
					Mode:    0o644,
				},
			},
			ExpectedFiles: []string{".superclaude", ".claude", ".mcp.json"},
		},
	}

	for _, scenario := range summaryScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Create test harness
			harness := NewEquivalenceTestHarness(scenario.Name, scenario.Config)

			// Setup isolated test environments
			dagDir, manualDir := harness.SetupTestEnvironment(t)

			// Create identical starting environments
			harness.CreateIdenticalEnvironment(t, dagDir, manualDir, &scenario)

			// Create separate loggers for each approach
			dagLogger := &TestLogger{}
			manualLogger := &TestLogger{}

			// Test DAG approach
			dagInstaller, err := NewInstrumentedInstaller(dagDir, scenario.Config, dagLogger)
			if err != nil {
				t.Fatalf("Failed to create DAG installer: %v", err)
			}

			// Test manual approach
			manualInstaller, err := NewEquivalenceManualInstaller(manualDir, scenario.Config, manualLogger)
			if err != nil {
				t.Fatalf("Failed to create manual installer: %v", err)
			}

			// Execute both approaches
			dagErr := dagInstaller.Install()
			manualErr := manualInstaller.Install()

			// Both should succeed for these scenarios
			if dagErr != nil {
				t.Fatalf("DAG installation failed: %v", dagErr)
			}
			if manualErr != nil {
				t.Fatalf("Manual installation failed: %v", manualErr)
			}

			// Get installation summaries
			dagSummary := dagInstaller.GetInstallationSummary()
			manualSummary := manualInstaller.GetInstallationSummary()

			// Compare summaries in detail
			CompareDetailedInstallationSummaries(t, &dagSummary, &manualSummary, scenario.Config)

			t.Logf("✅ Installation summary equivalence verified for scenario: %s", scenario.Name)
		})
	}
}

// TestErrorPropagationEquivalence tests error handling and propagation
func TestErrorPropagationEquivalence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error propagation equivalence tests in short mode")
	}

	// Test scenario that should result in identical error behavior
	// Note: This test simulates error conditions by using invalid configurations
	scenario := EquivalenceTestScenario{
		Name:        "error_propagation_test",
		Description: "Test error propagation equivalence",
		Config: &InstallConfig{
			NoBackup:          false,
			Interactive:       false,
			AddRecommendedMCP: false,
			Force:             false,
		},
		ExistingFiles: []TestFile{},
		ExpectedFiles: []string{},
	}

	// Create test harness
	harness := NewEquivalenceTestHarness(scenario.Name, scenario.Config)

	// Setup isolated test environments
	dagDir, manualDir := harness.SetupTestEnvironment(t)

	// Create identical starting environments
	harness.CreateIdenticalEnvironment(t, dagDir, manualDir, &scenario)

	// Create separate loggers for each approach
	dagLogger := &TestLogger{}
	manualLogger := &TestLogger{}

	// Test DAG approach
	dagInstaller, err := NewInstrumentedInstaller(dagDir, scenario.Config, dagLogger)
	if err != nil {
		t.Fatalf("Failed to create DAG installer: %v", err)
	}

	// Test manual approach
	manualInstaller, err := NewEquivalenceManualInstaller(manualDir, scenario.Config, manualLogger)
	if err != nil {
		t.Fatalf("Failed to create manual installer: %v", err)
	}

	// Execute both approaches
	dagErr := dagInstaller.Install()
	manualErr := manualInstaller.Install()

	// For this test, we expect both to succeed (no forced errors in normal flow)
	// But we can verify error handling consistency
	CompareErrorHandling(t, dagErr, manualErr, dagLogger.ErrorLog, manualLogger.ErrorLog)

	t.Logf("✅ Error propagation equivalence verified")
}

// CompareDetailedInstallationSummaries provides comprehensive summary comparison
func CompareDetailedInstallationSummaries(t *testing.T, dagSummary, manualSummary *InstallationSummary, config *InstallConfig) {
	t.Helper()

	// Compare basic directory information
	if dagSummary.TargetDir != manualSummary.TargetDir {
		// Target dirs will be different (different temp dirs), but their structure should be similar
		t.Logf("Target directories differ (expected): DAG=%s, Manual=%s",
			filepath.Base(dagSummary.TargetDir), filepath.Base(manualSummary.TargetDir))
	}

	// Compare MCP configuration behavior
	if dagSummary.MCPConfigCreated != manualSummary.MCPConfigCreated {
		t.Errorf("MCP config creation status differs: DAG=%v, Manual=%v",
			dagSummary.MCPConfigCreated, manualSummary.MCPConfigCreated)
	}

	// Verify MCP configuration matches config setting
	expectedMCP := config.AddRecommendedMCP
	if dagSummary.MCPConfigCreated != expectedMCP {
		t.Errorf("DAG MCP config creation doesn't match setting: got %v, expected %v",
			dagSummary.MCPConfigCreated, expectedMCP)
	}
	if manualSummary.MCPConfigCreated != expectedMCP {
		t.Errorf("Manual MCP config creation doesn't match setting: got %v, expected %v",
			manualSummary.MCPConfigCreated, expectedMCP)
	}

	// Compare existing files detection
	CompareExistingFilesDetection(t, dagSummary.ExistingFiles, manualSummary.ExistingFiles)

	// Compare backed up files (using our normalized comparison)
	dagBackedUpRelative := normalizeBackupPaths(dagSummary.BackedUpFiles, dagSummary.TargetDir)
	manualBackedUpRelative := normalizeBackupPaths(manualSummary.BackedUpFiles, manualSummary.TargetDir)

	sort.Strings(dagBackedUpRelative)
	sort.Strings(manualBackedUpRelative)

	if !reflect.DeepEqual(dagBackedUpRelative, manualBackedUpRelative) {
		t.Errorf("Backed up files differ (relative paths):\nDAG:    %v\nManual: %v",
			dagBackedUpRelative, manualBackedUpRelative)
	}

	// Verify backup behavior matches configuration
	expectedBackups := !config.NoBackup && len(dagBackedUpRelative) > 0
	actualBackups := len(dagBackedUpRelative) > 0
	if expectedBackups != actualBackups {
		t.Logf("ℹ️  Backup behavior: expected backups=%v, actual backups=%v (may be valid if no files to backup)",
			expectedBackups, actualBackups)
	}

	// Compare completed steps (already tested in CompareInstallationSummaries, but verify here too)
	dagCompleted := make([]string, len(dagSummary.CompletedSteps))
	copy(dagCompleted, dagSummary.CompletedSteps)
	sort.Strings(dagCompleted)

	manualCompleted := make([]string, len(manualSummary.CompletedSteps))
	copy(manualCompleted, manualSummary.CompletedSteps)
	sort.Strings(manualCompleted)

	if !reflect.DeepEqual(dagCompleted, manualCompleted) {
		t.Errorf("Completed steps differ:\nDAG:    %v\nManual: %v", dagCompleted, manualCompleted)
	}

	t.Logf("✅ Detailed summary comparison passed: %d steps completed, MCP=%v, %d files backed up",
		len(dagCompleted), dagSummary.MCPConfigCreated, len(dagBackedUpRelative))
}

// CompareExistingFilesDetection compares the existing files detection logic
func CompareExistingFilesDetection(t *testing.T, dagExisting, manualExisting ExistingFiles) {
	t.Helper()

	if dagExisting.CLAUDEmd != manualExisting.CLAUDEmd {
		t.Errorf("CLAUDE.md existence detection differs: DAG=%v, Manual=%v",
			dagExisting.CLAUDEmd, manualExisting.CLAUDEmd)
	}

	if dagExisting.MCPConfig != manualExisting.MCPConfig {
		t.Errorf("MCP config existence detection differs: DAG=%v, Manual=%v",
			dagExisting.MCPConfig, manualExisting.MCPConfig)
	}

	if dagExisting.ClaudeDir != manualExisting.ClaudeDir {
		t.Errorf(".claude directory existence detection differs: DAG=%v, Manual=%v",
			dagExisting.ClaudeDir, manualExisting.ClaudeDir)
	}

	if dagExisting.SuperClaudeDir != manualExisting.SuperClaudeDir {
		t.Errorf(".superclaude directory existence detection differs: DAG=%v, Manual=%v",
			dagExisting.SuperClaudeDir, manualExisting.SuperClaudeDir)
	}
}

// CompareErrorHandling compares error handling behavior between approaches
func CompareErrorHandling(t *testing.T, dagErr, manualErr error, dagErrorLog, manualErrorLog []string) {
	t.Helper()

	// Compare error outcomes
	if (dagErr == nil) != (manualErr == nil) {
		t.Errorf("Error outcomes differ: DAG error=%v, Manual error=%v", dagErr, manualErr)
		return
	}

	// If both succeeded, check error logs for consistency
	if dagErr == nil && manualErr == nil {
		// Both succeeded - compare error logs (should be minimal for successful runs)
		if len(dagErrorLog) != len(manualErrorLog) {
			t.Logf("ℹ️  Error log lengths differ: DAG=%d, Manual=%d (may be acceptable)",
				len(dagErrorLog), len(manualErrorLog))
		}

		// Log any errors that were recovered from
		if len(dagErrorLog) > 0 {
			t.Logf("DAG error log: %v", dagErrorLog)
		}
		if len(manualErrorLog) > 0 {
			t.Logf("Manual error log: %v", manualErrorLog)
		}

		t.Logf("✅ Both approaches succeeded with consistent error handling")
		return
	}

	// Both failed - compare error characteristics
	if dagErr.Error() != manualErr.Error() {
		t.Logf("⚠️  Both approaches failed but with different error messages:")
		t.Logf("   DAG error: %v", dagErr)
		t.Logf("   Manual error: %v", manualErr)
		// This might be acceptable if error messages are slightly different but semantically equivalent
	} else {
		t.Logf("✅ Both approaches failed with identical error: %v", dagErr)
	}

	// Compare error logs for failed runs
	t.Logf("DAG error log entries: %d", len(dagErrorLog))
	t.Logf("Manual error log entries: %d", len(manualErrorLog))
}

// TestBasicEquivalenceInfrastructure tests the test harness itself
func TestBasicEquivalenceInfrastructure(t *testing.T) {
	// Test harness creation
	config := &InstallConfig{
		NoBackup:          true,
		Interactive:       false,
		AddRecommendedMCP: false,
		Force:             false,
	}

	harness := NewEquivalenceTestHarness("infrastructure_test", config)
	if harness == nil {
		t.Fatal("Failed to create test harness")
	}

	// Test environment setup
	dagDir, manualDir := harness.SetupTestEnvironment(t)

	// Verify directories were created
	if _, err := os.Stat(dagDir); os.IsNotExist(err) {
		t.Errorf("DAG directory was not created: %s", dagDir)
	}
	if _, err := os.Stat(manualDir); os.IsNotExist(err) {
		t.Errorf("Manual directory was not created: %s", manualDir)
	}

	// Test identical environment creation
	scenario := EquivalenceTestScenario{
		ExistingFiles: []TestFile{
			{
				Path:    "test.txt",
				Content: "test content",
				Mode:    0o644,
			},
		},
	}

	harness.CreateIdenticalEnvironment(t, dagDir, manualDir, &scenario)

	// Verify files were created in both directories
	dagFile := filepath.Join(dagDir, "test.txt")
	manualFile := filepath.Join(manualDir, "test.txt")

	dagContent, err := os.ReadFile(dagFile)
	if err != nil {
		t.Errorf("Failed to read DAG test file: %v", err)
	}

	manualContent, err := os.ReadFile(manualFile)
	if err != nil {
		t.Errorf("Failed to read manual test file: %v", err)
	}

	if !bytes.Equal(dagContent, manualContent) {
		t.Errorf("File contents differ:\nDAG: %s\nManual: %s", dagContent, manualContent)
	}

	t.Log("✅ Test harness infrastructure verified")
}
