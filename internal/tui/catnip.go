package tui

import (
	"os"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// EasterEggMode represents special seasonal themes
type EasterEggMode int

const (
	EasterEggNone EasterEggMode = iota
	EasterEggHalloween
	EasterEggChristmas
)

// detectEasterEgg checks for seasonal dates or environment variable
// This is the catnip that makes our kittens change colors! 🎃🎄
func detectEasterEgg() EasterEggMode {
	// Check environment variable first
	if easterEgg := os.Getenv("EASTER_EGG"); easterEgg != "" {
		switch strings.ToLower(easterEgg) {
		case "halloween":
			return EasterEggHalloween
		case "xmas", "christmas":
			return EasterEggChristmas
		}
	}

	// Check actual date
	now := time.Now()
	month := now.Month()
	day := now.Day()

	// Halloween: October 31 🎃
	if month == time.October && day == 31 {
		return EasterEggHalloween
	}

	// Christmas: December 25 🎄
	if month == time.December && day == 25 {
		return EasterEggChristmas
	}

	return EasterEggNone
}

// getKittenStyles returns styled kittens based on easter egg mode
// Give the kittens some catnip and watch them change colors! 🐱✨
func getKittenStyles(logo string, mode EasterEggMode) (string, string) {
	switch mode {
	case EasterEggHalloween:
		// Both cats orange for Halloween 🎃
		orangeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
		return orangeStyle.Render(logo), orangeStyle.Render(logo)

	case EasterEggChristmas:
		// One red, one green for Christmas 🎄
		redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		return redStyle.Render(logo), greenStyle.Render(logo)

	default:
		// Default pink/magenta cats
		defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		return defaultStyle.Render(logo), defaultStyle.Render(logo)
	}
}
