use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use anyhow::Result;
use futures::StreamExt;
use k8s_openapi::api::core::v1::{Node, Pod};
use kube::runtime::reflector::Store;
use kube::runtime::{reflector, watcher, WatchStreamExt};
use kube::{Api, Client};
use tokio::sync::Mutex;

use crate::app::{DataHealth, SourceStatus};
use crate::data::gpu_metrics::GpuNodeMetrics;
use crate::data::node::{FleetNode, NodeStatus};
use crate::k8s::dcgm::DcgmScraper;

pub struct ClusterDataSource {
    node_store: Store<Node>,
    pod_store: Store<Pod>,
    dcgm: DcgmScraper,
    context_name: String,
    dcgm_failures: Arc<Mutex<u32>>,
}

impl ClusterDataSource {
    pub async fn new() -> Result<Self> {
        let config = kube::Config::infer().await?;
        let context_name = config.cluster_url.to_string();

        let client = Client::try_from(config)?;
        let dcgm = DcgmScraper::discover(&client).await;

        let node_api: Api<Node> = Api::all(client.clone());
        let pod_api: Api<Pod> = Api::all(client);

        let (node_store, node_writer) = reflector::store();
        let (pod_store, pod_writer) = reflector::store();

        let node_stream = reflector(node_writer, watcher(node_api, watcher::Config::default()))
            .applied_objects()
            .boxed();
        let pod_stream = reflector(pod_writer, watcher(pod_api, watcher::Config::default()))
            .applied_objects()
            .boxed();

        tokio::spawn(async move {
            futures::pin_mut!(node_stream);
            while node_stream.next().await.is_some() {}
        });
        tokio::spawn(async move {
            futures::pin_mut!(pod_stream);
            while pod_stream.next().await.is_some() {}
        });

        Ok(Self {
            node_store,
            pod_store,
            dcgm,
            context_name,
            dcgm_failures: Arc::new(Mutex::new(0)),
        })
    }

    pub fn context_name(&self) -> &str {
        &self.context_name
    }

    pub async fn server_version(&self) -> Result<String> {
        let config = kube::Config::infer().await?;
        let client = Client::try_from(config)?;
        let version = client.apiserver_version().await?;
        Ok(format!("v{}.{}", version.major, version.minor))
    }

    pub async fn fetch_fleet(&self) -> Result<(Vec<FleetNode>, DataHealth)> {
        let nodes: Vec<Arc<Node>> = self.node_store.state();
        let pods: Vec<Arc<Pod>> = self.pod_store.state();

        let (gpu_metrics, dcgm_status) = match self.dcgm.scrape().await {
            Ok(metrics) => {
                *self.dcgm_failures.lock().await = 0;
                let status = if metrics.is_empty() {
                    SourceStatus::Unavailable
                } else {
                    SourceStatus::Connected
                };
                (metrics, status)
            }
            Err(e) => {
                let mut failures = self.dcgm_failures.lock().await;
                *failures += 1;
                let status = if *failures >= 3 {
                    SourceStatus::Degraded(format!("unreachable ({}x)", failures))
                } else {
                    SourceStatus::Degraded("retrying".to_string())
                };
                tracing::warn!("DCGM scrape failed (attempt {}): {}", failures, e);
                (HashMap::new(), status)
            }
        };

        let health = DataHealth {
            dcgm_status,
            last_fetch_error: None,
        };

        let pod_gpu_map = build_pod_gpu_map(&pods);

        let fleet: Vec<FleetNode> = nodes
            .iter()
            .filter_map(|node| {
                let name = node.metadata.name.as_deref()?;
                let status = node.status.as_ref()?;
                let allocatable = status.allocatable.as_ref()?;

                let gpu_count = allocatable
                    .get("nvidia.com/gpu")
                    .and_then(|q| q.0.parse::<u32>().ok())
                    .unwrap_or(0);

                if gpu_count == 0 {
                    return None;
                }

                let labels = node.metadata.labels.as_ref();
                let instance_type = labels
                    .and_then(|l| l.get("node.kubernetes.io/instance-type"))
                    .or_else(|| labels.and_then(|l| l.get("beta.kubernetes.io/instance-type")))
                    .cloned()
                    .unwrap_or_else(|| "unknown".to_string());

                let gpu_model = labels
                    .and_then(|l| l.get("nvidia.com/gpu.product"))
                    .cloned();

                let node_pods = pod_gpu_map.get(name);
                let gpu_allocated: u32 = node_pods
                    .map(|pods| pods.iter().map(|p| p.gpu_request).sum())
                    .unwrap_or(0);

                let workload =
                    node_pods.and_then(|pods| pods.first().map(|p| p.workload_name.clone()));

                let ready = is_node_ready(node);

                // Issue 4: try exact match, then short hostname
                let metrics = gpu_metrics.get(name).or_else(|| {
                    let short = normalize_hostname(name);
                    gpu_metrics.get(short.as_str())
                });

                let node_status = determine_status(gpu_allocated, gpu_count, metrics, ready);

                let idle_duration = if node_status == NodeStatus::Idle {
                    Some(Duration::from_secs(0))
                } else {
                    None
                };

                Some(FleetNode {
                    name: name.to_string(),
                    instance_type,
                    gpu_model,
                    gpu_count,
                    gpu_allocated,
                    utilization: metrics.map(|m| m.avg_utilization),
                    memory_pct: metrics.map(|m| m.avg_memory_pct),
                    temperature: metrics.map(|m| m.max_temperature),
                    power_watts: metrics.map(|m| m.total_power_watts),
                    workload,
                    idle_duration,
                    status: node_status,
                    ready,
                })
            })
            .collect();

        Ok((fleet, health))
    }
}

fn normalize_hostname(h: &str) -> String {
    h.split('.').next().unwrap_or(h).to_string()
}

fn determine_status(
    allocated: u32,
    total: u32,
    metrics: Option<&GpuNodeMetrics>,
    ready: bool,
) -> NodeStatus {
    if !ready {
        return NodeStatus::Degraded;
    }
    if allocated == 0 {
        return NodeStatus::Idle;
    }
    if let Some(m) = metrics {
        if m.avg_utilization >= 95.0 {
            return NodeStatus::Saturated;
        }
    }
    if allocated >= total {
        NodeStatus::Saturated
    } else {
        NodeStatus::Busy
    }
}

fn is_node_ready(node: &Node) -> bool {
    node.status
        .as_ref()
        .and_then(|s| s.conditions.as_ref())
        .map(|conditions| {
            conditions
                .iter()
                .any(|c| c.type_ == "Ready" && c.status == "True")
        })
        .unwrap_or(false)
}

struct PodGpuInfo {
    workload_name: String,
    gpu_request: u32,
}

fn build_pod_gpu_map(pods: &[Arc<Pod>]) -> HashMap<String, Vec<PodGpuInfo>> {
    let mut map: HashMap<String, Vec<PodGpuInfo>> = HashMap::new();

    for pod in pods {
        let node_name = match pod.spec.as_ref().and_then(|s| s.node_name.as_ref()) {
            Some(n) => n.clone(),
            None => continue,
        };

        let gpu_request: u32 = pod
            .spec
            .as_ref()
            .map(|spec| {
                spec.containers
                    .iter()
                    .map(|c| {
                        c.resources
                            .as_ref()
                            .and_then(|r| r.requests.as_ref())
                            .and_then(|req| req.get("nvidia.com/gpu"))
                            .and_then(|q| q.0.parse::<u32>().ok())
                            .unwrap_or(0)
                    })
                    .sum()
            })
            .unwrap_or(0);

        if gpu_request == 0 {
            continue;
        }

        let workload_name = pod
            .metadata
            .owner_references
            .as_ref()
            .and_then(|owners| owners.first())
            .map(|o| o.name.clone())
            .unwrap_or_else(|| {
                pod.metadata
                    .name
                    .clone()
                    .unwrap_or_else(|| "unknown".to_string())
            });

        map.entry(node_name).or_default().push(PodGpuInfo {
            workload_name,
            gpu_request,
        });
    }

    map
}
