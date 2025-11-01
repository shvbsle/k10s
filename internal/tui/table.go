package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/tui/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// wrapTextAtWordBoundary wraps text at word boundaries when possible.
// Falls back to truncation for words longer than maxWidth.
// Preserves original whitespace formatting.
func wrapTextAtWordBoundary(text string, maxWidth int) []string {
	if maxWidth <= 0 || runewidth.StringWidth(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	currentLine := ""
	currentWidth := 0
	i := 0
	textRunes := []rune(text)

	for i < len(textRunes) {
		// Skip leading whitespace for new lines (except first line)
		if currentWidth == 0 && len(lines) > 0 {
			for i < len(textRunes) && textRunes[i] == ' ' {
				i++
			}
			if i >= len(textRunes) {
				break
			}
		}

		// Collect whitespace before the word (preserve spacing)
		whitespaceStart := i
		for i < len(textRunes) && textRunes[i] == ' ' {
			i++
		}
		whitespace := string(textRunes[whitespaceStart:i])
		whitespaceWidth := runewidth.StringWidth(whitespace)

		// Collect the word
		wordStart := i
		for i < len(textRunes) && textRunes[i] != ' ' {
			i++
		}
		word := string(textRunes[wordStart:i])
		wordWidth := runewidth.StringWidth(word)

		// Check if whitespace + word fits on current line
		neededWidth := whitespaceWidth + wordWidth
		if currentWidth > 0 && currentWidth+neededWidth > maxWidth {
			// Doesn't fit - save current line and start new one
			lines = append(lines, currentLine)
			currentLine = ""
			currentWidth = 0
			whitespace = "" // Don't carry over leading whitespace to new line
			whitespaceWidth = 0
		}

		// Add whitespace and word to current line
		if wordWidth > 0 {
			currentLine += whitespace + word
			currentWidth += whitespaceWidth + wordWidth
		}

		// Handle words longer than maxWidth
		if currentWidth > maxWidth && currentLine == whitespace+word {
			// This single word is too long, truncate it
			currentLine = runewidth.Truncate(currentLine, maxWidth, "…")
			lines = append(lines, currentLine)
			currentLine = ""
			currentWidth = 0
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		lines = []string{""}
	}

	return lines
}

// updateTableData updates the table rows based on the current page and data.
func (m *Model) updateTableData() {
	if m.resourceType == k8s.ResourceLogs && m.logLines != nil {
		m.updateTableDataForLogs()
	} else {
		m.updateTableDataForResources()
	}
}

// updateTableDataForResources updates table with Kubernetes resources.
func (m *Model) updateTableDataForResources() {
	// Bounds checking to prevent slice out of range
	if len(m.resources) == 0 {
		m.table.SetRows([]table.Row{})
		m.paginator.SetTotalPages(0)
		return
	}

	start := m.paginator.Page * m.paginator.PerPage
	if start >= len(m.resources) {
		start = 0
		m.paginator.Page = 0
	}

	end := start + m.paginator.PerPage
	if end > len(m.resources) {
		end = len(m.resources)
	}

	pageResources := m.resources[start:end]
	rows := make([]table.Row, len(pageResources))

	for i, res := range pageResources {
		rows[i] = table.Row(res)
	}

	m.table.SetRows(rows)
	m.paginator.SetTotalPages(len(m.resources))
}

// updateTableDataForLogs updates table with container logs.
func (m *Model) updateTableDataForLogs() {
	// Bounds checking to prevent slice out of range
	if len(m.logLines) == 0 {
		m.table.SetRows([]table.Row{})
		m.paginator.SetTotalPages(0)
		return
	}

	start := m.paginator.Page * m.paginator.PerPage
	if start >= len(m.logLines) {
		start = 0
		m.paginator.Page = 0
	}

	end := start + m.paginator.PerPage
	if end > len(m.logLines) {
		end = len(m.logLines)
	}

	pageLogLines := m.logLines[start:end]
	var rows []table.Row

	for _, logLine := range pageLogLines {
		logRows := m.formatLogLine(logLine)
		rows = append(rows, logRows...)
	}

	m.table.SetRows(rows)
	m.paginator.SetTotalPages(len(m.logLines))
}

// formatLogLine formats a single log line for table display with optional wrapping.
func (m *Model) formatLogLine(logLine k8s.LogLine) []table.Row {
	// Format line number prefix (e.g., "   1: ")
	lineNumPrefix := fmt.Sprintf("%4d: ", logLine.LineNum)
	lineNumWidth := runewidth.StringWidth(lineNumPrefix)

	var timestamp string
	var timestampWidth int

	// Add timestamp if enabled
	if m.logView.ShowTimestamps && logLine.Timestamp != "" {
		timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		timestamp = timestampStyle.Render(logLine.Timestamp) + " "
		timestampWidth = lipgloss.Width(timestamp)
	}

	var rows []table.Row

	if m.logView.WrapText {
		columns := m.table.Columns()
		logWidth := columns[0].Width

		// Calculate available width for actual log content
		// Account for timestamp, line number prefix, and continuation marker
		prefixWidth := timestampWidth + lineNumWidth
		availableWidth := logWidth - prefixWidth
		if availableWidth < 10 {
			availableWidth = 10
		}

		// Wrap only the log content (without line number prefix)
		wrappedLines := wrapTextAtWordBoundary(logLine.Content, availableWidth)

		for j, line := range wrappedLines {
			var displayLine string
			if j == 0 {
				// First line: timestamp + line number + content
				displayLine = timestamp + lineNumPrefix + line
			} else {
				// Continuation lines: indent to align with first line's content
				indent := strings.Repeat(" ", prefixWidth)
				displayLine = indent + line
			}
			rows = append(rows, table.Row{
				displayLine,
				"", "", "", "", "",
			})
		}
	} else {
		displayLine := timestamp + lineNumPrefix + logLine.Content
		rows = append(rows, table.Row{
			displayLine,
			"", "", "", "", "",
		})
	}

	return rows
}

// renderTableWithHeader renders the table with a custom header border containing the resource type.
func (m *Model) renderTableWithHeader(b *strings.Builder) {
	nsDisplay := m.currentNamespace
	if nsDisplay == metav1.NamespaceAll {
		nsDisplay = "all"
	}

	headerText := fmt.Sprintf(" %s[%s](%d) ", m.resourceType, nsDisplay, len(m.resources))
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	borderColor := lipgloss.Color("240")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Get column information from table
	columns := m.table.Columns()

	// Calculate total table width from column definitions
	tableWidth := 0
	for _, col := range columns {
		tableWidth += col.Width
	}
	// Add spacing between columns (1 space between each column)
	tableWidth += len(columns) - 1

	// Build custom top border with centered title
	topBorder := m.buildTopBorderWithTitle(headerText, tableWidth, borderColor, headerStyle)
	b.WriteString(topBorder)
	b.WriteString("\n")

	// Render column headers manually
	headerLine := ""
	for i, col := range columns {
		if i > 0 {
			headerLine += " "
		}
		// Truncate or pad to exact width
		title := col.Title
		if len(title) > col.Width {
			title = title[:col.Width]
		} else if len(title) < col.Width {
			title = title + strings.Repeat(" ", col.Width-len(title))
		}
		headerLine += title
	}

	headerLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	b.WriteString(borderStyle.Render("│"))
	b.WriteString(headerLineStyle.Render(headerLine))
	b.WriteString(borderStyle.Render("│"))
	b.WriteString("\n")

	// Render toggle status for logs view
	if m.resourceType == k8s.ResourceLogs {
		onStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		offStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

		autoscrollStatus := offStyle.Render("OFF")
		if m.logView.Autoscroll {
			autoscrollStatus = onStyle.Render("ON")
		}

		fullscreenStatus := offStyle.Render("OFF")
		if m.logView.Fullscreen {
			fullscreenStatus = onStyle.Render("ON")
		}

		timestampStatus := offStyle.Render("OFF")
		if m.logView.ShowTimestamps {
			timestampStatus = onStyle.Render("ON")
		}

		wrapStatus := offStyle.Render("OFF")
		if m.logView.WrapText {
			wrapStatus = onStyle.Render("ON")
		}

		toggleLine := fmt.Sprintf(" %s %s %s %s %s %s %s %s",
			labelStyle.Render("[Autoscroll:"),
			autoscrollStatus+labelStyle.Render("]"),
			labelStyle.Render("[Fullscreen:"),
			fullscreenStatus+labelStyle.Render("]"),
			labelStyle.Render("[Timestamps:"),
			timestampStatus+labelStyle.Render("]"),
			labelStyle.Render("[Wrap:"),
			wrapStatus+labelStyle.Render("]"),
		)

		// Pad or truncate to exact table width using ANSI-aware functions
		toggleLineLen := lipgloss.Width(toggleLine)
		if toggleLineLen > tableWidth {
			toggleLine = ansi.Truncate(toggleLine, tableWidth, "…")
		} else if toggleLineLen < tableWidth {
			toggleLine += strings.Repeat(" ", tableWidth-toggleLineLen)
		}

		b.WriteString(borderStyle.Render("│"))
		b.WriteString(toggleLine)
		b.WriteString(borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Render separator line
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	separator := "├" + strings.Repeat("─", tableWidth) + "┤"
	b.WriteString(separatorStyle.Render(separator))
	b.WriteString("\n")

	// Render data rows
	rows := m.table.Rows()
	selectedRow := m.table.Cursor()
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	normalStyle := lipgloss.NewStyle()

	for idx, row := range rows {
		rowLine := ""
		for i, cell := range row {
			if i > 0 {
				rowLine += " "
			}
			// Truncate or pad to exact width (ANSI-aware)
			cellText := cell
			visualWidth := lipgloss.Width(cellText)

			if visualWidth > columns[i].Width {
				// Truncate with ANSI-awareness
				cellText = ansi.Truncate(cellText, columns[i].Width, "…")
			} else if visualWidth < columns[i].Width {
				// Pad based on visual width
				cellText = cellText + strings.Repeat(" ", columns[i].Width-visualWidth)
			}
			rowLine += cellText
		}

		// Apply selection styling
		rowStyle := normalStyle
		if idx == selectedRow {
			rowStyle = selectedStyle
		}

		b.WriteString(borderStyle.Render("│"))
		b.WriteString(rowStyle.Render(rowLine))
		b.WriteString(borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Write bottom border
	bottomBorder := "└" + strings.Repeat("─", tableWidth) + "┘"
	b.WriteString(borderStyle.Render(bottomBorder))
}

// buildTopBorderWithTitle creates a top border with a centered title.
func (m *Model) buildTopBorderWithTitle(title string, width int, borderColor lipgloss.Color, titleStyle lipgloss.Style) string {
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Calculate centering - leftDashes + titleLen + rightDashes = width
	titleLen := runewidth.StringWidth(title)
	leftDashes := (width - titleLen) / 2
	rightDashes := width - titleLen - leftDashes

	if leftDashes < 1 {
		leftDashes = 1
	}
	if rightDashes < 1 {
		rightDashes = 1
	}

	// Build: ┌──── title ────┐
	var result strings.Builder
	result.WriteString(borderStyle.Render("┌"))
	result.WriteString(borderStyle.Render(strings.Repeat("─", leftDashes)))
	result.WriteString(titleStyle.Render(title))
	result.WriteString(borderStyle.Render(strings.Repeat("─", rightDashes)))
	result.WriteString(borderStyle.Render("┐"))

	return result.String()
}

// updateColumns updates the table columns based on the current width and resource type.
func (m *Model) updateColumns(width int) {
	// Rough calc for border renders:
	// Total overhead: 2 (borders) + 5 (column spacing) = 7
	totalWidth := width - 10
	if totalWidth < 90 {
		totalWidth = 90
	}
	columns := resources.GetColumns(totalWidth, m.resourceType)

	// we have to clear rows if we're going to update columns to avoid breaking
	// the model with inconsistent column count.
	// SAFETY: this function should always be called before updating rows.
	m.table.SetRows([]table.Row{})

	m.table.SetColumns(columns)
}

// renderPagination renders the pagination display based on configured style.
// Automatically switches to verbose style for logs with more than 5 pages.
func (m *Model) renderPagination(b *strings.Builder) {
	// For logs with more than 5 pages, always use verbose style
	if m.resourceType == k8s.ResourceLogs && m.paginator.TotalPages > 5 {
		paginatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pageInfo := fmt.Sprintf("Page %d/%d", m.paginator.Page+1, m.paginator.TotalPages)
		b.WriteString(paginatorStyle.Render(pageInfo))
		return
	}

	// Otherwise use configured style
	switch m.config.PaginationStyle {
	case config.PaginationStyleVerbose:
		// Text-based pagination: "Page 1/10"
		paginatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pageInfo := fmt.Sprintf("Page %d/%d", m.paginator.Page+1, m.paginator.TotalPages)
		b.WriteString(paginatorStyle.Render(pageInfo))
	case config.PaginationStyleBubbles:
		// Bubbles paginator component (dots)
		b.WriteString(m.paginator.View())
	default:
		// Default to bubbles style
		b.WriteString(m.paginator.View())
	}
}
