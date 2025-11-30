package tui

import (
	"strings"

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
	if m.currentGVR.Resource == k8s.ResourceDescribe && m.describeContent != "" {
		return len(strings.Split(m.describeContent, "\n"))
	}
	return len(m.resources)
}

func (m *Model) updateKeysForResourceType() {
	isLogs := m.currentGVR.Resource == k8s.ResourceLogs
	isDescribe := m.currentGVR.Resource == k8s.ResourceDescribe

	// Enable/disable log-specific keys
	m.keys.Fullscreen.SetEnabled(isLogs || isDescribe)
	m.keys.Autoscroll.SetEnabled(isLogs)
	m.keys.ToggleTime.SetEnabled(isLogs)
	m.keys.WrapText.SetEnabled(isLogs || isDescribe)
	m.keys.CopyLogs.SetEnabled(isLogs)
	m.keys.ToggleLineNums.SetEnabled(isDescribe)

	// Enable namespace keys only for namespace-aware resources
	canUseNS := m.isNamespaced(m.currentGVR.Resource)
	m.keys.AllNS.SetEnabled(canUseNS)
	m.keys.DefaultNS.SetEnabled(canUseNS)
}
