# Views Specification

## View Trait

```rust
pub trait View {
    fn update(&mut self, msg: &AppMsg, ctx: &AppContext) -> Option<Action>;
    fn render(&self, frame: &mut Frame, area: Rect, ctx: &AppContext);
    fn key_map(&self) -> &[KeyBinding];
    fn on_enter(&mut self, ctx: &AppContext);
    fn on_leave(&mut self);
    fn title(&self) -> &str;
}
```

## Planned Views

### Fleet View (default landing)

**Purpose:** GPU fleet health at a glance. Every GPU node in the cluster, sorted idle-first.

| Column | Source | Absent |
|--------|--------|--------|
| NODE | k8s API | never |
| MODEL | node labels / allocatable | "unknown" |
| GPUs | allocatable `nvidia.com/gpu` | 0 |
| UTIL | DCGM exporter | — |
| MEM | DCGM exporter | — |
| TEMP | DCGM exporter | — |
| WORKLOAD | pod scheduling | IDLE |
| IDLE | computed (no GPU pods scheduled) | — |

**Sort:** Idle (duration desc) → Degraded → Busy (util asc) → Saturated.  
**Highlight:** Amber for idle. Escalate after 6h.  
**Keys:** Enter → Node Detail, `j` → Jobs view, `/` → filter.

### Node Detail View

**Purpose:** Per-GPU breakdown for a single node.

Shows each GPU device index with utilization bar, memory %, temp, power, workload attribution, training rank.

**Keys:** Esc → back to Fleet, Enter → Workload Detail.

### Jobs View

**Purpose:** Training workloads grouped by parent CRD (Job/JobSet/PyTorchJob/RayJob/MPIJob).

| Column | Source |
|--------|--------|
| JOB | owner reference chain |
| TYPE | CRD kind |
| RANKS | pod count |
| RUNNING/FAILED/PENDING | pod phase |
| AGE | creation timestamp |
| QUEUE | Kueue annotation |

**Primary atom is the Rank.** Drill in to see per-rank comparison, sparklines, straggler detection.

**Keys:** Enter → Rank Detail, Esc → back.

### Queue View

**Purpose:** Kueue ClusterQueue/LocalQueue state + pending workloads.

Shows capacity, admitted vs pending, quota usage. Absent Kueue shows helpful "Kueue not detected" message.

### Help Overlay

**Purpose:** Context-sensitive key binding reference. Shows active view's key map + globals.

**Keys:** `?` toggles on/off.

## View Lifecycle

```
Navigate(ViewX { data }) →
  1. active_view.on_leave()
  2. push active_view onto stack
  3. new_view = ViewX::new(data)
  4. new_view.on_enter(ctx)
  5. active_view = new_view

Esc (pop) →
  1. active_view.on_leave()
  2. drop active_view
  3. active_view = stack.pop()
  4. active_view.on_enter(ctx)  // refresh
```
