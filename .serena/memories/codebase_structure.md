# Super Claude Lite Codebase Structure

## Directory Layout
```
super-claude-lite/
├── cmd/
│   └── super-claude-lite/
│       └── main.go              # Main entry point, CLI commands
├── internal/                    # Private packages
│   ├── installer/
│   │   ├── installer.go         # Core installation logic
│   │   ├── context.go           # Installation context management
│   │   └── steps.go             # Installation step definitions
│   ├── git/
│   │   └── operations.go        # Git operations (clone, checkout)
│   └── config/
│       └── constants.go         # Configuration constants
├── docs/                        # Documentation directory
├── bin/                         # Built binaries (created by make)
├── Makefile                     # Build automation
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── .golangci.yml               # Linter configuration
├── .pre-commit-config.yaml     # Pre-commit hooks configuration
├── demo.tape                   # VHS demo script
├── demo.gif                    # Generated demo animation
├── README.md                   # Project documentation
├── LICENSE                     # MIT license
├── CLAUDE.md                   # Claude Code instructions
└── .mcp.json                   # MCP server configuration
```

## Key Source Files

### Main Entry Point
- **cmd/super-claude-lite/main.go**: 
  - CLI command definitions using Cobra
  - Commands: init, status, clean, rollback
  - Uses Charmbracelet Fang for enhanced CLI experience

### Core Logic
- **internal/installer/installer.go**: 
  - `Installer` struct and methods
  - `NewInstaller()` constructor
  - `Install()`, `Rollback()` methods
  - Installation summary functionality

- **internal/installer/context.go**:
  - Installation context management
  - Target directory and configuration handling

- **internal/installer/steps.go**:
  - Step-by-step installation process
  - Dependency-aware installation logic

### Supporting Modules
- **internal/git/operations.go**:
  - Git repository operations
  - Clone and checkout functionality

- **internal/config/constants.go**:
  - Repository URLs and commit hashes
  - Directory and file name constants
  - Default MCP server configurations

## Key Constants and Configuration

### Repository Information
- **SuperClaude Framework**: https://github.com/SuperClaude-Org/SuperClaude_Framework.git
- **Fixed Commit**: 8f12b19a220aabf23680ee86b6d6ab21c46ba6f6
- **Branch**: master

### Directory Structure Created
- **.superclaude/** - Framework core files
- **.claude/** - Claude Code configuration
- **.mcp.json** - MCP server configuration
- **CLAUDE.md** - Claude instructions

### MCP Servers (when --add-mcp used)
- sequential-thinking
- context7  
- serena
- playwright

## Build Artifacts
- **bin/super-claude-lite** - Built binary
- **demo.gif** - Generated demo animation
- **test-run/** - Temporary directory for testing (cleaned up)

## No Test Files
The project currently has no `*_test.go` files, indicating tests should be added for proper development practices.

## Integration Points
- Integrates with SuperClaude Framework repository
- Configures Claude Code via .claude/ directory
- Sets up MCP servers for enhanced AI functionality
- Uses git for repository operations
- Creates symlinks for slash command integration