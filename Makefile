# SuperClaude Lite - Lightweight SuperClaude Framework Installer
# Build automation tool for Go project

.PHONY: build clean test fmt lint install uninstall dev info help deps

# Project variables
BINARY_NAME=super-claude-lite
MAIN_PATH=cmd/super-claude-lite/main.go
BIN_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built at $(BIN_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@go clean
	@echo "Clean completed"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	@cp $(BIN_DIR)/$(BINARY_NAME) $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Uninstall binary from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from GOPATH/bin..."
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

# Development mode - build and show help
dev: build
	@echo "Development build complete. Usage examples:"
	@echo "  ./$(BIN_DIR)/$(BINARY_NAME) --help"
	@echo "  ./$(BIN_DIR)/$(BINARY_NAME) init --dry-run"
	@echo "  ./$(BIN_DIR)/$(BINARY_NAME) status"

# Run the binary with test installation
test-install: build
	@echo "Testing installation in test directory..."
	@mkdir -p test-run
	@./$(BIN_DIR)/$(BINARY_NAME) init test-run --dry-run
	@rm -rf test-run

# Run pre-commit hooks manually
pre-commit:
	@echo "Running pre-commit hooks..."
	@pre-commit run --all-files

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@go mod download
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@pip install pre-commit
	@pre-commit install
	@echo "Setup complete!"

# Display project info
info:
	@echo "SuperClaude Lite Project Information"
	@echo "===================================="
	@echo "Go version: $(shell go version)"
	@echo "Binary name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Binary path: $(BIN_DIR)/$(BINARY_NAME)"
	@echo "Main file: $(MAIN_PATH)"

# Show help
help:
	@echo "SuperClaude Lite Makefile"
	@echo "========================="
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build the binary"
	@echo "  clean        Remove build artifacts"
	@echo "  test         Run tests"
	@echo "  fmt          Format Go code"
	@echo "  lint         Run golangci-lint"
	@echo "  deps         Install and tidy dependencies"
	@echo "  install      Install binary to GOPATH/bin"
	@echo "  uninstall    Remove binary from GOPATH/bin"
	@echo "  dev          Build and show development usage"
	@echo "  test-install Test installation with dry-run"
	@echo "  pre-commit   Run pre-commit hooks manually"
	@echo "  setup        Setup development environment"
	@echo "  info         Show project information"
	@echo "  help         Show this help message"