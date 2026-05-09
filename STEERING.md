# k10s Rust Rewrite — Steering Doc

This document is the architectural contract for the Rust rewrite of k10s. Any AI assistant, pair programmer, or future contributor MUST read this before writing code. It exists because the Go implementation failed due to vibe-coding without architectural guardrails.

## What k10s Is

A GPU-aware Kubernetes terminal dashboard. Single binary. Vim keybindings. The atoms are **GPUs and Jobs**, not pods. The signature view is a fleet-level GPU dashboard with idle detection, utilization bars, and workload attribution.

See `.kiro/specs/vision.md` and `.kiro/specs/design-redo.md` for full product context.

---

## Hard Rules (Non-Negotiable)

These rules exist because their violation killed the Go implementation. Every one maps to a concrete failure mode.

### 1. Views Are Types, Not Branches

**Rule:** Each view (fleet, node-detail, logs, describe, jobs, workloads, events) MUST be a separate type implementing a `View` trait. No `match current_view { ... }` sprawl in the app-level update/render.

**Why:** The Go version used `if m.currentGVR.Resource == "logs"` in 20+ places. Adding a view meant touching every conditional. Missing one caused a regression.

**Trait signature (approximate):**
```rust
trait View {
    fn update(&mut self, msg: &Msg, ctx: &mut AppContext) -> Option<Action>;
    fn render(&self, frame: &mut Frame, area: Rect, ctx: &AppContext);
    fn key_map(&self) -> &[KeyBinding];
    fn on_enter(&mut self, ctx: &mut AppContext);  // called when view becomes active
    fn on_leave(&mut self);                         // called when view is deactivated
}
```

**Test:** Can you add a new view by creating a single new file that implements `View`, without modifying any existing view? If not, the architecture is wrong.

### 2. No Shared Mutable State Between Views

**Rule:** Views own their data. The only shared state is read-only context (k8s client handle, cluster info, config). Views receive data via messages, not by reading mutable fields on a god object.

**Why:** The Go version had `m.resources`, `m.logLines`, `m.describeContent`, `m.allResources` all as fields on one struct. Switching views required manually nil'ing the right fields. Missing one leaked stale data.

**Pattern:** When navigating from fleet → node-detail, the fleet view sends a `Navigate(NodeDetail { node_name, node_data })` action. The app router creates/activates the node-detail view with that data. The fleet view's state is preserved on a stack for back-navigation.

### 3. Typed Data Models, Not Positional Arrays

**Rule:** Never represent a row as `Vec<String>`. Each resource type has a typed struct. Column rendering is a function from struct → cells, not positional indexing.

**Why:** The Go version used `[]string` for all resources. Sort code hardcoded `row[3]` as "Alloc" and `row[2]` as "Compute". Adding a column broke every index.

```rust
struct FleetNode {
    name: String,
    instance_type: String,
    gpu_model: Option<String>,
    gpu_count: u32,
    utilization: Option<f32>,  // None when DCGM unavailable
    memory_pct: Option<f32>,
    temperature: Option<u32>,
    power_watts: Option<u32>,
    workload: Option<String>,
    idle_duration: Option<Duration>,
    status: NodeStatus,
}
```

### 4. Async Events Become Messages on the Main Loop

**Rule:** Background tasks (k8s watch, DCGM polling, log streaming) MUST communicate with the TUI via a message channel. They MUST NOT hold references to or mutate view state directly.

**Why:** The Go version's watch goroutine mutated `m.resources` directly while `View()` read it on the main loop — a data race. The fix was a channel signal *after* mutation, which was still wrong.

**Pattern:**
```rust
enum AppMsg {
    WatchEvent(WatchEvent),
    DcgmMetrics(Vec<GpuMetrics>),
    LogLine(String),
    Tick,
    Input(KeyEvent),
}
```
The main loop `select!`s on input + message channel. Views never spawn tasks themselves — they return `Action::StartWatch(...)` or `Action::FetchLogs(...)` and the app runner handles it.

### 5. Key Bindings Are Scoped Per View

**Rule:** Each view declares its own key map via `fn key_map(&self) -> &[KeyBinding]`. The app router checks the active view's key map first, then falls through to global bindings (`:`, `ctrl+c`, `?`).

**Why:** The Go version had `s` meaning autoscroll in logs, shell in pods, and nothing elsewhere — all dispatched in one flat switch. Adding vision keys (`w`, `e`, `g`) collided with existing bindings.

**Pattern:**
```rust
struct KeyBinding {
    key: KeyEvent,
    action: Action,
    description: &'static str,
    // Display in help? Some bindings are internal
    show_in_help: bool,
}
```

Global bindings (always active): `:` (command), `ctrl+c` (quit), `?` (help), `esc` (back).
View bindings: declared by each view, override globals if there's a conflict within that view.

### 6. Navigation Is a Stack with Typed Transitions

**Rule:** Use an explicit navigation stack. Each entry stores the view type + its state. `Esc` pops. `Enter` pushes. The stack type is `Vec<Box<dyn View>>` or equivalent.

**Why:** The Go version's memento pattern was ad-hoc — it saved/restored 15 fields manually and forgot fleet-view-specific fields in early iterations.

**Pattern:**
```rust
struct App {
    view_stack: Vec<Box<dyn View>>,
    // Active view is always view_stack.last_mut()
}
```

Pushing: current view gets `on_leave()`, new view gets `on_enter()`, pushed onto stack.
Popping: current view is dropped, previous view gets `on_enter()` again (to refresh if needed).

### 7. Graceful Degradation Is a First-Class Concept

**Rule:** Every data field that comes from DCGM (utilization, memory, temp, power) MUST be `Option<T>`. Views render `—` or a hint when None. Never panic or show broken UI when DCGM is absent.

**Why:** Requiring DCGM to try k10s is a conversion killer. The fleet view must be useful with just the k8s API (GPU count from allocatable, workload from pod scheduling, idle from "no GPU-requesting pods on this node").

### 8. Rendering Is Stateless

**Rule:** `render()` is a pure function of view state → terminal frame. No side effects, no mutations, no fetching. All mutations happen in `update()`.

**Why:** The Go version's `View()` method read mutable state that the watch goroutine was concurrently modifying. Keeping render stateless eliminates an entire class of bugs.

---

## Recommended Crate Stack

| Concern | Crate | Why |
|---------|-------|-----|
| TUI framework | `ratatui` | De facto standard. Immediate-mode rendering. |
| Async runtime | `tokio` | k8s client needs async. |
| K8s client | `kube` | Mature, supports watch/informers, CRDs. |
| Terminal backend | `crossterm` | Cross-platform, works in all terminals. |
| Serialization | `serde` + `serde_json`/`serde_yaml` | Standard. |
| HTTP (DCGM scrape) | `reqwest` | For Prometheus endpoint scraping. |
| Prometheus parsing | `prometheus-parse` | Parse DCGM exporter metrics. |

---

## Project Structure

```
src/
├── main.rs              # Entry point, tokio runtime, app loop
├── app.rs               # App struct, message dispatch, navigation stack
├── config.rs            # Config file loading, key binding config
├── msg.rs               # All message types (AppMsg enum)
├── action.rs            # All action types views can return
├── views/
│   ├── mod.rs           # View trait definition
│   ├── fleet.rs         # Fleet view (default landing)
│   ├── node_detail.rs   # Per-GPU node detail
│   ├── jobs.rs          # Training jobs grouped view
│   ├── workloads.rs     # Workloads filtered to a node
│   ├── events.rs        # Events filtered to a node
│   ├── logs.rs          # Container log streaming
│   └── describe.rs      # Resource describe/yaml
├── data/
│   ├── mod.rs           # Data layer types
│   ├── node.rs          # FleetNode, NodeDetail structs
│   ├── gpu.rs           # GpuMetrics, GpuStatus
│   ├── job.rs           # TrainingJob, Rank
│   └── pod.rs           # Pod summary (for workload attribution)
├── k8s/
│   ├── mod.rs           # K8s client wrapper
│   ├── watcher.rs       # Resource watch → AppMsg
│   └── informer.rs      # Informer cache (optional optimization)
├── dcgm/
│   ├── mod.rs           # DCGM integration
│   └── scraper.rs       # Prometheus endpoint polling
├── ui/
│   ├── mod.rs           # Shared rendering utilities
│   ├── alloc_bar.rs     # Utilization bar widget
│   ├── table.rs         # Generic typed table widget
│   └── styles.rs        # Color palette, idle amber, etc.
└── keys.rs              # Global key bindings, KeyBinding struct
```

---

## The Test for Correctness

Before merging any PR, answer these questions:

1. **Can I add a new view without modifying existing views?** (Rule 1)
2. **If I delete a view file, does the rest of the app compile?** (Rule 2)
3. **Are all row fields accessed by name, never by index?** (Rule 3)
4. **Do background tasks only communicate via messages?** (Rule 4)
5. **Does a key binding in one view affect behavior in another?** (Rule 5 — answer should be NO)
6. **Can I press Esc from any depth and land back correctly?** (Rule 6)
7. **Does the app work (degraded) on a cluster with no DCGM?** (Rule 7)
8. **Does render() read only from `&self` with no side effects?** (Rule 8)

---

## Implementation Order

Match the design-redo's order, but foundation first:

1. **Scaffold** — App loop, View trait, message passing, navigation stack. No k8s yet — hardcode mock data. Verify rules 1-6 with two dummy views.
2. **Fleet view with k8s API only** — Connect `kube` client, list nodes, classify GPU/CPU, show fleet table. No DCGM. Verify rule 7.
3. **Idle detection + sort** — Compute idle from pod scheduling. Amber highlight. Idle-first sort.
4. **Node detail view** — Enter drills into per-GPU breakdown. Verify rule 6 (Esc returns correctly).
5. **DCGM integration** — Scrape Prometheus endpoint, fill `Option<T>` fields, fleet and node-detail views update.
6. **Jobs view** — Group pods by training CRD parent. Drill into ranks.
7. **Context-filtered jumps** — `w`, `e`, `g` from fleet/node views.
8. **Polish** — GIF, README, install story, launch.

---

## What NOT to Build

- Plugin system
- Multi-cluster support
- Custom resource view definitions (no JSON schema for resource views)
- Cost modeling
- Inference-specific views
- Every k9s feature (describe, edit, delete, logs for arbitrary resources)

Keep the surface area small. The tool does one thing well: GPU fleet awareness.
