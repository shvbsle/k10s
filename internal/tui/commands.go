package tui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"text/template"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	"k8s.io/apimachinery/pkg/watch"
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
	case "resource", "rs":
		if len(args) == 0 {
			return m.listAvailableResources()
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

	// Parse the requested GVR
	requestedGVR := schema.GroupVersionResource{}
	if found {
		requestedGVR.Version = after
	}
	gr := schema.ParseGroupResource(before)
	requestedGVR.Resource = gr.Resource
	requestedGVR.Group = gr.Group

	// Validate that the resource exists on the server
	validGVRs := cli.GetServerGVRs(m.k8sClient.Discovery())
	resourceExists := false
	for _, validGVR := range validGVRs {
		// Match on resource name and group (ignore version if not specified)
		if validGVR.Resource == requestedGVR.Resource &&
			validGVR.Group == requestedGVR.Group &&
			(requestedGVR.Version == "" || validGVR.Version == requestedGVR.Version) {
			// Use the server's preferred version if version wasn't specified
			if requestedGVR.Version == "" {
				requestedGVR.Version = validGVR.Version
			}
			resourceExists = true
			break
		}
	}

	if !resourceExists {
		return m.showCommandError(fmt.Sprintf("resource '%s' not found on the server", command))
	}

	// Only update the current GVR after validation succeeds
	m.currentGVR = requestedGVR

	namespace := cli.ParseNamespace(args)

	return m.commandWithPreflights(
		m.loadResourcesWithNamespace(m.currentGVR, namespace, metav1.ListOptions{}),
		m.requireConnection,
	)
}

// listAvailableResources displays all available Kubernetes resources in the cluster.
func (m *Model) listAvailableResources() tea.Cmd {
	return func() tea.Msg {
		// Get all available resources from the server
		validGVRs := cli.GetServerGVRs(m.k8sClient.Discovery())

		// Create a map to deduplicate by resource name (showing preferred version)
		resourceMap := make(map[string]schema.GroupVersionResource)
		for _, gvr := range validGVRs {
			key := gvr.Resource
			if gvr.Group != "" {
				key = gvr.Resource + "." + gvr.Group
			}
			// Only keep the first occurrence (preferred version)
			if _, exists := resourceMap[key]; !exists {
				resourceMap[key] = gvr
			}
		}

		// Convert to sorted slice for display
		resources := lo.Values(resourceMap)

		// Sort alphabetically by resource name
		sort.Slice(resources, func(i, j int) bool {
			iKey := resources[i].Resource
			if resources[i].Group != "" {
				iKey = resources[i].Resource + "." + resources[i].Group
			}
			jKey := resources[j].Resource
			if resources[j].Group != "" {
				jKey = resources[j].Resource + "." + resources[j].Group
			}
			return iKey < jKey
		})

		// Create rows for table display
		rows := lo.Map(resources, func(gvr schema.GroupVersionResource, _ int) k8s.OrderedResourceFields {
			resourceName := gvr.Resource
			if gvr.Group != "" {
				resourceName = gvr.Resource + "." + gvr.Group
			}
			return k8s.OrderedResourceFields{
				resourceName,
				gvr.Version,
				gvr.Group,
			}
		})

		return resourcesLoadedMsg{
			gvr:       schema.GroupVersionResource{Resource: "api-resources"},
			resources: rows,
		}
	}
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
// loadResourcesWithNamespace creates a command that loads the specified resource type from a specific namespace.
func (m *Model) loadResourcesWithNamespace(gvr schema.GroupVersionResource, namespace string, listOptions metav1.ListOptions) tea.Cmd {
	return func() tea.Msg {
		resourceList, err := m.k8sClient.Dynamic().Resource(gvr).Namespace(namespace).List(context.TODO(), listOptions)
		if err != nil {
			log.G().Error("failed to load resources", "gvr", gvr, "error", err)
			return errMsg{err}
		}

		return resourcesLoadedMsg{
			gvr:         gvr,
			namespace:   namespace,
			listOptions: listOptions,
			resources: lo.Map(resourceList.Items, func(object unstructured.Unstructured, _ int) k8s.OrderedResourceFields {
				return lo.Map(resources.GetResourceView(gvr.Resource).Fields, func(field resources.ResourceViewField, _ int) string {
					// TODO: handle more gracefully
					return lo.Must(field.Resolver.Resolve(&object))
				})
			}),
		}
	}
}

func (m *Model) watchResources(gvr schema.GroupVersionResource, namespace string) tea.Cmd {
	return func() tea.Msg {
		// we dont need to setup the watcher.
		if m.resourceWatcher != nil {
			return nil
		}

		w, err := m.k8sClient.Dynamic().Resource(gvr).Namespace(namespace).Watch(context.TODO(), m.listOptions)
		if err != nil {
			log.G().Error("failed to load resources", "gvr", gvr, "error", err)
			return errMsg{err}
		}

		m.resourceWatcher = w

		go func() {
			for e := range w.ResultChan() {
				obj, ok := e.Object.(*unstructured.Unstructured)
				if !ok {
					panic(fmt.Sprintf("did not get unstructured, got %T", e.Object))
				}

				_, index, _ := lo.FindIndexOf(m.resources, func(r k8s.OrderedResourceFields) bool {
					return lo.IndexOf(r, obj.GetName()) != -1 && lo.IndexOf(r, obj.GetNamespace()) != -1
				})

				fields := lo.Map(resources.GetResourceView(gvr.Resource).Fields, func(field resources.ResourceViewField, _ int) string {
					return lo.Must(field.Resolver.Resolve(obj))
				})

				switch e.Type {
				case watch.Added:
					if index == -1 {
						m.resources = append(m.resources, fields)

						// TODO: this is expensive, but we can find cheaper
						// or better alternative later.
						var (
							nameIndex, _      = k8s.NameColumn(m.table.Columns())
							namespaceIndex, _ = k8s.NamespaceColumn(m.table.Columns())
						)

						// TODO: this is how kubernetes resources are
						// assumed to be sorted. i.e. by name and namespace.
						sortIndex := func(index int) func(int, int) bool {
							return func(i, j int) bool { return strings.Compare(m.resources[i][index], m.resources[j][index]) < 0 }
						}
						sort.Slice(m.resources, sortIndex(nameIndex))
						sort.Slice(m.resources, sortIndex(namespaceIndex))
					}
				case watch.Modified:
					lo.Assert(index != -1, "cant update non-existing item")
					m.resources[index] = fields
				case watch.Deleted:
					lo.Assert(index != -1, "cant delete non-existing item")
					m.resources = slices.Delete(m.resources, index, index+1)
				}

				for !m.tryQueueTableUpdate() {
					// keep trying to queue until succeeds.
					// TODO: handle better and maybe update api.
				}
			}
		}()

		return nil
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
		// Execute the loadResources command to get the actual message
		return m.loadResources(m.currentGVR.Resource)()
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

// canDrillDown checks if drill-down is supported for the current resource type.
func (m *Model) canDrillDown() bool {
	// Special resources with hardcoded drill-down support
	switch m.currentGVR.Resource {
	case k8s.ResourcePods, k8s.ResourceContainers:
		return true
	case k8s.ResourceLogs, k8s.ResourceDescribe, k8s.ResourceAPIResources:
		return false
	}

	// Check if resource has drill-down configuration
	resourceView := resources.GetResourceView(m.currentGVR.Resource)
	return resourceView.DrillDown != nil
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

// describeCurrentResource creates a command that fetches and describes the currently selected resource in YAML format.
func (m *Model) describeCurrentResource() tea.Cmd {
	return func() tea.Msg {
		if len(m.resources) == 0 {
			return commandErrMsg{message: "no resource selected"}
		}

		actualIdx := m.paginator.Page*m.paginator.PerPage + m.table.Cursor()
		if actualIdx >= len(m.resources) {
			return commandErrMsg{message: "invalid selection"}
		}

		selectedResource := m.resources[actualIdx]

		// Extract name and namespace from the selected row
		var selectedName, selectedNamespace string
		if nameIndex, ok := k8s.NameColumn(m.table.Columns()); ok {
			selectedName = selectedResource[nameIndex]
		}
		if namespaceIndex, ok := k8s.NamespaceColumn(m.table.Columns()); ok {
			selectedNamespace = selectedResource[namespaceIndex]
		}

		// Use the current namespace if no namespace column exists (cluster-scoped resources)
		if selectedNamespace == "" {
			selectedNamespace = m.currentNamespace
		}

		log.G().Info("describing resource", "gvr", m.currentGVR, "name", selectedName, "namespace", selectedNamespace)

		// Use kubectl describe to get human-readable output
		var cmd *exec.Cmd
		resourceType := m.currentGVR.Resource

		if selectedNamespace != "" && selectedNamespace != metav1.NamespaceAll {
			cmd = exec.Command("kubectl", "describe", resourceType, selectedName, "-n", selectedNamespace)
		} else {
			cmd = exec.Command("kubectl", "describe", resourceType, selectedName)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.G().Error("failed to describe resource", "error", err, "output", string(output))
			return errMsg{fmt.Errorf("failed to describe resource: %w\n%s", err, string(output))}
		}

		return resourceDescribedMsg{
			yamlContent:  string(output),
			resourceName: selectedName,
			namespace:    selectedNamespace,
			gvr:          m.currentGVR,
		}
	}
}
