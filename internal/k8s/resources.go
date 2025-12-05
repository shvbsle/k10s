package k8s

import (
	"charm.land/bubbles/v2/table"
	"github.com/samber/lo"
)

// ResourceType represents the type of Kubernetes resource being displayed.
type ResourceType = string

const (
	// ResourcePods represents Kubernetes pods.
	ResourcePods ResourceType = "pods"
	// ResourceNodes represents Kubernetes nodes.
	ResourceNodes ResourceType = "nodes"
	// ResourceNamespaces represents Kubernetes namespaces.
	ResourceNamespaces ResourceType = "namespaces"
	// ResourceServices represents Kubernetes services.
	ResourceServices ResourceType = "services"
	// ResourceContainers represents containers within a pod.
	ResourceContainers ResourceType = "containers"
	// ResourceLogs represents logs for a specific container.
	ResourceLogs ResourceType = "logs"
	// ResourceAPIResources represents the list of available API resources.
	ResourceAPIResources ResourceType = "api-resources"
	// ResourceDescribe represents the YAML description of a resource.
	ResourceDescribe ResourceType = "describe"
)

// OrderedResourceFields represents a Kubernetes resource with common fields suitable for
// display in the TUI table view.
type OrderedResourceFields []string

func NamespaceColumn(columns []table.Column) (int, bool) {
	_, index, ok := lo.FindIndexOf(columns, func(col table.Column) bool {
		return col.Title == "Namespace"
	})
	return index, ok
}

func NameColumn(columns []table.Column) (int, bool) {
	_, index, ok := lo.FindIndexOf(columns, func(col table.Column) bool {
		return col.Title == "Name"
	})
	return index, ok
}
