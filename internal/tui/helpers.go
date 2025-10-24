package tui

import "github.com/shvbsle/k10s/internal/k8s"

// isNamespaceAware returns true if the resource type supports namespace filtering.
// Node, Container, and Log resource types are not namespace-aware.
func isNamespaceAware(resType k8s.ResourceType) bool {
	switch resType {
	case k8s.ResourceNodes, k8s.ResourceContainers, k8s.ResourceLogs:
		return false
	default:
		return true
	}
}

// getTotalItems returns the total number of items (resources or log lines) to paginate.
func (m Model) getTotalItems() int {
	if m.resourceType == k8s.ResourceLogs && m.logLines != nil {
		return len(m.logLines)
	}
	return len(m.resources)
}
