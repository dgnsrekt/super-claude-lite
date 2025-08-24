package config

const (
	// Repository information
	RepoURL     = "https://github.com/SuperClaude-Org/SuperClaude_Framework.git"
	FixedCommit = "13aa2ec"
	Branch      = "master"

	// Directory names
	SuperClaudeDir = ".superclaude"
	ClaudeDir      = ".claude"
	MCPConfigFile  = ".mcp.json"
	CLAUDEFile     = "CLAUDE.md"

	// Framework paths within the repository
	CoreSourcePath     = "SuperClaude/Core"
	CommandsSourcePath = "SuperClaude/Commands"
	AgentsSourcePath   = "SuperClaude/Agents"
	ModesSourcePath    = "SuperClaude/Modes"

	// Backup directory prefix
	BackupDirPrefix = ".superclaude-backup"
)

// SuperClaude import directive for CLAUDE.md
const SuperClaudeImport = `## SuperClaude Instructions

**Import SuperClaude Core, treat as if import is in the main CLAUDE.md file.**
@./.superclaude/CLAUDE.md`

// Default MCP servers to recommend
var RecommendedMCPServers = map[string]interface{}{
	"sequential-thinking": map[string]interface{}{
		"command": "npx",
		"args":    []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
	},
	"context7": map[string]interface{}{
		"command": "npx",
		"args":    []string{"-y", "@upstash/context7-mcp@latest"},
	},
	"serena": map[string]interface{}{
		"command": "uvx",
		"args":    []string{"--from", "git+https://github.com/oraios/serena", "serena", "start-mcp-server"},
	},
	"playwright": map[string]interface{}{
		"command": "npx",
		"args":    []string{"@playwright/mcp@latest"},
	},
}
