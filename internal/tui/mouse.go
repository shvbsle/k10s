package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ViewZone identifies a region of the terminal that can receive mouse events.
type ViewZone int

const (
	ZoneNone    ViewZone = iota
	ZoneHeader           // Top header area (cluster info, key hints, logo)
	ZoneTable            // Table data rows
	ZoneCommand          // Command palette at the bottom
)

// MouseAction describes what semantic action a mouse event maps to.
type MouseAction int

const (
	MouseActionNone MouseAction = iota
	MouseActionSelectRow
	MouseActionHoverRow
	MouseActionHoverClear
	MouseActionScrollUp
	MouseActionScrollDown
	MouseActionScrollLeft
	MouseActionScrollRight
)

// MouseEvent is the processed result of a raw terminal mouse event.
// It translates pixel/cell coordinates into semantic table actions.
type MouseEvent struct {
	Action MouseAction
	Row    int        // Table row index (valid for SelectRow, HoverRow)
	Zone   ViewZone   // Which zone the event occurred in
	Mod    tea.KeyMod // Modifier keys held during the event (shift, ctrl, alt)
}

// MouseHandler encapsulates all mouse interaction state and logic.
// It translates raw terminal mouse events into semantic actions
// that the Model can act on without knowing about coordinate math.
type MouseHandler struct {
	hoverRow   int // Currently hovered table row (-1 = none)
	dataStartY int // Terminal Y where table data rows begin (set during render)
	totalRows  int // Number of visible table rows (set during render)
	enabled    bool
}

// NewMouseHandler creates a MouseHandler with sensible defaults.
func NewMouseHandler() *MouseHandler {
	return &MouseHandler{
		hoverRow: -1,
		enabled:  true,
	}
}

// SetEnabled toggles mouse handling on or off.
func (mh *MouseHandler) SetEnabled(enabled bool) {
	mh.enabled = enabled
}

// BeginRender should be called at the start of table data row rendering.
// Pass the current builder content so the handler can compute the Y offset
// from the number of newlines already written.
func (mh *MouseHandler) BeginRender(contentSoFar string, totalRows int) {
	mh.dataStartY = strings.Count(contentSoFar, "\n")
	mh.totalRows = totalRows
}

// HoverRow returns the currently hovered row index, or -1 if none.
func (mh *MouseHandler) HoverRow() int {
	if !mh.enabled {
		return -1
	}
	return mh.hoverRow
}

// HandleEvent processes a raw bubbletea mouse message and returns a
// semantic MouseEvent. The Model uses this to decide what to do
// without embedding coordinate logic in the Update switch.
func (mh *MouseHandler) HandleEvent(msg tea.MouseMsg) MouseEvent {
	if !mh.enabled {
		return MouseEvent{Action: MouseActionNone}
	}

	m := msg.Mouse()

	switch msg.(type) {
	case tea.MouseClickMsg:
		row := mh.yToRow(m.Y)
		if row < 0 {
			return MouseEvent{Action: MouseActionNone, Zone: mh.yToZone(m.Y), Mod: m.Mod}
		}
		return MouseEvent{
			Action: MouseActionSelectRow,
			Row:    row,
			Zone:   ZoneTable,
			Mod:    m.Mod,
		}

	case tea.MouseMotionMsg:
		row := mh.yToRow(m.Y)
		if row < 0 {
			if mh.hoverRow != -1 {
				mh.hoverRow = -1
				return MouseEvent{Action: MouseActionHoverClear, Zone: mh.yToZone(m.Y), Mod: m.Mod}
			}
			return MouseEvent{Action: MouseActionNone, Zone: mh.yToZone(m.Y), Mod: m.Mod}
		}
		if row != mh.hoverRow {
			mh.hoverRow = row
			return MouseEvent{Action: MouseActionHoverRow, Row: row, Zone: ZoneTable, Mod: m.Mod}
		}
		return MouseEvent{Action: MouseActionNone, Zone: ZoneTable, Mod: m.Mod}

	case tea.MouseWheelMsg:
		switch m.Button {
		case tea.MouseWheelUp:
			return MouseEvent{Action: MouseActionScrollUp, Zone: mh.yToZone(m.Y), Mod: m.Mod}
		case tea.MouseWheelDown:
			return MouseEvent{Action: MouseActionScrollDown, Zone: mh.yToZone(m.Y), Mod: m.Mod}
		case tea.MouseWheelLeft:
			return MouseEvent{Action: MouseActionScrollLeft, Zone: mh.yToZone(m.Y), Mod: m.Mod}
		case tea.MouseWheelRight:
			return MouseEvent{Action: MouseActionScrollRight, Zone: mh.yToZone(m.Y), Mod: m.Mod}
		}
	}

	return MouseEvent{Action: MouseActionNone}
}

// yToRow converts a terminal Y coordinate to a table row index.
// Returns -1 if the Y is outside the table data area.
func (mh *MouseHandler) yToRow(y int) int {
	row := y - mh.dataStartY
	if row < 0 || row >= mh.totalRows {
		return -1
	}
	return row
}

// yToZone determines which view zone a Y coordinate falls in.
func (mh *MouseHandler) yToZone(y int) ViewZone {
	if y < mh.dataStartY {
		return ZoneHeader
	}
	if y < mh.dataStartY+mh.totalRows {
		return ZoneTable
	}
	return ZoneCommand
}

// RowStyle returns the appropriate lipgloss style for a given row index,
// considering selection and hover state.
func (mh *MouseHandler) RowStyle(rowIdx, selectedRow int, selected, hover, normal lipgloss.Style) lipgloss.Style {
	switch {
	case rowIdx == selectedRow:
		return selected
	case mh.enabled && rowIdx == mh.hoverRow:
		return hover
	default:
		return normal
	}
}

// IsHovered returns whether the given row index is currently hovered.
func (mh *MouseHandler) IsHovered(rowIdx int) bool {
	return mh.enabled && rowIdx == mh.hoverRow
}

// MouseMode returns the appropriate tea.MouseMode based on whether
// mouse handling is enabled.
func (mh *MouseHandler) MouseMode() tea.MouseMode {
	if mh.enabled {
		return tea.MouseModeAllMotion
	}
	return tea.MouseModeNone
}
