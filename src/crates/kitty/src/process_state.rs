use std::collections::{BTreeMap, HashSet};
use std::sync::Arc;
use std::time::Duration;

use tokio::sync::RwLock;
use tokio_util::sync::CancellationToken;
use tracing::info;

pub struct WatchedProcess {
    pub pid: u32,
    pub labels: BTreeMap<String, String>,
    pub local_ports: HashSet<u16>,
    pub tcp_tx_queue_bytes: u64,
}

pub type Snapshot = Arc<RwLock<Vec<WatchedProcess>>>;

pub struct ProcessState {
    snapshot: Snapshot,
}

impl ProcessState {
    pub fn new(
        pids: Arc<RwLock<Vec<u32>>>,
        refresh_interval: Duration,
        shutdown: CancellationToken,
    ) -> Self {
        let snapshot: Snapshot = Arc::new(RwLock::new(Vec::new()));
        let snapshot_writer = snapshot.clone();

        tokio::spawn(async move {
            let mut interval = tokio::time::interval(refresh_interval);
            interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);

            loop {
                tokio::select! {
                    _ = shutdown.cancelled() => break,
                    _ = interval.tick() => {
                        let current_pids = pids.read().await.clone();
                        let new_snapshot = refresh(&current_pids).await;
                        *snapshot_writer.write().await = new_snapshot;
                    }
                }
            }
        });

        info!(
            refresh_ms = refresh_interval.as_millis(),
            "process state started"
        );

        Self { snapshot }
    }

    pub fn snapshot(&self) -> Snapshot {
        self.snapshot.clone()
    }
}

async fn refresh(pids: &[u32]) -> Vec<WatchedProcess> {
    let mut processes = Vec::new();

    for &pid in pids {
        let Some(labels) = read_labels(pid).await else {
            continue;
        };

        let (local_ports, tcp_tx_queue_bytes) = parse_tcp_state(pid).await;

        processes.push(WatchedProcess {
            pid,
            labels,
            local_ports,
            tcp_tx_queue_bytes,
        });
    }

    processes
}

async fn read_labels(pid: u32) -> Option<BTreeMap<String, String>> {
    let data = tokio::fs::read(format!("/proc/{pid}/environ")).await.ok()?;
    let env_str = String::from_utf8_lossy(&data);
    let vars: Vec<&str> = env_str.split('\0').collect();

    if !vars.contains(&"KITTY_WATCH=1") {
        return None;
    }

    let mut labels = BTreeMap::new();
    labels.insert("pid".into(), pid.to_string());
    extract_framework_labels(&vars, &mut labels);

    Some(labels)
}

/// Extracts framework-specific labels from the process environment.
/// Supports multiple training frameworks by checking for known env var patterns.
/// Extend this function to add support for new frameworks.
fn extract_framework_labels(vars: &[&str], labels: &mut BTreeMap<String, String>) {
    for var in vars {
        // PyTorch DDP / torchrun
        if let Some(rank) = var.strip_prefix("RANK=") {
            labels.insert("rank".into(), rank.to_string());
        } else if let Some(job) = var.strip_prefix("JOB_NAME=") {
            labels.insert("job".into(), job.to_string());
        }
        // JAX / XLA
        else if let Some(rank) = var.strip_prefix("PROCESS_INDEX=") {
            labels
                .entry("rank".into())
                .or_insert_with(|| rank.to_string());
        }
        // General (Kubeflow TrainJob, MPI)
        else if let Some(rank) = var.strip_prefix("OMPI_COMM_WORLD_RANK=") {
            labels
                .entry("rank".into())
                .or_insert_with(|| rank.to_string());
        }
    }
}

async fn parse_tcp_state(pid: u32) -> (HashSet<u16>, u64) {
    let mut ports = HashSet::new();
    let mut tx_queue_total: u64 = 0;

    let Ok(content) = tokio::fs::read_to_string(format!("/proc/{pid}/net/tcp")).await else {
        return (ports, tx_queue_total);
    };

    for line in content.lines().skip(1) {
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() < 5 || fields[3] != "01" {
            continue;
        }

        if let Some(port_hex) = fields[1].split(':').nth(1) {
            if let Ok(port) = u16::from_str_radix(port_hex, 16) {
                ports.insert(port);
            }
        }

        if let Some((tx_hex, _)) = fields[4].split_once(':') {
            tx_queue_total += u64::from_str_radix(tx_hex, 16).unwrap_or(0);
        }
    }

    (ports, tx_queue_total)
}
