# k10s — Steering Doc

This document is the architectural contract for k10s. Any AI assistant, pair programmer, or future contributor MUST read this before writing code.

## What k10s Is

GPU-aware Kubernetes tooling in two independent components:

- **kitty** — an agent for GPU cluster operations
- **tui** — a terminal dashboard for GPU fleet visibility
- **e2e** — end-to-end tests for both components

## Project Structure

```
k10s/
├── Cargo.toml              # Workspace manifest + profiles
├── Cargo.lock              # Shared lockfile
├── src/crates/
│   ├── kitty/              # Agent crate
│   │   ├── Cargo.toml
│   │   └── src/
│   ├── tui/                # TUI crate
│   │   ├── Cargo.toml
│   │   └── src/
│   └── e2e/                # End-to-end test crate
│       ├── Cargo.toml
│       └── src/
├── CLAUDE.md               # AI assistant instructions
├── STEERING.md             # This file — architectural contract
├── README.md               # User-facing docs
└── LICENSE
```

## Workspace Design

The root `Cargo.toml` is a workspace manifest with no `[package]` section. It declares:
- Three member crates under `src/crates/`
- A `dev` profile (unoptimized, debug info)
- A `release` profile (opt-level 3, fat LTO, single codegen unit, stripped symbols, abort on panic)

The three crates are fully independent — they share no code. Each declares its own dependencies.

## Build & Run

```bash
cargo build                    # Build all crates
cargo build -p kitty           # Build agent only
cargo build -p tui             # Build TUI only
cargo build -p e2e             # Build e2e only
cargo run -p kitty             # Run agent
cargo run -p tui               # Run TUI
cargo run -p e2e               # Run e2e tests
cargo build --release          # Optimized release build
cargo clippy -- -D warnings    # Lint (must pass)
cargo fmt --check              # Format check (must pass)
```

---

## Hard Rules (Non-Negotiable)

### 1. Crates Do Not Share Code

The kitty agent and TUI are independent binaries. No shared library crate, no cross-imports. If both need similar functionality, they each implement it independently. This keeps them decoupled and deployable separately.

### 2. No Hardcoded/Mock Data

Both components connect to real infrastructure. Test infrastructure uses test doubles injected at the boundary, not fake cluster connections or hardcoded data baked into the binary.

### 3. Clippy + Fmt Gate

`cargo clippy -- -D warnings` and `cargo fmt --check` must pass before any commit. The `.githooks/pre-commit` hook enforces this.

---

## TUI Hard Rules

These rules exist because their violation killed the previous Go implementation.

### Views Are Types, Not Branches

Each view is a separate type implementing a `View` trait. No `match current_view { ... }` sprawl.

### No Shared Mutable State Between Views

Views own their data. Shared state is read-only context. Views receive data via messages.

### Typed Data Models, Not Positional Arrays

Never represent a row as `Vec<String>`. Each resource type has a typed struct.

### Async Events Become Messages on the Main Loop

Background tasks communicate with the TUI via a message channel. They never mutate view state directly.

### Navigation Is a Stack

Explicit navigation stack. `Esc` pops. `Enter` pushes.

### Graceful Degradation

Every DCGM-sourced field is `Option<T>`. Views render `—` when None. Never panic on missing data.

### Rendering Is Stateless

`render()` is a pure function of view state → terminal frame. No side effects, no mutations.

---