# Go v0.4.0 Feature Parity Tracker

Tracks every Go feature, its current Rust status, and which phase it belongs to.

Legend: ✅ Done | 🔧 Scaffolded (interfaces exist, not wired) | ❌ Missing

---

## UI / Navigation

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| j/k up/down | ✅ | 0 |
| h/l horizontal scroll | ✅ | 0 |
| g/G jump top/bottom | ✅ | 0 |
| `0` reset scroll | ✅ | 0 |
| Esc go back (nav stack) | ✅ | 0 |
| Mouse wheel scroll | ❌ | 1 |
| Mouse click-to-select | ❌ | 1 |
| Tab/Shift+Tab fleet tabs (GPU/CPU/All) | ❌ | 1 |
| Pagination (auto page size) | ❌ | 1 |
| Command history (up/down in `:`) | ❌ | 1 |
| Tab autocomplete in command mode | ❌ | 2 |
| Breadcrumb navigation path | ❌ | 2 |

## Commands

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| `:q` / `:quit` | ✅ | 0 |
| `:rs <resource>` / `:resource` | 🔧 (parser + dispatch + ResourceView exist) | 0 |
| `:rs <resource> -n <ns>` | 🔧 (parser handles it, namespace passed to DataSource) | 0 |
| `:help` | ✅ | 0 |
| `:ctx` — list/switch context | 🔧 (parsed, returns "not yet implemented") | 1 |
| `:ns` — list/switch namespace | 🔧 (parsed, returns "not yet implemented") | 1 |
| `:reconnect` / `:r` | ❌ | 1 |
| `:cplogs` / `:cp` — copy logs | ❌ | 2 |
| Plugin command registry | ❌ | 3+ |

## Views

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| Fleet view (GPU dashboard) | ✅ | 0 |
| Generic resource table | 🔧 (ResourceView exists, needs live wiring test) | 0 |
| Help overlay | ✅ | 0 |
| Log view (streaming) | ❌ | 2 |
| Describe view | ❌ | 2 |
| YAML view | ❌ | 2 |
| Containers view (pod drill-down) | ❌ | 2 |
| Contexts list view | ❌ | 1 |
| Namespaces list view | ❌ | 1 |
| API Resources list (`:rs` with no args) | ❌ | 1 |

## Fleet View Specifics

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| GPU node detection | ✅ | 0 |
| Idle-first sort + amber highlight | ✅ | 0 |
| GPU allocation from pod scheduling | ✅ | 0 |
| DCGM metrics integration | ✅ | 0 |
| Graceful DCGM degradation | ✅ | 0 |
| Tab filtering (GPU/CPU/All) | ❌ | 1 |
| Allocation bar rendering | ✅ (text bar `[####  ]`) | 0 |
| Enter → drill to pods on node | 🔧 (Action defined, not wired in fleet handle_key) | 0 |
| Instance type display | ✅ (in data, not shown in column) | 1 |
| Node readiness display | ✅ (affects status color) | 0 |

## Data / Streaming

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| Reflector/watch (delta updates) | ✅ (nodes + pods via kube reflector) | 0 |
| Periodic fetch loop (5s) | ✅ | 0 |
| Client-side age refresh | ❌ | 1 |
| Field selector filtering | 🔧 (ResourceFilter::FieldEquals defined) | 1 |
| Label selector filtering | ❌ | 1 |
| Dynamic API discovery | 🔧 (DataSource::discover_resources defined) | 0 |
| Watch for generic resources | ❌ (currently one-shot list, not watch) | 1 |
| Log streaming (pod logs) | ❌ | 2 |
| Reconnection on connection loss | ❌ | 1 |

## Configuration

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| Config file (`~/.config/k10s/config.toml`) | ❌ | 0 |
| Logo/ASCII art customization | ❌ | 0 |
| Keybinding overrides | ❌ | 1 |
| Refresh interval | ❌ (hardcoded 5s) | 0 |
| Log file path | ❌ (hardcoded `k10s.log`) | 0 |
| Default namespace | ❌ | 0 |
| Default landing view | ❌ | 1 |
| Pagination style | ❌ | 1 |
| Config auto-generation | ❌ | 0 |

## Actions / Operations

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| `d` — describe resource | ❌ | 2 |
| `e` — edit resource ($EDITOR) | ❌ | 3+ |
| `y` — view YAML | ❌ | 2 |
| `s` — shell exec into pod | ❌ | 3+ |
| `L` — view logs | ❌ | 2 |
| Copy to clipboard (OSC 52) | ❌ | 2 |

## Log View Features

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| Real-time log streaming | ❌ | 2 |
| Autoscroll toggle | ❌ | 2 |
| Fullscreen toggle (`f`) | ❌ | 2 |
| Word wrap toggle (`w`) | ❌ | 2 |
| Timestamp toggle (`t`) | ❌ | 2 |
| Line number toggle (`n`) | ❌ | 2 |
| Search/filter (`/`) | ❌ | 2 |
| Circular buffer (10K lines max) | ❌ | 2 |

## Infrastructure

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| Structured logging to file | ✅ (tracing + file writer) | 0 |
| Health status in header | ✅ (DCGM status indicator) | 0 |
| DataSource trait (live/mock) | ✅ | 0 |
| `--mock` CLI flag | ✅ | 0 |
| Mock data generator | ✅ (fleet + generic resources) | 0 |
| Pre-commit hook (fmt + clippy) | ✅ | 0 |
| Render snapshot tests (TestBackend) | ✅ (14 tests) | 0 |
| Plugin system | ❌ | 3+ |
| Error messages in command bar | ✅ | 0 |

## Rendering / Polish

| Feature | Rust Status | Phase |
|---------|-------------|-------|
| Header bar (context + kittens + hints) | ✅ | 0 |
| Colored status per node state | ✅ | 0 |
| Command bar at bottom (`:` mode) | ✅ | 0 |
| Status messages (errors, success) | ✅ (errors only) | 0 |
| Success messages with timeout | ❌ | 1 |
| ANSI color in log output | ❌ | 2 |
| Column auto-sizing | ❌ (known issue: Min() overflow) | 0 |

---

## Phase Summary

### Phase 0 (Current — Foundation)
Everything scaffolded. Fleet works live. Mock mode works. Command system parses. ResourceView renders. Tests pass.

**Remaining Phase 0 items:**
- [ ] Config file loading (`~/.config/k10s/config.toml`) with defaults
- [ ] Fix column width issue (known-issues.md)
- [ ] Wire Enter drill-down in fleet view
- [ ] Wire `:rs` end-to-end with live cluster (test manually)

### Phase 1 — Interactivity & Polish
- Context/namespace switching (`:ctx`, `:ns`)
- Fleet tab filtering (GPU/CPU/All)
- Mouse support
- Watch-based updates for ResourceView
- Reconnection handling
- Age refresh (client-side)
- Command history
- Keybinding config
- Pagination

### Phase 2 — Views & Operations
- Log view (streaming, autoscroll, search)
- Describe view
- YAML view
- Containers view
- `d`/`y`/`L` action keys
- Clipboard (OSC 52)
- `:cplogs`

### Phase 3+ — Extensions
- Shell exec (`s`)
- Edit resource (`e`)
- Plugin system
- Multi-cluster
- Custom themes
