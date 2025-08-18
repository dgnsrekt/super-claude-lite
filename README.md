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

A lightweight installer for the SuperClaude Framework that allows you to install it inside the project vs in your home dir.

## Demo

![Demo](demo.gif)

## Quick Start

```bash
# Install the tool
go install github.com/dgnsrekt/super-claude-lite/cmd/super-claude-lite@latest

# Check installation status
super-claude-lite status

# Install with recommended MCP servers
super-claude-lite init --add-mcp

# or Basic installation
super-claude-lite init

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

## Acknowledgments

This tool installs the [SuperClaude Framework](https://github.com/SuperClaude-Org/SuperClaude_Framework) created by [SuperClaude-Org](https://github.com/SuperClaude-Org). SuperClaude Framework is an open-source AI configuration framework that provides specialized commands, cognitive personas, and development methodologies for enhancing Claude Code workflows.

Special thanks to the SuperClaude-Org team for creating and maintaining this powerful framework that makes AI-assisted development more accessible and efficient.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---
