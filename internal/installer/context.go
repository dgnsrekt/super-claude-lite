package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InstallContext holds the state of the installation process
type InstallContext struct {
	TargetDir     string
	TempDir       string
	RepoPath      string
	BackupDir     string
	BackupManager *BackupManager
	Completed     []string
	Config        *InstallConfig
	ExistingFiles *ExistingFiles
	SkipClaudeDir bool
	DryRun        bool
}

// InstallConfig holds installation configuration options
type InstallConfig struct {
	Force             bool
	NoBackup          bool
	Interactive       bool
	AddRecommendedMCP bool
	BackupDir         string
}

// ExistingFiles tracks what files already exist before installation
type ExistingFiles struct {
	CLAUDEmd       bool
	MCPConfig      bool
	SuperClaudeDir bool
	ClaudeDir      bool
}

// BackupManager handles backing up existing files
type BackupManager struct {
	BackupDir string
	Files     map[string]string // original path -> backup path
}

// NewInstallContext creates a new installation context
func NewInstallContext(targetDir string, config *InstallConfig) (*InstallContext, error) {
	// Create backup directory if needed
	var backupDir string
	var backupManager *BackupManager

	if !config.NoBackup {
		timestamp := time.Now().Format("20060102-150405")
		backupDir = filepath.Join(targetDir, fmt.Sprintf(".superclaude-backup-%s", timestamp))

		if config.BackupDir != "" {
			backupDir = config.BackupDir
		}

		backupManager = &BackupManager{
			BackupDir: backupDir,
			Files:     make(map[string]string),
		}
	}

	ctx := &InstallContext{
		TargetDir:     targetDir,
		BackupDir:     backupDir,
		BackupManager: backupManager,
		Completed:     make([]string, 0),
		Config:        config,
		ExistingFiles: &ExistingFiles{},
	}

	return ctx, nil
}

// ScanExistingFiles checks what files already exist in the target directory
func (ctx *InstallContext) ScanExistingFiles() error {
	claudePath := filepath.Join(ctx.TargetDir, "CLAUDE.md")
	mcpPath := filepath.Join(ctx.TargetDir, ".mcp.json")
	superClaudePath := filepath.Join(ctx.TargetDir, ".superclaude")
	claudeDirPath := filepath.Join(ctx.TargetDir, ".claude")

	ctx.ExistingFiles.CLAUDEmd = fileExists(claudePath)
	ctx.ExistingFiles.MCPConfig = fileExists(mcpPath)
	ctx.ExistingFiles.SuperClaudeDir = fileExists(superClaudePath)
	ctx.ExistingFiles.ClaudeDir = fileExists(claudeDirPath)

	return nil
}

// CreateBackupDir creates the backup directory if it doesn't exist
func (bm *BackupManager) CreateBackupDir() error {
	if bm.BackupDir == "" {
		return nil
	}
	return os.MkdirAll(bm.BackupDir, 0755)
}

// BackupFile creates a backup of the specified file
func (bm *BackupManager) BackupFile(filePath string) error {
	if bm.BackupDir == "" {
		return nil // No backup configured
	}

	if !fileExists(filePath) {
		return nil // Nothing to backup
	}

	// Ensure backup directory exists
	if err := bm.CreateBackupDir(); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup path
	fileName := filepath.Base(filePath)
	backupPath := filepath.Join(bm.BackupDir, fileName)

	// Handle subdirectories (like .superclaude)
	if stat, err := os.Stat(filePath); err == nil && stat.IsDir() {
		return copyDir(filePath, backupPath)
	}

	// Copy file
	if err := copyFile(filePath, backupPath); err != nil {
		return fmt.Errorf("failed to backup %s: %w", filePath, err)
	}

	bm.Files[filePath] = backupPath
	return nil
}

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}
