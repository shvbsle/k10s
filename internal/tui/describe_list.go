package tui

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// yamlKeyRegex matches YAML keys (words with optional spaces followed by colon)
// Examples: "Namespace:", "Service Account:", "Node-Selectors:"
var yamlKeyRegex = regexp.MustCompile(`^(\s*)([A-Za-z0-9][A-Za-z0-9_ -]*):`)

// DescribeViewport wraps a viewport for scrollable describe output
type DescribeViewport struct {
	viewport        viewport.Model
	showLineNumbers bool
	width           int
	height          int
	resourceName    string
	namespace       string
	rawContent      string
}

// NewDescribeViewport creates a new describe viewport
func NewDescribeViewport() *DescribeViewport {
	vp := viewport.New(
		viewport.WithWidth(80),
		viewport.WithHeight(20),
	)

	return &DescribeViewport{
		viewport:        vp,
		showLineNumbers: true,
	}
}

// SetContent sets the describe content
func (d *DescribeViewport) SetContent(content, resourceName, namespace string) {
	d.rawContent = content
	d.resourceName = resourceName
	d.namespace = namespace
	d.updateRenderedContent()
}

// highlightYAMLLine applies syntax highlighting to a single line
func highlightYAMLLine(line string) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Check if line has a YAML key pattern
	match := yamlKeyRegex.FindStringSubmatchIndex(line)
	if match != nil {
		// match[0:2] = full match, match[2:4] = indent, match[4:6] = key
		indent := line[match[2]:match[3]]
		keyName := line[match[4]:match[5]]
		rest := line[match[1]:]

		return indent + keyStyle.Render(keyName+":") + valueStyle.Render(rest)
	}

	// No key found, render as plain value
	return valueStyle.Render(line)
}

// updateRenderedContent renders the content with syntax highlighting and line numbers
func (d *DescribeViewport) updateRenderedContent() {
	lines := strings.Split(d.rawContent, "\n")

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var rendered strings.Builder
	for i, line := range lines {
		if d.showLineNumbers {
			lineNumStr := lineNumStyle.Render(fmt.Sprintf("%5d ", i+1))
			rendered.WriteString(lineNumStr)
		}
		rendered.WriteString(highlightYAMLLine(line))
		if i < len(lines)-1 {
			rendered.WriteString("\n")
		}
	}

	d.viewport.SetContent(rendered.String())
}

// SetSize sets the viewport dimensions (accounting for header and footer)
func (d *DescribeViewport) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.SetWidth(width)
	// Reserve 2 lines: 1 for header, 1 for footer
	d.viewport.SetHeight(max(height-2, 1))
}

// SetShowLineNumbers toggles line number display
func (d *DescribeViewport) SetShowLineNumbers(show bool) {
	d.showLineNumbers = show
	d.updateRenderedContent()
}

// ToggleLineNumbers toggles line number display
func (d *DescribeViewport) ToggleLineNumbers() {
	d.showLineNumbers = !d.showLineNumbers
	d.updateRenderedContent()
}

// ShowLineNumbers returns whether line numbers are shown
func (d *DescribeViewport) ShowLineNumbers() bool {
	return d.showLineNumbers
}

// Update handles input for the describe viewport
func (d *DescribeViewport) Update(msg tea.Msg) (*DescribeViewport, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			d.viewport.GotoTop()
			return d, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			d.viewport.GotoBottom()
			return d, nil
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// View renders the describe viewport with header and footer
func (d *DescribeViewport) View() string {
	// Build header
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	var title string
	if d.namespace != "" {
		title = fmt.Sprintf("Describe: %s/%s", d.namespace, d.resourceName)
	} else {
		title = fmt.Sprintf("Describe: %s", d.resourceName)
	}

	// Scroll position indicator
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	scrollInfo := hintStyle.Render(fmt.Sprintf(" %d%%", int(d.viewport.ScrollPercent()*100)))

	header := titleStyle.Render(title) + scrollInfo

	// Build footer with hints
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	footer := keyStyle.Render("↑↓") + hintStyle.Render(" scroll  ") +
		keyStyle.Render("g/G") + hintStyle.Render(" top/bottom  ") +
		keyStyle.Render("n") + hintStyle.Render(" line numbers  ") +
		keyStyle.Render("esc") + hintStyle.Render(" go back")

	return header + "\n" + d.viewport.View() + "\n" + footer
}

// GotoTop scrolls to the top
func (d *DescribeViewport) GotoTop() {
	d.viewport.GotoTop()
}

// GotoBottom scrolls to the bottom
func (d *DescribeViewport) GotoBottom() {
	d.viewport.GotoBottom()
}

// Height returns the total height used by the viewport (including header/footer)
func (d *DescribeViewport) Height() int {
	return d.height
}
