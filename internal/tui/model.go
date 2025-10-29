package tui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	logLines           []k8s.LogLine
	resourceType       k8s.ResourceType
	clusterInfo        *k8s.ClusterInfo
	currentNamespace   string
	commandSuggestions []string
	selectedSuggestion int
	navigationHistory  *NavigationHistory
	logView            *LogViewState
	ready              bool
	width              int
	height             int
	err                error
	commandErr         string
	commandSuccess     string
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type resourcesLoadedMsg struct {
	resources []k8s.Resource
	resType   k8s.ResourceType
	namespace string // The namespace these resources were loaded from
}

type logsLoadedMsg struct {
	logLines  []k8s.LogLine
	namespace string
}

type commandErrMsg struct {
	message string
}

type clearCommandErrMsg struct{}

type commandSuccessMsg struct {
	message string
}

type clearCommandSuccessMsg struct{}

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

	suggestions := []string{"pods", "nodes", "namespaces", "services", "ns", "quit", "q", "reconnect", "r", "cplogs", "cp"}

	keys := newKeyMap()

	// Disable log-specific keys by default (enabled only in logs view)
	keys.Fullscreen.SetEnabled(false)
	keys.Autoscroll.SetEnabled(false)
	keys.ToggleTime.SetEnabled(false)
	keys.WrapText.SetEnabled(false)
	keys.CopyLogs.SetEnabled(false)

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
		currentNamespace:   "",
		commandSuggestions: suggestions,
		selectedSuggestion: -1,
		navigationHistory:  NewNavigationHistory(),
		logView:            NewLogViewState(),
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

// ShortHelp returns context-aware short help based on current view.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Command, m.keys.Quit}
}

// FullHelp returns context-aware full help based on current view.
// Log-specific keybindings are only shown when viewing logs.
func (m Model) FullHelp() [][]key.Binding {
	base := [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.GotoTop, m.keys.GotoBottom},
		{m.keys.Left, m.keys.Right, m.keys.AllNS, m.keys.DefaultNS},
		{m.keys.Enter, m.keys.Back, m.keys.Command, m.keys.Quit},
	}

	// Only show log-specific keys when viewing logs
	if m.resourceType == k8s.ResourceLogs {
		base = append(base, []key.Binding{
			m.keys.Fullscreen, m.keys.Autoscroll, m.keys.ToggleTime, m.keys.WrapText, m.keys.CopyLogs,
		})
	}

	return base
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
		m.logLines = nil // Clear log lines when loading resources
		m.resourceType = msg.resType
		m.currentNamespace = msg.namespace

		// Update key bindings for new resource type
		m.updateKeysForResourceType()

		if m.width > 0 {
			totalWidth := m.width - 7
			if totalWidth < 90 {
				totalWidth = 90
			}
			m.updateColumns(totalWidth)
		}
		m.updateTableData()
		return m, nil

	case logsLoadedMsg:
		m.logLines = msg.logLines
		m.resources = nil // Clear resources when loading logs
		m.resourceType = k8s.ResourceLogs
		m.currentNamespace = msg.namespace

		// Update key bindings for logs view
		m.updateKeysForResourceType()

		if m.width > 0 {
			totalWidth := m.width - 7
			if totalWidth < 90 {
				totalWidth = 90
			}
			m.updateColumns(totalWidth)
		}

		// Jump to last page for tailing behavior
		if len(m.logLines) > 0 {
			lastPage := (len(m.logLines) - 1) / m.paginator.PerPage
			m.paginator.Page = lastPage
		}

		m.updateTableData()

		if m.logView.Autoscroll {
			m.table.GotoBottom()
		}

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

	case commandSuccessMsg:
		m.commandSuccess = msg.message
		// Clear the success message after 5 seconds
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return clearCommandSuccessMsg{}
		})

	case clearCommandSuccessMsg:
		m.commandSuccess = ""
		return m, nil

	case logsCopiedMsg:
		if msg.success {
			m.commandSuccess = msg.message
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return clearCommandSuccessMsg{}
			})
		} else {
			m.commandErr = msg.message
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return clearCommandErrMsg{}
			})
		}

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
			case ":":
				m.viewMode = ViewModeCommand
				m.commandInput.Focus()
				return m, nil
			case "0":
				if !m.isConnected() {
					m.err = fmt.Errorf("not connected to cluster. Use :reconnect")
					return m, nil
				}
				if !isNamespaceAware(m.resourceType) {
					return m, nil // No-op for non-namespace-aware resources
				}
				m.currentNamespace = ""
				m.paginator.Page = 0
				m.err = nil
				return m, m.loadResourcesCmd(m.resourceType)
			case "d":
				if !m.isConnected() {
					m.err = fmt.Errorf("not connected to cluster. Use :reconnect")
					return m, nil
				}
				if !isNamespaceAware(m.resourceType) {
					return m, nil // No-op for non-namespace-aware resources
				}
				if m.clusterInfo != nil {
					m.currentNamespace = m.clusterInfo.Namespace
					m.paginator.Page = 0
					m.err = nil
					return m, m.loadResourcesCmd(m.resourceType)
				}
				return m, nil
			case "enter":
				if m.resourceType == k8s.ResourceLogs {
					return m, nil
				}

				if len(m.resources) == 0 {
					return m, nil
				}
				actualIdx := m.paginator.Page*m.paginator.PerPage + m.table.Cursor()
				if actualIdx >= len(m.resources) {
					return m, nil
				}
				selectedResource := m.resources[actualIdx]

				memento := m.saveToMemento(selectedResource.Name, selectedResource.Namespace)
				m.navigationHistory.Push(memento)

				return m, m.drillDown(selectedResource)

			case "esc", "escape":
				memento := m.navigationHistory.Pop()
				if memento != nil {
					m.restoreFromMemento(memento)
				} else {
					return m, m.loadResourcesWithNamespace(k8s.ResourcePods, "")
				}
				return m, nil
			case "f":
				if m.resourceType == k8s.ResourceLogs {
					m.logView.Fullscreen = !m.logView.Fullscreen
				}
				return m, nil
			case "s":
				if m.resourceType == k8s.ResourceLogs {
					m.logView.Autoscroll = !m.logView.Autoscroll
					if m.logView.Autoscroll {
						m.table.GotoBottom()
					}
				}
				return m, nil
			case "t":
				if m.resourceType == k8s.ResourceLogs {
					m.logView.ShowTimestamps = !m.logView.ShowTimestamps
					m.updateTableData()
				}
				return m, nil
			case "w":
				if m.resourceType == k8s.ResourceLogs {
					m.logView.WrapText = !m.logView.WrapText
					m.updateTableData()
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
				// Go to first line of first page (absolute first line)
				// Disable autoscroll when manually navigating to top
				if m.resourceType == k8s.ResourceLogs {
					m.logView.Autoscroll = false
				}
				m.paginator.Page = 0
				m.updateTableData()
				m.table.GotoTop()
				return m, nil
			case "G":
				// Go to last line of last page (absolute last line)
				if m.resourceType == k8s.ResourceLogs {
					// For logs, go to last page and enable autoscroll for tailing
					totalLogs := len(m.logLines)
					if totalLogs > 0 {
						lastPage := (totalLogs - 1) / m.paginator.PerPage
						m.paginator.Page = lastPage
						m.updateTableData()
						m.table.GotoBottom()
						m.logView.Autoscroll = true
					}
				} else {
					// For resources, go to last page
					totalResources := len(m.resources)
					if totalResources > 0 {
						lastPage := (totalResources - 1) / m.paginator.PerPage
						m.paginator.Page = lastPage
						m.updateTableData()
						m.table.GotoBottom()
					}
				}
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

	// Skip top header if fullscreen is enabled for logs
	if !m.logView.Fullscreen || m.resourceType != k8s.ResourceLogs {
		m.renderTopHeader(&b)
		b.WriteString("\n\n")
	}

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

	// Render breadcrumb navigation if we're in a drilled-down view
	if m.navigationHistory.Len() > 0 {
		b.WriteString("\n")
		m.renderBreadcrumb(&b)
	}

	// Render pagination based on configured style
	if m.getTotalItems() > m.paginator.PerPage {
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

	// Calculate command success height (if shown)
	commandSuccessLines := 0
	var commandSuccessContent string
	if m.commandSuccess != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
		commandSuccessContent = successStyle.Render(fmt.Sprintf("✓ %s", m.commandSuccess))
		commandSuccessLines = 2 // Success line + padding
	}

	// Fill remaining height to push command palette/error/success to bottom
	output := b.String()
	if m.height > 0 {
		renderedLines := strings.Count(output, "\n") + 1
		totalNeeded := m.height - commandPaletteLines - commandErrorLines - commandSuccessLines

		if renderedLines < totalNeeded {
			remainingLines := totalNeeded - renderedLines
			output += strings.Repeat("\n", remainingLines)
		}
	}

	if m.viewMode == ViewModeCommand {
		output += "\n" + commandPaletteContent + "\n"
	} else if m.commandErr != "" {
		output += "\n" + commandErrorContent + "\n"
	} else if m.commandSuccess != "" {
		output += "\n" + commandSuccessContent + "\n"
	}

	return output
}

// renderBreadcrumb renders the navigation breadcrumb showing the hierarchy path.
func (m Model) renderBreadcrumb(b *strings.Builder) {
	if m.navigationHistory.Len() == 0 {
		return
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	brightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var parts []string

	breadcrumb := m.navigationHistory.GetBreadcrumb()
	for i, level := range breadcrumb {
		var part string
		if i == len(breadcrumb)-1 {
			part = brightStyle.Render(string(level.ResourceType))
			if level.ResourceName != "" {
				part += separatorStyle.Render(" > ") + brightStyle.Render(level.ResourceName)
			}
		} else {
			part = dimStyle.Render(string(level.ResourceType))
			if level.ResourceName != "" {
				part += separatorStyle.Render(" > ") + dimStyle.Render(level.ResourceName)
			}
		}
		parts = append(parts, part)
	}

	parts = append(parts, brightStyle.Render(string(m.resourceType)))

	breadcrumbStr := strings.Join(parts, separatorStyle.Render(" > "))
	b.WriteString(dimStyle.Render("Path: ") + breadcrumbStr)
}

// saveToMemento creates and returns a memento containing the current state of the Model.
func (m Model) saveToMemento(selectedResourceName, selectedNamespace string) *ModelMemento {
	return &ModelMemento{
		resources:        m.resources,
		resourceType:     m.resourceType,
		currentNamespace: m.currentNamespace,
		tableCursor:      m.table.Cursor(),
		paginatorPage:    m.paginator.Page,
		err:              m.err,
		logView:          m.logView,
		resourceName:     selectedResourceName,
		namespace:        selectedNamespace,
	}
}

// restoreFromMemento restores the Model's state from a memento.
func (m *Model) restoreFromMemento(memento *ModelMemento) {
	if memento == nil {
		return
	}

	m.resources = memento.resources
	m.resourceType = memento.resourceType
	m.currentNamespace = memento.currentNamespace
	m.err = memento.err
	m.logView = memento.logView

	// Update key bindings for restored resource type
	m.updateKeysForResourceType()

	// Update pagination
	m.paginator.Page = memento.paginatorPage
	m.paginator.SetTotalPages(len(m.resources))

	// Update table columns and data
	if m.width > 0 {
		totalWidth := m.width - 7
		if totalWidth < 90 {
			totalWidth = 90
		}
		m.updateColumns(totalWidth)
	}
	m.updateTableData()

	maxCursor := len(m.table.Rows()) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}
	if memento.tableCursor > maxCursor {
		m.table.SetCursor(maxCursor)
	} else {
		m.table.SetCursor(memento.tableCursor)
	}
}
