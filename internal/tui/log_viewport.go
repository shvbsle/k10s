package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
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
}

// NewLogViewport creates a new log viewport
func NewLogViewport() *LogViewport {
	vp := viewport.New(
		viewport.WithWidth(80),
		viewport.WithHeight(20),
	)

	return &LogViewport{
		viewport:        vp,
		showLineNumbers: false,
		showTimestamps:  false,
		autoScroll:      true,
		wordWrap:        false,
		maxBufferSize:   DefaultMaxLogBuffer,
		logLines:        make([]k8s.LogLine, 0),
	}
}

// SetContent sets the initial log content
func (l *LogViewport) SetContent(lines []k8s.LogLine, podName, containerName, namespace string) {
	l.logLines = lines
	l.totalLines = len(lines)
	l.podName = podName
	l.containerName = containerName
	l.namespace = namespace
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

// updateRenderedContent renders the log content with optional line numbers and timestamps
func (l *LogViewport) updateRenderedContent() {
	if len(l.logLines) == 0 {
		l.viewport.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No logs available"))
		return
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	var rendered strings.Builder
	// Calculate the offset for line numbers when buffer has been trimmed
	lineNumOffset := l.totalLines - len(l.logLines)

	for i, line := range l.logLines {
		if l.showLineNumbers {
			actualLineNum := lineNumOffset + i + 1
			lineNumStr := lineNumStyle.Render(fmt.Sprintf("%6d ", actualLineNum))
			rendered.WriteString(lineNumStr)
		}

		if l.showTimestamps && line.Timestamp != "" {
			rendered.WriteString(timestampStyle.Render(line.Timestamp + " "))
		}

		content := line.Content
		if l.wordWrap && l.width > 0 {
			content = l.wrapText(content, l.width-10) // Account for line numbers and padding
		}
		rendered.WriteString(contentStyle.Render(content))

		if i < len(l.logLines)-1 {
			rendered.WriteString("\n")
		}
	}

	l.viewport.SetContent(rendered.String())
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

	header := titleStyle.Render(title) + scrollInfo + autoScrollIndicator

	// Build footer with hints
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	footer := keyStyle.Render("↑↓/jk") + hintStyle.Render(" scroll  ") +
		keyStyle.Render("g/G") + hintStyle.Render(" top/bottom  ") +
		keyStyle.Render("n") + hintStyle.Render(" line#  ") +
		keyStyle.Render("t") + hintStyle.Render(" time  ") +
		keyStyle.Render("s") + hintStyle.Render(" tail  ") +
		keyStyle.Render("w") + hintStyle.Render(" wrap  ") +
		keyStyle.Render("esc") + hintStyle.Render(" back")

	return header + "\n" + l.viewport.View() + "\n" + footer
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
