package tui

import (
	"charm.land/bubbles/v2/table"
	"github.com/shvbsle/k10s/internal/k8s"
)

// LogViewState holds configuration for log viewing.
type LogViewState struct {
	Autoscroll     bool
	Fullscreen     bool
	ShowTimestamps bool
	WrapText       bool
}

// NewLogViewState creates a LogViewState with sensible defaults.
func NewLogViewState() *LogViewState {
	return &LogViewState{
		Autoscroll:     true,
		Fullscreen:     false,
		ShowTimestamps: false,
		WrapText:       false,
	}
}

// DescribeViewState holds configuration for describe view.
type DescribeViewState struct {
	Fullscreen      bool
	WrapText        bool
	ShowLineNumbers bool
}

// NewDescribeViewState creates a DescribeViewState with sensible defaults.
func NewDescribeViewState() *DescribeViewState {
	return &DescribeViewState{
		Fullscreen:      true,
		WrapText:        false,
		ShowLineNumbers: true,
	}
}

// DisplayRow represents a single row that can be displayed in the TUI table.
// Different resource types implement this interface to provide consistent table rendering.
type DisplayRow interface {
	// ToTableRow converts the display row into a table.Row for rendering.
	ToTableRow(includeTimestamp bool) table.Row
}

// ResourceRow wraps a Kubernetes resource for display.
type ResourceRow struct {
	Resource k8s.OrderedResourceFields
}

// ToTableRow converts a Kubernetes resource into a table row.
func (r ResourceRow) ToTableRow(_ bool) table.Row {
	return table.Row(r.Resource)
}

// LogLine represents a single line of container logs with metadata.
type LogLine struct {
	LineNum   int
	Timestamp string
	Content   string
}

// ToTableRow converts a log line into a table row.
// When includeTimestamp is true and Timestamp is not empty, it's included.
func (l LogLine) ToTableRow(includeTimestamp bool) table.Row {
	var tsColumn string
	if includeTimestamp && l.Timestamp != "" {
		tsColumn = l.Timestamp
	}

	return table.Row{
		l.Content,
		tsColumn,
		"",
		"",
		"",
		"",
	}
}
