# Node-Level GPU Training Metrics

I ask myself a sharp question: "If I'm a binary running on the node, can I tell if an ML training job is running on the node without looking at any container logs?"

The motivation behind this question is a three step ladder:
1. Several ML trianing frameworks exists (Pytorch/Jax etc) and they will keep evolving so never build a solution that relies on the highest level of abstraction. 
2. Any new abstraction that comes up for ML training, at its core, will end up doing some compute on GPUs and then syncing gradients.
3. It should be sufficient to just look at spikes GPU metrics and networking metrics to re-construct the timeline of training.

Based on my research, all metrics observable from the node without framework instrumentation.

| Metric Name | How It's Derived | Spikes During | Component | eBPF Alternative? |
|---|---|---|---|---|
| gpu-sm-utilization | `nvidia-smi dmon -s u` or DCGM_FI_DEV_GPU_UTIL | COMPUTE | gpu | No (hardware counter) |
| gpu-memory-utilization | `nvidia-smi dmon -s u` (mem% column) | COMPUTE | gpu | No (hardware counter) |
| gpu-sm-active | DCGM_FI_PROF_SM_ACTIVE (field 1002) | COMPUTE | gpu | No (hardware counter) |
| gpu-sm-occupancy | DCGM_FI_PROF_SM_OCCUPANCY (field 1003) | COMPUTE | gpu | No (hardware counter) |
| gpu-tensor-core-active | DCGM_FI_PROF_PIPE_TENSOR_ACTIVE (field 1004) | COMPUTE | gpu | No (hardware counter) |
| gpu-fp32-pipe-active | DCGM_FI_PROF_PIPE_FP32_ACTIVE (field 1007) | COMPUTE | gpu | No (hardware counter) |
| gpu-fp16-pipe-active | DCGM_FI_PROF_PIPE_FP16_ACTIVE (field 1008) | COMPUTE | gpu | No (hardware counter) |
| gpu-dram-active | DCGM_FI_PROF_DRAM_ACTIVE (field 1005) | COMPUTE | gpu | No (hardware counter) |
| gpu-pcie-tx-bytes | `nvidia-smi dmon -s t` or DCGM_FI_PROF_PCIE_TX_BYTES (field 1009) | SYNC | gpu | No (hardware counter) |
| gpu-pcie-rx-bytes | `nvidia-smi dmon -s t` or DCGM_FI_PROF_PCIE_RX_BYTES (field 1010) | SYNC | gpu | No (hardware counter) |
| gpu-nvlink-tx-bytes | DCGM_FI_PROF_NVLINK_TX_BYTES (field 1011) | SYNC | gpu | No (hardware counter) |
| gpu-nvlink-rx-bytes | DCGM_FI_PROF_NVLINK_RX_BYTES (field 1012) | SYNC | gpu | No (hardware counter) |
| gpu-memory-used | `nvidia-smi --query-gpu=memory.used` | COMPUTE (stable) | gpu | No (hardware counter) |
| gpu-memory-free | `nvidia-smi --query-gpu=memory.free` | COMPUTE (stable) | gpu | No (hardware counter) |
| gpu-power-draw | `nvidia-smi dmon -s p` | COMPUTE | gpu | No (hardware counter) |
| gpu-temperature | `nvidia-smi dmon -s p` | COMPUTE | gpu | No (hardware counter) |
| gpu-compute-app-pid | `nvidia-smi --query-compute-apps=pid,used_gpu_memory` | - | gpu | No |
| net-interface-tx-bytes | `/sys/class/net/<iface>/statistics/tx_bytes` | SYNC | network | Yes - `tracepoint/net/net_dev_xmit` gives per-packet |
| net-interface-rx-bytes | `/sys/class/net/<iface>/statistics/rx_bytes` | SYNC | network | Yes - `tracepoint/net/netif_receive_skb` gives per-packet |
| net-interface-tx-packets | `/sys/class/net/<iface>/statistics/tx_packets` | SYNC | network | Yes - same tracepoint, count events |
| net-interface-rx-packets | `/sys/class/net/<iface>/statistics/rx_packets` | SYNC | network | Yes - same tracepoint |
| net-interface-tx-errors | `/sys/class/net/<iface>/statistics/tx_errors` | - (error) | network | Yes - `tracepoint/net/net_dev_xmit` status field |
| net-interface-rx-drops | `/sys/class/net/<iface>/statistics/rx_dropped` | - (error) | network | Yes - `tracepoint/skb/kfree_skb` with drop reason |
| net-tcp-send-queue | `ss -tn` Send-Q column (per connection) | SYNC | network | Yes - `kprobe/tcp_sendmsg` gives exact bytes per call |
| net-tcp-recv-queue | `ss -tn` Recv-Q column (per connection) | SYNC | network | Yes - `kprobe/tcp_recvmsg` gives exact bytes per call |
| net-tcp-retransmits | `/proc/net/tcp` retransmit counter | - (error) | network | Yes - `tracepoint/tcp/tcp_retransmit_skb` (event-driven, instant) |
| net-tcp-connection-state | `/proc/<pid>/net/tcp` state field | - | network | Yes - `tracepoint/sock/inet_sock_set_state` (state change events) |
| net-tcp-connection-count | `nsenter -t <pid> -n ss -tn | wc -l` | - (stable) | network | Yes - track via sock_set_state events |
| nccl-allreduce-count | NCCL_DEBUG=INFO logs (`AllReduce: opCount N`) | SYNC | network | No (use NCCL Profiler Plugin instead) |
| nccl-interface-selected | NCCL_DEBUG=INFO logs (`NET/Socket : Using ...`) | - (init) | network | No (log-only) |
| nccl-profiler-coll-duration | NCCL Profiler Plugin API (startEvent/endEvent) | SYNC | network | No (NCCL plugin API, not kernel) |
| proc-state | `/proc/<pid>/status` State field (R/S/D) | COMPUTE=R, SYNC=S | cpu | Yes - `tracepoint/sched/sched_switch` (event-driven) |
| proc-voluntary-ctx-switches | `/proc/<pid>/status` voluntary_ctxt_switches | SYNC | cpu | Yes - `tracepoint/sched/sched_switch` with prev_state |
| proc-nonvoluntary-ctx-switches | `/proc/<pid>/status` nonvoluntary_ctxt_switches | COMPUTE | cpu | Yes - `tracepoint/sched/sched_switch` preempt flag |
| proc-threads | `/proc/<pid>/status` Threads field | - (stable) | cpu | No (polling is fine) |
| proc-cpu-usage | `/proc/<pid>/stat` utime+stime fields | COMPUTE | cpu | Yes - `sched_switch` duration tracking |
| proc-vm-rss | `/proc/<pid>/status` VmRSS | - (stable) | memory | No (polling is fine) |
| proc-vm-size | `/proc/<pid>/status` VmSize | - (stable) | memory | No (polling is fine) |
| proc-io-read-bytes | `/proc/<pid>/io` read_bytes | COMPUTE (data loading) | disk | Yes - `tracepoint/block/block_rq_complete` |
| proc-io-write-bytes | `/proc/<pid>/io` write_bytes | - (checkpointing) | disk | Yes - same tracepoint |
| proc-io-syscr | `/proc/<pid>/io` syscr (read syscall count) | COMPUTE (data loading) | disk | Yes - `kprobe/vfs_read` |
| proc-fd-count | `ls /proc/<pid>/fd | wc -l` | - (stable) | cpu | No (polling is fine) |
| proc-socket-count | `ls /proc/<pid>/fd -la | grep socket | wc -l` | - (stable) | network | No (polling is fine) |
| sys-cpu-utilization | `/proc/stat` or `mpstat` | COMPUTE | cpu | Yes - `sched_switch` accounting |
| sys-memory-available | `/proc/meminfo` MemAvailable | - (stable) | memory | No (polling is fine) |
| sys-shm-usage | `df /dev/shm` (used vs total) | COMPUTE (dataloader) | memory | No (polling is fine) |

## eBPF-Only Metrics (not derivable from polling)

These require eBPF and have no `/proc` or `/sys` equivalent:

| Metric Name | How It's Derived (eBPF hook) | Spikes During | Component |
|---|---|---|---|
| net-tcp-bytes-per-connection | `kprobe/tcp_sendmsg` + `kprobe/tcp_recvmsg` with sk pointer | SYNC | network |
| net-tcp-rtt-per-connection | `tracepoint/tcp/tcp_probe` (srtt_us field) | SYNC | network |
| net-tcp-cwnd-per-connection | `tracepoint/tcp/tcp_probe` (snd_cwnd field) | SYNC | network |
| net-tcp-retransmit-instant | `tracepoint/tcp/tcp_retransmit_skb` (event, not counter) | - (error) | network |
| net-tcp-reset-event | `tracepoint/tcp/tcp_receive_reset` | - (error) | network |
| net-socket-buffer-overflow | `tracepoint/sock/sock_exceed_buf_limit` | SYNC (overload) | network |
| net-packet-drop-reason | `tracepoint/skb/kfree_skb` with reason enum | - (error) | network |
| proc-sendmsg-latency | `kprobe/tcp_sendmsg` entry/exit delta | SYNC | network |
| proc-recvmsg-latency | `kprobe/tcp_recvmsg` entry/exit delta | SYNC | network |
| proc-futex-wait-duration | `kprobe/do_futex` entry/exit delta | SYNC (GPU wait) | cpu |
| proc-ioctl-nvidia-calls | `uprobe` on `/dev/nvidia*` ioctl | COMPUTE | gpu |
| proc-cuda-malloc-events | `uprobe` on `cudaMalloc` in libcudart.so | - (init/OOM) | gpu |
| proc-cuda-memcpy-bytes | `uprobe` on `cudaMemcpy` in libcudart.so | COMPUTE | gpu |
| proc-sched-preempt-duration | `tracepoint/sched/sched_switch` off-cpu time per PID | COMPUTE (interference) | cpu |
| proc-sched-wakeup-latency | `tracepoint/sched/sched_wakeup` to `sched_switch` delta | SYNC (latency) | cpu |
| sys-oom-kill-event | `tracepoint/oom/oom_kill_process` | - (error) | memory |
| sys-page-fault-major | `tracepoint/exceptions/page_fault_user` (major only) | COMPUTE (thrashing) | memory |
| sys-irq-nic-latency | `tracepoint/irq/irq_handler_entry` + exit for NIC IRQ | SYNC | network |
| sys-softirq-net-duration | `tracepoint/irq/softirq_entry` (NET_RX/NET_TX) | SYNC | network |

