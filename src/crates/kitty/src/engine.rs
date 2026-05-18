use std::sync::Arc;

use tokio::sync::{mpsc, RwLock};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};

use crate::collector::Collector;
use crate::config::CollectorConfig;
use crate::emitter::Emitter;
use crate::sample::Sample;

pub struct Engine {
    shutdown: CancellationToken,
    sample_tx: mpsc::Sender<Vec<Sample>>,
    sample_rx: mpsc::Receiver<Vec<Sample>>,
    node_name: String,
}

pub struct CollectorEntry {
    pub collector: Box<dyn Collector>,
    pub config: CollectorConfig,
}

impl Engine {
    pub fn new(shutdown: CancellationToken, node_name: String) -> Self {
        let (sample_tx, sample_rx) = mpsc::channel(256);
        Self {
            shutdown,
            sample_tx,
            sample_rx,
            node_name,
        }
    }

    pub fn spawn_collector(&self, entry: CollectorEntry) {
        let tx = self.sample_tx.clone();
        let shutdown = self.shutdown.clone();
        let interval_duration = entry.config.interval;
        let mut collector = entry.collector;

        tokio::spawn(async move {
            let mut interval = tokio::time::interval(interval_duration);
            interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);

            loop {
                tokio::select! {
                    _ = shutdown.cancelled() => {
                        info!(collector = collector.name(), "shutting down");
                        break;
                    }
                    _ = interval.tick() => {
                        match collector.collect().await {
                            Ok(samples) if !samples.is_empty() => {
                                if tx.send(samples).await.is_err() {
                                    break;
                                }
                            }
                            Ok(_) => {}
                            Err(e) => {
                                warn!(collector = collector.name(), error = %e, "collection failed");
                            }
                        }
                    }
                }
            }
        });
    }

    pub async fn run(mut self, mut emitters: Vec<Box<dyn Emitter>>) {
        info!("engine started, waiting for samples");

        loop {
            tokio::select! {
                _ = self.shutdown.cancelled() => {
                    info!("engine shutting down");
                    break;
                }
                recv = self.sample_rx.recv() => {
                    match recv {
                        Some(mut samples) => {
                            for sample in &mut samples {
                                sample.labels.insert("node".into(), self.node_name.clone());
                            }
                            for emitter in &mut emitters {
                                if let Err(e) = emitter.emit(&samples).await {
                                    error!(emitter = emitter.name(), error = %e, "emission failed");
                                }
                            }
                        }
                        None => {
                            info!("all collector channels closed");
                            break;
                        }
                    }
                }
            }
        }
    }
}

/// Shared PID list that the GPU collector writes to and the process collector reads from.
pub fn shared_pid_list() -> Arc<RwLock<Vec<u32>>> {
    Arc::new(RwLock::new(Vec::new()))
}
