package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keybindings for the TUI
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	GotoTop    key.Binding
	GotoBottom key.Binding
	AllNS      key.Binding
	DefaultNS  key.Binding
	Enter      key.Binding
	Back       key.Binding
	Command    key.Binding
	Quit       key.Binding
	// Log view specific bindings
	Fullscreen key.Binding
	Autoscroll key.Binding
	ToggleTime key.Binding
	WrapText   key.Binding
	CopyLogs   key.Binding
}

// newKeyMap creates a new keyMap with all bindings configured
func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h", "pgup"),
			key.WithHelp("←/h", "previous"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l", "pgdown"),
			key.WithHelp("→/l", "next"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		AllNS: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "all ns"),
		),
		DefaultNS: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "default ns"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "drill down"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "escape"),
			key.WithHelp("esc", "go back"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Fullscreen: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fullscreen"),
		),
		Autoscroll: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "autoscroll"),
		),
		ToggleTime: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "timestamps"),
		),
		WrapText: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "wrap"),
		),
		CopyLogs: key.NewBinding(
			key.WithKeys(":cplogs"),
			key.WithHelp(":cplogs", "copy logs [all] [path]"),
		),
	}
}
