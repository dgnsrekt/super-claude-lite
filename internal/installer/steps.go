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

// selectMCPServers is a function variable that can be overridden for testing
var selectMCPServers = func(servers []MCPServer) ([]MCPServer, error) {
	return ShowMCPSelector(servers)
}

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
		"CopyAgentFiles":           {Name: "CopyAgentFiles", Execute: copyAgentFiles, Validate: validateAgentFiles},
		"CopyModeFiles":            {Name: "CopyModeFiles", Execute: copyModeFiles, Validate: validateModeFiles},
		"CopyMCPFiles":             {Name: "CopyMCPFiles", Execute: copyMCPFiles, Validate: nil},
		"MergeOrCreateCLAUDEmd":    {Name: "MergeOrCreateCLAUDEmd", Execute: mergeOrCreateCLAUDEmd, Validate: nil},
		"MergeOrCreateMCPConfig":   {Name: "MergeOrCreateMCPConfig", Execute: mergeOrCreateMCPConfig, Validate: nil},
		"CreateCommandSymlink":     {Name: "CreateCommandSymlink", Execute: createCommandSymlink, Validate: nil},
		"CreateAgentSymlink":       {Name: "CreateAgentSymlink", Execute: createAgentSymlink, Validate: nil},
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
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would create backups if needed\n")
		return nil
	}

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

	// Create .claude directory and subdirectories as needed
	claudeDir := filepath.Join(ctx.TargetDir, config.ClaudeDir)
	if !ctx.SkipClaudeDir {
		if !ctx.ExistingFiles.ClaudeDir {
			// Create the .claude directory itself
			dirs = append(dirs, claudeDir)
		}
		// Always ensure commands and agents subdirectories exist for Claude Code integration
		dirs = append(dirs, filepath.Join(claudeDir, "commands"), filepath.Join(claudeDir, "agents"))
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

	if err := copyMarkdownFiles(corePath, targetPath); err != nil {
		return err
	}

	// Create CLAUDE.md file with v4 import structure
	claudePath := filepath.Join(targetPath, "CLAUDE.md")
	if !fileExists(claudePath) {
		claudeContent := `# The superclaude CLAUDE.md file uses an import system to load multiple context files:

*MANDATORY*
@FLAGS.md # Flag definitions and triggers
@RULES.md # Core behavioral rules
@PRINCIPLES.md # Guiding principles
*CRITICAL*
@Modes/MODE_Brainstorming.md # Collaborative discovery mode
@Modes/MODE_Introspection.md # Transparent reasoning mode
@Modes/MODE_Task_Management.md # Task orchestration mode
@Modes/MODE_Orchestration.md # Tool coordination mode
@Modes/MODE_Token_Efficiency.md # Compressed communication mode
`
		if err := os.WriteFile(claudePath, []byte(claudeContent), 0644); err != nil {
			return fmt.Errorf("failed to create CLAUDE.md: %w", err)
		}
	}

	return nil
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

func copyAgentFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would copy agent files from %s\n", config.AgentsSourcePath)
		return nil
	}

	agentsPath := filepath.Join(ctx.RepoPath, config.AgentsSourcePath)
	targetPath := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Agents")

	return copyMarkdownFiles(agentsPath, targetPath)
}

func copyModeFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would copy mode files from %s\n", config.ModesSourcePath)
		return nil
	}

	modesPath := filepath.Join(ctx.RepoPath, config.ModesSourcePath)
	targetPath := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Modes")

	return copyMarkdownFiles(modesPath, targetPath)
}

func copyMCPFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would copy selected MCP files\n")
		return nil
	}

	// Only copy MCP files if user requested MCP support
	if !ctx.Config.AddRecommendedMCP {
		return nil
	}

	// Discover available MCP servers
	servers, err := DiscoverMCPServers(ctx.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to discover MCP servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Printf("No MCP servers found, skipping MCP installation\n")
		return nil
	}

	// Show TUI for server selection
	fmt.Printf("Select MCP servers to install:\n")
	selectedServers, err := selectMCPServers(servers)
	if err != nil {
		return fmt.Errorf("failed to select MCP servers: %w", err)
	}

	if len(selectedServers) == 0 {
		fmt.Printf("No MCP servers selected, skipping MCP installation\n")
		return nil
	}

	// Store selected servers in context for later use
	ctx.SelectedMCPServers = selectedServers

	// Create MCP target directory
	mcpTargetDir := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "MCP")
	if err := os.MkdirAll(mcpTargetDir, 0750); err != nil {
		return fmt.Errorf("failed to create MCP directory: %w", err)
	}

	// Copy selected MCP files
	mcpSourceDir := filepath.Join(ctx.RepoPath, "SuperClaude", "MCP")
	for _, server := range selectedServers {
		srcFile := filepath.Join(mcpSourceDir, server.MDFile)
		dstFile := filepath.Join(mcpTargetDir, server.MDFile)

		if err := copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("failed to copy MCP file %s: %w", server.MDFile, err)
		}
	}

	fmt.Printf("Copied %d MCP server files\n", len(selectedServers))
	return nil
}

func mergeOrCreateCLAUDEmd(ctx *InstallContext) error {
	// Main project CLAUDE.md (imports from .superclaude)
	mainClaudePath := filepath.Join(ctx.TargetDir, config.CLAUDEFile)
	// SuperClaude internal CLAUDE.md (gets MCP imports added)
	superClaudePath := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, config.CLAUDEFile)

	if ctx.DryRun {
		if ctx.ExistingFiles.CLAUDEmd {
			fmt.Printf("[DRY RUN] Would merge SuperClaude import into existing CLAUDE.md\n")
		} else {
			fmt.Printf("[DRY RUN] Would create new CLAUDE.md\n")
		}

		if len(ctx.SelectedMCPServers) > 0 {
			fmt.Printf("[DRY RUN] Would add MCP imports to .superclaude/CLAUDE.md for %d selected servers\n", len(ctx.SelectedMCPServers))
		}
		return nil
	}

	// Handle main project CLAUDE.md
	if ctx.ExistingFiles.CLAUDEmd {
		if err := mergeCLAUDEmd(mainClaudePath); err != nil { // No MCP imports in main file
			return err
		}
	} else {
		if err := createCLAUDEmd(mainClaudePath); err != nil { // No MCP imports in main file
			return err
		}
	}

	// Handle .superclaude/CLAUDE.md (add MCP imports here)
	return updateSuperClaudeMCPImports(superClaudePath, ctx.SelectedMCPServers)
}

func updateSuperClaudeMCPImports(superClaudePath string, selectedMCPServers []MCPServer) error {
	// Read existing .superclaude/CLAUDE.md
	content, err := os.ReadFile(superClaudePath)
	if err != nil {
		return fmt.Errorf("failed to read .superclaude/CLAUDE.md: %w", err)
	}

	contentStr := string(content)

	// Remove any existing MCP import section first
	contentStr = removeMCPImportsSection(contentStr)

	// Add MCP imports if any servers were selected
	if len(selectedMCPServers) > 0 {
		mcpSection := "\n*MCP_INTEGRATIONS*\n"
		for _, server := range selectedMCPServers {
			mcpSection += fmt.Sprintf("@MCP/%s\n", server.MDFile)
		}
		contentStr += mcpSection
	}

	// Write updated content back
	if err := os.WriteFile(superClaudePath, []byte(contentStr), 0o600); err != nil {
		return fmt.Errorf("failed to update .superclaude/CLAUDE.md: %w", err)
	}

	if len(selectedMCPServers) > 0 {
		fmt.Printf("Added MCP imports to .superclaude/CLAUDE.md for %d selected servers\n", len(selectedMCPServers))
	}

	return nil
}

func removeMCPImportsSection(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inMCPSection := false

	for _, line := range lines {
		if strings.HasPrefix(line, "*MCP_INTEGRATIONS*") {
			inMCPSection = true
			continue
		}

		// If we're in MCP section and hit a line that starts with @ and contains MCP/, skip it
		if inMCPSection && strings.HasPrefix(line, "@") && strings.Contains(line, "MCP/") {
			continue
		}

		// If we're in MCP section and hit an empty line or another section, exit MCP section
		if inMCPSection && (strings.TrimSpace(line) == "" || strings.HasPrefix(line, "*")) {
			inMCPSection = false
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
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
		return mergeMCPConfig(mcpPath, ctx.Config.AddRecommendedMCP, ctx.SelectedMCPServers, ctx.RepoPath)
	}

	return createMCPConfigWithSelected(mcpPath, ctx.SelectedMCPServers, ctx.RepoPath)
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

func createAgentSymlink(ctx *InstallContext) error {
	if ctx.DryRun {
		fmt.Printf("[DRY RUN] Would create symlink from .superclaude/Agents to .claude/agents/sc\n")
		return nil
	}

	targetPath := filepath.Join(ctx.TargetDir, config.ClaudeDir, "agents", "sc")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove existing agent symlink: %w", err)
		}
	}

	// Create symlink using relative path for portability
	relPath := "../../.superclaude/Agents"
	if err := os.Symlink(relPath, targetPath); err != nil {
		return fmt.Errorf("failed to create agent symlink: %w", err)
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
	expectedFiles := []string{"CLAUDE.md", "FLAGS.md", "PRINCIPLES.md", "RULES.md"}

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

func validateAgentFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		return nil
	}

	agentsDir := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Agents")

	// Check that agents directory exists
	if _, err := os.Stat(agentsDir); err != nil {
		return fmt.Errorf("agents directory missing: %s", agentsDir)
	}

	// Check that at least some agent files exist
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return fmt.Errorf("failed to read agents directory: %w", err)
	}

	mdCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			mdCount++
		}
	}

	if mdCount == 0 {
		return fmt.Errorf("no agent files found in %s", agentsDir)
	}

	return nil
}

func validateModeFiles(ctx *InstallContext) error {
	if ctx.DryRun {
		return nil
	}

	modesDir := filepath.Join(ctx.TargetDir, config.SuperClaudeDir, "Modes")

	// Check that modes directory exists
	if _, err := os.Stat(modesDir); err != nil {
		return fmt.Errorf("modes directory missing: %s", modesDir)
	}

	// Check that at least some mode files exist
	entries, err := os.ReadDir(modesDir)
	if err != nil {
		return fmt.Errorf("failed to read modes directory: %w", err)
	}

	mdCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			mdCount++
		}
	}

	if mdCount == 0 {
		return fmt.Errorf("no mode files found in %s", modesDir)
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

func mergeMCPConfig(mcpPath string, addRecommended bool, selectedServers []MCPServer, repoPath string) error {
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

	// Add selected servers if requested
	if addRecommended && len(selectedServers) > 0 {
		servers := existing["mcpServers"].(map[string]interface{})

		for _, mcpServer := range selectedServers {
			serverConfig, err := LoadMCPConfig(repoPath, mcpServer.ConfigFile)
			if err != nil {
				return fmt.Errorf("failed to load MCP config for %s: %w", mcpServer.Name, err)
			}

			// Merge the loaded config, but don't overwrite existing ones
			for key, value := range serverConfig {
				if _, exists := servers[key]; !exists {
					servers[key] = value
				}
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

func createMCPConfigWithSelected(mcpPath string, selectedServers []MCPServer, repoPath string) error {
	mcpServers := make(map[string]interface{})

	// Add only the selected servers by loading their config files
	for _, mcpServer := range selectedServers {
		serverConfig, err := LoadMCPConfig(repoPath, mcpServer.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load MCP config for %s: %w", mcpServer.Name, err)
		}

		// Merge the loaded config into mcpServers
		for key, value := range serverConfig {
			mcpServers[key] = value
		}
	}

	mcpConfig := map[string]interface{}{
		"mcpServers": mcpServers,
	}

	output, err := json.MarshalIndent(mcpConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal .mcp.json: %w", err)
	}

	return os.WriteFile(mcpPath, output, 0o600)
}
