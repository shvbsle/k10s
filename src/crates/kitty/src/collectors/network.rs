use std::collections::BTreeMap;
use std::time::SystemTime;

use async_trait::async_trait;

use crate::collector::Collector;
use crate::error::Result;
use crate::process_state::Snapshot;
use crate::sample::{MetricValue, Sample};

pub struct NetworkCollector {
    snapshot: Snapshot,
}

impl NetworkCollector {
    pub fn new(snapshot: Snapshot) -> Self {
        Self { snapshot }
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
        let processes = self.snapshot.read().await;

        for proc in processes.iter() {
            let snmp_path = format!("/proc/{}/net/snmp", proc.pid);
            samples.extend(collect_snmp_counters(&snmp_path, now, &proc.labels).await);

            samples.push(Sample {
                timestamp: now,
                metric: "net.tcp_tx_queue_bytes".into(),
                value: MetricValue::Gauge(proc.tcp_tx_queue_bytes as f64),
                labels: proc.labels.clone(),
            });
        }

        Ok(samples)
    }
}

async fn collect_snmp_counters(
    path: &str,
    now: SystemTime,
    labels: &BTreeMap<String, String>,
) -> Vec<Sample> {
    let Ok(content) = tokio::fs::read_to_string(path).await else {
        return Vec::new();
    };

    let Some((headers, values)) = parse_proc_pair(&content, "Tcp:") else {
        return Vec::new();
    };

    let fields: BTreeMap<&str, u64> = headers
        .iter()
        .zip(values.iter())
        .filter_map(|(h, v)| v.parse::<u64>().ok().map(|val| (*h, val)))
        .collect();

    let mut samples = Vec::new();

    if let Some(&val) = fields.get("RetransSegs") {
        samples.push(Sample {
            timestamp: now,
            metric: "net.tcp_retransmits".into(),
            value: MetricValue::Counter(val),
            labels: labels.clone(),
        });
    }

    if let Some(&val) = fields.get("OutSegs") {
        samples.push(Sample {
            timestamp: now,
            metric: "net.tcp_out_segs".into(),
            value: MetricValue::Counter(val),
            labels: labels.clone(),
        });
    }

    samples
}

fn parse_proc_pair<'a>(content: &'a str, prefix: &str) -> Option<(Vec<&'a str>, Vec<&'a str>)> {
    let mut lines = content.lines().filter(|l| l.starts_with(prefix));
    let header_line = lines.next()?;
    let value_line = lines.next()?;

    let headers: Vec<&str> = header_line.split_whitespace().skip(1).collect();
    let values: Vec<&str> = value_line.split_whitespace().skip(1).collect();

    if headers.len() != values.len() {
        return None;
    }

    Some((headers, values))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_snmp_tcp() {
        let content = "\
Ip: Forwarding DefaultTTL InReceives InHdrErrors
Ip: 1 64 1858250 0
Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
Tcp: 1 200 120000 -1 6706 961 176 1132 25 1858250 1927806 224 0 5209 0
Udp: InDatagrams NoPorts InErrors OutDatagrams
Udp: 100 0 0 100
";
        let (headers, values) = parse_proc_pair(content, "Tcp:").unwrap();
        let fields: BTreeMap<&str, &str> = headers
            .iter()
            .zip(values.iter())
            .map(|(h, v)| (*h, *v))
            .collect();

        assert_eq!(fields["RetransSegs"], "224");
        assert_eq!(fields["OutSegs"], "1927806");
    }

    #[test]
    fn parse_missing_prefix_returns_none() {
        let content = "Ip: Forwarding\nIp: 1\n";
        assert!(parse_proc_pair(content, "Tcp:").is_none());
    }
}
