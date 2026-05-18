use std::collections::BTreeMap;
use std::time::SystemTime;

use async_trait::async_trait;
use tracing::warn;

use crate::collector::Collector;
use crate::error::Result;
use crate::sample::{MetricValue, Sample};

pub struct NetworkCollector {
    interfaces: Vec<String>,
}

impl NetworkCollector {
    pub fn new(interfaces: Vec<String>) -> Result<Self> {
        let interfaces = if interfaces.is_empty() {
            discover_interfaces()?
        } else {
            interfaces
        };
        Ok(Self { interfaces })
    }
}

#[async_trait]
impl Collector for NetworkCollector {
    fn name(&self) -> &'static str {
        "network"
    }

    async fn collect(&mut self) -> Result<Vec<Sample>> {
        let now = SystemTime::now();
        let mut samples = Vec::new();

        let stats: &[(&str, &str)] = &[
            ("tx_bytes", "net.tx_bytes"),
            ("rx_bytes", "net.rx_bytes"),
            ("tx_packets", "net.tx_packets"),
            ("rx_packets", "net.rx_packets"),
            ("tx_errors", "net.tx_errors"),
            ("rx_dropped", "net.rx_dropped"),
        ];

        for iface in &self.interfaces {
            let mut labels = BTreeMap::new();
            labels.insert("interface".into(), iface.clone());
            let base = format!("/sys/class/net/{iface}/statistics");

            for (file, metric) in stats {
                let path = format!("{base}/{file}");
                match tokio::fs::read_to_string(&path).await {
                    Ok(content) => {
                        if let Ok(val) = content.trim().parse::<u64>() {
                            samples.push(Sample {
                                timestamp: now,
                                metric: (*metric).into(),
                                value: MetricValue::Counter(val),
                                labels: labels.clone(),
                            });
                        }
                    }
                    Err(e) => {
                        warn!(interface = %iface, file = %file, error = %e, "failed to read network stat");
                    }
                }
            }
        }

        Ok(samples)
    }
}

fn discover_interfaces() -> Result<Vec<String>> {
    let mut ifaces = Vec::new();
    for entry in std::fs::read_dir("/sys/class/net")? {
        let entry = entry?;
        let name = entry.file_name().to_string_lossy().to_string();
        if name != "lo" {
            ifaces.push(name);
        }
    }
    Ok(ifaces)
}
