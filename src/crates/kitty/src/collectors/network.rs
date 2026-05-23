use std::collections::BTreeMap;
use std::time::SystemTime;

use async_trait::async_trait;
use tracing::warn;

use crate::collector::Collector;
use crate::error::Result;
use crate::sample::{MetricValue, Sample};

enum ProcMetricKind {
    Counter,
    Gauge,
}

struct ProcMetric {
    field: &'static str,
    metric: &'static str,
    kind: ProcMetricKind,
}

static TCP_SNMP_METRICS: &[ProcMetric] = &[
    ProcMetric {
        field: "RetransSegs",
        metric: "net.tcp_retransmits",
        kind: ProcMetricKind::Counter,
    },
    ProcMetric {
        field: "InSegs",
        metric: "net.tcp_in_segs",
        kind: ProcMetricKind::Counter,
    },
    ProcMetric {
        field: "OutSegs",
        metric: "net.tcp_out_segs",
        kind: ProcMetricKind::Counter,
    },
    ProcMetric {
        field: "CurrEstab",
        metric: "net.tcp_estab_connections",
        kind: ProcMetricKind::Gauge,
    },
];

static TCP_NETSTAT_METRICS: &[ProcMetric] = &[
    ProcMetric {
        field: "TCPTimeouts",
        metric: "net.tcp_timeouts",
        kind: ProcMetricKind::Counter,
    },
    ProcMetric {
        field: "TCPFastRetrans",
        metric: "net.tcp_fast_retrans",
        kind: ProcMetricKind::Counter,
    },
    ProcMetric {
        field: "TCPLossProbes",
        metric: "net.tcp_loss_probes",
        kind: ProcMetricKind::Counter,
    },
];

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

        samples.extend(collect_proc_metrics("/proc/net/snmp", "Tcp:", TCP_SNMP_METRICS, now).await);
        samples.extend(
            collect_proc_metrics("/proc/net/netstat", "TcpExt:", TCP_NETSTAT_METRICS, now).await,
        );

        Ok(samples)
    }
}

async fn collect_proc_metrics(
    path: &str,
    prefix: &str,
    metrics: &[ProcMetric],
    now: SystemTime,
) -> Vec<Sample> {
    let content = match tokio::fs::read_to_string(path).await {
        Ok(c) => c,
        Err(e) => {
            warn!(path, error = %e, "failed to read procfs");
            return Vec::new();
        }
    };

    let Some((headers, values)) = parse_proc_pair(&content, prefix) else {
        warn!(path, prefix, "failed to parse procfs section");
        return Vec::new();
    };

    let fields: BTreeMap<&str, u64> = headers
        .iter()
        .zip(values.iter())
        .filter_map(|(h, v)| v.parse::<u64>().ok().map(|val| (*h, val)))
        .collect();

    metrics
        .iter()
        .filter_map(|m| {
            fields.get(m.field).map(|&val| Sample {
                timestamp: now,
                metric: m.metric.into(),
                value: match m.kind {
                    ProcMetricKind::Counter => MetricValue::Counter(val),
                    ProcMetricKind::Gauge => MetricValue::Gauge(val as f64),
                },
                labels: BTreeMap::new(),
            })
        })
        .collect()
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

/// Parses /proc/net/snmp and /proc/net/netstat format: two consecutive lines
/// with the same prefix, first is headers, second is values.
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
        assert_eq!(fields["CurrEstab"], "25");
        assert_eq!(fields["InSegs"], "1858250");
        assert_eq!(fields["OutSegs"], "1927806");
    }

    #[test]
    fn parse_netstat_tcpext() {
        let content = "\
TcpExt: SyncookiesSent TCPTimeouts TCPFastRetrans TCPLossProbes TCPSackRecovery
TcpExt: 0 6 92 126 11
IpExt: InNoRoutes InTruncatedPkts
IpExt: 0 0
";
        let (headers, values) = parse_proc_pair(content, "TcpExt:").unwrap();
        let fields: BTreeMap<&str, &str> = headers
            .iter()
            .zip(values.iter())
            .map(|(h, v)| (*h, *v))
            .collect();

        assert_eq!(fields["TCPTimeouts"], "6");
        assert_eq!(fields["TCPFastRetrans"], "92");
        assert_eq!(fields["TCPLossProbes"], "126");
    }

    #[test]
    fn parse_missing_prefix_returns_none() {
        let content = "Ip: Forwarding\nIp: 1\n";
        assert!(parse_proc_pair(content, "Tcp:").is_none());
    }
}
