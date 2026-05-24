mod cgroup;
mod collector;
mod collectors;
mod config;
mod emitter;
mod emitters;
mod engine;
mod error;
mod gpu;
mod process_state;
mod sample;

use tracing::{error, info, warn};

use crate::collectors::ebpf::EbpfCollector;
use crate::collectors::gpu::GpuCollector;
use crate::collectors::network::NetworkCollector;
use crate::collectors::process::ProcessCollector;
use crate::collectors::system::SystemCollector;
use crate::config::Config;
use crate::emitters::prometheus::PrometheusEmitter;
use crate::engine::{CollectorEntry, Engine};
use crate::gpu::nvidia::NvidiaProvider;
use crate::process_state::ProcessState;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_target(false)
        .with_thread_ids(true)
        .json()
        .with_writer(std::io::stderr)
        .init();

    let config = match Config::from_env() {
        Ok(c) => c,
        Err(e) => {
            error!(error = %e, "failed to load config");
            std::process::exit(1);
        }
    };

    info!(?config, "kitty agent starting");

    let shutdown = tokio_util::sync::CancellationToken::new();

    let shutdown_signal = shutdown.clone();
    tokio::spawn(async move {
        let ctrl_c = tokio::signal::ctrl_c();
        let mut sigterm = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to register SIGTERM handler");

        tokio::select! {
            _ = ctrl_c => { info!("received SIGINT"); }
            _ = sigterm.recv() => { info!("received SIGTERM"); }
        }
        shutdown_signal.cancel();
    });

    let engine = Engine::new(shutdown.clone(), config.node_name.clone());
    let pid_list = engine::shared_pid_list();
    let mut collector_count = 0;

    // Shared process state — refreshes every 5s, provides cached snapshot to collectors
    let process_state = ProcessState::new(
        pid_list.clone(),
        config.state_refresh_interval,
        shutdown.clone(),
    );
    let snapshot = process_state.snapshot();

    // GPU collector
    if config.collectors.gpu.enabled {
        match NvidiaProvider::new() {
            Ok(provider) => {
                let gpu_collector = GpuCollector::new(Box::new(provider), pid_list.clone());
                engine.spawn_collector(CollectorEntry {
                    collector: Box::new(gpu_collector),
                    config: config.collectors.gpu.clone(),
                });
                collector_count += 1;
                info!("gpu collector started");
            }
            Err(e) => {
                warn!(error = %e, "gpu collector unavailable");
            }
        }
    }

    // Network collector
    if config.collectors.network.enabled {
        let collector = NetworkCollector::new(snapshot.clone());
        engine.spawn_collector(CollectorEntry {
            collector: Box::new(collector),
            config: config.collectors.network.clone(),
        });
        collector_count += 1;
        info!("network collector started");
    }

    // Process collector
    if config.collectors.process.enabled {
        let collector = ProcessCollector::new(pid_list.clone());
        engine.spawn_collector(CollectorEntry {
            collector: Box::new(collector),
            config: config.collectors.process.clone(),
        });
        collector_count += 1;
        info!("process collector started");
    }

    // System collector
    if config.collectors.system.enabled {
        let collector = SystemCollector::new();
        engine.spawn_collector(CollectorEntry {
            collector: Box::new(collector),
            config: config.collectors.system.clone(),
        });
        collector_count += 1;
        info!("system collector started");
    }

    // eBPF collector
    if config.collectors.ebpf.enabled {
        match EbpfCollector::new(snapshot.clone()) {
            Ok(collector) => {
                engine.spawn_collector(CollectorEntry {
                    collector: Box::new(collector),
                    config: config.collectors.ebpf.clone(),
                });
                collector_count += 1;
                info!("ebpf collector started");
            }
            Err(e) => {
                warn!(error = %e, "ebpf collector unavailable (requires CAP_SYS_ADMIN)");
            }
        }
    }

    if collector_count == 0 {
        error!("no collectors could be initialized, exiting");
        std::process::exit(1);
    }

    info!(
        collector_count,
        "all collectors registered, starting engine"
    );

    let emitters: Vec<Box<dyn crate::emitter::Emitter>> = vec![Box::new(PrometheusEmitter::new(
        config.prometheus_port,
        shutdown.clone(),
    ))];

    engine.run(emitters).await;

    info!("kitty agent stopped");
}
