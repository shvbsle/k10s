# k10s Architecture

Living document. Evolves as we build. For the full rationale behind each rule, see `STEERING.md`.

## Core Invariants

1. **Views are types** — each view implements the `View` trait. No match-on-enum sprawl in the app loop.
2. **No shared mutable state** — views own their data. Shared context is read-only.
3. **Typed data models** — never `Vec<String>`. Structs with named fields, flattened to cells only at render time.
4. **Message passing** — async tasks communicate via `tokio::mpsc`. No direct mutation of view state.
5. **Scoped key maps** — each view declares its bindings. Unhandled keys bubble to global.
6. **Navigation stack** — `Vec<Box<dyn View>>`. Enter pushes, Esc pops. Views get lifecycle hooks.
7. **Graceful degradation** — every external data field is `Option<T>`. Absent → "—", not panic.
8. **Stateless rendering** — `render(&self, ...)` is pure. All mutations happen in `update()`.

## Message Flow

```
┌──────────────┐     AppMsg      ┌──────────┐    Action     ┌───────────┐
│  Data Sources │ ─────────────► │  App Loop │ ◄──────────── │   Views   │
│  (k8s watch,  │                │  select!  │ ─────────────►│  update() │
│   DCGM poll,  │                │           │    AppMsg     │  render() │
│   log stream) │                └──────────┘               └───────────┘
└──────────────┘                      │
                                      │ Action::Navigate(...)
                                      ▼
                                ┌──────────┐
                                │ NavStack  │
                                │ push/pop  │
                                └──────────┘
```

## Crate Dependencies (Phase 0)

| Crate | Purpose |
|-------|---------|
| `ratatui` | TUI rendering (immediate mode) |
| `crossterm` | Terminal backend |
| `tokio` | Async runtime (for future k8s/DCGM) |

Phase 1 adds: `kube`, `k8s-openapi`, `reqwest`, `serde`, `prometheus-parse`.

## App Loop Pseudocode

```rust
loop {
    terminal.draw(|f| app.render(f))?;
    
    select! {
        event = crossterm_events.next() => {
            if let Some(action) = app.handle_input(event) {
                app.dispatch(action);
            }
        }
        msg = msg_rx.recv() => {
            app.handle_msg(msg);
        }
    }
}
```

## Correctness Checklist (PR Gate)

- [ ] New view = one new file, no changes to existing views
- [ ] Deleting a view file still compiles
- [ ] Row fields accessed by name, never by index
- [ ] Background tasks only communicate via messages
- [ ] Key bindings in one view don't affect another
- [ ] Esc from any depth lands back correctly
- [ ] App works (degraded) without DCGM
- [ ] `render()` has no side effects
