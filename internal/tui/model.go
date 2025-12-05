package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/samber/lo"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/log"
	"github.com/shvbsle/k10s/internal/plugins"
	"github.com/shvbsle/k10s/internal/tui/cli"
	"github.com/shvbsle/k10s/internal/tui/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
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
	// 3rd party UI components
	table        table.Model
	paginator    paginator.Model
	commandInput textinput.Model
	help         help.Model

	// 1st part UI components
	config           *config.Config
	commandSuggester cli.Suggester
	commandHistory   cli.History
	keys             keyMap
	updateTableChan  chan struct{}

	// cluster info and state
	k8sClient         *k8s.Client
	currentGVR        schema.GroupVersionResource
	resourceWatcher   watch.Interface
	resources         []k8s.OrderedResourceFields
	listOptions       metav1.ListOptions
	clusterInfo       *k8s.ClusterInfo
	logLines          []k8s.LogLine
	describeContent   string
	currentNamespace  string
	navigationHistory *NavigationHistory
	logView           *LogViewState
	describeView      *DescribeViewState
	ready             bool
	viewMode          ViewMode
	viewWidth         int
	viewHeight        int
	err               error
	commandErr        string
	commandSuccess    string
	pluginRegistry    *plugins.Registry
	pluginToLaunch    plugins.Plugin
}

func (m *Model) tryQueueTableUpdate() bool {
	select {
	case m.updateTableChan <- struct{}{}:
		return true
	default:
		return false
	}
}

type updateTableMsg struct{}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type resourcesLoadedMsg struct {
	resources   []k8s.OrderedResourceFields
	gvr         schema.GroupVersionResource
	namespace   string
	listOptions metav1.ListOptions
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

type resourceDescribedMsg struct {
	yamlContent  string
	resourceName string
	namespace    string
	gvr          schema.GroupVersionResource
}

// New creates a new TUI model with the provided configuration and Kubernetes client.
// The client may be nil or disconnected - the TUI will handle this gracefully and
// display appropriate status messages.
func New(cfg *config.Config, client *k8s.Client, registry *plugins.Registry) *Model {
	if registry == nil {
		registry = plugins.NewRegistry()
	}

	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.CharLimit = 100
	ti.SetWidth(50)

	// Initial columnMap for pods (default resource type)
	columns := resources.GetColumns(100, k8s.ResourcePods)

	// Use a reasonable initial height (will be updated immediately on first WindowSizeMsg)
	// We use 20 as a temporary value just for initialization
	initialHeight := 20

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(initialHeight),
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
	p.PerPage = initialHeight
	p.ActiveDot = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("235"), Dark: lipgloss.Color("252")}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("250"), Dark: lipgloss.Color("238")}).Render("•")

	// Works even when disconnected
	var clusterInfo *k8s.ClusterInfo
	if client != nil {
		clusterInfo, _ = client.GetClusterInfo()
	}

	keys := newKeyMap()

	// Disable log-specific keys by default (enabled only in logs view)
	keys.Fullscreen.SetEnabled(false)
	keys.Autoscroll.SetEnabled(false)
	keys.ToggleTime.SetEnabled(false)
	keys.WrapText.SetEnabled(false)
	keys.CopyLogs.SetEnabled(false)
	keys.ToggleLineNums.SetEnabled(false)

	h := help.New()
	h.ShowAll = true // Show full help by default
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Fetch available resources once for both "resource" and "rs" commands
	availableResources := lo.Map(cli.GetServerGVRs(client.Discovery()), func(gvr schema.GroupVersionResource, _ int) string {
		return k8s.FormatGVR(gvr)
	})

	return &Model{
		config:           cfg,
		k8sClient:        client,
		table:            t,
		paginator:        p,
		commandInput:     ti,
		help:             h,
		keys:             keys,
		updateTableChan:  make(chan struct{}, 1000), // can only queue 1000
		viewMode:         ViewModeNormal,
		currentGVR:       schema.GroupVersionResource{Resource: k8s.ResourcePods},
		clusterInfo:      clusterInfo,
		currentNamespace: metav1.NamespaceAll,
		commandSuggester: cli.ParseSuggestionTree(
			lo.Assign(
				// built-ins
				map[string]any{
					"q":         struct{}{},
					"quit":      struct{}{},
					"r":         struct{}{},
					"reconnect": struct{}{},
					"cp":        struct{}{},
					"cplogs":    struct{}{},
				},
				// kubernetes resources
				map[string]any{
					"resource": availableResources,
					"rs":       availableResources,
				},
				// plugins
				lo.SliceToMap(registry.CommandSuggestions(), func(suggestion string) (string, any) {
					return suggestion, struct{}{}
				}),
			),
		),
		commandHistory:    cli.NewCommandHistory(100),
		navigationHistory: NewNavigationHistory(),
		logView:           NewLogViewState(),
		describeView:      NewDescribeViewState(),
		pluginRegistry:    registry,
	}
}

// Init initializes the TUI model and returns the initial command to run.
// It attempts to load pods if the client is connected.
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// bootstrap the update table event loop.
	cmds = append(cmds, func() tea.Msg { return updateTableMsg{} })

	// Only try to load resources if connected
	if m.isConnected() {
		cmds = append(cmds, m.loadResources(k8s.ResourcePods))
	}

	return tea.Batch(cmds...)
}

func (m *Model) GetPluginToLaunch() plugins.Plugin {
	return m.pluginToLaunch
}

// ShortHelp returns context-aware short help based on current view.
func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Command, m.keys.Quit}
}

// FullHelp returns context-aware full help based on current view.
// Log-specific keybindings are only shown when viewing logs.
func (m *Model) FullHelp() [][]key.Binding {
	base := [][]key.Binding{
		{m.keys.Up, m.keys.Down, m.keys.GotoTop, m.keys.GotoBottom},
		{m.keys.Left, m.keys.Right, m.keys.AllNS, m.keys.DefaultNS},
		{m.keys.Enter, m.keys.Back, m.keys.Command, m.keys.Quit},
	}

	// Only show log-specific keys when viewing logs
	if m.currentGVR.Resource == k8s.ResourceLogs {
		base = append(base, []key.Binding{
			m.keys.Fullscreen, m.keys.Autoscroll, m.keys.ToggleTime, m.keys.WrapText, m.keys.CopyLogs,
		})
	}

	return base
}

// Update handles messages and updates the model state accordingly.
// It implements the tea.Model interface for Bubble Tea.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case updateTableMsg:
		return m, func() tea.Msg {
			// block on someone sending the update message.
			<-m.updateTableChan
			// run the necessary table view update calls.
			m.updateColumns(m.viewWidth)
			m.updateTableData()
			// recursively send the update message to keep the request queued.
			return updateTableMsg{}
		}
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
		m.ready = true

		// Dynamic header height: 20 lines base, adjusted for very tall/short terminals
		baseHeaderHeight := 20
		headerHeight := baseHeaderHeight
		if m.viewHeight > 50 {
			headerHeight = 22
		} else if m.viewHeight < 30 {
			headerHeight = 15
		}

		tableHeight := max(m.viewHeight-headerHeight, 5)
		m.table.SetHeight(tableHeight)

		// Dynamic page size calculation:
		// - Describe view always uses full tableHeight
		// - If MaxPageSize is 0 (auto/default), use all available tableHeight
		// - If MaxPageSize is set to a specific number, use it as a ceiling (but never exceed tableHeight)
		if m.currentGVR.Resource == k8s.ResourceDescribe {
			// Describe view always uses full height
			m.paginator.PerPage = tableHeight
		} else if m.config.MaxPageSize == config.AutoPageSize || m.config.MaxPageSize == 0 {
			// Auto mode (default): use all available screen space
			m.paginator.PerPage = tableHeight
		} else {
			// User specified a maximum: use it as ceiling, but never exceed available height
			m.paginator.PerPage = min(m.config.MaxPageSize, tableHeight)
		}

		m.updateColumns(m.viewWidth)
		m.updateTableData()

		return m, func() tea.Msg { return tea.ClearScreen() }

	case resourcesLoadedMsg:
		m.resources = msg.resources
		m.logLines = nil // Clear log lines when loading resources

		// cleanup the resource watcher when we switch to a new resource view.
		if m.currentGVR != msg.gvr && m.resourceWatcher != nil {
			m.resourceWatcher.Stop()
			m.resourceWatcher = nil
		}
		m.currentGVR = msg.gvr
		m.currentNamespace = msg.namespace
		m.listOptions = msg.listOptions

		// Update key bindings for new resource type
		m.updateKeysForResourceType()

		m.updateColumns(m.viewWidth)
		m.updateTableData()
		m.table.SetCursor(0)

		return m, m.watchResources(msg.gvr, msg.namespace)

	case logsLoadedMsg:
		m.logLines = msg.logLines
		m.resources = nil // Clear resources when loading logs
		m.currentGVR.Resource = k8s.ResourceLogs
		m.currentNamespace = msg.namespace

		// Update key bindings for logs view
		m.updateKeysForResourceType()
		m.updateColumns(m.viewWidth)

		// Jump to last page for tailing behavior
		if len(m.logLines) > 0 {
			lastPage := (len(m.logLines) - 1) / m.paginator.PerPage
			m.paginator.Page = lastPage
		}

		m.updateTableData()
		m.table.SetCursor(0)

		if m.logView.Autoscroll {
			m.table.GotoBottom()
		}

		return m, nil

	case resourceDescribedMsg:
		m.describeContent = msg.yamlContent
		m.resources = nil // Clear resources when loading describe view
		m.logLines = nil  // Clear log lines when loading describe view
		m.currentGVR.Resource = k8s.ResourceDescribe
		m.currentNamespace = msg.namespace

		// Update key bindings for describe view
		m.updateKeysForResourceType()
		m.updateColumns(m.viewWidth)

		// Set pagination to use full table height for describe view
		// Use the same header height calculation as in WindowSizeMsg
		baseHeaderHeight := 20
		headerHeight := baseHeaderHeight
		if m.viewHeight > 50 {
			headerHeight = 22
		} else if m.viewHeight < 30 {
			headerHeight = 15
		}
		tableHeight := max(m.viewHeight-headerHeight, 5)
		m.paginator.PerPage = tableHeight

		// Reset to first page
		m.paginator.Page = 0
		m.updateTableData()
		m.table.SetCursor(0)

		return m, nil

	case errMsg:
		log.G().Error("error occurred", "error", msg.err)
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

	case launchPluginMsg:
		m.pluginToLaunch = msg.plugin
		return m, tea.Quit

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
		switch m.viewMode {
		case ViewModeCommand:
			switch msg.String() {
			case "enter":
				command := strings.TrimSpace(m.commandInput.Value())
				m.commandInput.Reset()
				m.commandHistory.Push(command)
				m.commandHistory.ResetIndex()

				m.viewMode = ViewModeNormal
				return m, m.executeCommand(command)
			case "esc", "ctrl+c":
				m.commandInput.Reset()
				m.commandHistory.ResetIndex()

				m.viewMode = ViewModeNormal
				return m, nil
			case "tab":
				// autocomplete with the first matching suggestion

				// TODO: the first command match will currently prevent
				// suggestions, which may be unintuitive.
				// e.g. type 'r', which maps to 'reconnect' but it wont have
				// any suggestions because it represents a full command.
				args := cli.ParseArgs(m.commandInput.Value())
				if option, ok := lo.First(m.commandSuggester.Suggestions(args.AsList()...)); ok {
					// adding the extra space after the autocorrect help with
					// repeatetive autosuggestions.
					newCommand := strings.Join(args.ReplaceLast(option).AsList(), " ") + " "
					m.commandInput.SetValue(newCommand)
					m.commandInput.SetCursor(len(newCommand))
				}
				return m, nil
			case "down":
				m.commandInput.SetValue(m.commandHistory.MoveIndex(-1))
				return m, nil
			case "up":
				m.commandInput.SetValue(m.commandHistory.MoveIndex(1))
				return m, nil
			default:
				m.commandInput, cmd = m.commandInput.Update(msg)
				return m, cmd
			}
		default:
			// Handle keys explicitly to prevent double-processing
			switch msg.String() {
			case ":":
				m.viewMode = ViewModeCommand
				m.commandInput.Focus()
				// Clear any previous error or success messages when entering command mode
				m.commandErr = ""
				m.commandSuccess = ""
				return m, nil
			case "enter":
				if m.currentGVR.Resource == k8s.ResourceLogs {
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

				// Check if drill-down is supported before modifying navigation history
				if !m.canDrillDown() {
					return m, nil
				}

				var selectedNamespace, selectedName string
				if nameIndex, ok := k8s.NameColumn(m.table.Columns()); ok {
					selectedName = selectedResource[nameIndex]
				}
				if namespaceIndex, ok := k8s.NamespaceColumn(m.table.Columns()); ok {
					selectedNamespace = selectedResource[namespaceIndex]
				}
				memento := m.saveToMemento(selectedName, selectedNamespace)
				m.navigationHistory.Push(memento)

				return m, m.commandWithPreflights(m.drillDown(selectedResource), m.requireConnection)
			case "esc", "escape":
				memento := m.navigationHistory.Pop()
				if memento != nil {
					m.restoreFromMemento(memento)
				} else {
					return m, m.loadResources(k8s.ResourcePods)
				}
				return m, nil
			case "f":
				switch m.currentGVR.Resource {
				case k8s.ResourceLogs:
					m.logView.Fullscreen = !m.logView.Fullscreen
				case k8s.ResourceDescribe:
					m.describeView.Fullscreen = !m.describeView.Fullscreen
				}
				return m, nil
			case "s":
				if m.currentGVR.Resource == k8s.ResourceLogs {
					m.logView.Autoscroll = !m.logView.Autoscroll
					if m.logView.Autoscroll {
						m.table.GotoBottom()
					}
				}
				return m, nil
			case "t":
				if m.currentGVR.Resource == k8s.ResourceLogs {
					m.logView.ShowTimestamps = !m.logView.ShowTimestamps
					m.updateTableData()
				}
				return m, nil
			case "w":
				switch m.currentGVR.Resource {
				case k8s.ResourceLogs:
					m.logView.WrapText = !m.logView.WrapText
					m.updateTableData()
				case k8s.ResourceDescribe:
					m.describeView.WrapText = !m.describeView.WrapText
					m.updateTableData()
				}
				return m, nil
			case "n":
				if m.currentGVR.Resource == k8s.ResourceDescribe {
					m.describeView.ShowLineNumbers = !m.describeView.ShowLineNumbers
					m.updateTableData()
				}
				return m, nil
			case "y":
				// Yank/copy selected row (if implemented in future)
				return m, nil
			case "d":
				// Describe the currently selected resource
				if m.currentGVR.Resource == k8s.ResourceLogs ||
					m.currentGVR.Resource == k8s.ResourceDescribe ||
					m.currentGVR.Resource == k8s.ResourceContainers ||
					m.currentGVR.Resource == k8s.ResourceAPIResources {
					return m, nil // Can't describe these resource types
				}
				if !m.isConnected() {
					m.err = fmt.Errorf("not connected to cluster. Use :reconnect")
					return m, nil
				}
				return m, m.commandWithPreflights(
					m.describeCurrentResource(),
					m.requireConnection,
				)
			case "j", "down":
				// Handle navigation directly to prevent double-processing
				// Check if at bottom of current page
				if m.table.Cursor() >= len(m.table.Rows())-1 {
					// At bottom of page, try to go to next page
					if m.paginator.Page < m.paginator.TotalPages-1 {
						m.paginator.NextPage()
						m.updateTableData()
						m.table.GotoTop() // Start at top of next page
					}
				} else {
					m.table.MoveDown(1)
				}
				return m, nil
			case "k", "up":
				// Handle navigation directly to prevent double-processing
				// Check if at top of current page
				if m.table.Cursor() <= 0 {
					// At top of page, try to go to previous page
					if m.paginator.Page > 0 {
						m.paginator.PrevPage()
						m.updateTableData()
						m.table.GotoBottom() // Start at bottom of previous page
					}
				} else {
					m.table.MoveUp(1)
				}
				return m, nil
			case "J", "shift+down":
				// Jump to bottom of current page
				m.table.GotoBottom()
				return m, nil
			case "K", "shift+up":
				// Jump to top of current page
				m.table.GotoTop()
				return m, nil
			case "g":
				// Go to first line of first page (absolute first line)
				// Disable autoscroll when manually navigating to top
				if m.currentGVR.Resource == k8s.ResourceLogs {
					m.logView.Autoscroll = false
				}
				m.paginator.Page = 0
				m.updateTableData()
				m.table.GotoTop()
				return m, nil
			case "G":
				// Go to last line of last page (absolute last line)
				switch m.currentGVR.Resource {
				case k8s.ResourceLogs:
					// For logs, go to last page and enable autoscroll for tailing
					totalLogs := len(m.logLines)
					if totalLogs > 0 {
						lastPage := (totalLogs - 1) / m.paginator.PerPage
						m.paginator.Page = lastPage
						m.updateTableData()
						m.table.GotoBottom()
						m.logView.Autoscroll = true
					}
				case k8s.ResourceDescribe:
					// For describe view, go to last page
					if m.describeContent != "" {
						lines := strings.Split(m.describeContent, "\n")
						totalLines := len(lines)
						if totalLines > 0 {
							lastPage := (totalLines - 1) / m.paginator.PerPage
							m.paginator.Page = lastPage
							m.updateTableData()
							m.table.GotoBottom()
						}
					}
				default:
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
			case "0":
				// Explicitly ignore this key to prevent fallthrough to table
				return m, nil
			}
			// For unhandled keys in normal mode, pass to table
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
	}

	// Only update table for non-key messages
	m.table, cmd = m.table.Update(msg)

	return m, tea.Batch(cmd)
}

func (m *Model) isConnected() bool {
	return m.k8sClient != nil && m.k8sClient.IsConnected()
}

// View renders the current state of the TUI.
// It implements the tea.Model interface for Bubble Tea v2.
func (m *Model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Initializing k10s...")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	var b strings.Builder

	b.WriteString("\n")

	// Skip top header if fullscreen is enabled for logs or describe
	skipHeader := (m.logView != nil && m.logView.Fullscreen && m.currentGVR.Resource == k8s.ResourceLogs) ||
		(m.describeView != nil && m.describeView.Fullscreen && m.currentGVR.Resource == k8s.ResourceDescribe)
	if !skipHeader {
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

	// Render breadcrumb navigation if we're in a drilled-down view
	if m.navigationHistory.Len() > 0 {
		b.WriteString("\n\n")
		m.renderBreadcrumb(&b)
	}

	// Render pagination based on configured style (more compact for describe/logs views)
	if m.getTotalItems() > m.paginator.PerPage {
		if m.currentGVR.Resource == k8s.ResourceDescribe || m.currentGVR.Resource == k8s.ResourceLogs {
			b.WriteString("\n") // Single newline for describe/logs
		} else {
			b.WriteString("\n\n") // Double newline for resource lists
		}
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
	if m.viewHeight > 0 {
		renderedLines := strings.Count(output, "\n") + 1
		// When in command mode, only reserve space for command palette (ignore error/success)
		// This ensures command input doesn't shift when replacing error messages
		var bottomReservedLines int
		if m.viewMode == ViewModeCommand {
			bottomReservedLines = commandPaletteLines
		} else if m.commandErr != "" {
			bottomReservedLines = commandErrorLines
		} else if m.commandSuccess != "" {
			bottomReservedLines = commandSuccessLines
		}

		totalNeeded := m.viewHeight - bottomReservedLines

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

	v := tea.NewView(output)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderBreadcrumb renders the navigation breadcrumb showing the hierarchy path.
func (m *Model) renderBreadcrumb(b *strings.Builder) {
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

	parts = append(parts, brightStyle.Render(k8s.FormatGVR(m.currentGVR)))

	breadcrumbStr := strings.Join(parts, separatorStyle.Render(" > "))
	b.WriteString(dimStyle.Render("Path: ") + breadcrumbStr)
}

// saveToMemento creates and returns a memento containing the current state of the Model.
func (m *Model) saveToMemento(selectedResourceName, selectedNamespace string) *ModelMemento {
	return &ModelMemento{
		resources:        m.resources,
		currentGVR:       m.currentGVR,
		currentNamespace: m.currentNamespace,
		listOptions:      m.listOptions,

		tableCursor:   m.table.Cursor(),
		paginatorPage: m.paginator.Page,
		err:           m.err,
		logView:       m.logView,

		resourceName: selectedResourceName,
		namespace:    selectedNamespace,
	}
}

// restoreFromMemento restores the Model's state from a memento.
func (m *Model) restoreFromMemento(memento *ModelMemento) {
	if memento == nil {
		return
	}

	m.resources = memento.resources
	m.currentGVR = memento.currentGVR
	m.currentNamespace = memento.currentNamespace
	m.listOptions = memento.listOptions
	m.err = memento.err
	m.logView = memento.logView

	// Update key bindings for restored resource type
	m.updateKeysForResourceType()

	// Update pagination
	m.paginator.Page = memento.paginatorPage
	m.paginator.SetTotalPages(len(m.resources))

	// Update table columns and data
	m.updateColumns(m.viewWidth)
	m.updateTableData()

	maxCursor := max(len(m.table.Rows())-1, 0)
	if memento.tableCursor > maxCursor {
		m.table.SetCursor(maxCursor)
	} else {
		m.table.SetCursor(memento.tableCursor)
	}
}
