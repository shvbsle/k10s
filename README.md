# k10s: A GPU-Aware Kubernetes Terminal Dashboard

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**k10s** is a GPU-aware Kubernetes TUI. See which GPUs are actually doing work, which are burning money idle, and why your training job's ranks are scattered across the cluster.

k9s treats a GPU node like any other node. It has no idea an H100 costs $3/hr and is sitting at 4% utilization. k10s closes that gap.

## Principles

- **GPUs and jobs are the atoms, not pods.** The default view is a fleet-level GPU dashboard. Pods are an implementation detail you drill into when needed.
- **Idle GPUs are loud, not quiet.** Idle nodes sort to the top and glow amber. An unallocated H100 is $3/hr on fire.
- **Training job awareness.** See 64 ranks as one logical unit, not 64 unrelated pods.
- **Works without DCGM.** GPU count and workload mapping come from the k8s API. DCGM exporter unlocks live utilization, memory, temp, and power.

## Status

Rewriting from scratch in Rust. The Go prototype (v0.4.0) is archived on the `archive/go-v0.4.0` branch.

## License

Apache 2.0. See [LICENSE](LICENSE) for details.
