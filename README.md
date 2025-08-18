# Super Claude Lite

```
  _____ _    _ _____  ______ _____   _____ _               _      _      _ _       
 / ____| |  | |  __ \|  ____|  __ \ / ____| |        /\   | |    | |    (_) |      
| (___ | |  | | |__) | |__  | |__) | |    | |       /  \  | |    | |     _| |_ ___ 
 \___ \| |  | |  ___/|  __| |  _  /| |    | |      / /\ \ | |    | |    | | __/ _ \
 ____) | |__| | |    | |____| | \ \| |____| |____ / ____ \| |____| |____| | ||  __/
|_____/ \____/|_|    |______|_|  \_\\_____|______/_/    \_\______|______|_|\__\___|
```

A lightweight installer for the SuperClaude Framework that gives you control over where and how it's installed.

## Quick Start

```bash
# Install the tool
go install github.com/dgnsrekt/super-claude-lite/cmd/super-claude-lite@latest

# Basic installation
super-claude-lite init

# Install with recommended MCP servers
super-claude-lite init --add-mcp

# Check installation status
super-claude-lite status
```

## Commands

- `init` - Install SuperClaude framework files
- `status` - Check installation status
- `clean` - Remove installed files
- `rollback` - Restore from backup

## Features

- Install SuperClaude Framework at any location (not forced to home directory)
- Optional MCP server configuration with `--add-mcp` flag
- Automatic backup and merge of existing files
- Symlink integration with Claude Code slash commands
- DAG-based dependency resolution
- Dry-run support

## MCP Servers

When using `--add-mcp`, includes these recommended servers:
- sequential-thinking
- context7
- serena
- playwright

---

*Remember: Home directory installations are optional. This tool makes your SuperClaude setup speak for itself.*