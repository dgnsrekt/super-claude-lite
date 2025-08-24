package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MCPServer represents an MCP server with its metadata
type MCPServer struct {
	Name        string `json:"name"`        // e.g., "Context7"
	DisplayName string `json:"displayName"` // e.g., "Context7 MCP integration"
	MDFile      string `json:"mdFile"`      // e.g., "MCP_Context7.md"
	ConfigFile  string `json:"configFile"`  // e.g., "context7.json"
	Selected    bool   `json:"selected"`    // Selection state for TUI
}

// DiscoverMCPServers scans the SuperClaude/MCP directory and returns available MCP servers
func DiscoverMCPServers(repoPath string) ([]MCPServer, error) {
	mcpDir := filepath.Join(repoPath, "SuperClaude", "MCP")
	
	// Check if MCP directory exists
	if _, err := os.Stat(mcpDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("MCP directory not found: %s", mcpDir)
	}

	// Read MCP directory
	entries, err := os.ReadDir(mcpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP directory: %w", err)
	}

	var servers []MCPServer

	// Look for MCP_*.md files
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "MCP_") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Extract server name: MCP_Context7.md -> Context7
		name := strings.TrimPrefix(entry.Name(), "MCP_")
		name = strings.TrimSuffix(name, ".md")
		
		// Expected config file: Context7 -> context7.json
		configFile := strings.ToLower(name) + ".json"
		configPath := filepath.Join(mcpDir, "configs", configFile)
		
		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Skip servers without config files
			continue
		}

		server := MCPServer{
			Name:        name,
			DisplayName: fmt.Sprintf("%s MCP integration", name),
			MDFile:      entry.Name(),
			ConfigFile:  configFile,
			Selected:    false,
		}

		servers = append(servers, server)
	}

	// Sort servers by name for consistent display
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers, nil
}

// LoadMCPConfig loads an MCP server configuration from its JSON file
func LoadMCPConfig(repoPath, configFile string) (map[string]interface{}, error) {
	configPath := filepath.Join(repoPath, "SuperClaude", "MCP", "configs", configFile)
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config %s: %w", configFile, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config %s: %w", configFile, err)
	}

	return config, nil
}

// GetSelectedServers returns only the servers that are marked as selected
func GetSelectedServers(servers []MCPServer) []MCPServer {
	var selected []MCPServer
	for _, server := range servers {
		if server.Selected {
			selected = append(selected, server)
		}
	}
	return selected
}