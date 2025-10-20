package tui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
)

// Version is the current version of k10s.
const Version = "v0.1.0"

// ViewMode represents the current input mode of the TUI.
type ViewMode int

const (
	// ViewModeNormal is the default mode with vim-style navigation keybindings.
	ViewModeNormal ViewMode = iota
	// ViewModeCommand is the command entry mode activated by pressing ':'.
	ViewModeCommand
)

// Model represents the state of the k10s TUI application, including the current
// view, resource data, cluster connection status, and UI components.
type Model struct {
	config             *config.Config
	k8sClient          *k8s.Client
	table              table.Model
	paginator          paginator.Model
	commandInput       textinput.Model
	help               help.Model
	keys               keyMap
	viewMode           ViewMode
	resources          []k8s.Resource
	resourceType       k8s.ResourceType
	clusterInfo        *k8s.ClusterInfo
	currentNamespace   string // "" = all namespaces, otherwise specific namespace
	commandSuggestions []string
	selectedSuggestion int
	ready              bool
	width              int
	height             int
	err                error
	commandErr         string // Command-specific error shown at bottom for 5s
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type resourcesLoadedMsg struct {
	resources []k8s.Resource
	resType   k8s.ResourceType
}

type commandErrMsg struct {
	message string
}

type clearCommandErrMsg struct{}

// New creates a new TUI model with the provided configuration and Kubernetes client.
// The client may be nil or disconnected - the TUI will handle this gracefully and
// display appropriate status messages.
func New(cfg *config.Config, client *k8s.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.CharLimit = 100
	ti.Width = 50

	// Initial columns for pods (default resource type)
	titles := getColumnTitles(k8s.ResourcePods)
	columns := []table.Column{
		{Title: titles[0], Width: 35},
		{Title: titles[1], Width: 15},
		{Title: titles[2], Width: 20},
		{Title: titles[3], Width: 12},
		{Title: titles[4], Width: 8},
		{Title: titles[5], Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(cfg.PageSize),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.HiddenBorder()).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = cfg.PageSize
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")

	// Works even when disconnected
	var clusterInfo *k8s.ClusterInfo
	if client != nil {
		clusterInfo, _ = client.GetClusterInfo()
	}

	suggestions := []string{"pods", "nodes", "namespaces", "services", "ns", "quit", "q", "reconnect", "r"}

	keys := newKeyMap()

	h := help.New()
	h.ShowAll = true // Show full help by default
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	return Model{
		config:             cfg,
		k8sClient:          client,
		table:              t,
		paginator:          p,
		commandInput:       ti,
		help:               h,
		keys:               keys,
		viewMode:           ViewModeNormal,
		resourceType:       k8s.ResourcePods,
		clusterInfo:        clusterInfo,
		currentNamespace:   "", // Start with all namespaces
		commandSuggestions: suggestions,
		selectedSuggestion: -1,
	}
}

// Init initializes the TUI model and returns the initial command to run.
// It attempts to load pods if the client is connected.
func (m Model) Init() tea.Cmd {
	// Only try to load resources if connected
	if m.isConnected() {
		return tea.Batch(
			m.loadResourcesCmd(k8s.ResourcePods),
		)
	}
	return nil
}

// Update handles messages and updates the model state accordingly.
// It implements the tea.Model interface for Bubble Tea.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Dynamic header height: 20 lines base, adjusted for very tall/short terminals
		baseHeaderHeight := 20
		headerHeight := baseHeaderHeight
		if m.height > 50 {
			headerHeight = 22
		} else if m.height < 30 {
			headerHeight = 15
		}

		tableHeight := m.height - headerHeight
		if tableHeight < 5 {
			tableHeight = 5
		}
		m.table.SetHeight(tableHeight)

		// Rough calc for border renders:
		// Total overhead: 2 (borders) + 5 (column spacing) = 7
		totalWidth := m.width - 7
		if totalWidth < 90 {
			totalWidth = 90
		}

		m.updateColumns(totalWidth)

		return m, func() tea.Msg {
			return tea.ClearScreen()
		}

	case resourcesLoadedMsg:
		m.resources = msg.resources
		m.resourceType = msg.resType
		// Update column headers based on new resource type
		if m.width > 0 {
			totalWidth := m.width - 7
			if totalWidth < 90 {
				totalWidth = 90
			}
			m.updateColumns(totalWidth)
		}
		m.updateTableData()
		return m, nil

	case errMsg:
		log.Printf("TUI: Error occurred: %v", msg.err)
		m.err = msg.err
		return m, nil

	case commandErrMsg:
		m.commandErr = msg.message
		// Clear the error after 5 seconds
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return clearCommandErrMsg{}
		})

	case clearCommandErrMsg:
		m.commandErr = ""
		return m, nil

	case tea.KeyMsg:
		if m.viewMode == ViewModeCommand {
			switch msg.String() {
			case "enter":
				command := strings.TrimSpace(m.commandInput.Value())
				m.commandInput.Reset()
				m.viewMode = ViewModeNormal
				return m, m.executeCommand(command)
			case "esc":
				m.commandInput.Reset()
				m.viewMode = ViewModeNormal
				return m, nil
			case "tab":
				// Autocomplete with the first matching suggestion
				filtered := m.getFilteredSuggestions()
				if len(filtered) > 0 {
					m.commandInput.SetValue(filtered[0])
					m.commandInput.SetCursor(len(filtered[0]))
				}
				return m, nil
			default:
				m.commandInput, cmd = m.commandInput.Update(msg)
				return m, cmd
			}
		} else {
			// Handle keys explicitly to prevent double-processing
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case ":":
				m.viewMode = ViewModeCommand
				m.commandInput.Focus()
				return m, nil
			case "0":
				if !m.isConnected() {
					m.err = fmt.Errorf("not connected to cluster. Use :reconnect")
					log.Printf("TUI: User attempted to switch to all namespaces while disconnected")
					return m, nil
				}
				m.currentNamespace = ""
				m.paginator.Page = 0 // Avoid slice bounds panic
				m.err = nil
				log.Printf("TUI: Switched to all namespaces")
				return m, m.loadResourcesCmd(m.resourceType)
			case "d":
				if !m.isConnected() {
					m.err = fmt.Errorf("not connected to cluster. Use :reconnect")
					log.Printf("TUI: User attempted to switch to default namespace while disconnected")
					return m, nil
				}
				if m.clusterInfo != nil {
					m.currentNamespace = m.clusterInfo.Namespace
					m.paginator.Page = 0 // Avoid slice bounds panic
					m.err = nil
					log.Printf("TUI: Switched to default namespace (%s)", m.clusterInfo.Namespace)
					return m, m.loadResourcesCmd(m.resourceType)
				}
				return m, nil
			case "y":
				// Yank/copy selected row (if implemented in future)
				return m, nil
			case "j", "down":
				// Handle navigation directly to prevent double-processing
				m.table.MoveDown(1)
				return m, nil
			case "k", "up":
				// Handle navigation directly to prevent double-processing
				m.table.MoveUp(1)
				return m, nil
			case "g":
				// Handle navigation directly to prevent double-processing
				m.table.GotoTop()
				return m, nil
			case "G":
				// Handle navigation directly to prevent double-processing
				m.table.GotoBottom()
				return m, nil
			case "h", "left", "pgup":
				if m.paginator.Page > 0 {
					m.paginator.PrevPage()
					m.updateTableData()
				}
				return m, nil
			case "l", "right", "pgdown":
				if m.paginator.Page < m.paginator.TotalPages-1 {
					m.paginator.NextPage()
					m.updateTableData()
				}
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			// For unhandled keys in normal mode, pass to table
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
	}

	// Only update table for non-key messages
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) isConnected() bool {
	return m.k8sClient != nil && m.k8sClient.IsConnected()
}

// View renders the current state of the TUI to a string.
// It implements the tea.Model interface for Bubble Tea.
func (m Model) View() string {
	if !m.ready {
		return "Initializing k10s..."
	}

	var b strings.Builder

	b.WriteString("\n")

	m.renderTopHeader(&b)
	b.WriteString("\n\n")

	// Only show connection errors at the top (command errors shown in command palette)
	if !m.isConnected() {
		if m.err != nil {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
			b.WriteString(errorStyle.Render(fmt.Sprintf("⚠ Error: %v", m.err)))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" (try :reconnect)"))
			b.WriteString("\n\n")
		} else {
			warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
			b.WriteString(warningStyle.Render("⚠ Disconnected from cluster"))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" - use :reconnect to connect"))
			b.WriteString("\n")
		}
	}

	m.renderTableWithHeader(&b)
	b.WriteString("\n")

	// Render pagination based on configured style
	if len(m.resources) > m.paginator.PerPage {
		b.WriteString("\n")
		m.renderPagination(&b)
	}

	// Calculate command palette height (if shown)
	commandPaletteLines := 0
	var commandPaletteContent string
	if m.viewMode == ViewModeCommand {
		var cmdBuilder strings.Builder
		m.renderCommandInput(&cmdBuilder)
		commandPaletteContent = cmdBuilder.String()
		// Count actual newlines in the content plus padding (before and after)
		commandPaletteLines = strings.Count(commandPaletteContent, "\n") + 2
	}

	// Calculate command error height (if shown)
	commandErrorLines := 0
	var commandErrorContent string
	if m.commandErr != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
		commandErrorContent = errorStyle.Render(fmt.Sprintf("⚠ %s", m.commandErr))
		commandErrorLines = 2 // Error line + padding
	}

	// Fill remaining height to push command palette/error to bottom
	output := b.String()
	if m.height > 0 {
		renderedLines := strings.Count(output, "\n") + 1
		totalNeeded := m.height - commandPaletteLines - commandErrorLines

		if renderedLines < totalNeeded {
			remainingLines := totalNeeded - renderedLines
			output += strings.Repeat("\n", remainingLines)
		}
	}

	if m.viewMode == ViewModeCommand {
		output += "\n" + commandPaletteContent + "\n"
	} else if m.commandErr != "" {
		output += "\n" + commandErrorContent + "\n"
	}

	return output
}
