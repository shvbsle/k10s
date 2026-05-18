use std::collections::BTreeMap;
use std::sync::Arc;
use std::time::SystemTime;

use async_trait::async_trait;
use tokio::sync::RwLock;
use tracing::warn;

use crate::collector::Collector;
use crate::error::Result;
use crate::gpu::GpuProvider;
use crate::sample::{MetricValue, Sample};

pub struct GpuCollector {
    provider: Box<dyn GpuProvider>,
    pids: Arc<RwLock<Vec<u32>>>,
}

impl GpuCollector {
    pub fn new(provider: Box<dyn GpuProvider>, pids: Arc<RwLock<Vec<u32>>>) -> Self {
        Self { provider, pids }
    }
}

#[async_trait]
impl Collector for GpuCollector {
    fn name(&self) -> &'static str {
        "gpu"
    }

    async fn collect(&mut self) -> Result<Vec<Sample>> {
        let now = SystemTime::now();
        let device_count = self.provider.device_count()?;
        let vendor = self.provider.vendor();
        let mut samples = Vec::new();
        let mut all_pids: Vec<u32> = Vec::new();

        for idx in 0..device_count {
            let mut labels = BTreeMap::new();
            labels.insert("gpu_id".into(), idx.to_string());
            labels.insert("vendor".into(), vendor.into());

            if let Ok(util) = self.provider.utilization(idx).await {
                samples.push(sample(
                    now,
                    "gpu.sm_utilization",
                    MetricValue::Gauge(util.sm),
                    &labels,
                ));
                samples.push(sample(
                    now,
                    "gpu.memory_utilization",
                    MetricValue::Gauge(util.memory),
                    &labels,
                ));
            } else {
                warn!(gpu_id = idx, "failed to read utilization");
            }

            if let Ok(mem) = self.provider.memory_info(idx).await {
                samples.push(sample(
                    now,
                    "gpu.memory_used",
                    MetricValue::Gauge(mem.used as f64),
                    &labels,
                ));
                samples.push(sample(
                    now,
                    "gpu.memory_free",
                    MetricValue::Gauge(mem.free as f64),
                    &labels,
                ));
                samples.push(sample(
                    now,
                    "gpu.memory_total",
                    MetricValue::Gauge(mem.total as f64),
                    &labels,
                ));
            } else {
                warn!(gpu_id = idx, "failed to read memory info");
            }

            if let Ok(watts) = self.provider.power_usage(idx).await {
                samples.push(sample(
                    now,
                    "gpu.power_draw_watts",
                    MetricValue::Gauge(watts),
                    &labels,
                ));
            } else {
                warn!(gpu_id = idx, "failed to read power usage");
            }

            if let Ok(temp) = self.provider.temperature(idx).await {
                samples.push(sample(
                    now,
                    "gpu.temperature_c",
                    MetricValue::Gauge(temp),
                    &labels,
                ));
            } else {
                warn!(gpu_id = idx, "failed to read temperature");
            }

            if let Ok(pcie) = self.provider.pcie_throughput(idx).await {
                samples.push(sample(
                    now,
                    "gpu.pcie_tx_bytes",
                    MetricValue::Counter(pcie.tx_bytes),
                    &labels,
                ));
                samples.push(sample(
                    now,
                    "gpu.pcie_rx_bytes",
                    MetricValue::Counter(pcie.rx_bytes),
                    &labels,
                ));
            } else {
                warn!(gpu_id = idx, "failed to read pcie throughput");
            }

            match self.provider.nvlink_throughput(idx).await {
                Ok(Some(nvlink)) => {
                    samples.push(sample(
                        now,
                        "gpu.nvlink_tx_bytes",
                        MetricValue::Counter(nvlink.tx_bytes),
                        &labels,
                    ));
                    samples.push(sample(
                        now,
                        "gpu.nvlink_rx_bytes",
                        MetricValue::Counter(nvlink.rx_bytes),
                        &labels,
                    ));
                }
                Ok(None) => {}
                Err(_) => {
                    warn!(gpu_id = idx, "failed to read nvlink throughput");
                }
            }

            match self.provider.running_compute_processes(idx).await {
                Ok(procs) => {
                    for proc in &procs {
                        all_pids.push(proc.pid);
                        let mut proc_labels = labels.clone();
                        proc_labels.insert("pid".into(), proc.pid.to_string());
                        if let Some(mem) = proc.gpu_memory_used {
                            samples.push(sample(
                                now,
                                "gpu.process_memory_used",
                                MetricValue::Gauge(mem as f64),
                                &proc_labels,
                            ));
                        }
                    }
                }
                Err(_) => {
                    warn!(gpu_id = idx, "failed to read compute processes");
                }
            }
        }

        all_pids.sort_unstable();
        all_pids.dedup();
        *self.pids.write().await = all_pids;

        Ok(samples)
    }
}

fn sample(
    timestamp: SystemTime,
    metric: &str,
    value: MetricValue,
    labels: &BTreeMap<String, String>,
) -> Sample {
    Sample {
        timestamp,
        metric: metric.into(),
        value,
        labels: labels.clone(),
    }
}
