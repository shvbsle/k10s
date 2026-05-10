# Known Issues

## Open

### NODE column clipped at narrow terminals

**Status:** Open  
**Introduced:** 2026-05-10  
**Affected:** Fleet view at terminals < 120 cols

**Problem:** `COL_WIDTHS` uses `Constraint::Min(N)` for all 9 columns, summing to 112. At 80 cols (78 usable inside borders), ratatui compresses all columns proportionally. NODE gets ~10 chars, truncating `ip-172-31-10-5` to `ip-172-31-`.

**Evidence** (from `cargo test fleet_renders_at_80x24 -- --nocapture`):
```
│ip-172-31- Tesla-T4   8x    0/8     —          —     —      —       IDLE      │
```

**Root cause:** Using `Min()` when the sum exceeds available width. Ratatui treats `Min` as a preference, not a guarantee. When total demand > supply, all columns shrink proportionally.

**Fix:** Replace column constraints with:
- `Length(N)` for short fixed-width columns (GPUs=4, ALLOC=5, MEM=4, TEMP=4, POWER=5)
- `Fill(1)` for elastic columns (NODE, MODEL, UTIL, WORKLOAD) so they share remaining space

**Verification:** After fix, `cargo test fleet_renders_at_80x24 -- --nocapture` must show at least 15 chars of node name (e.g., `ip-172-31-10-5`).

---

### Horizontal scroll has no visible effect

**Status:** Open  
**Introduced:** 2026-05-10  
**Affected:** Fleet view

**Problem:** The horizontal scroll implementation hides entire columns from the left, but since all columns are already compressed to fit the terminal width, removing a column from the left just gives remaining columns more room — it doesn't reveal new content on the right that was previously hidden.

**Root cause:** Horizontal scroll was designed for a virtual table wider than the terminal. But because all columns already fit (squeezed) in the terminal, there's nothing off-screen to scroll into.

**Fix:** The table's virtual width must exceed the terminal width. Columns should be sized at their *desired* width (not compressed), and only the visible viewport window rendered. This requires either:
1. Rendering columns at full `Length(N)` widths and clipping the viewport window (shift which columns are visible), or
2. Using ratatui's upcoming viewport/scroll features if available in 0.29+

**Depends on:** Fixing the column constraint issue above first.

---

## Resolved

(None yet)
