package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// renderTopHeader renders a minimal header with just the logo and essential info.
// Press ? to see full help and cluster information.
func (m *Model) renderTopHeader(b *strings.Builder) {
	m.renderMinimalHeader(b)
}

// renderMinimalHeader shows a clean 3-row header matching the logo height.
// Column 1: Context, Cluster, k10s Ver
// Column 2: Key hints (?, :, esc)
// Column 3: Logo (kittens)
func (m *Model) renderMinimalHeader(b *strings.Builder) {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	statusColor := "46" // green
	if !m.isConnected() {
		statusColor = "203" // red
	}
	statusIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Bold(true).
		Render("●")

	// Column 1: Cluster info (3 rows)
	var infoLines [3]string
	if m.clusterInfo != nil {
		infoLines[0] = labelStyle.Render("Context: ") + valueStyle.Render(m.clusterInfo.Context)
		infoLines[1] = labelStyle.Render("Cluster: ") + valueStyle.Render(m.clusterInfo.Cluster)
		infoLines[2] = labelStyle.Render("k10s:    ") + valueStyle.Render(Version)
	} else {
		infoLines[0] = valueStyle.Render("disconnected")
		infoLines[1] = ""
		infoLines[2] = labelStyle.Render("k10s: ") + valueStyle.Render(Version)
	}
	infoBlock := statusIndicator + " " + infoLines[0] + "\n  " + infoLines[1] + "\n  " + infoLines[2]

	// Column 2: Key hints (3 rows)
	keyHints := keyStyle.Render("?") + hintStyle.Render(" help") + "\n" +
		keyStyle.Render(":") + hintStyle.Render(" command") + "\n" +
		keyStyle.Render("esc") + hintStyle.Render(" go back")

	// Column 3: Logo (kittens)
	easterEgg := detectEasterEgg()
	kitten1, kitten2 := getKittenStyles(m.config.Logo, easterEgg)
	doubleKitten := lipgloss.JoinHorizontal(lipgloss.Top, kitten1, " ", kitten2)

	termWidth := max(m.viewWidth, 80)

	infoBlockWidth := lipgloss.Width(infoBlock)
	keyHintsWidth := lipgloss.Width(keyHints)
	doubleKittenWidth := lipgloss.Width(doubleKitten)

	const minGap = 4
	totalContentWidth := infoBlockWidth + keyHintsWidth + doubleKittenWidth + (minGap * 2)

	if totalContentWidth <= termWidth {
		gap1 := minGap
		gap2 := max(termWidth-infoBlockWidth-keyHintsWidth-doubleKittenWidth-gap1, minGap)

		header := lipgloss.JoinHorizontal(lipgloss.Top,
			infoBlock,
			strings.Repeat(" ", gap1),
			keyHints,
			strings.Repeat(" ", gap2),
			doubleKitten,
		)
		b.WriteString(header)
	} else {
		// Constrained layout for narrow terminals - skip kittens
		header := lipgloss.JoinHorizontal(lipgloss.Top,
			infoBlock,
			strings.Repeat(" ", minGap),
			keyHints,
		)
		b.WriteString(header)
	}
}
