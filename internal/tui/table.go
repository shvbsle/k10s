package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
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

// applyHorizontalScroll takes a full-width line, skips `offset` visual characters,
// then truncates/pads to exactly `viewWidth` visual characters.
// This enables horizontal scrolling of table content.
// Handles both plain text and ANSI-styled text.
func applyHorizontalScroll(line string, offset, viewWidth int) string {
	if viewWidth <= 0 {
		return ""
	}

	lineWidth := lipgloss.Width(line)

	// Apply offset: skip leading characters
	display := line
	if offset > 0 {
		if offset >= lineWidth {
			// Entire content is scrolled past
			display = ""
		} else {
			// Use ANSI-aware truncation: cut the first `offset` visual chars
			// by taking the right portion of the string
			display = ansi.Cut(line, offset, lineWidth)
		}
	}

	// Truncate or pad to exact viewWidth
	displayWidth := lipgloss.Width(display)
	if displayWidth > viewWidth {
		display = ansi.Truncate(display, viewWidth, "")
	} else if displayWidth < viewWidth {
		display = display + strings.Repeat(" ", viewWidth-displayWidth)
	}

	return display
}

// statusColor returns a lipgloss style that colorizes pod/container status values.
// Green for running/healthy, red for errors, yellow for pending, gray for completed, etc.
func statusColor(value string) lipgloss.Style {
	s := strings.ToLower(strings.TrimSpace(value))
	switch {
	case s == "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")) // green
	case s == "succeeded" || s == "completed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // gray
	case s == "pending" || s == "containercreating" || s == "podinitialized" || s == "init" || s == "waiting":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // yellow/orange
	case s == "failed" || s == "error" || s == "crashloopbackoff" || s == "imagepullbackoff" || s == "errimagepull" || s == "oomkilled" || s == "terminated":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // red
	case s == "terminating" || s == "unknown":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange
	default:
		return lipgloss.NewStyle()
	}
}

// isStatusColumn returns true if the column title represents a status/phase field.
func isStatusColumn(title string) bool {
	t := strings.ToLower(title)
	return t == "phase" || t == "status"
}

// updateTableData updates the table rows based on the current page and data.
func (m *Model) updateTableData() {
	if m.currentGVR.Resource == k8s.ResourceLogs && m.logLines != nil {
		m.updateTableDataForLogs()
	} else if (m.currentGVR.Resource == k8s.ResourceDescribe || m.currentGVR.Resource == k8s.ResourceYaml) && m.describeContent != "" {
		m.updateTableDataForDescribe()
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

	end := min(start+m.paginator.PerPage, len(m.resources))

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

	end := min(start+m.paginator.PerPage, len(m.logLines))

	pageLogLines := m.logLines[start:end]
	var rows []table.Row

	for _, logLine := range pageLogLines {
		logRows := m.formatLogLine(logLine)
		rows = append(rows, logRows...)
	}

	m.table.SetRows(rows)
	m.paginator.SetTotalPages(len(m.logLines))
}

// updateTableDataForDescribe updates table with YAML content from describe.
func (m *Model) updateTableDataForDescribe() {
	// Bounds checking to prevent slice out of range
	if m.describeContent == "" {
		m.table.SetRows([]table.Row{})
		m.paginator.SetTotalPages(0)
		return
	}

	// Split YAML content into lines
	lines := strings.Split(m.describeContent, "\n")

	// Calculate pagination
	start := m.paginator.Page * m.paginator.PerPage
	if start >= len(lines) {
		start = 0
		m.paginator.Page = 0
	}

	end := min(start+m.paginator.PerPage, len(lines))
	pageLines := lines[start:end]

	// Convert lines to table rows with syntax highlighting
	rows := make([]table.Row, len(pageLines))
	for i, line := range pageLines {
		highlightedLine := m.highlightDescribeLine(line)

		if m.describeView.ShowLineNumbers {
			// Add line number prefix (1-indexed, relative to whole document)
			lineNum := start + i + 1
			rows[i] = table.Row{fmt.Sprintf("%4d: %s", lineNum, highlightedLine)}
		} else {
			rows[i] = table.Row{highlightedLine}
		}
	}

	m.table.SetRows(rows)
	m.paginator.SetTotalPages(len(lines))
}

// highlightDescribeLine applies syntax highlighting to a kubectl describe output line.
// It highlights keys (text ending with ':') in a different color.
func (m *Model) highlightDescribeLine(line string) string {
	// Find the position of the first colon
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return line // No colon, return as-is
	}

	beforeColon := line[:colonIdx]
	trimmedBefore := strings.TrimLeft(beforeColon, " \t")

	// Check if this looks like a key:
	// 1. Not empty
	// 2. Starts at the beginning of the line (after whitespace)
	// 3. Doesn't contain special characters that indicate it's not a label
	if trimmedBefore == "" {
		return line
	}

	// Check if the line contains characters that suggest it's not a key
	invalidChars := []string{"\"", "'", "(", ")", "[", "]", "{", "}", "=", "<", ">"}
	for _, char := range invalidChars {
		if strings.Contains(trimmedBefore, char) {
			return line // Contains invalid characters, not a key
		}
	}

	// Additional check: if there's text after the colon on the same line,
	// and it starts immediately (no space), it's probably not a label
	// (e.g., "http://example.com:8080")
	if colonIdx+1 < len(line) && line[colonIdx+1] != ' ' && line[colonIdx+1] != '\t' && line[colonIdx+1] != '\n' {
		return line
	}

	// Get the leading whitespace
	leadingSpace := beforeColon[:len(beforeColon)-len(trimmedBefore)]
	afterColon := line[colonIdx:]

	// Highlight the key in cyan and bold
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	highlightedKey := keyStyle.Render(trimmedBefore)

	value      := strings.TrimSpace(strings.TrimPrefix(afterColon, ":"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	coloredValue := highlightDescribeValue(value, afterColon, valueStyle, dimStyle)

	return leadingSpace + highlightedKey + coloredValue
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
		availableWidth := max(logWidth-prefixWidth, 10)

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
			rows = append(rows, table.Row{displayLine})
		}
	} else {
		displayLine := timestamp + lineNumPrefix + logLine.Content
		rows = append(rows, table.Row{displayLine})
	}

	return rows
}

// renderTableWithHeader renders the table with a custom header border containing the resource type.
func (m *Model) renderTableWithHeader(b *strings.Builder) {
	nsDisplay := m.currentNamespace
	if nsDisplay == metav1.NamespaceAll {
		nsDisplay = "all"
	}

	headerText := fmt.Sprintf(" %s [%s] (%d) ", k8s.FormatGVR(m.currentGVR), nsDisplay, len(m.resources))
	if m.horizontalOffset > 0 {
		headerText = fmt.Sprintf(" %s [%s] (%d) ◀ scroll:%d ", k8s.FormatGVR(m.currentGVR), nsDisplay, len(m.resources), m.horizontalOffset)
	}
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
	if len(columns) > 1 {
		tableWidth += len(columns) - 1
	}

	// Build custom top border with centered title
	topBorder := m.buildTopBorderWithTitle(headerText, tableWidth, borderColor, headerStyle)
	b.WriteString(topBorder)
	b.WriteString("\n")

	// Render column headers manually (skip for logs, describe, and yaml views as they don't need columns)
	// We need to compute effective column widths first (accounting for cell overflow)
	// so that headers and data rows use the same widths and stay aligned.
	rows := m.table.Rows()

	// Calculate effective column widths: max of defined width and widest cell in that column
	effectiveWidths := make([]int, len(columns))
	for i, col := range columns {
		effectiveWidths[i] = col.Width
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(effectiveWidths) {
				break
			}
			cellWidth := runewidth.StringWidth(cell)
			if cellWidth > effectiveWidths[i] {
				effectiveWidths[i] = cellWidth
			}
		}
	}

	// Calculate the total content width using effective widths
	maxContentWidth := 0
	for i, w := range effectiveWidths {
		if i > 0 {
			maxContentWidth++ // column separator space
		}
		maxContentWidth += w
	}

	// Clamp horizontal offset so we can't scroll past the content
	maxOffset := maxContentWidth - tableWidth
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.horizontalOffset > maxOffset {
		m.horizontalOffset = maxOffset
	}

	if m.currentGVR.Resource != k8s.ResourceLogs && m.currentGVR.Resource != k8s.ResourceDescribe && m.currentGVR.Resource != k8s.ResourceYaml {
		// Build full header line with each column padded to its effective width
		fullHeaderLine := ""
		for i, col := range columns {
			if i > 0 {
				fullHeaderLine += " "
			}
			title := col.Title
			titleWidth := runewidth.StringWidth(title)
			if titleWidth < effectiveWidths[i] {
				title = title + strings.Repeat(" ", effectiveWidths[i]-titleWidth)
			}
			fullHeaderLine += title
		}

		// Apply horizontal scroll to header
		headerLine := applyHorizontalScroll(fullHeaderLine, m.horizontalOffset, tableWidth)

		headerLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
		b.WriteString(borderStyle.Render("│"))
		b.WriteString(headerLineStyle.Render(headerLine))
		b.WriteString(borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Render toggle status for logs view
	if m.currentGVR.Resource == k8s.ResourceLogs {
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

	// Render toggle status for describe/yaml view
	if m.currentGVR.Resource == k8s.ResourceDescribe || m.currentGVR.Resource == k8s.ResourceYaml {
		onStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		offStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)

		fullscreenStatus := offStyle.Render("OFF")
		if m.describeView.Fullscreen {
			fullscreenStatus = onStyle.Render("ON")
		}

		wrapStatus := offStyle.Render("OFF")
		if m.describeView.WrapText {
			wrapStatus = onStyle.Render("ON")
		}

		lineNumsStatus := offStyle.Render("OFF")
		if m.describeView.ShowLineNumbers {
			lineNumsStatus = onStyle.Render("ON")
		}

		togglePart := fmt.Sprintf(" %s %s %s %s %s %s",
			labelStyle.Render("[Fullscreen:"),
			fullscreenStatus+labelStyle.Render("]"),
			labelStyle.Render("[Wrap:"),
			wrapStatus+labelStyle.Render("]"),
			labelStyle.Render("[Lines:"),
			lineNumsStatus+labelStyle.Render("]"),
		)

		hintPart := hintStyle.Render("esc to go back ")

		// Calculate spacing between toggle and hint
		toggleLen := lipgloss.Width(togglePart)
		hintLen := lipgloss.Width(hintPart)
		spacing := tableWidth - toggleLen - hintLen
		if spacing < 1 {
			spacing = 1
		}

		toggleLine := togglePart + strings.Repeat(" ", spacing) + hintPart

		// Truncate if still too long
		if lipgloss.Width(toggleLine) > tableWidth {
			toggleLine = ansi.Truncate(toggleLine, tableWidth, "…")
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
	selectedRow := m.table.Cursor()
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	normalStyle := lipgloss.NewStyle()

	for idx, row := range rows {
		isSelected := idx == selectedRow

		// Build the full row with each cell padded to its effective width
		// Cells that overflow their defined column width are kept at full length
		fullRowLine := ""
		for i, cell := range row {
			if i > 0 {
				fullRowLine += " "
			}
			cellText := cell

			// Colorize status/phase columns (skip for selected row so highlight extends fully)
			if !isSelected && i < len(columns) && isStatusColumn(columns[i].Title) {
				style := statusColor(cell)
				cellText = style.Render(cell)
			}

			cellWidth := lipgloss.Width(cellText)
			if i < len(effectiveWidths) && cellWidth < effectiveWidths[i] {
				// Pad to effective column width
				cellText = cellText + strings.Repeat(" ", effectiveWidths[i]-cellWidth)
			}
			fullRowLine += cellText
		}

		// Apply horizontal offset and truncate to table width
		displayLine := applyHorizontalScroll(fullRowLine, m.horizontalOffset, tableWidth)

		// Apply selection styling
		rowStyle := normalStyle
		if idx == selectedRow {
			rowStyle = selectedStyle
		}

		b.WriteString(borderStyle.Render("│"))
		b.WriteString(rowStyle.Render(displayLine))
		b.WriteString(borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Write bottom border
	bottomBorder := "└" + strings.Repeat("─", tableWidth) + "┘"
	b.WriteString(borderStyle.Render(bottomBorder))
}

// buildTopBorderWithTitle creates a top border with a centered title.
func (m *Model) buildTopBorderWithTitle(title string, width int, borderColor color.Color, titleStyle lipgloss.Style) string {
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
	// Account for borders and margin:
	// 2 chars for left/right borders (│) + 4 chars margin = 6 total overhead
	totalWidth := width - 4
	columns := resources.GetColumns(totalWidth, m.currentGVR.Resource)

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
	if m.currentGVR.Resource == k8s.ResourceLogs && m.paginator.TotalPages > 5 {
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
