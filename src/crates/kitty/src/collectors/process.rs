use std::collections::{BTreeMap, HashMap};
use std::sync::Arc;
use std::time::{Instant, SystemTime};

use async_trait::async_trait;
use tokio::sync::RwLock;
use tracing::warn;

use crate::cgroup::{self, CgroupInfo};
use crate::collector::Collector;
use crate::error::Result;
use crate::sample::{MetricValue, Sample};

pub struct ProcessCollector {
    pids: Arc<RwLock<Vec<u32>>>,
    prev_cpu: HashMap<u32, PrevCpuState>,
    prev_time: Option<Instant>,
    cgroup_cache: HashMap<u32, Option<CgroupInfo>>,
}

struct PrevCpuState {
    utime: u64,
    stime: u64,
}

impl ProcessCollector {
    pub fn new(pids: Arc<RwLock<Vec<u32>>>) -> Self {
        Self {
            pids,
            prev_cpu: HashMap::new(),
            prev_time: None,
            cgroup_cache: HashMap::new(),
        }
    }
}

#[async_trait]
impl Collector for ProcessCollector {
    fn name(&self) -> &'static str {
        "process"
    }

    async fn collect(&mut self) -> Result<Vec<Sample>> {
        let now = SystemTime::now();
        let wall_now = Instant::now();
        let pids = self.pids.read().await.clone();
        let mut samples = Vec::new();

        let elapsed_secs = self
            .prev_time
            .map(|t| wall_now.duration_since(t).as_secs_f64())
            .unwrap_or(0.0);

        for pid in &pids {
            let mut labels = BTreeMap::new();
            labels.insert("pid".into(), pid.to_string());

            // Resolve cgroup info (cached)
            if !self.cgroup_cache.contains_key(pid) {
                self.cgroup_cache.insert(*pid, cgroup::resolve(*pid).await);
            }
            if let Some(Some(info)) = self.cgroup_cache.get(pid) {
                labels.insert("pod_uid".into(), info.pod_uid.clone());
                if !info.container_id.is_empty() {
                    labels.insert("container_id".into(), info.container_id.clone());
                }
            }

            if let Some(status_samples) = read_proc_status(*pid, now, &labels).await {
                samples.extend(status_samples);
            }

            if let Some((cpu_samples, new_state)) =
                read_proc_stat(*pid, now, &labels, self.prev_cpu.get(pid), elapsed_secs).await
            {
                samples.extend(cpu_samples);
                self.prev_cpu.insert(*pid, new_state);
            }

            if let Some(io_samples) = read_proc_io(*pid, now, &labels).await {
                samples.extend(io_samples);
            }
        }

        // Clean up stale PIDs
        self.prev_cpu.retain(|pid, _| pids.contains(pid));
        self.cgroup_cache.retain(|pid, _| pids.contains(pid));
        self.prev_time = Some(wall_now);

        Ok(samples)
    }
}

async fn read_proc_status(
    pid: u32,
    now: SystemTime,
    labels: &BTreeMap<String, String>,
) -> Option<Vec<Sample>> {
    let content = tokio::fs::read_to_string(format!("/proc/{pid}/status"))
        .await
        .ok()?;

    let mut samples = Vec::new();
    for line in content.lines() {
        let parts: Vec<&str> = line.splitn(2, ':').collect();
        if parts.len() != 2 {
            continue;
        }
        let key = parts[0].trim();
        let val = parts[1].trim();

        match key {
            "State" => {
                let state = val.chars().next().unwrap_or('?').to_string();
                let mut state_labels = labels.clone();
                state_labels.insert("state".into(), state);
                samples.push(Sample {
                    timestamp: now,
                    metric: "proc.state".into(),
                    value: MetricValue::Gauge(1.0),
                    labels: state_labels,
                });
            }
            "VmRSS" => {
                if let Some(kb) = parse_kb_value(val) {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.vm_rss_bytes".into(),
                        value: MetricValue::Gauge((kb * 1024) as f64),
                        labels: labels.clone(),
                    });
                }
            }
            "VmSize" => {
                if let Some(kb) = parse_kb_value(val) {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.vm_size_bytes".into(),
                        value: MetricValue::Gauge((kb * 1024) as f64),
                        labels: labels.clone(),
                    });
                }
            }
            "Threads" => {
                if let Ok(n) = val.parse::<u64>() {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.threads".into(),
                        value: MetricValue::Gauge(n as f64),
                        labels: labels.clone(),
                    });
                }
            }
            "voluntary_ctxt_switches" => {
                if let Ok(n) = val.parse::<u64>() {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.voluntary_ctx_switches".into(),
                        value: MetricValue::Counter(n),
                        labels: labels.clone(),
                    });
                }
            }
            "nonvoluntary_ctxt_switches" => {
                if let Ok(n) = val.parse::<u64>() {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.nonvoluntary_ctx_switches".into(),
                        value: MetricValue::Counter(n),
                        labels: labels.clone(),
                    });
                }
            }
            _ => {}
        }
    }
    Some(samples)
}

async fn read_proc_stat(
    pid: u32,
    now: SystemTime,
    labels: &BTreeMap<String, String>,
    prev: Option<&PrevCpuState>,
    elapsed_secs: f64,
) -> Option<(Vec<Sample>, PrevCpuState)> {
    let content = tokio::fs::read_to_string(format!("/proc/{pid}/stat"))
        .await
        .ok()?;

    // Fields: pid (comm) state ppid ... field[13]=utime field[14]=stime
    let fields: Vec<&str> = content.split_whitespace().collect();
    if fields.len() < 15 {
        warn!(pid, "unexpected /proc/{}/stat format", pid);
        return None;
    }

    let utime = fields[13].parse::<u64>().ok()?;
    let stime = fields[14].parse::<u64>().ok()?;
    let new_state = PrevCpuState { utime, stime };

    let mut samples = Vec::new();

    if let Some(prev) = prev {
        if elapsed_secs > 0.0 {
            let ticks_per_sec = 100.0_f64; // sysconf(_SC_CLK_TCK), typically 100
            let delta_ticks =
                (utime.saturating_sub(prev.utime) + stime.saturating_sub(prev.stime)) as f64;
            let cpu_pct = (delta_ticks / ticks_per_sec) / elapsed_secs * 100.0;
            samples.push(Sample {
                timestamp: now,
                metric: "proc.cpu_percent".into(),
                value: MetricValue::Gauge(cpu_pct),
                labels: labels.clone(),
            });
        }
    }

    Some((samples, new_state))
}

async fn read_proc_io(
    pid: u32,
    now: SystemTime,
    labels: &BTreeMap<String, String>,
) -> Option<Vec<Sample>> {
    let content = tokio::fs::read_to_string(format!("/proc/{pid}/io"))
        .await
        .ok()?;

    let mut samples = Vec::new();
    for line in content.lines() {
        let parts: Vec<&str> = line.splitn(2, ':').collect();
        if parts.len() != 2 {
            continue;
        }
        let key = parts[0].trim();
        let val = parts[1].trim();

        match key {
            "read_bytes" => {
                if let Ok(n) = val.parse::<u64>() {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.io_read_bytes".into(),
                        value: MetricValue::Counter(n),
                        labels: labels.clone(),
                    });
                }
            }
            "write_bytes" => {
                if let Ok(n) = val.parse::<u64>() {
                    samples.push(Sample {
                        timestamp: now,
                        metric: "proc.io_write_bytes".into(),
                        value: MetricValue::Counter(n),
                        labels: labels.clone(),
                    });
                }
            }
            _ => {}
        }
    }
    Some(samples)
}

fn parse_kb_value(val: &str) -> Option<u64> {
    val.split_whitespace().next()?.parse::<u64>().ok()
}
