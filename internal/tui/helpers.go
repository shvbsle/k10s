package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/shvbsle/k10s/internal/k8s"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var namespacedCache = map[string]bool{}

// resolvedResource holds the result of resolving a resource name against the
// server's discovery API: the full GVR and whether it's namespace-scoped.
type resolvedResource struct {
	GVR        schema.GroupVersionResource
	Namespaced bool
}

// resolveGVR looks up a resource name in the server's preferred resources and
// returns the fully qualified GVR along with its namespace scope. It populates
// namespacedCache as a side effect so that isNamespaced stays consistent.
func (m *Model) resolveGVR(resource string) resolvedResource {
	result := resolvedResource{
		GVR:        schema.GroupVersionResource{Resource: resource},
		Namespaced: true,
	}

	apiResourceLists, err := m.k8sClient.Discovery().ServerPreferredResources()
	if err != nil && len(apiResourceLists) == 0 {
		return result
	}

	for _, apiResourceList := range apiResourceLists {
		gv, _ := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		for _, apiResource := range apiResourceList.APIResources {
			if apiResource.Name == resource {
				result.GVR = gv.WithResource(apiResource.Name)
				result.Namespaced = apiResource.Namespaced
				namespacedCache[resource] = apiResource.Namespaced
				return result
			}
		}
	}

	return result
}

// isNamespaced returns whether the resource is cluster or namespace scoped
func (m *Model) isNamespaced(resource string) bool {
	if supports, ok := namespacedCache[resource]; ok {
		return supports
	}

	resolved := m.resolveGVR(resource)
	return resolved.Namespaced
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
