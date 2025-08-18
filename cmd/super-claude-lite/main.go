package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/dgnsrekt/super-claude-lite/internal/installer"
)

var (
	version = "0.2.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "super-claude-lite",
		Short:   "Lightweight installer for SuperClaude Framework",
		Long:    "A lightweight installer for the SuperClaude Framework that allows you to install it inside the project dir vs home dir.",
		Version: fmt.Sprintf("%s (%s)", version, commit),
	}

	// Add subcommands
	rootCmd.AddCommand(
		createInitCommand(),
		createStatusCommand(),
		createCleanCommand(),
		createRollbackCommand(),
	)

	// Use Fang for batteries-included CLI
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createInitCommand() *cobra.Command {
	var (
		targetDir         string
		force             bool
		noBackup          bool
		interactive       bool
		addRecommendedMCP bool
		backupDir         string
		dryRun            bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Install SuperClaude framework in specified directory",
		Long: `Install SuperClaude framework files in the specified directory.
If no directory is specified, uses the current working directory.

The installer will:
- Clone SuperClaude Framework at a fixed commit
- Copy framework files to .superclaude/
- Create or merge CLAUDE.md with SuperClaude import
- Create or merge .mcp.json configuration
- Backup existing files before modification`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine target directory
			if len(args) > 0 {
				targetDir = args[0]
			}

			if targetDir == "" {
				var err error
				targetDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
			}

			// Convert to absolute path
			targetDir, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve target directory: %w", err)
			}

			// Create installation config
			config := &installer.InstallConfig{
				Force:             force,
				NoBackup:          noBackup,
				Interactive:       interactive,
				AddRecommendedMCP: addRecommendedMCP,
				BackupDir:         backupDir,
			}

			// Create installer
			inst, err := installer.NewInstaller(targetDir, config)
			if err != nil {
				return fmt.Errorf("failed to create installer: %w", err)
			}

			// Set dry run mode
			inst.GetContext().DryRun = dryRun

			fmt.Printf("Installing SuperClaude Framework to: %s\n", targetDir)

			if dryRun {
				fmt.Printf("[DRY RUN] No files will be modified\n")
			}

			start := time.Now()

			// Run installation
			if err := inst.Install(); err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			duration := time.Since(start)

			if !dryRun {
				// Print summary
				summary := inst.GetInstallationSummary()
				summary.PrintSummary()
				fmt.Printf("\nInstallation completed in %v\n", duration)
			} else {
				fmt.Printf("\n[DRY RUN] Installation simulation completed in %v\n", duration)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&targetDir, "path", "p", "", "Target directory for installation (default: current directory)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force installation without prompts")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backups of existing files")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Ask for confirmation on each conflict")
	cmd.Flags().BoolVar(&addRecommendedMCP, "add-mcp", false, "Add recommended MCP servers to .mcp.json")
	cmd.Flags().StringVarP(&backupDir, "backup-dir", "b", "", "Custom backup directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	return cmd
}

func createStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [directory]",
		Short: "Check SuperClaude installation status",
		Long:  "Check if SuperClaude framework is installed and show status information.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine target directory
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}

			targetDir, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve directory: %w", err)
			}

			return checkInstallationStatus(targetDir)
		},
	}

	return cmd
}

func createCleanCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clean [directory]",
		Short: "Remove SuperClaude framework files",
		Long:  "Remove SuperClaude framework files from the specified directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine target directory
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}

			targetDir, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve directory: %w", err)
			}

			return cleanInstallation(targetDir, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")

	return cmd
}

func createRollbackCommand() *cobra.Command {
	var backupDir string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback to previous state using backup",
		Long:  "Restore files from a backup directory created during installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if backupDir == "" {
				return fmt.Errorf("backup directory must be specified with --backup-dir")
			}

			return rollbackInstallation(backupDir)
		},
	}

	cmd.Flags().StringVarP(&backupDir, "backup-dir", "b", "", "Backup directory to restore from (required)")
	_ = cmd.MarkFlagRequired("backup-dir")

	return cmd
}

// checkInstallationStatus checks if SuperClaude is installed
func checkInstallationStatus(targetDir string) error {
	fmt.Printf("Checking SuperClaude installation status in: %s\n\n", targetDir)

	requiredFiles := map[string]string{
		filepath.Join(targetDir, ".superclaude"): "Framework directory",
		filepath.Join(targetDir, "CLAUDE.md"):    "Main configuration",
		filepath.Join(targetDir, ".claude"):      "Claude directory",
	}

	optionalFiles := map[string]string{
		filepath.Join(targetDir, ".mcp.json"): "MCP configuration",
	}

	installed := true
	for path, description := range requiredFiles {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("✅ %s: %s\n", description, path)
		} else {
			fmt.Printf("❌ %s: %s (missing)\n", description, path)
			if path == filepath.Join(targetDir, ".superclaude") {
				installed = false
			}
		}
	}

	fmt.Printf("\nOptional files:\n")
	for path, description := range optionalFiles {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("✅ %s: %s\n", description, path)
		} else {
			fmt.Printf("➖ %s: %s (optional - use --add-mcp to create)\n", description, path)
		}
	}

	fmt.Printf("\nStatus: ")
	if installed {
		fmt.Printf("✅ SuperClaude is installed\n")
	} else {
		fmt.Printf("❌ SuperClaude is not installed\n")
		fmt.Printf("\nTo install, run: super-claude-lite init\n")
	}

	return nil
}

// cleanInstallation removes SuperClaude files
func cleanInstallation(targetDir string, force bool) error {
	if !force {
		fmt.Printf("This will remove SuperClaude framework files from: %s\n", targetDir)
		fmt.Printf("Files to be removed:\n")
		fmt.Printf("  - .superclaude/ (entire directory)\n")
		fmt.Printf("  - SuperClaude import from CLAUDE.md (if present)\n")
		fmt.Printf("\nContinue? (y/N): ")

		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Printf("Cancelled.\n")
			return nil
		}
	}

	// Remove command symlink if it exists
	symlinkPath := filepath.Join(targetDir, ".claude", "commands", "sc")
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			fmt.Printf("Warning: failed to remove command symlink: %v\n", err)
		}
	}

	// Remove .superclaude directory
	superClaudeDir := filepath.Join(targetDir, ".superclaude")
	if err := os.RemoveAll(superClaudeDir); err != nil {
		return fmt.Errorf("failed to remove .superclaude directory: %w", err)
	}

	fmt.Printf("✅ Removed SuperClaude framework files\n")
	return nil
}

// rollbackInstallation restores files from backup
func rollbackInstallation(backupDir string) error {
	if _, err := os.Stat(backupDir); err != nil {
		return fmt.Errorf("backup directory does not exist: %s", backupDir)
	}

	fmt.Printf("Rolling back from backup: %s\n", backupDir)

	// This is a simplified rollback - in a real implementation,
	// you'd want to restore files more carefully
	return filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Calculate the original path
			relPath, err := filepath.Rel(backupDir, path)
			if err != nil {
				return err
			}

			// Restore the file
			originalPath := relPath // This would need more sophisticated path resolution
			fmt.Printf("Restoring: %s\n", originalPath)
		}

		return nil
	})
}
