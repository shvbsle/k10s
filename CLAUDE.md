# k10s — GPU-Aware Kubernetes Tools

## Quick Reference

```bash
cargo build                    # Build all crates
cargo build -p kitty           # Build agent only
cargo build -p tui             # Build TUI only
cargo build -p e2e             # Build e2e only
cargo clippy -- -D warnings    # Lint (must pass before commit)
cargo fmt --check              # Format check
cargo test                     # Run all tests
cargo run -p kitty             # Run agent
cargo run -p tui               # Run TUI
cargo run -p e2e               # Run e2e tests
cargo build --release          # Optimized release build
```

## Project Structure

This is a Cargo workspace with three independent crates:

```
src/crates/
├── kitty/           # Agent for GPU cluster operations
│   ├── Cargo.toml
│   └── src/
├── tui/             # Terminal dashboard for GPU fleet visibility
│   ├── Cargo.toml
│   └── src/
└── e2e/             # End-to-end tests
    ├── Cargo.toml
    └── src/
```

The crates do not share code with each other. Each has its own dependencies declared in its own `Cargo.toml`.

## Workspace Cargo.toml

The root `Cargo.toml` is a workspace manifest only — it has no `[package]` section. It defines:
- `[workspace]` with the three member crates
- `[profile.dev]` — unoptimized, full debug info for fast iteration
- `[profile.release]` — max optimization (opt-level 3, fat LTO, single codegen unit, stripped symbols)

## Development Workflow

1. `cargo clippy -- -D warnings` must pass before commit.
2. `cargo fmt --check` must pass before commit.
3. The `.githooks/pre-commit` hook enforces both.

## File Ownership

| Path | Owns |
|------|------|
| `Cargo.toml` | Workspace manifest, profiles |
| `src/crates/kitty/` | Agent crate |
| `src/crates/tui/` | TUI crate |
| `src/crates/e2e/` | End-to-end test crate |
