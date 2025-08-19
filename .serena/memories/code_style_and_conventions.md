# Code Style and Conventions for Super Claude Lite

## Go Code Style

### Package Structure
- `cmd/super-claude-lite/` - Main entry point
- `internal/` - Private packages not for external use
  - `internal/installer/` - Installation logic
  - `internal/git/` - Git operations
  - `internal/config/` - Configuration constants

### Naming Conventions
- **Package names**: lowercase, single words when possible
- **Function names**: CamelCase for exported, camelCase for private
- **Variable names**: camelCase for local, CamelCase for exported
- **Constants**: CamelCase for exported, camelCase for private
- **Struct names**: CamelCase for exported types

### Error Handling
- Use `fmt.Errorf()` with `%w` verb for error wrapping
- Example: `return nil, fmt.Errorf("failed to create install context: %w", err)`

### Code Organization
- Use receiver methods for struct operations: `(*Installer).Install`
- Constructor pattern: `NewInstaller()` functions
- Keep public API minimal, use `internal/` packages for implementation

### Imports
- Standard library imports first
- Third-party imports second  
- Local project imports last
- Group with blank lines between sections

### Documentation
- Public functions and types should have comments
- Comments should start with the function/type name
- Use `// Package packagename` for package documentation

## Linting Configuration

### Enabled Linters (golangci-lint)
- **goconst** - Find repeated strings that could be constants
- **gocritic** - Comprehensive meta-linter  
- **gocyclo** - Cyclomatic complexity (max 15)
- **gosec** - Security analysis
- **misspell** - Spelling mistakes
- **unparam** - Unused function parameters
- **unconvert** - Unnecessary type conversions
- **whitespace** - Whitespace issues
- **prealloc** - Slice preallocation
- **predeclared** - Misuse of predeclared identifiers

### Exceptions
- Test files (`*_test.go`) exclude: gosec, goconst, gocyclo
- File operations in `internal/installer/` exclude G304 security warnings
- G204 (subprocess with variable) excluded for git commands

## Formatting
- **gofmt** - Standard Go formatting
- **goimports** - Automatic import management

## Pre-commit Hooks
- go-fmt, go-imports, go-mod-tidy
- go-vet-mod, go-build-mod, golangci-lint-mod
- trailing-whitespace, end-of-file-fixer
- check-yaml, check-json, check-toml
- check-merge-conflict, check-added-large-files

## Build Configuration
- Uses semantic versioning via git tags
- Version and commit embedded via ldflags during build
- Binary name: `super-claude-lite`
- Main path: `cmd/super-claude-lite/main.go`