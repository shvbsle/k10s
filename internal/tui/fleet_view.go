package tui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
	"github.com/shvbsle/k10s/internal/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FleetTab represents which node filter is active.
type FleetTab int

const (
	FleetTabGPU FleetTab = iota
	FleetTabCPU
	FleetTabAll
)

// FleetView holds the state for the GPU-first fleet view of nodes.
type FleetView struct {
	ActiveTab FleetTab
	GPUCount  int
	CPUCount  int
}

func NewFleetView() *FleetView {
	return &FleetView{
		ActiveTab: FleetTabGPU,
	}
}

// NextTab cycles forward: GPU → CPU → All → GPU
func (f *FleetView) NextTab() {
	f.ActiveTab = (f.ActiveTab + 1) % 3
}

// PrevTab cycles backward: GPU → All → CPU → GPU
func (f *FleetView) PrevTab() {
	f.ActiveTab = (f.ActiveTab + 2) % 3
}

// TabName returns the display name for a tab.
func TabName(tab FleetTab) string {
	switch tab {
	case FleetTabGPU:
		return "GPU"
	case FleetTabCPU:
		return "CPU"
	case FleetTabAll:
		return "All"
	}
	return ""
}

// RenderCountBadge renders "(N GPU · M CPU · N+M All)" for the header with
// active tab highlighting. The active tab's label and count are rendered in
// its accent color; inactive parts are dim.
func (f *FleetView) RenderCountBadge() string {
	amberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	blueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	total := f.GPUCount + f.CPUCount
	sep := sepStyle.Render(" · ")

	var gpuPart, cpuPart, allPart string

	switch f.ActiveTab {
	case FleetTabGPU:
		gpuPart = amberStyle.Render(fmt.Sprintf("%d GPU", f.GPUCount))
		cpuPart = dimStyle.Render(fmt.Sprintf("%d CPU", f.CPUCount))
		allPart = dimStyle.Render(fmt.Sprintf("%d All", total))
	case FleetTabCPU:
		gpuPart = dimStyle.Render(fmt.Sprintf("%d GPU", f.GPUCount))
		cpuPart = blueStyle.Render(fmt.Sprintf("%d CPU", f.CPUCount))
		allPart = dimStyle.Render(fmt.Sprintf("%d All", total))
	case FleetTabAll:
		gpuPart = dimStyle.Render(fmt.Sprintf("%d GPU", f.GPUCount))
		cpuPart = dimStyle.Render(fmt.Sprintf("%d CPU", f.CPUCount))
		allPart = amberStyle.Render(fmt.Sprintf("%d All", total))
	}

	return fmt.Sprintf("(%s%s%s%s%s)", gpuPart, sep, cpuPart, sep, allPart)
}

// ClassifyAndCount classifies all nodes and updates the GPU/CPU counts.
func (f *FleetView) ClassifyAndCount(nodes []*unstructured.Unstructured) {
	gpu, cpu := 0, 0
	for _, node := range nodes {
		if k8s.ClassifyNode(node) == k8s.NodeClassCPU {
			cpu++
		} else {
			gpu++
		}
	}
	f.GPUCount = gpu
	f.CPUCount = cpu

	// Auto-switch to CPU tab if no GPU nodes exist
	if gpu == 0 && f.ActiveTab == FleetTabGPU {
		f.ActiveTab = FleetTabCPU
	}
}

// allocBarModel is a reusable progress bar for rendering allocation bars.
// Created once to avoid per-call allocation overhead.
var allocBarModel = progress.New(
	progress.WithWidth(15),
	progress.WithoutPercentage(),
	progress.WithColorFunc(func(total, current float64) color.Color {
		switch {
		case total >= 0.9:
			return lipgloss.Color("#04B575") // green — fully used
		case total >= 0.5:
			return lipgloss.Color("#FFFF00") // yellow — mid
		default:
			return lipgloss.Color("#FF6600") // orange — underused
		}
	}),
)

// RenderAllocBar renders a progress bar using the bubbles progress component.
func RenderAllocBar(pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1.0 {
		pct = 1.0
	}
	allocBarModel.SetWidth(width)
	return allocBarModel.ViewAs(pct)
}

// rawObjectPtrs returns a slice of pointers to the model's rawObjects.
// Used by the fleet view for classification and counting.
func (m *Model) rawObjectPtrs() []*unstructured.Unstructured {
	ptrs := make([]*unstructured.Unstructured, len(m.rawObjects))
	for i := range m.rawObjects {
		ptrs[i] = &m.rawObjects[i]
	}
	return ptrs
}

// updateFleetView filters and re-sorts the table data based on the active
// fleet tab. This operates entirely on in-memory data — no API calls.
func (m *Model) updateFleetView() {
	if m.currentGVR.Resource != k8s.ResourceNodes || m.fleetView == nil {
		return
	}

	m.applyFleetFilter()
	m.updateColumns(m.viewWidth)
	m.updateTableData()
	m.table.SetCursor(0)
}

// applyFleetFilter filters m.resources from allResources based on the active
// fleet tab. rawObjects and allResources must be the same length and in the
// same order.
func (m *Model) applyFleetFilter() {
	if len(m.allResources) == 0 || len(m.rawObjects) == 0 {
		return
	}

	if m.fleetView.ActiveTab == FleetTabAll {
		// Copy so we can sort without mutating the originals.
		filtered := make([]k8s.OrderedResourceFields, len(m.allResources))
		copy(filtered, m.allResources)
		filteredTimes := make([]time.Time, len(m.allCreationTimes))
		copy(filteredTimes, m.allCreationTimes)
		sortFilteredResources(filtered, filteredTimes, FleetTabAll)
		m.resources = filtered
		m.creationTimes = filteredTimes
		return
	}

	filtered := make([]k8s.OrderedResourceFields, 0, len(m.allResources))
	filteredTimes := make([]time.Time, 0, len(m.allCreationTimes))

	for i, obj := range m.rawObjects {
		class := k8s.ClassifyNode(&obj)
		include := false
		switch m.fleetView.ActiveTab {
		case FleetTabGPU:
			include = class != k8s.NodeClassCPU
		case FleetTabCPU:
			include = class == k8s.NodeClassCPU
		}
		if include {
			filtered = append(filtered, m.allResources[i])
			if i < len(m.allCreationTimes) {
				filteredTimes = append(filteredTimes, m.allCreationTimes[i])
			}
		}
	}

	// Sort the filtered results and keep filteredTimes in sync.
	sortFilteredResources(filtered, filteredTimes, m.fleetView.ActiveTab)

	m.resources = filtered
	m.creationTimes = filteredTimes
}

// sortFilteredResources sorts the filtered resource rows (and their
// corresponding creation times) based on the active fleet tab.
//   - GPU tab: sort by the Alloc column value (index 3) ascending
//   - CPU tab: sort by the Name column (index 0) ascending
//   - All tab: GPU nodes first (sorted by Alloc), then CPU nodes (sorted by Name)
func sortFilteredResources(rows []k8s.OrderedResourceFields, times []time.Time, tab FleetTab) {
	if len(rows) <= 1 {
		return
	}

	// Build an index slice so we can sort rows and times together.
	indices := make([]int, len(rows))
	for i := range indices {
		indices[i] = i
	}

	sort.SliceStable(indices, func(a, b int) bool {
		ra := rows[indices[a]]
		rb := rows[indices[b]]

		switch tab {
		case FleetTabGPU:
			// Sort by Alloc column (index 3) ascending — string comparison
			// works because the bar format is consistent width.
			allocA, allocB := "", ""
			if len(ra) > 3 {
				allocA = ra[3]
			}
			if len(rb) > 3 {
				allocB = rb[3]
			}
			return allocA < allocB

		case FleetTabCPU:
			// Sort by Name column (index 0) ascending.
			nameA, nameB := "", ""
			if len(ra) > 0 {
				nameA = ra[0]
			}
			if len(rb) > 0 {
				nameB = rb[0]
			}
			return nameA < nameB

		case FleetTabAll:
			// GPU nodes first, then CPU nodes.
			// Within GPU: sort by Alloc (index 3).
			// Within CPU: sort by Name (index 0).
			computeA, computeB := "", ""
			if len(ra) > 2 {
				computeA = ra[2]
			}
			if len(rb) > 2 {
				computeB = rb[2]
			}
			aIsGPU := strings.HasPrefix(computeA, "gpu")
			bIsGPU := strings.HasPrefix(computeB, "gpu")

			if aIsGPU != bIsGPU {
				return aIsGPU // GPU nodes come first
			}
			if aIsGPU {
				// Both GPU — sort by Alloc column
				allocA, allocB := "", ""
				if len(ra) > 3 {
					allocA = ra[3]
				}
				if len(rb) > 3 {
					allocB = rb[3]
				}
				return allocA < allocB
			}
			// Both CPU — sort by Name column
			nameA, nameB := "", ""
			if len(ra) > 0 {
				nameA = ra[0]
			}
			if len(rb) > 0 {
				nameB = rb[0]
			}
			return nameA < nameB
		}
		return false
	})

	// Apply the sorted order to both slices.
	sortedRows := make([]k8s.OrderedResourceFields, len(rows))
	sortedTimes := make([]time.Time, len(times))
	for i, idx := range indices {
		sortedRows[i] = rows[idx]
		if idx < len(times) {
			sortedTimes[i] = times[idx]
		}
	}
	copy(rows, sortedRows)
	if len(times) > 0 {
		copy(times, sortedTimes)
	}
}
