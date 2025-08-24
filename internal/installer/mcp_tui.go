package installer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define key bindings
var keys = struct {
	up     key.Binding
	down   key.Binding
	space  key.Binding
	enter  key.Binding
	quit   key.Binding
	help   key.Binding
}{
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#02BA84")).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	checkboxChecked   = "[✓]"
	checkboxUnchecked = "[ ]"
)

// MCPSelectorModel represents the TUI state for MCP server selection
type MCPSelectorModel struct {
	servers  []MCPServer
	cursor   int
	selected map[int]bool
	quitting bool
	confirmed bool
}

// NewMCPSelector creates a new MCP selector TUI model
func NewMCPSelector(servers []MCPServer) MCPSelectorModel {
	return MCPSelectorModel{
		servers:  servers,
		cursor:   0,
		selected: make(map[int]bool),
		quitting: false,
		confirmed: false,
	}
}

func (m MCPSelectorModel) Init() tea.Cmd {
	return nil
}

func (m MCPSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.down):
			if m.cursor < len(m.servers)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.space):
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]

		case key.Matches(msg, keys.enter):
			// Confirm selections
			m.confirmed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m MCPSelectorModel) View() string {
	if m.quitting {
		return "\nCancelled MCP server selection.\n"
	}

	if m.confirmed {
		selectedCount := 0
		for _, isSelected := range m.selected {
			if isSelected {
				selectedCount++
			}
		}
		return fmt.Sprintf("\nSelected %d MCP servers for installation.\n", selectedCount)
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Select MCP Servers"))
	b.WriteString("\n\n")

	// Instructions
	b.WriteString(helpStyle.Render("Use ↑/↓ to navigate, space to select/deselect, enter to confirm"))
	b.WriteString("\n\n")

	// Server list
	for i, server := range m.servers {
		// Determine checkbox state
		checkbox := checkboxUnchecked
		if m.selected[i] {
			checkbox = checkboxChecked
		}

		// Determine style based on cursor position
		style := unselectedStyle
		if i == m.cursor {
			style = selectedStyle
		}

		// Format line
		line := fmt.Sprintf("%s %s", checkbox, server.DisplayName)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Footer help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press 'q' to quit, 'enter' to proceed with selection"))

	return b.String()
}

// GetSelectedServers returns the selected servers with their selection state updated
func (m MCPSelectorModel) GetSelectedServers() []MCPServer {
	var selected []MCPServer
	for i, server := range m.servers {
		if m.selected[i] {
			server.Selected = true
			selected = append(selected, server)
		}
	}
	return selected
}

// ShowMCPSelector displays the TUI and returns the selected servers
func ShowMCPSelector(servers []MCPServer) ([]MCPServer, error) {
	model := NewMCPSelector(servers)
	
	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run MCP selector: %w", err)
	}

	// Extract results from final model
	final := finalModel.(MCPSelectorModel)
	
	if final.quitting {
		return nil, fmt.Errorf("user cancelled MCP selection")
	}

	return final.GetSelectedServers(), nil
}