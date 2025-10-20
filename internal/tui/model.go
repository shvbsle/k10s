package tui

import (
	"fmt"
	"log"
	"strings"

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

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	GotoTop    key.Binding
	GotoBottom key.Binding
	AllNS      key.Binding
	DefaultNS  key.Binding
	Command    key.Binding
	Quit       key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.Command, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.GotoTop, k.GotoBottom},
		{k.Left, k.Right, k.AllNS, k.DefaultNS},
		{k.Command, k.Quit},
	}
}

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
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type resourcesLoadedMsg struct {
	resources []k8s.Resource
	resType   k8s.ResourceType
}

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

	keys := keyMap{
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
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}

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

func (m Model) loadResourcesCmd(resType k8s.ResourceType) tea.Cmd {
	return func() tea.Msg {
		var resources []k8s.Resource
		var err error

		ns := m.currentNamespace
		if ns == "" {
			ns = "all"
		}

		switch resType {
		case k8s.ResourcePods:
			log.Printf("TUI: Loading pods from namespace: %s", ns)
			resources, err = m.k8sClient.ListPods(m.currentNamespace)
		case k8s.ResourceNodes:
			log.Printf("TUI: Loading nodes")
			resources, err = m.k8sClient.ListNodes()
		case k8s.ResourceNamespaces:
			log.Printf("TUI: Loading namespaces")
			resources, err = m.k8sClient.ListNamespaces()
		case k8s.ResourceServices:
			log.Printf("TUI: Loading services from namespace: %s", ns)
			resources, err = m.k8sClient.ListServices(m.currentNamespace)
		default:
			log.Printf("TUI: Loading pods (default) from namespace: %s", ns)
			resources, err = m.k8sClient.ListPods(m.currentNamespace)
		}

		if err != nil {
			log.Printf("TUI: Failed to load %s: %v", resType, err)
			return errMsg{err}
		}

		log.Printf("TUI: Successfully loaded %d %s", len(resources), resType)
		return resourcesLoadedMsg{
			resources: resources,
			resType:   resType,
		}
	}
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

		// Account for borders and padding
		totalWidth := m.width - 6
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
			totalWidth := m.width - 6
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

func (m *Model) updateTableData() {
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

func (m Model) isConnected() bool {
	return m.k8sClient != nil && m.k8sClient.IsConnected()
}

func (m Model) requireConnection(cmd tea.Cmd) tea.Cmd {
	if !m.isConnected() {
		return func() tea.Msg {
			return errMsg{fmt.Errorf("not connected to cluster. Use :reconnect")}
		}
	}
	return cmd
}

func (m Model) executeCommand(command string) tea.Cmd {
	command = strings.ToLower(command)
	log.Printf("TUI: Executing command: %s", command)

	switch command {
	case "quit", "q":
		log.Printf("TUI: User quit application")
		return tea.Quit
	case "reconnect", "r":
		log.Printf("TUI: User requested reconnect")
		return m.reconnectCmd()
	case "pods", "pod", "po":
		log.Printf("TUI: Loading pods")
		return m.requireConnection(m.loadResourcesCmd(k8s.ResourcePods))
	case "nodes", "node", "no":
		log.Printf("TUI: Loading nodes")
		return m.requireConnection(m.loadResourcesCmd(k8s.ResourceNodes))
	case "namespaces", "namespace", "ns":
		log.Printf("TUI: Loading namespaces")
		return m.requireConnection(m.loadResourcesCmd(k8s.ResourceNamespaces))
	case "services", "service", "svc":
		log.Printf("TUI: Loading services")
		return m.requireConnection(m.loadResourcesCmd(k8s.ResourceServices))
	default:
		log.Printf("TUI: Unknown command: %s", command)
		return nil
	}
}

func (m Model) reconnectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.k8sClient == nil {
			log.Printf("TUI: Reconnect failed: no client available")
			return errMsg{fmt.Errorf("no client available")}
		}

		log.Printf("TUI: Attempting to reconnect to cluster...")
		err := m.k8sClient.Reconnect()
		if err != nil {
			log.Printf("TUI: Reconnect failed: %v", err)
			return errMsg{fmt.Errorf("reconnect failed: %w", err)}
		}

		log.Printf("TUI: Reconnect successful, loading pods...")
		resources, err := m.k8sClient.ListPods("")
		if err != nil {
			log.Printf("TUI: Failed to load pods after reconnect: %v", err)
			return errMsg{err}
		}

		log.Printf("TUI: Loaded %d pods after reconnect", len(resources))
		return resourcesLoadedMsg{
			resources: resources,
			resType:   k8s.ResourcePods,
		}
	}
}

// View renders the current state of the TUI to a string.
// It implements the tea.Model interface for Bubble Tea.
func (m Model) View() string {
	if !m.ready {
		return "Initializing k10s..."
	}

	var b strings.Builder

	m.renderTopHeader(&b)
	b.WriteString("\n\n")

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
		b.WriteString(errorStyle.Render(fmt.Sprintf("⚠ Error: %v", m.err)))
		if !m.isConnected() {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" (try :reconnect)"))
		}
		b.WriteString("\n\n")
	} else if !m.isConnected() {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		b.WriteString(warningStyle.Render("⚠ Disconnected from cluster"))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" - use :reconnect to connect"))
		b.WriteString("\n\n")
	}

	m.renderTableWithHeader(&b)
	b.WriteString("\n")

	// Render pagination based on configured style
	if len(m.resources) > m.paginator.PerPage {
		b.WriteString("\n")
		m.renderPagination(&b)
	}

	if m.viewMode == ViewModeCommand {
		b.WriteString("\n")
		m.renderCommandInput(&b)
	}

	// Fill remaining height to prevent resize artifacts
	output := b.String()
	if m.height > 0 {
		renderedLines := strings.Count(output, "\n") + 1
		if renderedLines < m.height {
			remainingLines := m.height - renderedLines
			output += strings.Repeat("\n", remainingLines)
		}
	}

	return output
}

func (m Model) renderTopHeader(b *strings.Builder) {
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
		nsDisplay := m.currentNamespace
		if nsDisplay == "" {
			nsDisplay = "all"
		}
		infoContent.WriteString(labelStyle.Render("Namespace: ") + valueStyle.Render(nsDisplay) + "\n")
		infoContent.WriteString(labelStyle.Render("K10s Ver: ") + valueStyle.Render(Version) + "\n")
		infoContent.WriteString(labelStyle.Render("K8s Ver: ") + valueStyle.Render(m.clusterInfo.K8sVersion) + "\n")
	}
	infoContent.WriteString(labelStyle.Render("CPU: ") + errorStyle.Render("n/a") + "\n")
	infoContent.WriteString(labelStyle.Render("MEM: ") + errorStyle.Render("n/a"))

	infoBlock := statusIndicator + " " + infoContent.String()
	helpBlock := m.help.View(m.keys)

	kittenStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	kitten1 := kittenStyle.Render(m.config.Logo)
	kitten2 := kittenStyle.Render(m.config.Logo)
	doubleKitten := lipgloss.JoinHorizontal(lipgloss.Top, kitten1, "  ", kitten2)

	termWidth := m.width
	if termWidth < 80 {
		termWidth = 80
	}

	infoBlockWidth := lipgloss.Width(infoBlock)
	helpBlockWidth := lipgloss.Width(helpBlock)
	doubleKittenWidth := lipgloss.Width(doubleKitten)

	const minGap = 2
	totalContentWidth := infoBlockWidth + helpBlockWidth + doubleKittenWidth + (minGap * 2)

	// Use natural widths if content fits, otherwise constrain with max widths
	if totalContentWidth <= termWidth {
		gap1 := minGap
		gap2 := termWidth - infoBlockWidth - helpBlockWidth - doubleKittenWidth - gap1
		if gap2 < minGap {
			gap2 = minGap
		}

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

		remainingSpace := termWidth - actualInfoWidth - actualHelpWidth - kittenSpace
		if remainingSpace < 0 {
			remainingSpace = 0
		}

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

// stripAnsi removes ANSI escape sequences for accurate length calculation.
func stripAnsi(s string) string {
	result := ""
	inEscape := false
	escapeStart := false

	for i := 0; i < len(s); i++ {
		r := rune(s[i])

		if r == '\x1b' {
			inEscape = true
			escapeStart = true
			continue
		}

		if inEscape {
			if escapeStart && r == '[' {
				escapeStart = false
				continue
			}
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
				escapeStart = false
				continue
			}
			continue
		}

		if !inEscape {
			result += string(r)
		}
	}
	return result
}

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
			// Truncate or pad to exact width
			cellText := cell
			if len(cellText) > columns[i].Width {
				cellText = cellText[:columns[i].Width]
			} else if len(cellText) < columns[i].Width {
				cellText = cellText + strings.Repeat(" ", columns[i].Width-len(cellText))
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

func (m Model) buildTopBorderWithTitle(title string, width int, borderColor lipgloss.Color, titleStyle lipgloss.Style) string {
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Calculate centering - leftDashes + titleLen + rightDashes = width
	titleLen := len(stripAnsi(title))
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

func (m Model) renderCommandInput(b *strings.Builder) {
	// Simple command input with inline autocomplete
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	b.WriteString(promptStyle.Render(":"))
	b.WriteString(m.commandInput.View())

	// Show autocomplete suggestions inline
	if len(m.commandInput.Value()) > 0 {
		filtered := m.getFilteredSuggestions()
		if len(filtered) > 0 {
			b.WriteString("  ")
			b.WriteString(suggestionStyle.Render(fmt.Sprintf("(%s)", strings.Join(filtered[:min(3, len(filtered))], ", "))))
		}
	}
}

func (m Model) getFilteredSuggestions() []string {
	input := strings.ToLower(m.commandInput.Value())
	var filtered []string

	for _, suggestion := range m.commandSuggestions {
		if strings.HasPrefix(suggestion, input) {
			filtered = append(filtered, suggestion)
		}
	}

	return filtered
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	default:
		return []string{"Name", "Namespace", "Node", "Status", "Age", "IP"}
	}
}

// updateColumns updates the table columns based on the current width and resource type.
func (m *Model) updateColumns(totalWidth int) {
	nameWidth := int(float64(totalWidth) * 0.30)
	nsWidth := int(float64(totalWidth) * 0.13)
	nodeWidth := int(float64(totalWidth) * 0.18)
	statusWidth := int(float64(totalWidth) * 0.12)
	ageWidth := int(float64(totalWidth) * 0.08)
	ipWidth := totalWidth - nameWidth - nsWidth - nodeWidth - statusWidth - ageWidth

	titles := getColumnTitles(m.resourceType)

	m.table.SetColumns([]table.Column{
		{Title: titles[0], Width: nameWidth},
		{Title: titles[1], Width: nsWidth},
		{Title: titles[2], Width: nodeWidth},
		{Title: titles[3], Width: statusWidth},
		{Title: titles[4], Width: ageWidth},
		{Title: titles[5], Width: ipWidth},
	})
}
