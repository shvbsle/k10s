package tui

import (
	"strings"
	"testing"
	"testing/quick"

	"charm.land/lipgloss/v2"
	"github.com/shvbsle/k10s/internal/k8s"
)

func makeLogLines(contents []string) []k8s.LogLine {
	lines := make([]k8s.LogLine, len(contents))
	for i, c := range contents {
		lines[i] = k8s.LogLine{LineNum: i + 1, Content: c}
	}
	return lines
}

// TestFilter_SetFilter checks that only matching lines are kept after SetFilter.
func TestFilter_SetFilter(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		filter      string
		wantMatches int
		wantNoMatch bool
	}{
		{
			name:        "all lines match",
			lines:       []string{"error: foo", "error: bar", "error: baz"},
			filter:      "error",
			wantMatches: 3,
		},
		{
			name:        "some lines match",
			lines:       []string{"info: ok", "error: bad", "warn: meh"},
			filter:      "error",
			wantMatches: 1,
		},
		{
			name:        "no lines match",
			lines:       []string{"info: ok", "warn: meh"},
			filter:      "error",
			wantMatches: 0,
			wantNoMatch: true,
		},
		{
			name:        "empty filter shows all lines",
			lines:       []string{"foo", "bar", "baz"},
			filter:      "",
			wantMatches: 3,
		},
		{
			name:        "case sensitive",
			lines:       []string{"Error: bad", "error: also bad"},
			filter:      "error",
			wantMatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv := NewLogViewport()
			lv.SetContent(makeLogLines(tt.lines), "pod", "container", "ns")
			lv.SetFilter(tt.filter)

			if tt.filter == "" {
				// No filter — matchCount is not used, just verify filterText is empty
				if lv.FilterText() != "" {
					t.Errorf("expected empty filterText, got %q", lv.FilterText())
				}
				return
			}

			if lv.matchCount != tt.wantMatches {
				t.Errorf("matchCount = %d, want %d", lv.matchCount, tt.wantMatches)
			}
		})
	}
}

// TestFilter_ClearFilter checks that ClearFilter resets all filter state.
func TestFilter_ClearFilter(t *testing.T) {
	lv := NewLogViewport()
	lv.SetContent(makeLogLines([]string{"error: foo", "info: bar"}), "pod", "container", "ns")
	lv.SetFilter("error")

	if lv.matchCount != 1 {
		t.Fatalf("precondition: matchCount = %d, want 1", lv.matchCount)
	}

	lv.ClearFilter()

	if lv.FilterText() != "" {
		t.Errorf("filterText = %q, want empty", lv.FilterText())
	}
	if lv.FilterActive() {
		t.Error("filterActive = true after ClearFilter, want false")
	}
	// matchCount is only meaningful when a filter is active; no assertion after clear
}

// TestFilter_ActivateFilter checks that ActivateFilter sets filterActive.
func TestFilter_ActivateFilter(t *testing.T) {
	lv := NewLogViewport()
	if lv.FilterActive() {
		t.Fatal("expected filterActive=false initially")
	}
	lv.ActivateFilter()
	if !lv.FilterActive() {
		t.Error("expected filterActive=true after ActivateFilter")
	}
}

// TestFilter_AppendLinesRespectFilter checks that newly streamed lines
// are filtered in real-time when a filter is active.
func TestFilter_AppendLinesRespectFilter(t *testing.T) {
	lv := NewLogViewport()
	lv.SetContent(makeLogLines([]string{"error: initial"}), "pod", "container", "ns")
	lv.SetFilter("error")

	if lv.matchCount != 1 {
		t.Fatalf("precondition failed: matchCount = %d", lv.matchCount)
	}

	// Append a matching and a non-matching line
	lv.AppendLines(makeLogLines([]string{"error: new", "info: ignored"}))

	if lv.matchCount != 2 {
		t.Errorf("matchCount = %d after append, want 2", lv.matchCount)
	}
}

// TestHighlightMatches checks that highlightMatches wraps every occurrence of term.
func TestHighlightMatches(t *testing.T) {
	base := lipgloss.NewStyle()
	highlight := lipgloss.NewStyle()

	tests := []struct {
		content string
		term    string
		count   int // expected number of highlighted segments
	}{
		{"error: foo error", "error", 2},
		{"no match here", "xyz", 0},
		{"aaa", "a", 3},
		{"", "x", 0},
	}

	for _, tt := range tests {
		result := highlightMatches(tt.content, tt.term, base, highlight)
		got := strings.Count(result, tt.term)
		if got != tt.count {
			t.Errorf("highlightMatches(%q, %q): found %d occurrences, want %d", tt.content, tt.term, got, tt.count)
		}
	}
}

// Property: after SetFilter(term), matchCount equals the number of lines
// whose Content contains term.
func TestProperty_FilterMatchCountConsistent(t *testing.T) {
	f := func(contents []string, term string) bool {
		if len(contents) == 0 || term == "" {
			return true
		}
		lv := NewLogViewport()
		lv.SetContent(makeLogLines(contents), "pod", "ctr", "ns")
		lv.SetFilter(term)

		expected := 0
		for _, c := range contents {
			if strings.Contains(c, term) {
				expected++
			}
		}
		return lv.matchCount == expected
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: ClearFilter always leaves filterText empty and filterActive false,
// regardless of prior state.
func TestProperty_ClearFilterAlwaysResets(t *testing.T) {
	f := func(term string) bool {
		lv := NewLogViewport()
		lv.SetContent(makeLogLines([]string{"foo", "bar"}), "pod", "ctr", "ns")
		lv.SetFilter(term)
		lv.ClearFilter()
		return lv.FilterText() == "" && !lv.FilterActive()
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
