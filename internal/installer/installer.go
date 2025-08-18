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

// Install executes the installation process
func (i *Installer) Install() error {
	log.Printf("Starting SuperClaude installation")

	// Get topological ordering using manual traversal
	executed := make(map[string]bool)
	var installationError error

	// Continue until all steps are executed or we hit an error
	for len(executed) < len(i.steps) && installationError == nil {
		progress := false

		// Find steps that can be executed (all dependencies satisfied)
		for stepName, step := range i.steps {
			if executed[stepName] {
				continue // Already executed
			}

			// Check if all dependencies are satisfied
			canExecute := true
			dependencies := i.getDependencies(stepName)
			for _, dep := range dependencies {
				if !executed[dep] {
					canExecute = false
					break
				}
			}

			// Skip logging individual dependency waits to reduce noise

			if canExecute {
				log.Printf("Executing step: %s", step.Name)

				// Execute the step
				if err := step.Execute(i.context); err != nil {
					installationError = fmt.Errorf("execution failed for step %s: %w", step.Name, err)
					break
				}

				// Run validation if defined (after execution)
				if step.Validate != nil {
					if err := step.Validate(i.context); err != nil {
						installationError = fmt.Errorf("validation failed for step %s: %w", step.Name, err)
						break
					}
				}

				// Mark step as completed
				executed[stepName] = true
				i.context.Completed = append(i.context.Completed, step.Name)
				log.Printf("Completed step: %s", step.Name)
				progress = true
			}
		}

		// If we didn't make progress, we have a circular dependency or missing step
		if !progress {
			installationError = fmt.Errorf("cannot resolve dependencies - possible circular dependency")
			break
		}
	}

	return installationError
}

// getDependencies returns the dependencies for a given step
func (i *Installer) getDependencies(stepName string) []string {
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
	}

	// ValidateInstallation dependencies change based on whether MCP config is enabled
	if stepName == "ValidateInstallation" {
		validateDeps := []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink"}
		if i.context.Config.AddRecommendedMCP {
			validateDeps = append(validateDeps, "MergeOrCreateMCPConfig")
		}
		return validateDeps
	}

	// CleanupTempFiles runs after everything else including validation
	if stepName == "CleanupTempFiles" {
		cleanupDeps := []string{"CopyCoreFiles", "CopyCommandFiles", "MergeOrCreateCLAUDEmd", "CreateCommandSymlink", "ValidateInstallation"}
		if i.context.Config.AddRecommendedMCP {
			cleanupDeps = append(cleanupDeps, "MergeOrCreateMCPConfig")
		}
		return cleanupDeps
	}

	if deps, ok := dependencies[stepName]; ok {
		return deps
	}
	return []string{}
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
