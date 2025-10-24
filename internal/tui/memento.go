package tui

import (
	"github.com/shvbsle/k10s/internal/k8s"
)

// ModelMemento captures Model state for drill-down navigation.
type ModelMemento struct {
	resources        []k8s.Resource
	resourceType     k8s.ResourceType
	currentNamespace string
	tableCursor      int
	paginatorPage    int
	err              error
	logView          *LogViewState
	resourceName     string
	namespace        string
}

// NavigationHistory manages navigation state as a stack.
type NavigationHistory struct {
	mementos []*ModelMemento
}

func NewNavigationHistory() *NavigationHistory {
	return &NavigationHistory{
		mementos: make([]*ModelMemento, 0),
	}
}

func (h *NavigationHistory) Push(memento *ModelMemento) {
	h.mementos = append(h.mementos, memento)
}

func (h *NavigationHistory) Pop() *ModelMemento {
	if len(h.mementos) == 0 {
		return nil
	}

	lastIdx := len(h.mementos) - 1
	memento := h.mementos[lastIdx]
	h.mementos = h.mementos[:lastIdx]
	return memento
}

func (h *NavigationHistory) Peek() *ModelMemento {
	if len(h.mementos) == 0 {
		return nil
	}
	return h.mementos[len(h.mementos)-1]
}

func (h *NavigationHistory) Len() int {
	return len(h.mementos)
}

func (h *NavigationHistory) Clear() {
	h.mementos = make([]*ModelMemento, 0)
}

// GetBreadcrumb returns navigation path for UI display.
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

// FindMementoByResourceType searches backwards for a memento with the given type.
func (h *NavigationHistory) FindMementoByResourceType(resType k8s.ResourceType) (*ModelMemento, int) {
	for i := len(h.mementos) - 1; i >= 0; i-- {
		if h.mementos[i].resourceType == resType {
			return h.mementos[i], i
		}
	}
	return nil, -1
}
