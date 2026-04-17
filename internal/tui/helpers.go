package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/shvbsle/k10s/internal/k8s"
)

var namespacedCache = map[string]bool{}

// isNamespaced returns whether the resource is cluster or namespace scoped
func (m *Model) isNamespaced(resource string) bool {
	if supports, ok := namespacedCache[resource]; ok {
		return supports
	}

	_, apiResourceLists, err := m.k8sClient.Discovery().ServerGroupsAndResources()
	if err != nil {
		// TODO: handle?
		return true
	}

	for _, apiResourceList := range apiResourceLists {
		for _, apiResource := range apiResourceList.APIResources {
			if apiResource.Name == resource {
				namespacedCache[resource] = apiResource.Namespaced
				return namespacedCache[resource]
			}
		}
	}

	// just assume things are namespaced for now
	return true
}

// getTotalItems returns the total number of items (resources or log lines) to paginate.
func (m *Model) getTotalItems() int {
	if m.currentGVR.Resource == k8s.ResourceLogs && m.logLines != nil {
		return len(m.logLines)
	}
	if (m.currentGVR.Resource == k8s.ResourceDescribe || m.currentGVR.Resource == k8s.ResourceYaml) && m.describeContent != "" {
		return len(strings.Split(m.describeContent, "\n"))
	}
	return len(m.resources)
}

func (m *Model) updateKeysForResourceType() {
	isLogs := m.currentGVR.Resource == k8s.ResourceLogs
	isDescribe := m.currentGVR.Resource == k8s.ResourceDescribe
	isYaml := m.currentGVR.Resource == k8s.ResourceYaml

	// Enable/disable view-specific keys
	m.keys.Fullscreen.SetEnabled(isLogs || isDescribe || isYaml)
	m.keys.Autoscroll.SetEnabled(isLogs)
	m.keys.ToggleTime.SetEnabled(isLogs)
	m.keys.WrapText.SetEnabled(isLogs || isDescribe || isYaml)
	m.keys.CopyLogs.SetEnabled(isLogs)
	m.keys.ToggleLineNums.SetEnabled(isDescribe || isYaml)

	// Enable namespace keys only for namespace-aware resources
	canUseNS := m.isNamespaced(m.currentGVR.Resource)
	m.keys.AllNS.SetEnabled(canUseNS)
	m.keys.DefaultNS.SetEnabled(canUseNS)

	// Enable shell key only for pods and containers views
	isPods := m.currentGVR.Resource == k8s.ResourcePods
	isContainers := m.currentGVR.Resource == k8s.ResourceContainers
	m.keys.Shell.SetEnabled(isPods || isContainers)

	m.keys.FilterLogs.SetEnabled(isLogs)
}

// getLogPodName returns the pod name from navigation history for the current log view
func (m *Model) getLogPodName() string {
	// Check if we came from containers view (pod → containers → logs)
	if memento, ok := m.navigationHistory.FindMementoByResourceType(k8s.ResourcePods); ok {
		return memento.resourceName
	}
	return ""
}

// getLogContainerName returns the container name from navigation history for the current log view
func (m *Model) getLogContainerName() string {
	// Check if we came from containers view
	if memento, ok := m.navigationHistory.FindMementoByResourceType(k8s.ResourceContainers); ok {
		return memento.resourceName
	}
	// If we came directly from pods (single container), the container name is in the log viewport
	if m.logViewport != nil {
		return m.logViewport.containerName
	}
	return ""
}

// isContainerRunning checks if a container status string indicates a Running state.
func isContainerRunning(status string) bool {
	return status == "Running"
}

// shellQuote wraps a string in single quotes for safe shell interpolation,
// escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// resourceSorter implements sort.Interface, sorting m.resources and
// m.creationTimes in lockstep by the value at the given column index.
// This avoids allocating temporary index slices on every sort.
type resourceSorter struct {
	resources     []k8s.OrderedResourceFields
	creationTimes []time.Time
	col           int
}

func (s resourceSorter) Len() int { return len(s.resources) }
func (s resourceSorter) Less(i, j int) bool {
	return s.resources[i][s.col] < s.resources[j][s.col]
}
func (s resourceSorter) Swap(i, j int) {
	s.resources[i], s.resources[j] = s.resources[j], s.resources[i]
	if i < len(s.creationTimes) && j < len(s.creationTimes) {
		s.creationTimes[i], s.creationTimes[j] = s.creationTimes[j], s.creationTimes[i]
	}
}

// sortResourcesByColumn sorts m.resources and m.creationTimes in place by the
// string value at the given column index. Both slices stay in sync.
func (m *Model) sortResourcesByColumn(col int) {
	sort.Stable(resourceSorter{
		resources:     m.resources,
		creationTimes: m.creationTimes,
		col:           col,
	})
}
