# Data Sources

## Overview

k10s joins multiple data sources into a unified GPU-centric view. Each source is optional — the tool degrades gracefully when a source is unavailable.

```
┌─────────────┐   ┌──────────────────┐   ┌─────────────┐   ┌────────────┐
│  k8s API    │   │  DCGM Exporter   │   │    Kueue    │   │  Pod Logs  │
│  (required) │   │  (optional)      │   │  (optional) │   │ (optional) │
└──────┬──────┘   └────────┬─────────┘   └──────┬──────┘   └─────┬──────┘
       │                   │                     │                 │
       ▼                   ▼                     ▼                 ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           DataStore (AppMsg)                              │
└──────────────────────────────────────────────────────────────────────────┘
```

## k8s API (Required)

**Client:** `kube` crate with `kube::Client::try_default()` (uses kubeconfig).

### What we get from k8s alone (no DCGM):
- Node list with labels (`nvidia.com/gpu.product`, instance type)
- GPU count per node from `allocatable["nvidia.com/gpu"]`
- Pod → node scheduling (workload attribution)
- Training CRD discovery (PyTorchJob, RayJob, MPIJob, JobSet)
- Owner reference chains (pod → ReplicaSet → Job → CRD)

### Watch Strategy
- Nodes: watch with resource version tracking
- Pods: watch, filtered to GPU-requesting pods (label selector or field selector TBD)
- Training CRDs: watch per detected kind

## DCGM Exporter (Optional)

**Protocol:** HTTP scrape of Prometheus metrics endpoint.

**Discovery:** Look for pods with label `app=nvidia-dcgm-exporter` or service `nvidia-dcgm-exporter` in `gpu-operator` / `monitoring` namespace. Fall back to configurable endpoint.

**Key metrics:**
| Metric | Maps to |
|--------|---------|
| `DCGM_FI_DEV_GPU_UTIL` | utilization % |
| `DCGM_FI_DEV_FB_USED` / `DCGM_FI_DEV_FB_FREE` | memory % |
| `DCGM_FI_DEV_GPU_TEMP` | temperature |
| `DCGM_FI_DEV_POWER_USAGE` | power watts |

**Polling interval:** 5s (configurable). Metrics are per-GPU-per-node, keyed by `gpu` (device index) and `Hostname` labels.

**Absent behavior:** All DCGM fields are `Option<T>`. Fleet/Node views render "—" for absent metrics.

## Kueue (Optional)

**Detection:** Check if `ClusterQueue` CRD exists in the cluster.

**Resources watched:**
- `ClusterQueue` — capacity, admitted workloads
- `LocalQueue` — per-namespace queue state
- `Workload` — pending/admitted workloads with priority

**Absent behavior:** Queue view shows "Kueue not installed — this view requires kueue.x-k8s.io CRDs" with a helpful link.

## Pod Logs (On-Demand)

**Purpose:** Extract per-step training metrics (step number, throughput, loss) from stdout.

**Strategy:** Configurable regex patterns. Defaults cover common frameworks:
- PyTorch: `Step (\d+).*loss[= :]([0-9.]+).*throughput[= :]([0-9.]+)`
- User-configurable via `~/.config/k10s/log_patterns.yaml`

**Streaming:** `kube::api::Api::log_stream()` — tailed, not bulk-fetched.

**Scale concern:** 64+ pods tailing simultaneously. Mitigation: only tail active/focused rank, prefetch on drill-down.

## Open Design Questions

1. **DCGM: HTTP scrape vs NVML exec** — Scraping the exporter is simpler and non-privileged. Direct NVML (exec into node) gives per-process GPU mapping but requires privileged access.
2. **Log parsing vs structured protocol** — Regex is fragile but requires zero instrumentation. A structured sidecar protocol is robust but adds adoption friction.
3. **Watch vs poll for DCGM** — Prometheus metrics don't support watch. Poll interval tradeoff: responsiveness vs load.
