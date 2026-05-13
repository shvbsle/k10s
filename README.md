# k10s: GPU-Aware Kubernetes Toolkit

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/kubernetes-1.25%2B-326ce5.svg)](https://kubernetes.io/)
[![Discord](https://img.shields.io/badge/Discord-Join%20Us-5865F2?logo=discord&logoColor=white)](https://discord.gg/rngaJustFD)


**k10s** is two things:
- **kitty** - a Daemonset that lives on your kubernetes cluster that collects node-level GPU + Network diagnostics
- **k10s** - (kittens) a TUI that shows the ML training jobs in your cluster and surfaces ranks that are misbehaving.

The outcomes will be:
1. Your idle / misbehaving GPUs are LOUD so you know if you are burning $$
2. You know exactly WHY your training job is messed up (stragger ranks, oom issues, network chokes etc)
3. You don't have to leave your terminal

These are problems that I have and thats the main reason to build this. If you also have these problems join our discord and consider becoming a contributor and shape this tool: [![Discord](https://img.shields.io/badge/Discord-Join%20Us-5865F2?logo=discord&logoColor=white)](https://discord.gg/rngaJustFD)

Other separate motivations for building k10s on the dev log: [why build k10s?](https://blog.k10s.dev/why-build-k10s/)

## Project Structure

```
src/crates/
├── kitty/    # daemonset agent
├── tui/      # tui duh
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

---

### What happened to the Go version?

There was a vibe-coded go-version of the TUI. Still available for use here:
https://github.com/shvbsle/k10s/tree/archive/go-v0.4.0

It became unmaintainable so I've archived that branch and decided to hand-write this TUI again from scratch in Rust.
I speak more about it here: [Blog: I'm going back to writing code by hand](https://blog.k10s.dev/im-going-back-to-writing-code-by-hand/)

