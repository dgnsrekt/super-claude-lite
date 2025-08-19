# Super Claude Lite - Project Overview

## Project Purpose
Super Claude Lite is a lightweight installer for the SuperClaude Framework that allows you to install it inside the project directory instead of the home directory. It provides a CLI tool that manages installation, status checking, cleaning, and rollback operations for the SuperClaude Framework.

## What It Does
- Installs SuperClaude Framework files locally within projects
- Optionally configures MCP (Model Context Protocol) servers
- Provides backup and rollback functionality
- Manages symlink integration with Claude Code slash commands
- Supports dependency-aware installation steps
- Offers dry-run support for safe testing

## Tech Stack
- **Language**: Go 1.24.0+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **UI/Styling**: Charmbracelet Fang (github.com/charmbracelet/fang)
- **Build Tool**: Make
- **Linting**: golangci-lint
- **Pre-commit**: pre-commit hooks (if available)
- **Demo Creation**: VHS (for generating demo.gif)

## Key Dependencies
- github.com/spf13/cobra - CLI framework
- github.com/charmbracelet/fang - Batteries-included CLI enhancements

## Core Concepts
- Installs SuperClaude Framework from: https://github.com/SuperClaude-Org/SuperClaude_Framework.git
- Fixed commit: 8f12b19a220aabf23680ee86b6d6ab21c46ba6f6
- Creates `.superclaude/` and `.claude/` directories
- Manages MCP server configurations in `.mcp.json`
- Uses backup strategy with `.superclaude-backup` prefix