# k10s Metrics Reference

Kitty is a lightweight node-level agent that exposes GPU, network, and system metrics via a Prometheus-compatible `/metrics` endpoint.

This document covers what kitty emits today, what's planned next, and the full research backlog.

---

## Emitted Metrics (v1.0.0)

These are live and available at `http://<node-ip>:9100/metrics`.

### GPU Metrics

Collected via NVML (NVIDIA Management Library). Each metric is labeled with `gpu_id`, `node`, and `vendor`.

| Metric | Description | Unit |
|--------|-------------|------|
| `gpu.memory_free` | Free GPU memory | bytes |
| `gpu.memory_total` | Total GPU memory | bytes |
| `gpu.memory_used` | Used GPU memory | bytes |
| `gpu.memory_utilization` | Memory controller utilization | percent (0-100) |
| `gpu.sm_utilization` | Streaming multiprocessor utilization | percent (0-100) |
| `gpu.pcie_rx_bytes` | PCIe receive throughput | bytes |
| `gpu.pcie_tx_bytes` | PCIe transmit throughput | bytes |
| `gpu.power_draw_watts` | Current power draw | watts |
| `gpu.temperature_c` | GPU temperature | celsius |

### Network Metrics (per-process straggler detection)

These metrics detect network-induced stragglers in distributed training. They are emitted **per watched process** — only processes with `KITTY_WATCH=1` in their environment are tracked. Labeled with `pid`, `node`, and optionally `rank` and `job` if the training framework sets them.

**Requires:** Training pods must set `KITTY_WATCH=1` env var. Kitty discovers PIDs via the GPU collector (NVML), then filters to those with this env var.

#### Procfs metrics (collected every 500ms)

| Metric | Source | Detects |
|--------|--------|---------|
| `net.tcp_retransmits` | `/proc/<pid>/net/snmp` RetransSegs | Packet loss |
| `net.tcp_out_segs` | `/proc/<pid>/net/snmp` OutSegs | Bandwidth throttle (rate collapse) |
| `net.tcp_tx_queue_bytes` | `/proc/<pid>/net/tcp` tx_queue sum | Bandwidth throttle (queue buildup on recovery) |

These read from the **pod's network namespace** (via `/proc/<pid>/net/`), not the host. This is critical — host-level counters are blind to per-pod network faults on Kubernetes with VPC CNI or similar.

#### eBPF metric (collected every 100ms)

| Metric | Source | Detects |
|--------|--------|---------|
| `net.tcp_rtt_us` | kprobe on `tcp_rtt_estimator` (arg1 = raw RTT sample) | Latency |

This captures the **raw, unsmoothed RTT** at the moment each ACK is processed. Unlike SRTT from `/proc/net/tcp` (which uses exponential smoothing and doesn't decay), this value jumps immediately when latency is introduced and drops immediately when removed.

The BPF program reads the socket's local port from `struct sock.__sk_common.skc_num` and emits events to a ring buffer. Userspace matches events to the correct training process by comparing ports against `/proc/<pid>/net/tcp`.

#### Detection matrix

| Network fault | Primary signal | Behavior |
|---------------|---------------|----------|
| Latency (e.g., 50ms delay) | `tcp_rtt_us` jumps from ~2ms to ~52ms | Instant response, instant recovery |
| Packet loss (e.g., 5%) | `tcp_retransmits` rate spikes | Counter climbs during fault, flattens after |
| Bandwidth throttle (e.g., 100Mbit cap) | `tcp_out_segs` flatlines to zero | Process stops sending, recovers after minutes |

### System Metrics

Collected from `/proc/stat` and `/proc/meminfo`. Labeled with `node`.

| Metric | Description | Unit |
|--------|-------------|------|
| `sys.cpu_utilization` | Node CPU utilization | ratio (0.0-1.0) |
| `sys.memory_available_bytes` | Available memory | bytes |
| `sys.shm_total_bytes` | Total shared memory (/dev/shm) | bytes |
| `sys.shm_used_bytes` | Used shared memory | bytes |

---

## Planned: GPU Profiling Metrics (in progress)

These require DCGM (Data Center GPU Manager) field queries. Coming in a future release.

| Metric | DCGM Field | Spikes During |
|--------|------------|---------------|
| `gpu.sm_active` | DCGM_FI_PROF_SM_ACTIVE (1002) | COMPUTE |
| `gpu.sm_occupancy` | DCGM_FI_PROF_SM_OCCUPANCY (1003) | COMPUTE |
| `gpu.tensor_core_active` | DCGM_FI_PROF_PIPE_TENSOR_ACTIVE (1004) | COMPUTE |
| `gpu.fp32_pipe_active` | DCGM_FI_PROF_PIPE_FP32_ACTIVE (1007) | COMPUTE |
| `gpu.fp16_pipe_active` | DCGM_FI_PROF_PIPE_FP16_ACTIVE (1008) | COMPUTE |
| `gpu.dram_active` | DCGM_FI_PROF_DRAM_ACTIVE (1005) | COMPUTE |
| `gpu.nvlink_tx_bytes` | DCGM_FI_PROF_NVLINK_TX_BYTES (1011) | SYNC |
| `gpu.nvlink_rx_bytes` | DCGM_FI_PROF_NVLINK_RX_BYTES (1012) | SYNC |

---

## Planned: Process Metrics (in progress)

Per-process metrics for GPU compute apps, collected from `/proc/<pid>/`. Useful for identifying straggler ranks.

| Metric | Source | Spikes During |
|--------|--------|---------------|
| `proc.state` | `/proc/<pid>/status` State | COMPUTE=R, SYNC=S |
| `proc.voluntary_ctx_switches` | `/proc/<pid>/status` | SYNC |
| `proc.nonvoluntary_ctx_switches` | `/proc/<pid>/status` | COMPUTE |
| `proc.cpu_usage` | `/proc/<pid>/stat` utime+stime | COMPUTE |
| `proc.threads` | `/proc/<pid>/status` Threads | stable |
| `proc.vm_rss` | `/proc/<pid>/status` VmRSS | stable |
| `proc.io_read_bytes` | `/proc/<pid>/io` read_bytes | data loading |
| `proc.io_write_bytes` | `/proc/<pid>/io` write_bytes | checkpointing |
| `proc.fd_count` | `/proc/<pid>/fd` | stable |
| `proc.socket_count` | `/proc/<pid>/fd` (socket links) | stable |

---

## Implemented: Network Straggler Detection

See "Network Metrics" above. Fully implemented and validated against real fault injection on a 4-node DDP training cluster.

---

## Implemented: eBPF RTT Measurement

`net.tcp_rtt_us` is collected via kprobe on `tcp_rtt_estimator`. See "Network Metrics" above for details.

Approaches we tried and rejected during research:
- `tcp_probe` tracepoint: only gives smoothed SRTT, same decay problem as `/proc/net/tcp`
- kprobe on `tcp_sendmsg` duration: sendmsg returns before qdisc delay is applied
- kprobe on `tcp_recvmsg` duration: data already in socket buffer when recvmsg runs
- SRTT from `/proc/<pid>/net/tcp`: exponential smoothing prevents decay after fault removal

---

## Planned: eBPF Metrics (research / future)

| Metric | eBPF Hook | Use case |
|--------|-----------|----------|
| `proc.futex_wait_duration` | `kprobe/do_futex` entry/exit | GPU synchronization wait |
| `proc.sched_preempt_duration` | `tracepoint/sched/sched_switch` off-cpu | CPU interference detection |
| `proc.sched_wakeup_latency` | `tracepoint/sched/sched_wakeup` → switch | Scheduling latency |
| `sys.oom_kill_event` | `tracepoint/oom/oom_kill_process` | OOM detection |
| `net.packet_drop_reason` | `tracepoint/skb/kfree_skb` | Network error diagnosis |

---

## Design Philosophy

The core insight: you don't need framework-level instrumentation to detect training problems. Any ML training workload, regardless of framework (PyTorch, JAX, etc.), ultimately does compute on GPUs and syncs gradients over the network. By observing GPU utilization patterns and network traffic at the node level, kitty can reconstruct the training timeline and identify:

- **Straggler ranks**: GPUs with lower utilization or longer sync phases
- **Network bottlenecks**: high retransmits, queue buildup, or asymmetric throughput
- **Resource waste**: idle GPUs still drawing power
- **OOM risks**: memory pressure building before a crash

See the [blog post](https://blog.k10s.dev/why-build-k10s/) for more on the motivation.
