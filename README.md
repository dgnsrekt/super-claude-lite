# Super Claude Lite

```
   _____                          ________                __
  / ___/__  ______  ___  _____   / ____/ /___ ___  ______/ /__
  \__ \/ / / / __ \/ _ \/ ___/  / /   / / __ `/ / / / __  / _ \
 ___/ / /_/ / /_/ /  __/ /     / /___/ / /_/ / /_/ / /_/ /  __/
/____/\__,_/ .___/\___/_/      \____/_/\__,_/\__,_/\__,_/\___/
    __    /_/_
   / /   (_) /____
  / /   / / __/ _ \
 / /___/ / /_/  __/
/_____/_/\__/\___/

```

A lightweight installer for the [SuperClaude Framework](https://github.com/SuperClaude-Org/SuperClaude_Framework) that allows you to install it inside the project dir vs home dir.

## Demo

![Demo](demo.gif)

## Quick Start

```bash
# Install the tool
go install github.com/dgnsrekt/super-claude-lite/cmd/super-claude-lite@latest

# Check installation status
super-claude-lite status

# Interactive installation with MCP server selection
super-claude-lite init --add-mcp

# Basic installation (framework only)
super-claude-lite init

```

## Commands

- `init` - Install SuperClaude framework files
- `status` - Check installation status
- `clean` - Remove installed files
- `rollback` - Restore from backup

## Features

### Core Installation
- Install SuperClaude Framework v4 at any location (not forced to home directory)
- DAG-based dependency resolution for reliable installation order
- Automatic backup and merge of existing files
- Dry-run support for safe testing

### SuperClaude Framework Integration
- Installs SuperClaude Framework v4 with all components
- See [SuperClaude Framework docs](https://github.com/SuperClaude-Org/SuperClaude_Framework) for framework details

### MCP Server Selection
- **Interactive TUI**: Keyboard-navigable interface for server selection
- **Smart Integration**: Automatic `.mcp.json` configuration merging
- **Framework Integration**: MCP imports added to SuperClaude's internal CLAUDE.md
- **Selective Installation**: Choose only the MCP servers you need

### Claude Code Integration  
- Symlink integration with Claude Code slash commands
- Automatic CLAUDE.md import configuration
- MCP server auto-configuration for immediate use

## MCP Server Selection

The `--add-mcp` flag launches an interactive terminal interface where you can select from available MCP servers:

### Available Servers
Available MCP servers vary by SuperClaude Framework version. The interactive TUI will show all servers available in your installation.

### Interactive Selection
```bash
super-claude-lite init --add-mcp
```

Use **↑/↓** or **j/k** to navigate, **Space** to toggle selection, **Enter** to confirm.

Selected servers are automatically:
- Added to your `.mcp.json` configuration
- Integrated into SuperClaude's import system
- Ready to use immediately in Claude Code

## Acknowledgments

This tool installs the [SuperClaude Framework](https://github.com/SuperClaude-Org/SuperClaude_Framework) created by [SuperClaude-Org](https://github.com/SuperClaude-Org). 

For detailed information about SuperClaude Framework features, commands, agents, and capabilities, please visit the [SuperClaude Framework documentation](https://github.com/SuperClaude-Org/SuperClaude_Framework).

Special thanks to the SuperClaude-Org team for creating and maintaining this powerful framework.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---
