# Development Commands and Tools for Super Claude Lite

## System Information
- **Platform**: Linux (arm64)
- **Go Version**: go1.24.6 linux/arm64
- **Available Tools**: golangci-lint ✓, pre-commit ✗

## Core Development Commands

### Building
```bash
# Build the project
make build                    # Creates bin/super-claude-lite

# Development build with usage examples  
make dev                      # Build + show usage examples

# Install to GOPATH for testing
make install                  # Copies to $GOPATH/bin

# Clean build artifacts
make clean                    # Removes bin/ directory
```

### Code Quality
```bash
# Format Go code (required before commits)
make fmt                      # Runs go fmt ./...

# Run linter (required before commits)
make lint                     # Runs golangci-lint run

# Install and tidy dependencies
make deps                     # go mod download && go mod tidy
```

### Testing
```bash
# Run all tests
make test                     # go test -v ./...

# Test installation process
make test-install            # Test with dry-run in test directory
```

### Development Environment
```bash
# Setup development environment
make setup                   # Install tools and configure pre-commit

# Manual pre-commit (if available)
make pre-commit              # Run all pre-commit hooks
```

### Project Information
```bash
# Show project details
make info                    # Version, commit, paths, Go version

# Show all available targets
make help                    # Display Makefile help
```

## Go Commands (Direct)
```bash
# Module management
go mod download              # Download dependencies
go mod tidy                  # Clean up go.mod/go.sum
go mod graph                 # Show dependency graph

# Code formatting and vetting
go fmt ./...                 # Format all Go files
go vet ./...                 # Run go vet

# Building
go build -o bin/super-claude-lite cmd/super-claude-lite/main.go

# Testing
go test ./...                # Run tests
go test -v ./...             # Verbose test output
```

## Linting with golangci-lint
```bash
# Run linter (available at /home/dgnsrekt/go/bin/golangci-lint)
golangci-lint run            # Run all configured linters
golangci-lint run --fix      # Auto-fix issues where possible
golangci-lint config path    # Show config file path
golangci-lint linters        # List available linters
```

## Application Commands (After Building)
```bash
# Basic usage
./bin/super-claude-lite --help
./bin/super-claude-lite --version

# Core operations  
./bin/super-claude-lite status              # Check installation status
./bin/super-claude-lite init                # Basic installation
./bin/super-claude-lite init --add-mcp      # Install with MCP servers
./bin/super-claude-lite init --dry-run      # Test installation
./bin/super-claude-lite clean               # Remove installation
./bin/super-claude-lite rollback            # Restore from backup
```

## Version and Build Information
- Uses git tags for versioning
- Embeds version and commit via ldflags
- Current version: 0.2.1
- Binary built with: `go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"`

## Git Workflow
```bash
# Standard development workflow
git status                   # Check current state
git add .                    # Stage changes
git commit -m "message"      # Commit with message
git push origin branch       # Push to remote

# Version tagging
git tag v0.x.x              # Create version tag
git push origin v0.x.x      # Push tag
```

## Demo Generation (VHS)
```bash
# Generate demo.gif (requires vhs tool)
vhs demo.tape               # Creates demo.gif from demo.tape script
```

## Essential Workflow
1. `make fmt` - Format code
2. `make lint` - Check linting
3. `make test` - Run tests  
4. `make build` - Build binary
5. `make test-install` - Test functionality
6. Git commit and push