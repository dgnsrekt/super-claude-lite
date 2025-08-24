# VHS Demo for local Super Claude Lite
Output demo.local.gif

# Terminal configuration
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 50ms

# Clear terminal and start demo
Type "clear"
Enter
Sleep 1s

# Cleaning up local super-claude-lite
Type "# Clean local Super Claude Lite"
Sleep 500ms
Enter
Type "make clean"
Sleep 500ms
Enter
Sleep 4s

# Install local super-claude-lite
Type "# Install local Super Claude Lite"
Sleep 500ms
Enter
Type "make build"
Sleep 500ms
Enter
Sleep 4s

# Create demo directory
Type "# Creating a demo directory for Super Claude Lite"
Sleep 500ms
Enter
Type "mkdir -p super-claude-demo && cp ./bin/super-claude-lite super-claude-demo && cd super-claude-demo"
Sleep 500ms
Enter
Sleep 1s

# Show version
Type "# Check version"
Sleep 500ms
Enter
Type "./super-claude-lite --version"
Sleep 500ms
Enter
Sleep 2s

# Show help
Type "# Display help information"
Sleep 500ms
Enter
Type "./super-claude-lite --help"
Sleep 500ms
Enter
Sleep 4s

# Clear and check status
Type "clear"
Enter
Sleep 500ms

# Check status before installation
Type "# Check current installation status"
Sleep 500ms
Enter
Type "./super-claude-lite status"
Sleep 500ms
Enter
Sleep 3s

# Initialize with MCP servers
Type "# Install SuperClaude Framework with MCP servers"
Sleep 500ms
Enter
Type "./super-claude-lite init --add-mcp"
Sleep 500ms
Enter
Sleep 5s
Space
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Space
Sleep 500ms
Down
Sleep 500ms
Space
Sleep 500ms
Enter
Sleep 4s

# Show the resulting structure
Type "# View the installed file structure"
Sleep 500ms
Enter
Type "tree -L 2 -a"
Sleep 500ms
Enter
Sleep 4s

# Final pause to admire the result
Sleep 3s