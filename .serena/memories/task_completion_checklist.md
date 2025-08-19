# Task Completion Checklist for Super Claude Lite

## Before Marking Any Task Complete

### 1. Code Quality Checks
```bash
# Format the code (required)
make fmt

# Run the linter (must pass)
make lint

# Run all tests (must pass)  
make test

# Build successfully
make build
```

### 2. Functional Testing
```bash
# Test the installation process
make test-install

# Test the actual binary
./bin/super-claude-lite --help
./bin/super-claude-lite status
./bin/super-claude-lite init --dry-run
```

### 3. Pre-commit Validation (if available)
```bash
# Run pre-commit hooks manually
make pre-commit
```

## All Steps Must Pass

- **Code formatting**: `make fmt` must not change any files
- **Linting**: `make lint` must report no issues
- **Tests**: `make test` must pass all tests
- **Build**: `make build` must complete successfully
- **Functional test**: `make test-install` must work correctly

## Additional Considerations

### For New Features
- Ensure the feature works as expected through manual testing
- Check that help text is updated if new flags/commands are added
- Verify error handling works correctly
- Test edge cases and invalid inputs

### For Bug Fixes
- Verify the bug is actually fixed
- Check that the fix doesn't break existing functionality
- Consider if tests should be added to prevent regression

### For Refactoring
- Ensure all existing functionality still works
- Check that the public API hasn't changed unexpectedly
- Verify performance hasn't degraded

## Documentation Updates
- Update README.md if user-facing changes were made
- Update help text in CLI commands if needed
- Add or update code comments for complex logic

## Git Workflow
```bash
# Before committing
git add .
git status  # Review what will be committed
git commit -m "descriptive commit message"

# After committing
git log --oneline -5  # Verify commit looks correct
```

## Integration with SuperClaude Framework
Since this tool installs the SuperClaude Framework, ensure:
- Installation creates correct directory structure
- MCP configuration is valid JSON
- Symlinks are created correctly
- Backup/rollback functionality works
- Framework files are at the expected commit/version