package installer

import (
	"fmt"
	"log"
)

// Installer manages the SuperClaude installation process
type Installer struct {
	steps   map[string]*InstallStep
	context *InstallContext
}

// NewInstaller creates a new installer instance
func NewInstaller(targetDir string, config *InstallConfig) (*Installer, error) {
	context, err := NewInstallContext(targetDir, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create install context: %w", err)
	}

	installer := &Installer{
		steps:   GetInstallSteps(),
		context: context,
	}

	return installer, nil
}

// Install executes the installation process using DAG-based topological sorting
func (i *Installer) Install() error {
	log.Printf("Starting SuperClaude installation")

	// Create dependency graph and build installation graph
	dependencyGraph := NewDependencyGraph()
	if err := dependencyGraph.BuildInstallationGraph(i.context.Config); err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Get topological ordering from the dependency graph
	executionOrder, err := dependencyGraph.GetTopologicalOrder()
	if err != nil {
		return fmt.Errorf("failed to determine execution order: %w", err)
	}

	// Execute steps in topological order
	for _, stepName := range executionOrder {
		step, exists := i.steps[stepName]
		if !exists {
			return fmt.Errorf("step '%s' not found in available steps", stepName)
		}

		log.Printf("Executing step: %s", step.Name)

		// Execute the step
		if err := step.Execute(i.context); err != nil {
			return fmt.Errorf("execution failed for step %s: %w", step.Name, err)
		}

		// Run validation if defined (after execution)
		if step.Validate != nil {
			if err := step.Validate(i.context); err != nil {
				return fmt.Errorf("validation failed for step %s: %w", step.Name, err)
			}
		}

		// Mark step as completed
		i.context.Completed = append(i.context.Completed, step.Name)
		log.Printf("Completed step: %s", step.Name)
	}

	return nil
}

// GetInstallationSummary returns a summary of the installation
func (i *Installer) GetInstallationSummary() InstallationSummary {
	summary := InstallationSummary{
		TargetDir:        i.context.TargetDir,
		BackupDir:        i.context.BackupDir,
		CompletedSteps:   i.context.Completed,
		ExistingFiles:    *i.context.ExistingFiles,
		MCPConfigCreated: i.context.Config.AddRecommendedMCP,
	}

	if i.context.BackupManager != nil {
		summary.BackedUpFiles = make([]string, 0, len(i.context.BackupManager.Files))
		for original := range i.context.BackupManager.Files {
			summary.BackedUpFiles = append(summary.BackedUpFiles, original)
		}
	}

	return summary
}

// InstallationSummary provides information about what was installed
type InstallationSummary struct {
	TargetDir        string
	BackupDir        string
	CompletedSteps   []string
	BackedUpFiles    []string
	ExistingFiles    ExistingFiles
	MCPConfigCreated bool
}

// PrintSummary displays a human-readable installation summary
func (s *InstallationSummary) PrintSummary() {
	fmt.Printf("\n✅ SuperClaude installation completed successfully!\n\n")

	fmt.Printf("Installation directory: %s\n", s.TargetDir)

	if len(s.BackedUpFiles) > 0 {
		fmt.Printf("\nBacked up files to: %s\n", s.BackupDir)
		for _, file := range s.BackedUpFiles {
			fmt.Printf("  - %s\n", file)
		}
	}

	fmt.Printf("\nFiles created/modified:\n")

	if s.ExistingFiles.CLAUDEmd {
		fmt.Printf("  - CLAUDE.md (merged with SuperClaude import)\n")
	} else {
		fmt.Printf("  - CLAUDE.md (created)\n")
	}

	if s.MCPConfigCreated {
		if s.ExistingFiles.MCPConfig {
			fmt.Printf("  - .mcp.json (merged with recommended servers)\n")
		} else {
			fmt.Printf("  - .mcp.json (created with recommended servers)\n")
		}
	}

	fmt.Printf("  - .superclaude/ (framework files)\n")

	if !s.ExistingFiles.ClaudeDir {
		fmt.Printf("  - .claude/ (created)\n")
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Review CLAUDE.md to ensure imports are correct\n")
	fmt.Printf("2. Restart Claude Code to load new configuration\n")
	fmt.Printf("3. Use SuperClaude commands and features in Claude Code\n")
}

// GetContext returns the installation context (for accessing dry run mode, etc.)
func (i *Installer) GetContext() *InstallContext {
	return i.context
}

// Rollback attempts to restore files from backup
func (i *Installer) Rollback() error {
	if i.context.BackupManager == nil || len(i.context.BackupManager.Files) == 0 {
		return fmt.Errorf("no backup available for rollback")
	}

	log.Printf("Rolling back installation...")

	for original, backup := range i.context.BackupManager.Files {
		log.Printf("Restoring %s from %s", original, backup)

		if err := copyFile(backup, original); err != nil {
			return fmt.Errorf("failed to restore %s: %w", original, err)
		}
	}

	log.Printf("Rollback completed")
	return nil
}
