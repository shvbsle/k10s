# Roadmap

## Phase 0: Skeleton ← CURRENT

**Goal:** TUI connects, shows static table, validates architecture rules 1-6 with two dummy views.

**Deliverables:**
- [x] Cargo.toml with ratatui + crossterm + tokio
- [x] View trait defined
- [x] App router with navigation stack
- [x] Message passing infrastructure (AppMsg, Action)
- [x] Fleet view (hardcoded mock data)
- [x] Help overlay (second view — proves Rule 1)
- [x] Header bar (cluster context + help hints + kitten ASCII art)
- [x] Horizontal scroll for table views (left/right arrow keys)
- [x] Pre-commit hook (cargo fmt --check + clippy)
- [ ] CI (cargo check + clippy + test)
- [ ] Verify: adding a view requires no changes to existing views

**Pattern established:** View trait, App router, message passing, NavStack.

---

## Phase 1: Fleet View (Live)

**Goal:** Live GPU fleet dashboard from k8s API. Launchable and demo-able.

**Deliverables:**
- [ ] `kube` client connection (kubeconfig auto-detect)
- [ ] Node watcher → AppMsg pipeline
- [ ] FleetNode typed struct populated from k8s API
- [ ] GPU count from `allocatable`
- [ ] Workload attribution from pod scheduling
- [ ] Idle detection (no GPU-requesting pods)
- [ ] Idle-first sort + amber highlight
- [ ] Graceful error states (no cluster, no access)
- [ ] 10-second GIF demo

**Pattern established:** Data source → message → DataStore → View pipeline.

---

## Phase 2: Jobs + Rank Detail

**Goal:** Per-rank comparison with sparklines, straggler detection.

**Deliverables:**
- [ ] Training CRD discovery (PyTorchJob, RayJob, MPIJob, JobSet)
- [ ] Owner reference chain resolution
- [ ] Jobs view with grouped display
- [ ] Rank detail drill-down
- [ ] TimeSeries ring buffer for GPU util sparklines
- [ ] Per-step duration from log parsing
- [ ] Straggler detection (rolling median comparison)
- [ ] Cross-rank comparison table

**Pattern established:** Drill-down navigation, TimeSeries, log-based extraction.

---

## Phase 3: Queue + `y` Diagnostic

**Goal:** Queue visibility and the "why?" killer feature.

**Deliverables:**
- [ ] Kueue CRD detection + watch
- [ ] Queue view (ClusterQueue capacity, admitted vs pending)
- [ ] `y` key → diagnostic overlay
- [ ] Rule engine: taint mismatch, resource fit, affinity, scheduling failures
- [ ] "Why idle?" for fleet view nodes
- [ ] "Why pending?" for queue workloads

**Pattern established:** Extensible diagnostic rule engine, optional data sources.

---

## Phase 4: Polish + Launch

**Goal:** v1.0.0 release.

**Deliverables:**
- [ ] Config file (~/.config/k10s/config.yaml)
- [ ] Help overlay with full key reference
- [ ] Error states polished (connection lost, permission denied)
- [ ] Binary releases (Linux, macOS ARM/x86)
- [ ] brew install formula
- [ ] README with GIF, install, features
- [ ] Show HN post draft

---

## Phase 5+: Post-Launch

Extends existing patterns without refactoring:
- Node detail view (DCGM per-GPU breakdown)
- Failure attribution ("rank 47 OOM'd, killed by kubelet")
- Network health (NCCL timeout detection)
- Multi-framework log patterns
- Write actions (delete/preempt) behind `--allow-writes`
