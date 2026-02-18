package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HelpModal represents the help modal state
type HelpModal struct {
	viewport viewport.Model
	visible  bool
	width    int
	height   int
}

// NewHelpModal creates a new help modal
func NewHelpModal() *HelpModal {
	return &HelpModal{
		viewport: viewport.New(
			viewport.WithWidth(80),
			viewport.WithHeight(20),
		),
		visible: false,
	}
}

// IsVisible returns whether the modal is currently visible
func (h *HelpModal) IsVisible() bool {
	return h.visible
}

// Toggle toggles the modal visibility
func (h *HelpModal) Toggle() {
	h.visible = !h.visible
}

// Show shows the modal
func (h *HelpModal) Show() {
	h.visible = true
}

// Hide hides the modal
func (h *HelpModal) Hide() {
	h.visible = false
}

// SetSize sets the modal size based on terminal dimensions
func (h *HelpModal) SetSize(width, height int) {
	h.width = width
	h.height = height
	// Modal takes up 80% of screen, with some padding
	modalWidth := min(width-4, 80)
	modalHeight := min(height-6, 40)
	h.viewport.SetWidth(modalWidth - 4) // Account for border
	h.viewport.SetHeight(modalHeight - 4)
}

// Update handles input for the help modal
func (h *HelpModal) Update(msg tea.Msg) (*HelpModal, tea.Cmd) {
	var cmd tea.Cmd
	h.viewport, cmd = h.viewport.Update(msg)
	return h, cmd
}

// SetContent sets the viewport content
func (h *HelpModal) SetContent(content string) {
	h.viewport.SetContent(content)
}

// View renders the help modal
func (h *HelpModal) View() string {
	if !h.visible {
		return ""
	}

	modalWidth := min(h.width-4, 80)
	modalHeight := min(h.height-6, 40)

	// Modal border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	// Footer hint
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	content := titleStyle.Render("k10s Help") + "\n\n" +
		h.viewport.View() + "\n\n" +
		footerStyle.Render("Press ? or Esc to close • ↑/↓ to scroll")

	modal := borderStyle.Render(content)

	// Center the modal
	return lipgloss.Place(
		h.width,
		h.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// BuildHelpContent builds the help content for the modal
func (m *Model) BuildHelpContent() string {
	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true).
		Underline(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Width(10)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Section 1: Cluster Info
	b.WriteString(sectionStyle.Render("Cluster Information"))
	b.WriteString("\n\n")

	if m.clusterInfo != nil {
		b.WriteString(labelStyle.Render("Context:  ") + valueStyle.Render(m.clusterInfo.Context) + "\n")
		b.WriteString(labelStyle.Render("Cluster:  ") + valueStyle.Render(m.clusterInfo.Cluster) + "\n")
		b.WriteString(labelStyle.Render("K8s Ver:  ") + valueStyle.Render(m.clusterInfo.K8sVersion) + "\n")
	} else {
		b.WriteString(dimStyle.Render("Not connected to a cluster\n"))
	}
	b.WriteString(labelStyle.Render("k10s Ver: ") + valueStyle.Render(Version) + "\n")

	// Section 2: Key Bindings
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Key Bindings"))
	b.WriteString("\n\n")

	// Navigation
	b.WriteString(dimStyle.Render("Navigation") + "\n")
	b.WriteString(keyStyle.Render("↑/k") + descStyle.Render("Move up") + "\n")
	b.WriteString(keyStyle.Render("↓/j") + descStyle.Render("Move down") + "\n")
	b.WriteString(keyStyle.Render("←/h") + descStyle.Render("Previous page") + "\n")
	b.WriteString(keyStyle.Render("→/l") + descStyle.Render("Next page") + "\n")
	b.WriteString(keyStyle.Render("g") + descStyle.Render("Go to top") + "\n")
	b.WriteString(keyStyle.Render("G") + descStyle.Render("Go to bottom") + "\n")
	b.WriteString(keyStyle.Render("Enter") + descStyle.Render("Drill down into resource") + "\n")
	b.WriteString(keyStyle.Render("Esc") + descStyle.Render("Go back / close modal") + "\n")

	// Actions
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Actions") + "\n")
	b.WriteString(keyStyle.Render(":") + descStyle.Render("Open command palette") + "\n")
	b.WriteString(keyStyle.Render("d") + descStyle.Render("Describe selected resource (YAML)") + "\n")
	b.WriteString(keyStyle.Render("s") + descStyle.Render("Shell into pod/container") + "\n")
	b.WriteString(keyStyle.Render("?") + descStyle.Render("Toggle this help modal") + "\n")
	b.WriteString(keyStyle.Render("Ctrl+C") + descStyle.Render("Quit") + "\n")

	// Log view specific
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Log View") + "\n")
	b.WriteString(keyStyle.Render("f") + descStyle.Render("Toggle fullscreen") + "\n")
	b.WriteString(keyStyle.Render("s") + descStyle.Render("Toggle autoscroll") + "\n")
	b.WriteString(keyStyle.Render("t") + descStyle.Render("Toggle timestamps") + "\n")
	b.WriteString(keyStyle.Render("w") + descStyle.Render("Toggle word wrap") + "\n")
	b.WriteString(keyStyle.Render("n") + descStyle.Render("Toggle line numbers") + "\n")

	// Describe view specific
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Describe View") + "\n")
	b.WriteString(keyStyle.Render("f") + descStyle.Render("Toggle fullscreen") + "\n")
	b.WriteString(keyStyle.Render("w") + descStyle.Render("Toggle word wrap") + "\n")
	b.WriteString(keyStyle.Render("n") + descStyle.Render("Toggle line numbers") + "\n")

	// Section 3: Commands
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Commands"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Type : to enter command mode") + "\n\n")

	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Width(18)
	b.WriteString(cmdStyle.Render(":q, :quit") + descStyle.Render("Quit k10s") + "\n")
	b.WriteString(cmdStyle.Render(":r, :reconnect") + descStyle.Render("Reconnect to cluster") + "\n")
	b.WriteString(cmdStyle.Render(":ctx") + descStyle.Render("List available contexts") + "\n")
	b.WriteString(cmdStyle.Render(":ctx <name>") + descStyle.Render("Switch to context") + "\n")
	b.WriteString(cmdStyle.Render(":rs, :resource") + descStyle.Render("List available resources") + "\n")
	b.WriteString(cmdStyle.Render(":rs <type>") + descStyle.Render("View resource (e.g., :rs deployments)") + "\n")
	b.WriteString(cmdStyle.Render(":rs <type> -n <ns>") + descStyle.Render("View resource in namespace") + "\n")
	b.WriteString(cmdStyle.Render(":cplogs [path]") + descStyle.Render("Copy logs to file") + "\n")
	b.WriteString(cmdStyle.Render(":cp [path]") + descStyle.Render("Alias for :cplogs") + "\n")

	// Section 4: Current Settings
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Current Settings"))
	b.WriteString("\n\n")

	pageSizeStr := "auto"
	if m.config.MaxPageSize > 0 {
		pageSizeStr = fmt.Sprintf("%d", m.config.MaxPageSize)
	}
	b.WriteString(labelStyle.Render("Page Size:        ") + valueStyle.Render(pageSizeStr) + "\n")
	b.WriteString(labelStyle.Render("Log Tail Lines:   ") + valueStyle.Render(fmt.Sprintf("%d", m.config.LogTailLines)) + "\n")
	b.WriteString(labelStyle.Render("Pagination Style: ") + valueStyle.Render(string(m.config.PaginationStyle)) + "\n")

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Config file: ~/.k10s.conf"))

	return b.String()
}
