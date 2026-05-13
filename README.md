# k10s: GPU-Aware Kubernetes Tools

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**k10s** makes GPU infrastructure in Kubernetes visible and manageable. It consists of two independent components:

- **kitty** — an agent for GPU cluster operations
- **tui** — a terminal dashboard for GPU fleet visibility

k9s treats a GPU node like any other node. It has no idea an H100 costs $3/hr and is sitting at 4% utilization. k10s closes that gap.

## Project Structure

```
src/crates/
├── kitty/    # Agent
├── tui/      # Terminal dashboard
└── e2e/      # End-to-end tests
```

## Quick Start

```bash
cargo build                          # Build all crates
cargo run -p kitty                   # Run the agent
cargo run -p tui                     # Run the TUI
cargo run -p e2e                     # Run e2e tests
cargo build --release                # Optimized release build
```

## License

Apache 2.0. See [LICENSE](LICENSE) for details.
