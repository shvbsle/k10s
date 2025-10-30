package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTopHeader renders the appropriate header based on terminal height.
// Three stages: Full (≥30 lines), Compact (20-29 lines), Minimal (<20 lines).
func (m *Model) renderTopHeader(b *strings.Builder) {
	// Stage 1 (Full) = >= 30: everything including CPU/MEM
	// Stage 2 (Compact) = 20-30: 4 lines - info + help + kittens (no CPU/MEM)
	// Stage 3 (Minimal) = < 20: just context + hint (future implementation)
	if m.viewHeight < 30 {
		m.renderCompactHeader(b)
	} else {
		m.renderFullHeader(b)
	}
}

// renderCompactHeader shows 4-line header: info + help + kittens (no CPU/MEM).
func (m *Model) renderCompactHeader(b *strings.Builder) {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	statusColor := "46" // green
	if !m.isConnected() {
		statusColor = "203" // red
	}
	statusIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Bold(true).
		Render("●")

	// Build compact info (only 4 lines, no CPU/MEM)
	var infoContent strings.Builder
	if m.clusterInfo != nil {
		infoContent.WriteString(labelStyle.Render("Context: ") + valueStyle.Render(m.clusterInfo.Context) + "\n")
		infoContent.WriteString(labelStyle.Render("Cluster: ") + valueStyle.Render(m.clusterInfo.Cluster) + "\n")
		infoContent.WriteString(labelStyle.Render("K10s Ver: ") + valueStyle.Render(Version))
	}

	infoBlock := statusIndicator + " " + infoContent.String()
	helpBlock := m.help.View(m)

	// Apply easter egg colors! 🎃🎄
	easterEgg := detectEasterEgg()
	kitten1, kitten2 := getKittenStyles(m.config.Logo, easterEgg)
	doubleKitten := lipgloss.JoinHorizontal(lipgloss.Top, kitten1, " ", kitten2)

	termWidth := max(m.viewWidth, 80)

	infoBlockWidth := lipgloss.Width(infoBlock)
	helpBlockWidth := lipgloss.Width(helpBlock)
	doubleKittenWidth := lipgloss.Width(doubleKitten)

	const minGap = 2
	totalContentWidth := infoBlockWidth + helpBlockWidth + doubleKittenWidth + (minGap * 2)

	// Use natural widths if content fits, otherwise constrain with max widths
	if totalContentWidth <= termWidth {
		gap1 := minGap
		gap2 := max(termWidth-infoBlockWidth-helpBlockWidth-doubleKittenWidth-gap1, minGap)

		header := lipgloss.JoinHorizontal(lipgloss.Top,
			infoBlock,
			strings.Repeat(" ", gap1),
			helpBlock,
			strings.Repeat(" ", gap2),
			doubleKitten,
		)
		b.WriteString(header)
	} else {
		maxInfoWidth := int(float64(termWidth) * 0.25)
		maxHelpWidth := int(float64(termWidth) * 0.45)
		kittenSpace := doubleKittenWidth + minGap

		if maxInfoWidth < 20 {
			maxInfoWidth = 20
		}
		if maxHelpWidth < 30 {
			maxHelpWidth = 30
		}

		infoStyled := lipgloss.NewStyle().MaxWidth(maxInfoWidth).Render(infoBlock)
		helpStyled := lipgloss.NewStyle().MaxWidth(maxHelpWidth).Render(helpBlock)

		actualInfoWidth := lipgloss.Width(infoStyled)
		actualHelpWidth := lipgloss.Width(helpStyled)

		remainingSpace := max(termWidth-actualInfoWidth-actualHelpWidth-kittenSpace, 0)

		header := lipgloss.JoinHorizontal(lipgloss.Top,
			infoStyled,
			strings.Repeat(" ", minGap),
			helpStyled,
			strings.Repeat(" ", remainingSpace),
			doubleKitten,
		)
		b.WriteString(header)
	}
}

// renderFullHeader shows everything including kittens (for large terminals).
func (m *Model) renderFullHeader(b *strings.Builder) {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	statusColor := "46" // green
	if !m.isConnected() {
		statusColor = "203" // red
	}
	statusIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Bold(true).
		Render("●")

	var infoContent strings.Builder
	if m.clusterInfo != nil {
		infoContent.WriteString(labelStyle.Render("Context: ") + valueStyle.Render(m.clusterInfo.Context) + "\n")
		infoContent.WriteString(labelStyle.Render("Cluster: ") + valueStyle.Render(m.clusterInfo.Cluster) + "\n")
		infoContent.WriteString(labelStyle.Render("K10s Ver: ") + valueStyle.Render(Version) + "\n")
		infoContent.WriteString(labelStyle.Render("K8s Ver: ") + valueStyle.Render(m.clusterInfo.K8sVersion) + "\n")
	}
	if len(m.listOptions.FieldSelector) > 0 {
		infoContent.WriteString(labelStyle.Render("FieldSelector: ") + valueStyle.Render(m.listOptions.FieldSelector) + "\n")
	}
	if len(m.listOptions.LabelSelector) > 0 {
		infoContent.WriteString(labelStyle.Render("LabelSelector: ") + valueStyle.Render(m.listOptions.LabelSelector) + "\n")
	}
	infoContent.WriteString(labelStyle.Render("CPU: ") + errorStyle.Render("n/a") + "\n")
	infoContent.WriteString(labelStyle.Render("MEM: ") + errorStyle.Render("n/a"))

	infoBlock := statusIndicator + " " + infoContent.String()
	helpBlock := m.help.View(m)

	// Apply easter egg colors! 🎃🎄
	easterEgg := detectEasterEgg()
	kitten1, kitten2 := getKittenStyles(m.config.Logo, easterEgg)
	doubleKitten := lipgloss.JoinHorizontal(lipgloss.Top, kitten1, " ", kitten2)

	termWidth := max(m.viewWidth, 80)

	infoBlockWidth := lipgloss.Width(infoBlock)
	helpBlockWidth := lipgloss.Width(helpBlock)
	doubleKittenWidth := lipgloss.Width(doubleKitten)

	const minGap = 2
	totalContentWidth := infoBlockWidth + helpBlockWidth + doubleKittenWidth + (minGap * 2)

	// Use natural widths if content fits, otherwise constrain with max widths
	if totalContentWidth <= termWidth {
		gap1 := minGap
		gap2 := max(termWidth-infoBlockWidth-helpBlockWidth-doubleKittenWidth-gap1, minGap)

		header := lipgloss.JoinHorizontal(lipgloss.Top,
			infoBlock,
			strings.Repeat(" ", gap1),
			helpBlock,
			strings.Repeat(" ", gap2),
			doubleKitten,
		)
		b.WriteString(header)
	} else {
		maxInfoWidth := int(float64(termWidth) * 0.25)
		maxHelpWidth := int(float64(termWidth) * 0.45)
		kittenSpace := doubleKittenWidth + minGap

		if maxInfoWidth < 20 {
			maxInfoWidth = 20
		}
		if maxHelpWidth < 30 {
			maxHelpWidth = 30
		}

		infoStyled := lipgloss.NewStyle().MaxWidth(maxInfoWidth).Render(infoBlock)
		helpStyled := lipgloss.NewStyle().MaxWidth(maxHelpWidth).Render(helpBlock)

		actualInfoWidth := lipgloss.Width(infoStyled)
		actualHelpWidth := lipgloss.Width(helpStyled)

		remainingSpace := max(termWidth-actualInfoWidth-actualHelpWidth-kittenSpace, 0)

		header := lipgloss.JoinHorizontal(lipgloss.Top,
			infoStyled,
			strings.Repeat(" ", minGap),
			helpStyled,
			strings.Repeat(" ", remainingSpace),
			doubleKitten,
		)
		b.WriteString(header)
	}
}
