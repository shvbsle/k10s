use std::collections::BTreeMap;
use std::time::{Instant, SystemTime};

use crate::collector::Collector;
use crate::error::Result;
use crate::sample::{MetricValue, Sample};
use async_trait::async_trait;
use nix::sys::statvfs::statvfs;

pub struct SystemCollector {
    prev_cpu: Option<CpuTimes>,
    prev_time: Option<Instant>,
}

#[derive(Clone)]
struct CpuTimes {
    user: u64,
    nice: u64,
    system: u64,
    idle: u64,
    iowait: u64,
    irq: u64,
    softirq: u64,
    steal: u64,
}

impl CpuTimes {
    fn total(&self) -> u64 {
        self.user
            + self.nice
            + self.system
            + self.idle
            + self.iowait
            + self.irq
            + self.softirq
            + self.steal
    }

    fn busy(&self) -> u64 {
        self.total() - self.idle - self.iowait
    }
}

impl SystemCollector {
    pub fn new() -> Self {
        Self {
            prev_cpu: None,
            prev_time: None,
        }
    }
}

#[async_trait]
impl Collector for SystemCollector {
    fn name(&self) -> &'static str {
        "system"
    }

    async fn collect(&mut self) -> Result<Vec<Sample>> {
        let now = SystemTime::now();
        let wall_now = Instant::now();
        let labels = BTreeMap::new();
        let mut samples = Vec::new();

        // CPU utilization from /proc/stat
        if let Some(cpu_samples) = self.collect_cpu(now, &labels).await {
            samples.extend(cpu_samples);
        }

        // Memory from /proc/meminfo
        if let Some(mem_samples) = collect_meminfo(now, &labels).await {
            samples.extend(mem_samples);
        }

        // Shared memory from /dev/shm
        if let Some(shm_samples) = collect_shm(now, &labels) {
            samples.extend(shm_samples);
        }

        self.prev_time = Some(wall_now);
        Ok(samples)
    }
}

impl SystemCollector {
    async fn collect_cpu(
        &mut self,
        now: SystemTime,
        labels: &BTreeMap<String, String>,
    ) -> Option<Vec<Sample>> {
        let content = tokio::fs::read_to_string("/proc/stat").await.ok()?;
        let first_line = content.lines().next()?;
        if !first_line.starts_with("cpu ") {
            return None;
        }

        let fields: Vec<u64> = first_line
            .split_whitespace()
            .skip(1) // skip "cpu"
            .filter_map(|f| f.parse().ok())
            .collect();

        if fields.len() < 8 {
            return None;
        }

        let current = CpuTimes {
            user: fields[0],
            nice: fields[1],
            system: fields[2],
            idle: fields[3],
            iowait: fields[4],
            irq: fields[5],
            softirq: fields[6],
            steal: fields[7],
        };

        let mut samples = Vec::new();

        if let Some(prev) = &self.prev_cpu {
            let delta_total = current.total().saturating_sub(prev.total());
            if delta_total > 0 {
                let delta_busy = current.busy().saturating_sub(prev.busy());
                let utilization = delta_busy as f64 / delta_total as f64 * 100.0;
                samples.push(Sample {
                    timestamp: now,
                    metric: "sys.cpu_utilization".into(),
                    value: MetricValue::Gauge(utilization),
                    labels: labels.clone(),
                });
            }
        }

        self.prev_cpu = Some(current);
        Some(samples)
    }
}

async fn collect_meminfo(
    now: SystemTime,
    labels: &BTreeMap<String, String>,
) -> Option<Vec<Sample>> {
    let content = tokio::fs::read_to_string("/proc/meminfo").await.ok()?;
    let mut samples = Vec::new();

    for line in content.lines() {
        if let Some(val) = line.strip_prefix("MemAvailable:") {
            if let Ok(kb) = val.split_whitespace().next()?.parse::<u64>() {
                samples.push(Sample {
                    timestamp: now,
                    metric: "sys.memory_available_bytes".into(),
                    value: MetricValue::Gauge((kb * 1024) as f64),
                    labels: labels.clone(),
                });
            }
            break;
        }
    }

    Some(samples)
}

fn collect_shm(now: SystemTime, labels: &BTreeMap<String, String>) -> Option<Vec<Sample>> {
    let stat = statvfs("/dev/shm").ok()?;
    let total = stat.blocks() * stat.fragment_size();
    let free = stat.blocks_available() * stat.fragment_size();
    let used = total.saturating_sub(free);

    Some(vec![
        Sample {
            timestamp: now,
            metric: "sys.shm_used_bytes".into(),
            value: MetricValue::Gauge(used as f64),
            labels: labels.clone(),
        },
        Sample {
            timestamp: now,
            metric: "sys.shm_total_bytes".into(),
            value: MetricValue::Gauge(total as f64),
            labels: labels.clone(),
        },
    ])
}
