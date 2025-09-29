// cmd/k10s/main.go
package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	ready bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.ready = true
		return m, nil
	default:
		return m, nil
	}
}

func (m model) View() string {
	if !m.ready {
		return "k10s: initializing...\n"
	}
	return "k10s: coming soon (press any key to exit)\n"
}

func main() {
	if err := tea.NewProgram(model{}).Start(); err != nil {
		log.Fatal(err)
	}
}
