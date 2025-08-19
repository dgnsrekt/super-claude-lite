# Suggested Commands for Super Claude Lite Development

## Daily Development Commands

### Building and Testing
```bash
# Build the binary
make build

# Run all tests
make test

# Format code (must be run before commits)
make fmt

# Run linter (must pass before commits)
make lint

# Install dependencies and tidy modules
make deps

# Development build with usage examples
make dev

# Test installation with dry-run
make test-install
```

### Project Management
```bash
# Clean build artifacts
make clean

# Install binary to GOPATH/bin for testing
make install

# Uninstall from GOPATH/bin
make uninstall

# Show project information
make info

# Show all available make targets
make help
```

### Quality Assurance
```bash
# Run pre-commit hooks manually (if pre-commit is installed)
make pre-commit

# Setup development environment
make setup
```

### Git and Version Control
```bash
# Standard git workflow
git add .
git commit -m "descriptive commit message"
git push origin branch-name
```

### Running the Tool
```bash
# After building
./bin/super-claude-lite --help
./bin/super-claude-lite status
./bin/super-claude-lite init --dry-run
./bin/super-claude-lite init --add-mcp

# After installing to GOPATH
super-claude-lite --version
super-claude-lite status
super-claude-lite init
super-claude-lite clean
super-claude-lite rollback
```

## Essential Commands When Task is Complete

1. **Format code**: `make fmt`
2. **Run linter**: `make lint` 
3. **Run tests**: `make test`
4. **Build**: `make build`
5. **Test functionality**: `make test-install`

All of these must pass before code can be considered complete.