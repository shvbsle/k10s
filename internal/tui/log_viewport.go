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

	// Line selection state for copy support
	selectionAnchor int // First line clicked (-1 = no selection)
	selectionEnd    int // Last line in selection range (-1 = no selection)

	// renderedToLogIndex maps rendered line indices to logLines indices.
	// Needed because filters can skip lines, so rendered line N may not
	// correspond to logLines[N].
	renderedToLogIndex []int

	// viewStartY is the terminal Y coordinate where the log viewport content
	// begins (after the log header line). Set by the parent during render so
	// that click Y coordinates can be mapped to rendered line indices.
	viewStartY int

	// Reusable styles — allocated once, not per-render-frame.
	styles logStyles
}

// logStyles holds pre-allocated lipgloss styles for log rendering.
type logStyles struct {
	lineNum   lipgloss.Style
	timestamp lipgloss.Style
	content   lipgloss.Style
	match     lipgloss.Style
	selected  lipgloss.Style
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
		selectionAnchor: -1,
		selectionEnd:    -1,
		styles: logStyles{
			lineNum:   lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
			timestamp: lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
			content:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
			match:     lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true),
			selected:  lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")),
		},
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
	l.selectionAnchor = -1
	l.selectionEnd = -1
	l.renderedToLogIndex = nil
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
// Selected lines get a highlight background applied to plain text (no nested ANSI)
// to avoid garbled escape sequences when the viewport truncates mid-sequence.
func (l *LogViewport) updateRenderedContent() {
	if len(l.logLines) == 0 {
		l.renderedToLogIndex = nil
		l.viewport.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No logs available"))
		return
	}

	// Compute selection range in logLines indices
	selLo, selHi := l.selectionRange()

	var rendered strings.Builder
	l.renderedToLogIndex = l.renderedToLogIndex[:0] // reuse backing array
	// Calculate the offset for line numbers when buffer has been trimmed
	lineNumOffset := l.totalLines - len(l.logLines)
	matchCount := 0
	renderedCount := 0

	for i, line := range l.logLines {
		if l.filterText != "" && !strings.Contains(line.Content, l.filterText) {
			continue
		}
		matchCount++

		isSelected := selLo >= 0 && i >= selLo && i <= selHi

		// Build prefix (line number + timestamp) as plain text
		var prefix string
		if l.showLineNumbers {
			actualLineNum := lineNumOffset + i + 1
			prefix += fmt.Sprintf("%6d ", actualLineNum)
		}
		if l.showTimestamps && line.Timestamp != "" {
			prefix += line.Timestamp + " "
		}

		content := line.Content

		// Word wrap: one log line may produce multiple rendered lines.
		// Each rendered line maps back to the same logLines index so that
		// click-to-select works correctly in word wrap mode.
		var lineTexts []string
		if l.wordWrap && l.width > 0 {
			wrapped := l.wrapText(content, l.width-10)
			lineTexts = strings.Split(wrapped, "\n")
		} else {
			lineTexts = []string{content}
		}

		for j, text := range lineTexts {
			if renderedCount > 0 {
				rendered.WriteString("\n")
			}
			l.renderedToLogIndex = append(l.renderedToLogIndex, i)
			renderedCount++

			if isSelected {
				// Selected: render as plain text with a single highlight style.
				// No nested ANSI codes — avoids garbled escape sequences.
				var plainLine string
				if j == 0 {
					plainLine = prefix + text
				} else {
					plainLine = strings.Repeat(" ", len(prefix)) + text
				}
				rendered.WriteString(l.styles.selected.Render(plainLine))
			} else {
				// Normal rendering with per-segment styles
				l.renderStyledLine(&rendered, i, j, text, prefix, lineNumOffset)
			}
		}
	}

	l.matchCount = matchCount
	if rendered.Len() == 0 {
		l.renderedToLogIndex = nil
		l.viewport.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No matching lines"))
	} else {
		l.viewport.SetContent(rendered.String())
	}
}

// renderStyledLine writes a single styled (non-selected) line to the builder.
func (l *LogViewport) renderStyledLine(b *strings.Builder, logIdx, wrapIdx int, text, prefix string,
	lineNumOffset int) {

	if wrapIdx == 0 {
		if l.showLineNumbers {
			actualLineNum := lineNumOffset + logIdx + 1
			b.WriteString(l.styles.lineNum.Render(fmt.Sprintf("%6d ", actualLineNum)))
		}
		if l.showTimestamps && l.logLines[logIdx].Timestamp != "" {
			b.WriteString(l.styles.timestamp.Render(l.logLines[logIdx].Timestamp + " "))
		}
	} else {
		b.WriteString(l.styles.content.Render(strings.Repeat(" ", len(prefix))))
	}

	if l.filterText != "" {
		b.WriteString(highlightMatches(text, l.filterText, l.styles.content, l.styles.match))
	} else {
		b.WriteString(l.styles.content.Render(text))
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
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			// Scrolling up pauses autoscroll so the user can read history
			// without new lines yanking them back to the bottom.
			l.autoScroll = false
		case tea.MouseWheelDown:
			// If we've scrolled to the bottom, re-enable tailing
			if l.viewport.AtBottom() {
				l.autoScroll = true
			}
		}
		// Delegate to the viewport for actual scroll movement
		var cmd tea.Cmd
		l.viewport, cmd = l.viewport.Update(msg)
		return l, cmd

	case tea.MouseClickMsg:
		l.handleClick(msg.Y, msg.Mod)
		return l, nil

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
	} else if l.HasSelection() {
		selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		footer = selStyle.Render(fmt.Sprintf("%d lines selected", l.SelectedLineCount())) +
			hintStyle.Render("  ") + keyStyle.Render("y/c") + hintStyle.Render(" yank (copy)  ") +
			keyStyle.Render("shift+click") + hintStyle.Render(" extend  ") +
			keyStyle.Render("esc") + hintStyle.Render(" clear")
	} else {
		footer = keyStyle.Render("↑↓/jk") + hintStyle.Render(" scroll  ") +
			keyStyle.Render("g/G") + hintStyle.Render(" top/bottom  ") +
			keyStyle.Render("n") + hintStyle.Render(" line#  ") +
			keyStyle.Render("t") + hintStyle.Render(" time  ") +
			keyStyle.Render("s") + hintStyle.Render(" tail  ") +
			keyStyle.Render("/") + hintStyle.Render(" filter  ") +
			keyStyle.Render("w") + hintStyle.Render(" wrap  ") +
			keyStyle.Render("click") + hintStyle.Render(" select  ") +
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

// SetViewStartY sets the terminal Y coordinate where the viewport content
// begins. Called by the parent View function so click coordinates can be
// translated to rendered line indices without magic numbers.
func (l *LogViewport) SetViewStartY(y int) {
	l.viewStartY = y
}

// handleClick processes a mouse click in the log viewport.
// Plain click selects a single line; shift+click extends the selection
// from the anchor to the clicked line.
func (l *LogViewport) handleClick(y int, mod tea.KeyMod) {
	// Map terminal Y to a rendered line index.
	// viewStartY points to the first line of viewport content (after the
	// log header line). The viewport's YOffset is the scroll position.
	renderedLine := l.viewport.YOffset() + (y - l.viewStartY)
	if renderedLine < 0 || renderedLine >= len(l.renderedToLogIndex) {
		return
	}

	logIdx := l.renderedToLogIndex[renderedLine]

	if mod&tea.ModShift != 0 && l.selectionAnchor >= 0 {
		// Shift+click: extend selection from anchor to this line
		l.selectionEnd = logIdx
	} else {
		// Plain click: start new selection (single line)
		l.selectionAnchor = logIdx
		l.selectionEnd = logIdx
	}

	l.updateRenderedContent()
}

// ClearSelection removes any active line selection.
func (l *LogViewport) ClearSelection() {
	if l.selectionAnchor < 0 {
		return
	}
	l.selectionAnchor = -1
	l.selectionEnd = -1
	l.updateRenderedContent()
}

// HasSelection returns true if one or more lines are selected.
func (l *LogViewport) HasSelection() bool {
	return l.selectionAnchor >= 0
}

// SelectedLines returns the log lines in the current selection range.
// Returns nil if nothing is selected.
func (l *LogViewport) SelectedLines() []k8s.LogLine {
	lo, hi := l.selectionRange()
	if lo < 0 {
		return nil
	}
	// Clamp to valid range
	if hi >= len(l.logLines) {
		hi = len(l.logLines) - 1
	}
	return l.logLines[lo : hi+1]
}

// selectionRange returns the normalized (low, high) indices into logLines.
// Returns (-1, -1) if no selection is active.
func (l *LogViewport) selectionRange() (int, int) {
	if l.selectionAnchor < 0 || l.selectionEnd < 0 {
		return -1, -1
	}
	lo, hi := l.selectionAnchor, l.selectionEnd
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi
}

// SelectedLineCount returns the number of lines currently selected.
func (l *LogViewport) SelectedLineCount() int {
	lo, hi := l.selectionRange()
	if lo < 0 {
		return 0
	}
	return hi - lo + 1
}
