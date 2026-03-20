package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/shvbsle/k10s/internal/k8s"
)

const (
	// DefaultMaxLogBuffer is the maximum number of log lines to keep in memory
	DefaultMaxLogBuffer = 10000
	// MinTailLines is the minimum initial lines to fetch
	MinTailLines = 100
	// TailLinesMultiplier is used to calculate initial tail lines from viewport height
	TailLinesMultiplier = 2
)

// LogViewport wraps a viewport for scrollable log output with streaming support
type LogViewport struct {
	viewport        viewport.Model
	showLineNumbers bool
	showTimestamps  bool
	autoScroll      bool
	wordWrap        bool
	width           int
	height          int
	podName         string
	containerName   string
	namespace       string
	logLines        []k8s.LogLine
	maxBufferSize   int
	totalLines      int // Total lines received (for accurate line numbering after trimming)
	filterText      string
	filterActive    bool
	filterInput     textinput.Model
	matchCount      int
}

// NewLogViewport creates a new log viewport
func NewLogViewport() *LogViewport {
	vp := viewport.New(
		viewport.WithWidth(80),
		viewport.WithHeight(20),
	)

	fi := textinput.New()
	fi.Placeholder = "filter logs..."

	return &LogViewport{
		viewport:        vp,
		showLineNumbers: false,
		showTimestamps:  false,
		autoScroll:      true,
		wordWrap:        false,
		maxBufferSize:   DefaultMaxLogBuffer,
		logLines:        make([]k8s.LogLine, 0),
		filterInput:     fi,
	}
}

// SetContent sets the initial log content
func (l *LogViewport) SetContent(lines []k8s.LogLine, podName, containerName, namespace string) {
	l.logLines = lines
	l.totalLines = len(lines)
	l.podName = podName
	l.containerName = containerName
	l.namespace = namespace
	l.filterText = ""
	l.filterActive = false
	l.filterInput.SetValue("")
	l.matchCount = 0
	l.updateRenderedContent()

	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

// AppendLines appends new log lines (for streaming)
func (l *LogViewport) AppendLines(lines []k8s.LogLine) {
	l.logLines = append(l.logLines, lines...)
	l.totalLines += len(lines)

	// Trim buffer if exceeds max size
	if len(l.logLines) > l.maxBufferSize {
		excess := len(l.logLines) - l.maxBufferSize
		l.logLines = l.logLines[excess:]
	}

	l.updateRenderedContent()

	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

// SetSize sets the viewport dimensions
func (l *LogViewport) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.viewport.SetWidth(width)
	// Reserve 2 lines: 1 for header, 1 for footer
	l.viewport.SetHeight(max(height-2, 1))
	l.updateRenderedContent()
}

// GetTailLines calculates the initial tail lines based on viewport height
func (l *LogViewport) GetTailLines() int {
	tailLines := l.height * TailLinesMultiplier
	if tailLines < MinTailLines {
		tailLines = MinTailLines
	}
	return tailLines
}

// updateRenderedContent renders the log content with optional line numbers and timestamps.
// When a filter is active, only matching lines are shown with matches highlighted.
func (l *LogViewport) updateRenderedContent() {
	if len(l.logLines) == 0 {
		l.viewport.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No logs available"))
		return
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	matchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)

	var rendered strings.Builder
	// Calculate the offset for line numbers when buffer has been trimmed
	lineNumOffset := l.totalLines - len(l.logLines)
	matchCount := 0
	renderedCount := 0

	for i, line := range l.logLines {
		if l.filterText != "" && !strings.Contains(line.Content, l.filterText) {
			continue
		}
		matchCount++

		if renderedCount > 0 {
			rendered.WriteString("\n")
		}
		renderedCount++

		if l.showLineNumbers {
			actualLineNum := lineNumOffset + i + 1
			rendered.WriteString(lineNumStyle.Render(fmt.Sprintf("%6d ", actualLineNum)))
		}

		if l.showTimestamps && line.Timestamp != "" {
			rendered.WriteString(timestampStyle.Render(line.Timestamp + " "))
		}

		content := line.Content
		if l.wordWrap && l.width > 0 {
			content = l.wrapText(content, l.width-10)
		}

		if l.filterText != "" {
			rendered.WriteString(highlightMatches(content, l.filterText, contentStyle, matchStyle))
		} else {
			rendered.WriteString(contentStyle.Render(content))
		}
	}

	l.matchCount = matchCount
	if rendered.Len() == 0 {
		l.viewport.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No matching lines"))
	} else {
		l.viewport.SetContent(rendered.String())
	}
}

// highlightMatches renders content with occurrences of term highlighted.
func highlightMatches(content, term string, base, highlight lipgloss.Style) string {
	var b strings.Builder
	for {
		idx := strings.Index(content, term)
		if idx < 0 {
			b.WriteString(base.Render(content))
			break
		}
		if idx > 0 {
			b.WriteString(base.Render(content[:idx]))
		}
		b.WriteString(highlight.Render(content[idx : idx+len(term)]))
		content = content[idx+len(term):]
	}
	return b.String()
}

// wrapText wraps text to the specified width
func (l *LogViewport) wrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var result strings.Builder
	for len(text) > width {
		result.WriteString(text[:width])
		result.WriteString("\n")
		text = text[width:]
	}
	result.WriteString(text)
	return result.String()
}

// Update handles input for the log viewport
func (l *LogViewport) Update(msg tea.Msg) (*LogViewport, tea.Cmd) {
	// When filter input is active, route key messages to it
	if l.filterActive {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				l.ClearFilter()
				return l, nil
			default:
				return l.UpdateFilter(msg)
			}
		}
		return l.UpdateFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			l.autoScroll = false
			l.viewport.GotoTop()
			return l, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			l.autoScroll = true
			l.viewport.GotoBottom()
			return l, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			l.autoScroll = false
			// Let the viewport handle the scroll via its Update method
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			// Let the viewport handle the scroll via its Update method
		}
	}

	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

// View renders the log viewport with header and footer
func (l *LogViewport) View() string {
	// Build header
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	var title string
	if l.namespace != "" {
		title = fmt.Sprintf("Logs: %s/%s/%s", l.namespace, l.podName, l.containerName)
	} else {
		title = fmt.Sprintf("Logs: %s/%s", l.podName, l.containerName)
	}

	// Scroll position indicator
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	scrollInfo := hintStyle.Render(fmt.Sprintf(" %d%%", int(l.viewport.ScrollPercent()*100)))

	// Auto-scroll indicator
	autoScrollIndicator := ""
	if l.autoScroll {
		autoScrollIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(" [TAILING]")
	} else {
		autoScrollIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" [PAUSED]")
	}

	// Filter match indicator
	filterIndicator := ""
	if l.filterText != "" {
		if l.matchCount > 0 {
			filterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(fmt.Sprintf(" [%d matches]", l.matchCount))
		} else {
			filterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(" [no matches]")
		}
	}

	header := titleStyle.Render(title) + scrollInfo + autoScrollIndicator + filterIndicator

	// Build footer with hints
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	var footer string
	if l.filterActive {
		footer = keyStyle.Render("/") + hintStyle.Render(" ") + l.filterInput.View() +
			hintStyle.Render("   ") + keyStyle.Render("esc") + hintStyle.Render(" clear")
	} else {
		footer = keyStyle.Render("↑↓/jk") + hintStyle.Render(" scroll  ") +
			keyStyle.Render("g/G") + hintStyle.Render(" top/bottom  ") +
			keyStyle.Render("n") + hintStyle.Render(" line#  ") +
			keyStyle.Render("t") + hintStyle.Render(" time  ") +
			keyStyle.Render("s") + hintStyle.Render(" tail  ") +
			keyStyle.Render("/") + hintStyle.Render(" filter  ") +
			keyStyle.Render("w") + hintStyle.Render(" wrap  ") +
			keyStyle.Render("esc") + hintStyle.Render(" back")
	}

	return header + "\n" + l.viewport.View() + "\n" + footer
}

// SetFilter applies a filter to the log view, showing only matching lines.
func (l *LogViewport) SetFilter(text string) {
	l.filterText = text
	l.updateRenderedContent()
	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

// ClearFilter removes the active filter and closes the filter input.
func (l *LogViewport) ClearFilter() {
	l.filterText = ""
	l.filterActive = false
	l.filterInput.SetValue("")
	l.filterInput.Blur()
	l.matchCount = 0
	l.updateRenderedContent()
}

// ActivateFilter opens the filter input bar.
func (l *LogViewport) ActivateFilter() tea.Cmd {
	l.filterActive = true
	return l.filterInput.Focus()
}

// FilterActive returns whether the filter input is currently open.
func (l *LogViewport) FilterActive() bool { return l.filterActive }

// FilterText returns the current filter string.
func (l *LogViewport) FilterText() string { return l.filterText }

// UpdateFilter passes a message to the filter input and updates the filter from its value.
func (l *LogViewport) UpdateFilter(msg tea.Msg) (*LogViewport, tea.Cmd) {
	var cmd tea.Cmd
	l.filterInput, cmd = l.filterInput.Update(msg)
	l.SetFilter(l.filterInput.Value())
	return l, cmd
}

// Toggle methods
func (l *LogViewport) ToggleLineNumbers() {
	l.showLineNumbers = !l.showLineNumbers
	l.updateRenderedContent()
}

func (l *LogViewport) ToggleTimestamps() {
	l.showTimestamps = !l.showTimestamps
	l.updateRenderedContent()
}

func (l *LogViewport) ToggleAutoScroll() {
	l.autoScroll = !l.autoScroll
	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

func (l *LogViewport) ToggleWordWrap() {
	l.wordWrap = !l.wordWrap
	l.updateRenderedContent()
}

// Getters
func (l *LogViewport) ShowLineNumbers() bool   { return l.showLineNumbers }
func (l *LogViewport) ShowTimestamps() bool    { return l.showTimestamps }
func (l *LogViewport) AutoScroll() bool        { return l.autoScroll }
func (l *LogViewport) WordWrap() bool          { return l.wordWrap }
func (l *LogViewport) LogLines() []k8s.LogLine { return l.logLines }
func (l *LogViewport) TotalLines() int         { return l.totalLines }

// GotoTop scrolls to the top
func (l *LogViewport) GotoTop() {
	l.autoScroll = false
	l.viewport.GotoTop()
}

// GotoBottom scrolls to the bottom and enables auto-scroll
func (l *LogViewport) GotoBottom() {
	l.autoScroll = true
	l.viewport.GotoBottom()
}

// Height returns the total height used by the viewport
func (l *LogViewport) Height() int {
	return l.height
}
