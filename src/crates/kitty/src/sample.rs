use serde::Serialize;
use std::collections::BTreeMap;
use std::time::SystemTime;

#[derive(Debug, Clone, Serialize)]
pub struct Sample {
    pub timestamp: SystemTime,
    pub metric: String,
    pub value: MetricValue,
    pub labels: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type", content = "value")]
pub enum MetricValue {
    Gauge(f64),
    Counter(u64),
}
