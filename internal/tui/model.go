package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
)

const Version = "v0.1.0"

type ViewMode int

const (
	ViewModeNormal ViewMode = iota
	ViewModeCommand
)

type Model struct {
	config       *config.Config
	k8sClient    *k8s.Client
	table        table.Model
	paginator    paginator.Model
	commandInput textinput.Model
	viewMode     ViewMode
	resources    []k8s.Resource
	resourceType k8s.ResourceType
	ready        bool
	width        int
	height       int
	err          error
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type resourcesLoadedMsg struct {
	resources []k8s.Resource
	resType   k8s.ResourceType
}

func New(cfg *config.Config, client *k8s.Client) Model {
	// Create command input
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.CharLimit = 100
	ti.Width = 50

	// Create table
	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "Namespace", Width: 20},
		{Title: "Status", Width: 15},
		{Title: "Age", Width: 10},
		{Title: "Extra", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(cfg.PageSize),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// Create paginator
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = cfg.PageSize
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")

	return Model{
		config:       cfg,
		k8sClient:    client,
		table:        t,
		paginator:    p,
		commandInput: ti,
		viewMode:     ViewModeNormal,
		resourceType: k8s.ResourcePods,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadResourcesCmd(m.k8sClient, k8s.ResourcePods),
	)
}

func loadResourcesCmd(client *k8s.Client, resType k8s.ResourceType) tea.Cmd {
	return func() tea.Msg {
		var resources []k8s.Resource
		var err error

		switch resType {
		case k8s.ResourcePods:
			resources, err = client.ListPods("")
		case k8s.ResourceNodes:
			resources, err = client.ListNodes()
		case k8s.ResourceNamespaces:
			resources, err = client.ListNamespaces()
		default:
			resources, err = client.ListPods("")
		}

		if err != nil {
			return errMsg{err}
		}

		return resourcesLoadedMsg{
			resources: resources,
			resType:   resType,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case resourcesLoadedMsg:
		m.resources = msg.resources
		m.resourceType = msg.resType
		m.updateTableData()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.viewMode == ViewModeCommand {
			switch msg.String() {
			case "enter":
				// Execute command
				command := strings.TrimSpace(m.commandInput.Value())
				m.commandInput.Reset()
				m.viewMode = ViewModeNormal
				return m, m.executeCommand(command)
			case "esc":
				// Cancel command
				m.commandInput.Reset()
				m.viewMode = ViewModeNormal
				return m, nil
			default:
				m.commandInput, cmd = m.commandInput.Update(msg)
				return m, cmd
			}
		} else {
			// Normal mode
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case ":":
				m.viewMode = ViewModeCommand
				m.commandInput.Focus()
				return m, nil
			case "j", "down":
				m.table.MoveDown(1)
			case "k", "up":
				m.table.MoveUp(1)
			case "g":
				m.table.GotoTop()
			case "G":
				m.table.GotoBottom()
			case "h", "left", "pgup":
				if m.paginator.Page > 0 {
					m.paginator.PrevPage()
					m.updateTableData()
				}
			case "l", "right", "pgdown":
				if m.paginator.Page < m.paginator.TotalPages-1 {
					m.paginator.NextPage()
					m.updateTableData()
				}
			case "ctrl+c":
				return m, tea.Quit
			}
		}
	}

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
			res.Status,
			res.Age,
			res.Extra,
		}
	}

	m.table.SetRows(rows)
	m.paginator.SetTotalPages(len(m.resources))
}

func (m Model) executeCommand(command string) tea.Cmd {
	command = strings.ToLower(command)

	switch command {
	case "quit", "q":
		return tea.Quit
	case "pods", "pod", "po":
		return loadResourcesCmd(m.k8sClient, k8s.ResourcePods)
	case "nodes", "node", "no":
		return loadResourcesCmd(m.k8sClient, k8s.ResourceNodes)
	case "namespaces", "namespace", "ns":
		return loadResourcesCmd(m.k8sClient, k8s.ResourceNamespaces)
	default:
		return nil
	}
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing k10s..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	var b strings.Builder

	// Logo and version
	logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	logoLines := strings.Split(m.config.Logo, "\n")
	for i, line := range logoLines {
		b.WriteString(logoStyle.Render(line))
		if i == 0 {
			b.WriteString(versionStyle.Render(" " + Version))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Resource type header
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	b.WriteString(headerStyle.Render(fmt.Sprintf("Resource: %s (Total: %d)", m.resourceType, len(m.resources))))
	b.WriteString("\n\n")

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Paginator
	if len(m.resources) > m.paginator.PerPage {
		paginatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pageInfo := fmt.Sprintf("Page %d/%d", m.paginator.Page+1, m.paginator.TotalPages)
		b.WriteString(paginatorStyle.Render(pageInfo))
		b.WriteString("\n")
	}

	// Command input or help
	if m.viewMode == ViewModeCommand {
		b.WriteString("\n:")
		b.WriteString(m.commandInput.View())
	} else {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		helpText := "j/k: nav | h/l: page | :: cmd | q: quit | Commands: pods, nodes, ns"
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(helpText))
	}

	return b.String()
}
