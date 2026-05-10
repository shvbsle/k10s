# k10s — GPU-Aware Kubernetes TUI

## Quick Reference

```bash
cargo build                    # Build
cargo clippy -- -D warnings    # Lint (must pass before commit)
cargo fmt --check              # Format check
cargo test                     # Run all tests including render snapshots
cargo run                      # Run TUI (requires kubeconfig with GPU cluster)
cargo run -- --render-once     # Render one frame to stdout and exit (for debugging)
```

## Architecture Summary

- **View trait** (`src/views/mod.rs`): Each screen implements `View`. Stateless render, mutations in `update()`.
- **App router** (`src/app.rs`): Navigation stack, message dispatch, layout (header + active view).
- **Header** (`src/ui/header.rs`): Fixed 4-row bar — cluster context, help hints, kitten ASCII art.
- **Data pipeline**: `ClusterDataSource` → `tokio::mpsc` → `AppMsg` → `View::update()` → re-render.
- **Styles** (`src/ui/styles.rs`): Semantic color functions. Idle=yellow, Busy=green, Saturated=red, Degraded=magenta.

## TUI Testing & Verification Protocol

**You cannot run the TUI interactively.** You must verify rendering correctness through these mechanisms:

### 1. Render snapshot tests (`src/tests/`)

Use ratatui's `TestBackend` to render views into an in-memory buffer at specific terminal dimensions, then assert the buffer content.

```rust
use ratatui::{Terminal, backend::TestBackend};

fn render_fleet_at(width: u16, height: u16, nodes: Vec<FleetNode>) -> String {
    let backend = TestBackend::new(width, height);
    let mut terminal = Terminal::new(backend).unwrap();
    // ... set up app, feed nodes, render frame
    // ... extract buffer as string
}
```

**Always test at these dimensions:**
- 80×24 (minimum supported terminal)
- 120×40 (typical developer terminal)
- 200×50 (wide ultrawide monitor)

### 2. The `--render-once` flag

When implemented, `cargo run -- --render-once` renders a single frame and prints the buffer to stdout. Use this via bash to inspect actual rendering:

```bash
cargo run -- --render-once 2>/dev/null | cat -v
```

### 3. Before any TUI layout change, verify by:

1. Run `cargo test` — snapshot tests catch regressions.
2. Calculate column widths manually: sum all `Min()` constraints + column spacing. If total > 80, the layout WILL clip on standard terminals.
3. Check the render test output at 80 cols to confirm no truncation of essential content.

## Visual Design Rules

### Column Layout

- **Minimum supported terminal width: 80 columns.**
- Column widths must sum to ≤ 78 (80 minus 2 for block borders) at the default scroll position.
- If columns cannot fit in 80 cols, use `Constraint::Percentage` or `Constraint::Ratio` instead of `Min()`. Columns should shrink proportionally, not clip arbitrarily.
- The NODE column shows the **hostname** portion only (strip `.compute.internal` or similar suffixes). Long names truncate with `…` on the right, never clip silently.
- Columns that hold short fixed-width data (GPUs: "8x", ALLOC: "4/8") use `Constraint::Length(N)`.
- Columns that hold variable-width data (NODE, WORKLOAD) use `Constraint::Percentage` or `Constraint::Fill`.

### Horizontal Scroll

- Horizontal scroll shifts the viewport window over a virtual table wider than the terminal.
- Scrolling must visibly change what's displayed: columns that were partially visible become fully visible, and leftmost columns disappear.
- The scroll offset is shown in the block title: `◄ scroll:N`.
- `h`/Left = scroll left, `l`/Right = scroll right, `0` = reset to start.

### Header Bar

- Fixed 4 rows (3 content + 1 separator/padding).
- Left: green dot + context, K8s version, k10s version.
- Center: help hints (? help, : command, esc go back).
- Right: two kitten ASCII art figures (magenta, right-aligned).
- The context shown should be the **cluster name** from kubeconfig (e.g., `arn:aws:eks:...cluster/gpu-research`), not the raw URL.

## System Design Audit Checklist

Before submitting any code change, audit it against these criteria as a system design specialist:

### Correctness

- [ ] Does the change maintain all 8 core invariants from `docs/architecture.md`?
- [ ] Are there any panics possible? (unwrap on None, index out of bounds, division by zero)
- [ ] Does `render()` remain pure (no side effects, no mutations)?
- [ ] Does the change work when data is absent? (all external fields are `Option<T>` → display "—")

### Performance

- [ ] No allocations inside `render()` that grow with frame count (leaking vecs, string accumulation).
- [ ] No blocking calls on the main thread (all I/O in tokio tasks, communicating via mpsc).
- [ ] Data structures are O(n) where n = number of nodes/pods, not O(n²).
- [ ] No unnecessary clones of large data (Vec<FleetNode> etc.) — pass references where possible.

### Resilience

- [ ] What happens when the cluster connection drops? (Should show error state, not panic)
- [ ] What happens at 0 nodes? 1 node? 1000 nodes? (Bounds on display, no overflow)
- [ ] What happens at terminal resize? (Ratatui handles this, but verify no stale cached dimensions)
- [ ] What happens with missing DCGM? (Graceful degradation — shows "—" for metrics)

### Extensibility

- [ ] Can a new view be added without modifying existing views? (Rule 1)
- [ ] Can a new column be added without refactoring the table infrastructure?
- [ ] Can a new data source be added without touching the render loop?
- [ ] Are types precise? (No `String` where an enum would prevent invalid states)

### UX Consistency

- [ ] Does navigation follow the stack model? (Enter pushes, Esc pops)
- [ ] Are keybindings consistent with existing views?
- [ ] Do colors follow the semantic style system? (Don't hardcode `Color::Yellow`, use `styles::idle_style()`)
- [ ] Is the rendering correct at 80×24? (Test, don't assume)

### Code Quality

- [ ] `cargo clippy -- -D warnings` passes.
- [ ] `cargo fmt --check` passes.
- [ ] No `#[allow(unused)]` on new code (allowed only on Phase 0 scaffolding for future phases).
- [ ] No unnecessary abstractions (three similar lines > premature generic).
- [ ] No comments explaining WHAT (the code is clear), only WHY if non-obvious.

## Common Pitfalls (Learned from Experience)

1. **`Constraint::Min(N)` does not guarantee N columns.** If the sum of all `Min()` values exceeds available width, ratatui distributes space proportionally and columns get less than their minimum. Use `Length()` for truly fixed columns, `Percentage()`/`Fill(1)` for flexible ones. **Evidence**: With `COL_WIDTHS` summing to 112, an 80-col terminal gives NODE only ~10 chars, truncating `ip-172-31-0-0` to `ip-172-31-`. The fix is: use `Length()` for short fixed cols (GPUs=5, ALLOC=5, MEM=4, TEMP=4, POWER=5) and `Fill(1)` for elastic cols (NODE, MODEL, UTIL, WORKLOAD).

2. **Node names from AWS EKS are long** (`ip-172-31-10-5.us-west-2.compute.internal`). Strip the domain suffix before display, or the NAME column alone consumes 40+ chars.

3. **TestBackend dimensions must match real terminals.** Test at 80×24 minimum. A test that only passes at 200 cols is useless.

4. **The header consumes 4 rows.** Available height for the active view is `terminal_height - 4`. At 24 rows, that's only 20 rows for the table (including borders and header row = 17 data rows max).

5. **Horizontal scroll must be column-aware.** Pixel-level scroll doesn't exist in terminal UIs. Scroll by hiding/showing whole columns or by adjusting the first-column offset.

6. **Never use hardcoded/mock data.** The user has a real GPU cluster. Always connect live. Test infrastructure uses `TestBackend` with synthetic `FleetNode` structs injected via the message pipeline, not fake cluster connections.

## File Ownership

| Path | Owns |
|------|------|
| `src/app.rs` | App struct, AppContext, navigation, render layout |
| `src/ui/header.rs` | Header bar rendering |
| `src/ui/styles.rs` | Semantic color/style functions |
| `src/views/fleet.rs` | Fleet GPU dashboard view |
| `src/views/help.rs` | Help overlay popup |
| `src/views/mod.rs` | View trait definition |
| `src/k8s/cluster.rs` | Kubernetes data fetching |
| `src/k8s/dcgm.rs` | DCGM metric scraping |
| `src/data/node.rs` | FleetNode struct + NodeStatus enum |
| `src/data/gpu_metrics.rs` | GpuNodeMetrics struct |
| `src/msg.rs` | AppMsg enum |
| `src/action.rs` | Action enum (user intents) |
| `docs/roadmap.md` | Phase tracking |
| `docs/architecture.md` | Core invariants + message flow |

## Development Workflow

1. **Before coding**: Read the relevant view file + `docs/architecture.md` invariants.
2. **During coding**: Run `cargo clippy -- -D warnings` frequently. Fix warnings immediately.
3. **After coding**: Run full test suite. Verify render snapshots at 80×24 pass.
4. **Before committing**: The pre-commit hook runs fmt + clippy. If it fails, fix and retry.
5. **Design audit**: Walk through the System Design Audit Checklist above for every change.
