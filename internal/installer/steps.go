package installer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgnsrekt/super-claude-lite/internal/config"
	"github.com/dgnsrekt/super-claude-lite/internal/git"
)

// InstallStep represents a single step in the installation process
type InstallStep struct {
	Name     string
	Execute  func(*InstallContext) error
	Validate func(*InstallContext) error
}

// GetInstallSteps returns all installation steps
func GetInstallSteps() map[string]*InstallStep {
	return map[string]*InstallStep{
		"CheckPrerequisites":       {Name: "CheckPrerequisites", Execute: checkPrerequisites, Validate: nil},
		"ScanExistingFiles":        {Name: "ScanExistingFiles", Execute: scanExistingFiles, Validate: nil},
		"CreateBackups":            {Name: "CreateBackups", Execute: createBackups, Validate: nil},
		"CheckTargetDirectory":     {Name: "CheckTargetDirectory", Execute: checkTargetDirectory, Validate: nil},
		"CloneRepository":          {Name: "CloneRepository", Execute: cloneRepository, Validate: validateRepoCloned},
		"CreateDirectoryStructure": {Name: "CreateDirectoryStructure", Execute: createDirectoryStructure, Validate: nil},
		"CopyCoreFiles":            {Name: "CopyCoreFiles", Execute: copyCoreFiles, Validate: validateCoreFiles},
		"CopyCommandFiles":         {Name: "CopyCommandFiles", Execute: copyCommandFiles, Validate: validateCommandFiles},
		"MergeOrCreateCLAUDEmd":    {Name: "MergeOrCreateCLAUDEmd", Execute: mergeOrCreateCLAUDEmd, Validate: nil},
		"MergeOrCreateMCPConfig":   {Name: "MergeOrCreateMCPConfig", Execute: mergeOrCreateMCPConfig, Validate: nil},
		"CreateCommandSymlink":     {Name: "CreateCommandSymlink", Execute: createCommandSymlink, Validate: nil},
		"ValidateInstallation":     {Name: "ValidateInstallation", Execute: validateInstallation, Validate: nil},
		"CleanupTempFiles":         {Name: "CleanupTempFiles", Execute: cleanupTempFiles, Validate: nil},
	}
}

func checkPrerequisites(ctx *InstallContext) error {
	// Check if git is installed
	if err := git.ValidateGitInstalled(); err != nil {
		return err
	}

	// Check if target directory is writable
	if err := checkWritePermissions(ctx.TargetDir); err != nil {
		return fmt.Errorf("target directory is not writable: %w", err)
	}

	return nil
}

func scanExistingFiles(ctx *InstallContext) error {
	return ctx.ScanExistingFiles()
}

func createBackups(ctx *InstallContext) error {
	if ctx.Config.NoBackup || ctx.BackupManager == nil {
		return nil
	}

	filesToBackup := []string{
		filepath.Join(ctx.TargetDir, config.CLAUDEFile),
		filepath.Join(ctx.TargetDir, config.MCPConfigFile),
		filepath.Join(ctx.TargetDir, config.SuperClaudeDir),
		filepath.Join(ctx.TargetDir, config.ClaudeDir),
	}

	for _, file := range filesToBackup {
		if err := ctx.BackupManager.BackupFile(file); err != nil {
			return fmt.Errorf("failed to backup %s: %w", file, err)
		}
	}

	return nil
}

func checkTargetDirectory(ctx *InstallContext) error {
	// Ensure target directory exists
	if err := os.MkdirAll(ctx.TargetDir, 0o750); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	return nil
}

func cloneRepository(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would clone repository to temp directory\n")
		return nil
	}

	tempDir, err := git.GetTempCloneDir()
	if err != nil {
		return err
	}

	ctx.TempDir = tempDir
	ctx.RepoPath = tempDir

	return git.CloneRepository(tempDir)
}

func createDirectoryStructure(ctx *InstallContext) error {
	dirs := []string{
		filepath.Join(ctx.TargetDir, config.SuperClaudeDir),
		filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Commands"),
	}

	// Create .claude directory only if it doesn't exist or is empty
	claudeDir := filepath.Join(ctx.TargetDir, config.ClaudeDir)
	if !ctx.SkipClaudeDir && !ctx.ExistingFiles.ClaudeDir {
		// Also create commands directory for Claude Code integration
		dirs = append(dirs, claudeDir, filepath.Join(claudeDir, "commands"))
	}

	for _, dir := range dirs {
		if ctx.DryRun {
			fmt.Printf("[DRY RUN] Would create directory: %s\n", dir)
			continue
		}

		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func copyCoreFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would copy core files from %s\n", config.CoreSourcePath)
		return nil
	}

	corePath, _ := git.GetSourcePaths(ctx.RepoPath)
	targetPath := filepath.Join(ctx.TargetDir, config.SuperClaudeDir)

	return copyMarkdownFiles(corePath, targetPath)
}

func copyCommandFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would copy command files from %s\n", config.CommandsSourcePath)
		return nil
	}

	_, commandsPath := git.GetSourcePaths(ctx.RepoPath)
	targetPath := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Commands")

	return copyMarkdownFiles(commandsPath, targetPath)
}

func mergeOrCreateCLAUDEmd(ctx *InstallContext) error {
	claudePath := filepath.Join(ctx.TargetDir, config.CLAUDEFile)

	if ctx.DryRun {
		if ctx.ExistingFiles.CLAUDEmd {
			fmt.Printf("[DRY RUN] Would merge SuperClaude import into existing CLAUDE.md\n")
		} else {
			fmt.Printf("[DRY RUN] Would create new CLAUDE.md\n")
		}
		return nil
	}

	if ctx.ExistingFiles.CLAUDEmd {
		return mergeCLAUDEmd(claudePath)
	}

	return createCLAUDEmd(claudePath)
}

func mergeOrCreateMCPConfig(ctx *InstallContext) error {
	// Only create/modify MCP config if the --add-mcp flag was used
	if !ctx.Config.AddRecommendedMCP {
		return nil
	}

	mcpPath := filepath.Join(ctx.TargetDir, config.MCPConfigFile)

	if ctx.DryRun {
		if ctx.ExistingFiles.MCPConfig {
			fmt.Printf("[DRY RUN] Would merge MCP configuration with recommended servers\n")
		} else {
			fmt.Printf("[DRY RUN] Would create new .mcp.json with recommended servers\n")
		}
		return nil
	}

	if ctx.ExistingFiles.MCPConfig {
		return mergeMCPConfig(mcpPath, ctx.Config.AddRecommendedMCP)
	}

	return createMCPConfigWithRecommended(mcpPath)
}

func createCommandSymlink(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would create symlink from .superclaude/Commands to .claude/commands/sc\n")
		return nil
	}

	targetPath := filepath.Join(ctx.TargetDir, config.ClaudeDir, "commands", "sc")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create symlink using relative path for portability
	relPath := "../../.superclaude/Commands"
	if err := os.Symlink(relPath, targetPath); err != nil {
		return fmt.Errorf("failed to create command symlink: %w", err)
	}

	return nil
}

func validateInstallation(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would validate installation files\n")
		return nil
	}

	// Check that core files exist
	requiredFiles := []string{
		filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "CLAUDE.md"),
		filepath.Join(ctx.TargetDir, config.CLAUDEFile),
	}

	for _, file := range requiredFiles {
		if !fileExists(file) {
			return fmt.Errorf("required file missing: %s", file)
		}
	}

	// Check that command symlink exists if .claude directory was created
	if !ctx.SkipClaudeDir && !ctx.ExistingFiles.ClaudeDir {
		symlinkPath := filepath.Join(ctx.TargetDir, config.ClaudeDir, "commands", "sc")
		if _, err := os.Lstat(symlinkPath); err != nil {
			return fmt.Errorf("command symlink missing: %s", symlinkPath)
		}
	}

	// Check that .mcp.json exists only if --add-mcp flag was used
	if ctx.Config.AddRecommendedMCP {
		mcpPath := filepath.Join(ctx.TargetDir, config.MCPConfigFile)
		if !fileExists(mcpPath) {
			return fmt.Errorf("MCP config file missing (expected due to --add-mcp flag): %s", mcpPath)
		}
	}

	return nil
}

func cleanupTempFiles(ctx *InstallContext) error {
	if ctx.TempDir != "" {
		if ctx.DryRun {
			fmt.Printf("[DRY RUN] Would cleanup temp directory: %s\n", ctx.TempDir)
			return nil
		}
		return git.CleanupTempDir(ctx.TempDir)
	}
	return nil
}

// Validation functions
func validateRepoCloned(ctx *InstallContext) error {
	if ctx.DryRun {
		return nil
	}

	// This validation runs after the cloneRepository step has executed,
	// so we check the result rather than pre-conditions
	if ctx.RepoPath == "" {
		return fmt.Errorf("repository path not set after cloning")
	}

	corePath, commandsPath := git.GetSourcePaths(ctx.RepoPath)

	if !fileExists(corePath) {
		return fmt.Errorf("core source directory not found: %s", corePath)
	}

	if !fileExists(commandsPath) {
		return fmt.Errorf("commands source directory not found: %s", commandsPath)
	}

	return nil
}

func validateCoreFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		return nil
	}

	coreDir := filepath.Join(ctx.TargetDir, config.SuperClaudeDir)
	expectedFiles := []string{"CLAUDE.md", "COMMANDS.md", "FLAGS.md", "PRINCIPLES.md", "RULES.md"}

	for _, file := range expectedFiles {
		filePath := filepath.Join(coreDir, file)
		if !fileExists(filePath) {
			return fmt.Errorf("core file missing: %s", filePath)
		}
	}

	return nil
}

func validateCommandFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		return nil
	}

	commandsDir := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Commands")

	// Check that at least some command files exist
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return fmt.Errorf("failed to read commands directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no command files found in %s", commandsDir)
	}

	return nil
}

// Helper functions
func checkWritePermissions(dir string) error {
	testFile := filepath.Join(dir, ".write-test")
	file, err := os.Create(testFile)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		log.Printf("failed to close test file: %v", err)
	}
	return os.Remove(testFile)
}

func copyMarkdownFiles(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only copy markdown files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil // Skip non-markdown files
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
			return err
		}

		return copyFile(path, dstPath)
	})
}

func mergeCLAUDEmd(claudePath string) error {
	content, err := os.ReadFile(claudePath)
	if err != nil {
		return fmt.Errorf("failed to read existing CLAUDE.md: %w", err)
	}

	contentStr := string(content)

	// Check if SuperClaude import already exists
	if strings.Contains(contentStr, "@./.superclaude/CLAUDE.md") {
		return nil // Already imported
	}

	// Append SuperClaude section
	newContent := contentStr + "\n\n" + config.SuperClaudeImport + "\n"

	return os.WriteFile(claudePath, []byte(newContent), 0o600)
}

func createCLAUDEmd(claudePath string) error {
	content := `# Claude Code Instructions

` + config.SuperClaudeImport + "\n"

	return os.WriteFile(claudePath, []byte(content), 0o600)
}

func mergeMCPConfig(mcpPath string, addRecommended bool) error {
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		return fmt.Errorf("failed to read existing .mcp.json: %w", err)
	}

	var existing map[string]interface{}
	if err := json.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("failed to parse existing .mcp.json: %w", err)
	}

	// Ensure mcpServers exists
	if _, ok := existing["mcpServers"]; !ok {
		existing["mcpServers"] = make(map[string]interface{})
	}

	// Add recommended servers if requested
	if addRecommended {
		servers := existing["mcpServers"].(map[string]interface{})

		for name, serverConfig := range config.RecommendedMCPServers {
			if _, exists := servers[name]; !exists {
				servers[name] = serverConfig
			}
		}
	}

	// Write merged config
	output, err := json.MarshalIndent(existing, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal .mcp.json: %w", err)
	}

	return os.WriteFile(mcpPath, output, 0o600)
}

func createMCPConfigWithRecommended(mcpPath string) error {
	mcpConfig := map[string]interface{}{
		"mcpServers": config.RecommendedMCPServers,
	}

	output, err := json.MarshalIndent(mcpConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal .mcp.json: %w", err)
	}

	return os.WriteFile(mcpPath, output, 0o600)
}
