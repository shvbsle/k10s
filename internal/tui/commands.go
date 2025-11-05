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
	"github.com/shvbsle/k10s/internal/plugins"
	"github.com/shvbsle/k10s/internal/tui/cli"
	"github.com/shvbsle/k10s/internal/tui/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type launchPluginMsg struct {
	plugin plugins.Plugin
}

// executeCommand processes a command string and returns the appropriate tea command.
func (m *Model) executeCommand(command string) tea.Cmd {
	originalCommand := command
	command = strings.ToLower(strings.TrimSpace(command))
	log.G().Info("executing command", "command", command)

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	baseCommand := parts[0]
	args := parts[1:]

	if plugin, ok := m.pluginRegistry.GetByCommand(baseCommand); ok {
		return m.launchPluginCmd(plugin)
	}

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
	before, after, found := strings.Cut(command, "/")
	if found {
		m.currentGVR.Version = after
	}
	gr := schema.ParseGroupResource(before)
	m.currentGVR.Resource = gr.Resource
	m.currentGVR.Group = gr.Group

	namespace := cli.ParseNamespace(args)

	return m.commandWithPreflights(
		m.loadResourcesWithNamespace(m.currentGVR, namespace, metav1.ListOptions{}),
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
	return m.loadResourcesWithNamespace(metav1.Unversioned.WithResource(resource), m.currentNamespace, metav1.ListOptions{})
}

// loadResources creates a command that loads the specified resource type using current namespace.
func (m *Model) loadResourcesGVR(gvr schema.GroupVersionResource) tea.Cmd {
	return m.loadResourcesWithNamespace(gvr, m.currentNamespace, metav1.ListOptions{})
}

// loadResourcesWithNamespace creates a command that loads the specified resource type from a specific namespace.
func (m *Model) loadResourcesWithNamespace(gvr schema.GroupVersionResource, namespace string, listOptions metav1.ListOptions) tea.Cmd {
	return func() tea.Msg {
		resourceList, err := m.k8sClient.Dynamic().
			Resource(gvr).
			Namespace(namespace).
			List(context.TODO(), listOptions)
		if err != nil {
			log.G().Error("failed to load resources", "gvr", gvr, "error", err)
			return errMsg{err}
		}

		return resourcesLoadedMsg{
			gvr:         gvr,
			namespace:   namespace,
			listOptions: listOptions,
			resources: lo.Map(resourceList.Items, func(object unstructured.Unstructured, _ int) k8s.OrderedResourceFields {
				return k8s.OrderedResourceFields(lo.Map(resources.GetResourceView(gvr.Resource).Fields, func(field resources.ResourceViewField, _ int) string {
					// TODO: handle more gracefully
					return lo.Must(field.Resolver.Resolve(object))
				}))
			}),
		}
	}
}

// reconnectCmd creates a command that attempts to reconnect to the cluster.
func (m *Model) reconnectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.k8sClient == nil {
			log.G().Warn("reconnect failed", "reason", "no client available")
			return errMsg{fmt.Errorf("no client available")}
		}

		log.G().Info("attempting to reconnect to cluster")
		err := m.k8sClient.Reconnect()
		if err != nil {
			log.G().Error("reconnect failed", "error", err)
			return errMsg{fmt.Errorf("reconnect failed: %w", err)}
		}

		log.G().Info("reconnect successful, loading resources")
		return m.loadResources(m.currentGVR.Resource)
	}
}

func (m *Model) launchPluginCmd(plugin plugins.Plugin) tea.Cmd {
	return func() tea.Msg {
		log.G().Info("launching plugin command", "plugin", plugin.Name())
		return launchPluginMsg{plugin: plugin}
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
		args := cli.ParseArgs(m.commandInput.Value())
		suggestions := m.commandSuggester.Suggestions(args.AsList()...)
		if len(suggestions) > 0 {
			b.WriteString(suggestionStyle.Render(fmt.Sprintf("(%s)", strings.Join(suggestions[:min(3, len(suggestions))], ", "))))
		}
	}
}

// drillDown handles drilling down into a selected resource.
// TODO: refactor to not use ordered fields.
func (m *Model) drillDown(selectedResource k8s.OrderedResourceFields) tea.Cmd {
	var selectedNamespace, selectedName string
	if nameIndex, ok := k8s.NameColumn(m.table.Columns()); ok {
		selectedName = selectedResource[nameIndex]
	}
	if namespaceIndex, ok := k8s.NamespaceColumn(m.table.Columns()); ok {
		selectedNamespace = selectedResource[namespaceIndex]
	}

	// overrides for certain views
	switch m.currentGVR.Resource {
	// TODO: maybe could pick another action for pod drill down via config
	// override?
	case k8s.ResourcePods:
		return func() tea.Msg {
			resources, err := m.k8sClient.ListContainersForPod(selectedName, selectedNamespace)
			if err != nil {
				log.TUI().Error("Failed to load containers", "error", err)
				return errMsg{err}
			}
			return resourcesLoadedMsg{
				resources: resources,
				// TODO: this gets weird because containers and logs arent
				// kuberetes resources. this overall needs to be stored as a
				// different concept for views and for resources.
				gvr:       schema.GroupVersionResource{Resource: k8s.ResourceContainers},
				namespace: selectedNamespace,
			}
		}
	case k8s.ResourceContainers:
		return func() tea.Msg {
			memento, ok := m.navigationHistory.FindMementoByResourceType(k8s.ResourcePods)
			if !ok {
				log.TUI().Error("Failed to get pod info from outer memento")
				return errMsg{fmt.Errorf("failed to get pod info")}
			}
			var (
				podName      = memento.resourceName
				podNamespace = memento.namespace
			)
			logLines, err := m.k8sClient.GetContainerLogs(podName, podNamespace, selectedName, m.config.LogTailLines, true)
			if err != nil {
				log.TUI().Error("Failed to load logs", "error", err)
				return errMsg{err}
			}
			return logsLoadedMsg{
				logLines:  logLines,
				namespace: selectedNamespace,
			}
		}
	case k8s.ResourceLogs:
		// TODO: noop, cant select logs
		return nil
	}

	resourceView := resources.GetResourceView(m.currentGVR.Resource)

	if resourceView.DrillDown == nil {
		log.TUI().Warn("drill down not supported for this resource", "GVR", m.currentGVR)
		return func() tea.Msg {
			return errMsg{err: fmt.Errorf("drill down not supported for this type: %s", m.currentGVR)}
		}
	}

	object, _ := m.k8sClient.Dynamic().
		Resource(m.currentGVR).
		Namespace(m.currentNamespace).
		Get(context.TODO(), selectedName, metav1.GetOptions{})

	fieldSelector := fields.AndSelectors(lo.Map(resourceView.DrillDown.SelectorTemplates, func(selectorTemplate string, _ int) fields.Selector {
		var fieldSelectorBuffer bytes.Buffer
		lo.Must0(template.Must(template.New("").Parse(selectorTemplate)).Execute(&fieldSelectorBuffer, object.UnstructuredContent()))
		return fields.ParseSelectorOrDie(fieldSelectorBuffer.String())
	})...)

	m.currentNamespace = metav1.NamespaceAll
	if m.isNamespaced(m.currentGVR.Resource) {
		m.currentNamespace = selectedNamespace
	}

	return m.loadResourcesWithNamespace(
		metav1.Unversioned.WithResource(resourceView.DrillDown.Resource),
		m.currentNamespace,
		metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		},
	)
}

func (m *Model) commandWithPreflights(cmd tea.Cmd, preflights ...func() error) tea.Cmd {
	for _, preflight := range preflights {
		if err := preflight(); err != nil {
			return func() tea.Msg {
				return commandErrMsg{message: err.Error()}
			}
		}
	}
	return cmd
}
