package tui

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shvbsle/k10s/internal/k8s"
)

// executeCommand processes a command string and returns the appropriate tea command.
func (m Model) executeCommand(command string) tea.Cmd {
	originalCommand := command
	command = strings.ToLower(strings.TrimSpace(command))
	log.Printf("TUI: Executing command: %s", command)

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	baseCommand := parts[0]
	args := parts[1:]

	switch baseCommand {
	case "quit", "q":
		log.Printf("TUI: User quit application")
		return tea.Quit
	case "reconnect", "r":
		log.Printf("TUI: User requested reconnect")
		return m.reconnectCmd()
	case "pods", "pod", "po":
		namespace := m.parseNamespaceArgs(args)
		log.Printf("TUI: Loading pods from namespace: %s", namespace)
		return m.requireConnection(m.loadResourcesWithNamespace(k8s.ResourcePods, namespace))
	case "nodes", "node", "no":
		log.Printf("TUI: Loading nodes")
		return m.requireConnection(m.loadResourcesCmd(k8s.ResourceNodes))
	case "namespaces", "namespace", "ns":
		log.Printf("TUI: Loading namespaces")
		return m.requireConnection(m.loadResourcesCmd(k8s.ResourceNamespaces))
	case "services", "service", "svc":
		namespace := m.parseNamespaceArgs(args)
		log.Printf("TUI: Loading services from namespace: %s", namespace)
		return m.requireConnection(m.loadResourcesWithNamespace(k8s.ResourceServices, namespace))
	default:
		log.Printf("TUI: Unknown command: %s", command)
		return m.showCommandError(fmt.Sprintf("did not recognize command `%s`", originalCommand))
	}
}

// parseNamespaceArgs parses namespace from command arguments.
// Supports patterns:
//   - "-n <namespace>" or "--namespace <namespace>"
//   - "<namespace>" (direct)
//   - "in <namespace>"
//   - "all" or "-n all" (for all namespaces)
//
// Returns "" for all namespaces, or the specific namespace name.
func (m Model) parseNamespaceArgs(args []string) string {
	if len(args) == 0 {
		// No args means all namespaces
		return ""
	}

	// Check for "-n <namespace>", "--namespace <namespace>", or "in <namespace>" patterns
	for i := 0; i < len(args); i++ {
		if (args[i] == "-n" || args[i] == "--namespace" || args[i] == "in") && i+1 < len(args) {
			ns := args[i+1]
			if ns == "all" {
				return ""
			}
			return ns
		}
	}

	// Check if first arg is "all"
	if args[0] == "all" {
		return ""
	}

	// Otherwise, treat first arg as namespace
	return args[0]
}

// showCommandError returns a command that sets the command error and clears it after 5 seconds.
func (m Model) showCommandError(errMsg string) tea.Cmd {
	return func() tea.Msg {
		// First, update the model with the error
		// This will be handled as a special commandErrMsg
		return commandErrMsg{errMsg}
	}
}

// loadResourcesCmd creates a command that loads the specified resource type using current namespace.
func (m Model) loadResourcesCmd(resType k8s.ResourceType) tea.Cmd {
	return m.loadResourcesWithNamespace(resType, m.currentNamespace)
}

// loadResourcesWithNamespace creates a command that loads the specified resource type from a specific namespace.
func (m Model) loadResourcesWithNamespace(resType k8s.ResourceType, namespace string) tea.Cmd {
	return func() tea.Msg {
		var resources []k8s.Resource
		var err error

		ns := namespace
		if ns == "" {
			ns = "all"
		}

		switch resType {
		case k8s.ResourcePods:
			log.Printf("TUI: Loading pods from namespace: %s", ns)
			resources, err = m.k8sClient.ListPods(namespace)
		case k8s.ResourceNodes:
			log.Printf("TUI: Loading nodes")
			resources, err = m.k8sClient.ListNodes()
		case k8s.ResourceNamespaces:
			log.Printf("TUI: Loading namespaces")
			resources, err = m.k8sClient.ListNamespaces()
		case k8s.ResourceServices:
			log.Printf("TUI: Loading services from namespace: %s", ns)
			resources, err = m.k8sClient.ListServices(namespace)
		default:
			log.Printf("TUI: Loading pods (default) from namespace: %s", ns)
			resources, err = m.k8sClient.ListPods(namespace)
		}

		if err != nil {
			log.Printf("TUI: Failed to load %s: %v", resType, err)
			return errMsg{err}
		}

		log.Printf("TUI: Successfully loaded %d %s", len(resources), resType)
		return resourcesLoadedMsg{
			resources: resources,
			resType:   resType,
			namespace: namespace,
		}
	}
}

// reconnectCmd creates a command that attempts to reconnect to the cluster.
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
			namespace: "", // All namespaces after reconnect
		}
	}
}

// requireConnection wraps a command to only execute if connected to a cluster.
func (m Model) requireConnection(cmd tea.Cmd) tea.Cmd {
	if !m.isConnected() {
		return func() tea.Msg {
			return errMsg{fmt.Errorf("not connected to cluster. Use :reconnect")}
		}
	}
	return cmd
}

// renderCommandInput renders the command input field with suggestions.
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

// getFilteredSuggestions returns command suggestions matching the current input.
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
