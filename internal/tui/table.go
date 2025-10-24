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
)

// wrapTextAtWordBoundary wraps text at word boundaries when possible.
// Falls back to truncation for words longer than maxWidth.
func wrapTextAtWordBoundary(text string, maxWidth int) []string {
	if maxWidth <= 0 || runewidth.StringWidth(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	currentLine := ""
	currentWidth := 0

	words := strings.Fields(text)
	for _, word := range words {
		wordWidth := runewidth.StringWidth(word)
		spaceWidth := 1

		// Check if word fits on current line
		neededWidth := wordWidth
		if currentWidth > 0 {
			neededWidth += spaceWidth
		}

		if currentWidth+neededWidth <= maxWidth {
			// Word fits
			if currentWidth > 0 {
				currentLine += " "
				currentWidth += spaceWidth
			}
			currentLine += word
			currentWidth += wordWidth
		} else {
			// Word doesn't fit
			if currentLine != "" {
				lines = append(lines, currentLine)
			}

			// Start new line with this word
			if wordWidth <= maxWidth {
				currentLine = word
				currentWidth = wordWidth
			} else {
				// Word too long - truncate with ellipsis
				currentLine = runewidth.Truncate(word, maxWidth, "…")
				currentWidth = maxWidth
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
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
	start := m.paginator.Page * m.paginator.PerPage
	end := start + m.paginator.PerPage
	if end > len(m.resources) {
		end = len(m.resources)
	}

	pageResources := m.resources[start:end]
	rows := make([]table.Row, len(pageResources))

	for i, res := range pageResources {
		rows[i] = table.Row{
			res.Name,
			res.Namespace,
			res.Node,
			res.Status,
			res.Age,
			res.Extra,
		}
	}

	m.table.SetRows(rows)
	m.paginator.SetTotalPages(len(m.resources))
}

// updateTableDataForLogs updates table with container logs.
func (m *Model) updateTableDataForLogs() {
	start := m.paginator.Page * m.paginator.PerPage
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
	// Format line number with content
	logContent := fmt.Sprintf("%4d: %s", logLine.LineNum, logLine.Content)

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

		// Calculate available width for content
		availableWidth := logWidth - timestampWidth
		if availableWidth < 10 {
			availableWidth = 10
		}

		wrappedLines := wrapTextAtWordBoundary(logContent, availableWidth)

		for j, line := range wrappedLines {
			var displayLine string
			if j == 0 {
				displayLine = timestamp + line
			} else {
				indent := strings.Repeat(" ", timestampWidth)
				displayLine = indent + "↳ " + line
			}
			rows = append(rows, table.Row{
				displayLine,
				"", "", "", "", "",
			})
		}
	} else {
		displayLine := timestamp + logContent
		rows = append(rows, table.Row{
			displayLine,
			"", "", "", "", "",
		})
	}

	return rows
}

// renderTableWithHeader renders the table with a custom header border containing the resource type.
func (m Model) renderTableWithHeader(b *strings.Builder) {
	nsDisplay := m.currentNamespace
	if nsDisplay == "" {
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
func (m Model) buildTopBorderWithTitle(title string, width int, borderColor lipgloss.Color, titleStyle lipgloss.Style) string {
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

// getColumnTitles returns the appropriate column titles based on the resource type.
func getColumnTitles(resType k8s.ResourceType) []string {
	switch resType {
	case k8s.ResourcePods:
		return []string{"Name", "Namespace", "Node", "Status", "Age", "Pod IP"}
	case k8s.ResourceNodes:
		return []string{"Name", "", "", "Status", "Age", "Node IP"}
	case k8s.ResourceNamespaces:
		return []string{"Name", "", "", "Status", "Age", ""}
	case k8s.ResourceServices:
		return []string{"Name", "Namespace", "", "Type", "Age", "Cluster-IP/Ports"}
	case k8s.ResourceContainers:
		return []string{"Name", "Type", "Image", "Status", "Restarts", "Ready"}
	case k8s.ResourceLogs:
		return []string{"", "", "", "", "", ""}
	default:
		return []string{"Name", "Namespace", "Node", "Status", "Age", "IP"}
	}
}

// updateColumns updates the table columns based on the current width and resource type.
func (m *Model) updateColumns(totalWidth int) {
	titles := getColumnTitles(m.resourceType)

	if m.resourceType == k8s.ResourceLogs {
		logWidth := totalWidth
		m.table.SetColumns([]table.Column{
			{Title: titles[0], Width: logWidth},
			{Title: titles[1], Width: 0},
			{Title: titles[2], Width: 0},
			{Title: titles[3], Width: 0},
			{Title: titles[4], Width: 0},
			{Title: titles[5], Width: 0},
		})
		return
	}

	nameWidth := int(float64(totalWidth) * 0.30)
	nsWidth := int(float64(totalWidth) * 0.13)
	nodeWidth := int(float64(totalWidth) * 0.18)
	statusWidth := int(float64(totalWidth) * 0.12)
	ageWidth := int(float64(totalWidth) * 0.08)
	ipWidth := totalWidth - nameWidth - nsWidth - nodeWidth - statusWidth - ageWidth

	m.table.SetColumns([]table.Column{
		{Title: titles[0], Width: nameWidth},
		{Title: titles[1], Width: nsWidth},
		{Title: titles[2], Width: nodeWidth},
		{Title: titles[3], Width: statusWidth},
		{Title: titles[4], Width: ageWidth},
		{Title: titles[5], Width: ipWidth},
	})
}

// renderPagination renders the pagination display based on configured style.
func (m Model) renderPagination(b *strings.Builder) {
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
