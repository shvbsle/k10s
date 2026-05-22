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

### Network Metrics

Collected from `/sys/class/net/<iface>/statistics/`. Each metric is labeled with `interface` and `node`.

| Metric | Description | Unit |
|--------|-------------|------|
| `net.rx_bytes` | Bytes received | bytes |
| `net.tx_bytes` | Bytes transmitted | bytes |
| `net.rx_packets` | Packets received | count |
| `net.tx_packets` | Packets transmitted | count |
| `net.rx_dropped` | Receive packets dropped | count |
| `net.tx_errors` | Transmit errors | count |

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

## Planned: Advanced Network Metrics (in progress)

TCP-level metrics for detecting gradient sync bottlenecks.

| Metric | Source | Spikes During |
|--------|--------|---------------|
| `net.tcp_send_queue` | `ss -tn` Send-Q | SYNC |
| `net.tcp_recv_queue` | `ss -tn` Recv-Q | SYNC |
| `net.tcp_retransmits` | `/proc/net/tcp` | error |
| `net.tcp_connection_count` | `ss -tn` | stable |

---

## Planned: eBPF Metrics (research / future)

These metrics are only derivable via eBPF. No `/proc` or `/sys` polling equivalent exists. They provide event-driven, per-connection granularity that's impossible to get otherwise.

### Network (eBPF)

| Metric | eBPF Hook | Spikes During |
|--------|-----------|---------------|
| `net.tcp_bytes_per_connection` | `kprobe/tcp_sendmsg` + `kprobe/tcp_recvmsg` | SYNC |
| `net.tcp_rtt_per_connection` | `tracepoint/tcp/tcp_probe` (srtt_us) | SYNC |
| `net.tcp_cwnd_per_connection` | `tracepoint/tcp/tcp_probe` (snd_cwnd) | SYNC |
| `net.tcp_retransmit_instant` | `tracepoint/tcp/tcp_retransmit_skb` | error |
| `net.tcp_reset_event` | `tracepoint/tcp/tcp_receive_reset` | error |
| `net.socket_buffer_overflow` | `tracepoint/sock/sock_exceed_buf_limit` | SYNC (overload) |
| `net.packet_drop_reason` | `tracepoint/skb/kfree_skb` | error |

### Process (eBPF)

| Metric | eBPF Hook | Spikes During |
|--------|-----------|---------------|
| `proc.sendmsg_latency` | `kprobe/tcp_sendmsg` entry/exit | SYNC |
| `proc.recvmsg_latency` | `kprobe/tcp_recvmsg` entry/exit | SYNC |
| `proc.futex_wait_duration` | `kprobe/do_futex` entry/exit | SYNC (GPU wait) |
| `proc.ioctl_nvidia_calls` | `uprobe` on `/dev/nvidia*` ioctl | COMPUTE |
| `proc.cuda_malloc_events` | `uprobe` on `cudaMalloc` | init/OOM |
| `proc.cuda_memcpy_bytes` | `uprobe` on `cudaMemcpy` | COMPUTE |
| `proc.sched_preempt_duration` | `tracepoint/sched/sched_switch` off-cpu | COMPUTE (interference) |
| `proc.sched_wakeup_latency` | `tracepoint/sched/sched_wakeup` → switch | SYNC (latency) |

### System (eBPF)

| Metric | eBPF Hook | Spikes During |
|--------|-----------|---------------|
| `sys.oom_kill_event` | `tracepoint/oom/oom_kill_process` | error |
| `sys.page_fault_major` | `tracepoint/exceptions/page_fault_user` | COMPUTE (thrashing) |
| `sys.irq_nic_latency` | `tracepoint/irq/irq_handler_entry` + exit | SYNC |
| `sys.softirq_net_duration` | `tracepoint/irq/softirq_entry` (NET_RX/TX) | SYNC |

---

## Design Philosophy

The core insight: you don't need framework-level instrumentation to detect training problems. Any ML training workload, regardless of framework (PyTorch, JAX, etc.), ultimately does compute on GPUs and syncs gradients over the network. By observing GPU utilization patterns and network traffic at the node level, kitty can reconstruct the training timeline and identify:

- **Straggler ranks**: GPUs with lower utilization or longer sync phases
- **Network bottlenecks**: high retransmits, queue buildup, or asymmetric throughput
- **Resource waste**: idle GPUs still drawing power
- **OOM risks**: memory pressure building before a crash

See the [blog post](https://blog.k10s.dev/why-build-k10s/) for more on the motivation.
