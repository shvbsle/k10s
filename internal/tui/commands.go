package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/log"
	"github.com/shvbsle/k10s/internal/tui/cli"
	"github.com/shvbsle/k10s/internal/tui/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// executeCommand processes a command string and returns the appropriate tea command.
func (m *Model) executeCommand(command string) tea.Cmd {
	originalCommand := command
	command = strings.ToLower(strings.TrimSpace(command))
	log.TUI("Executing command: %s", command)

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	baseCommand := parts[0]
	args := parts[1:]

	switch baseCommand {
	case "quit", "q":
		return tea.Quit
	case "reconnect", "r":
		return m.reconnectCmd()
	case "resource":
		if len(args) == 0 {
			return m.showCommandError(fmt.Sprintf("not enough arguments for command `%s`", originalCommand))
		}
		return m.resourceCommand(args[0], lo.Drop(args, 1))
	case "cplogs", "cp":
		// For cplogs, we need to preserve case in file paths, so use original args
		args := lo.Drop(strings.Fields(originalCommand), 1)
		return m.executeCplogsCommand(args)
	}

	return m.showCommandError(fmt.Sprintf("did not recognize command `%s`", originalCommand))
}

func (m *Model) resourceCommand(command string, args []string) tea.Cmd {
	gvr := schema.GroupVersionResource{
		// grabbing "v1" api version from a reputable source
		Version: metav1.Unversioned.Version,
	}
	before, after, found := strings.Cut(command, "/")
	if found {
		gvr.Version = after
	}
	gr := schema.ParseGroupResource(before)
	gvr.Resource = gr.Resource
	gvr.Group = gr.Group

	namespace := cli.ParseNamespace(args)

	return m.CommandWithPreflights(
		m.loadResourcesWithNamespace(gvr, namespace),
		m.requireConnection,
	)
}

// showCommandError returns a command that sets the command error and clears it after 5 seconds.
func (*Model) showCommandError(errMsg string) tea.Cmd {
	return func() tea.Msg {
		return commandErrMsg{errMsg}
	}
}

// loadResources creates a command that loads the specified resource type using current namespace.
func (m *Model) loadResources(resource string) tea.Cmd {
	return m.loadResourcesWithNamespace(metav1.Unversioned.WithResource(resource), m.currentNamespace)
}

// loadResourcesWithNamespace creates a command that loads the specified resource type from a specific namespace.
func (m *Model) loadResourcesWithNamespace(gvr schema.GroupVersionResource, namespace string) tea.Cmd {
	return func() tea.Msg {
		resourceList, err := m.k8sClient.Dynamic().
			Resource(gvr).
			Namespace(namespace).
			List(context.TODO(), m.listOptions)
		if err != nil {
			log.TUI("Failed to load %s: %v", gvr, err)
			return commandErrMsg{message: fmt.Errorf("listing resource %+v: %w", gvr, err).Error()}
		}

		return resourcesLoadedMsg{
			resources: lo.Map(resourceList.Items, func(object unstructured.Unstructured, _ int) k8s.OrderedResourceFields {
				return k8s.OrderedResourceFields(lo.Map(resources.GetResourceView(gvr.Resource).Fields, func(field resources.ResourceViewField, _ int) string {
					var fieldBuffer bytes.Buffer
					lo.Must0(template.Must(template.New("").Parse(field.PathTemplate)).Execute(&fieldBuffer, object.UnstructuredContent()))
					return fieldBuffer.String()
				}))
			}),
			resource:  gvr.Resource,
			namespace: namespace,
		}
	}
}

// reconnectCmd creates a command that attempts to reconnect to the cluster.
func (m *Model) reconnectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.k8sClient == nil {
			log.TUI("Reconnect failed: no client available")
			return errMsg{fmt.Errorf("no client available")}
		}

		log.TUI("Attempting to reconnect to cluster...")
		err := m.k8sClient.Reconnect()
		if err != nil {
			log.TUI("Reconnect failed: %v", err)
			return errMsg{fmt.Errorf("reconnect failed: %w", err)}
		}

		return resourcesLoadedMsg{
			resources: []k8s.OrderedResourceFields{},
			resource:  k8s.ResourcePods,
			namespace: metav1.NamespaceAll, // All namespaces after reconnect
		}
	}
}

// requireConnection wraps a command to only execute if connected to a cluster.
func (m *Model) requireConnection() error {
	if !m.isConnected() {
		return fmt.Errorf("not connected to cluster. Use :reconnect")
	}
	return nil
}

// renderCommandInput renders the command input field with suggestions.
func (m *Model) renderCommandInput(b *strings.Builder) {
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
func (m *Model) getFilteredSuggestions() []string {
	input := strings.ToLower(m.commandInput.Value())
	var filtered []string

	for _, suggestion := range m.commandSuggestions {
		if strings.HasPrefix(suggestion, input) {
			filtered = append(filtered, suggestion)
		}
	}

	return filtered
}

// drillDown handles drilling down into a selected resource.
// TODO: refactor to not use ordered fields.
func (m *Model) drillDown(selectedResource k8s.OrderedResourceFields) tea.Cmd {
	// overrides for certain resource kinds.
	switch m.resourceType {
	case
		k8s.ResourceContainers,
		k8s.ResourceLogs:
		// TODO: things...
	}

	resourceSchema := resources.GetResourceView(m.resourceType)
	object, _ := m.k8sClient.Dynamic().
		Resource(metav1.Unversioned.WithResource(m.resourceType)).
		Namespace(m.currentNamespace).
		Get(context.TODO(), selectedResource[0], metav1.GetOptions{})

	if resourceSchema.DrillDown == nil {
		log.TUI("Drill down not supported for this type: %s", m.resourceType)
		return nil
	}

	fieldSelector := fields.AndSelectors(lo.Map(resourceSchema.DrillDown.SelectorTemplates, func(selectorTemplate string, _ int) fields.Selector {
		var fieldSelectorBuffer bytes.Buffer
		lo.Must0(template.Must(template.New("").Parse(selectorTemplate)).Execute(&fieldSelectorBuffer, object.UnstructuredContent()))
		return fields.ParseSelectorOrDie(fieldSelectorBuffer.String())
	})...)

	namespace := metav1.NamespaceAll
	if m.isNamespaced(m.resourceType) {
		// TODO: search for the namespace field
		namespace = selectedResource[1]
	}

	// TODO: unset list options
	m.listOptions = metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	}

	return m.loadResourcesWithNamespace(
		metav1.Unversioned.WithResource(resourceSchema.DrillDown.Resource),
		namespace,
	)
}

func (m *Model) CommandWithPreflights(cmd tea.Cmd, preflights ...func() error) tea.Cmd {
	for _, preflight := range preflights {
		if err := preflight(); err != nil {
			return func() tea.Msg {
				return commandErrMsg{message: err.Error()}
			}
		}
	}
	return cmd
}
