package tui

import (
	"github.com/shvbsle/k10s/internal/k8s"
)

// ModelMemento represents an immutable snapshot of the Model's state for navigation.
type ModelMemento struct {
	resources        []k8s.Resource
	resourceType     k8s.ResourceType
	currentNamespace string
	tableCursor      int
	paginatorPage    int
	err              error
	logAutoscroll    bool
	logFullscreen    bool
	logTimestamps    bool
	resourceName     string
	namespace        string
}

// NavigationHistory manages a stack of mementos for navigation.
type NavigationHistory struct {
	mementos []*ModelMemento
}

// NewNavigationHistory creates a new empty navigation history.
func NewNavigationHistory() *NavigationHistory {
	return &NavigationHistory{
		mementos: make([]*ModelMemento, 0),
	}
}

// Push adds a new memento to the history stack.
func (h *NavigationHistory) Push(memento *ModelMemento) {
	h.mementos = append(h.mementos, memento)
}

// Pop removes and returns the most recent memento from the history stack.
// Returns nil if the stack is empty.
func (h *NavigationHistory) Pop() *ModelMemento {
	if len(h.mementos) == 0 {
		return nil
	}

	lastIdx := len(h.mementos) - 1
	memento := h.mementos[lastIdx]
	h.mementos = h.mementos[:lastIdx]
	return memento
}

// Peek returns the most recent memento without removing it from the stack.
// Returns nil if the stack is empty.
func (h *NavigationHistory) Peek() *ModelMemento {
	if len(h.mementos) == 0 {
		return nil
	}
	return h.mementos[len(h.mementos)-1]
}

// Len returns the number of mementos in the history.
func (h *NavigationHistory) Len() int {
	return len(h.mementos)
}

// Clear removes all mementos from the history.
func (h *NavigationHistory) Clear() {
	h.mementos = make([]*ModelMemento, 0)
}

// GetBreadcrumb returns a slice of resource information for breadcrumb display.
// Each element contains the resource type and name at that navigation level.
func (h *NavigationHistory) GetBreadcrumb() []struct {
	ResourceType k8s.ResourceType
	ResourceName string
} {
	breadcrumb := make([]struct {
		ResourceType k8s.ResourceType
		ResourceName string
	}, len(h.mementos))

	for i, memento := range h.mementos {
		breadcrumb[i].ResourceType = memento.resourceType
		breadcrumb[i].ResourceName = memento.resourceName
	}

	return breadcrumb
}

// GetMementoAt returns the memento at the specified index.
// Returns nil if the index is out of bounds.
func (h *NavigationHistory) GetMementoAt(index int) *ModelMemento {
	if index < 0 || index >= len(h.mementos) {
		return nil
	}
	return h.mementos[index]
}

// FindMementoByResourceType searches backwards through the history for the first
// memento matching the given resource type. Returns the memento and its index,
// or nil and -1 if not found.
func (h *NavigationHistory) FindMementoByResourceType(resType k8s.ResourceType) (*ModelMemento, int) {
	for i := len(h.mementos) - 1; i >= 0; i-- {
		if h.mementos[i].resourceType == resType {
			return h.mementos[i], i
		}
	}
	return nil, -1
}
