use std::collections::HashMap;
use std::time::Duration;

use anyhow::Result;
use async_trait::async_trait;
use serde_json::json;

use crate::app::{DataHealth, SourceStatus};
use crate::data::node::{FleetNode, NodeStatus};
use crate::k8s::gvr::Gvr;

use super::{DataSource, ResourceList, ResourceObject};

pub struct MockConfig {
    pub resource: String,
    pub count: usize,
}

impl MockConfig {
    pub fn parse(s: &str) -> Result<Self> {
        let (resource, count_str) = s.split_once(':').ok_or_else(|| {
            anyhow::anyhow!("mock format: <resource>:<count> (e.g., fleet:10000)")
        })?;
        let count = count_str.parse::<usize>()?;
        Ok(Self {
            resource: resource.to_string(),
            count,
        })
    }
}

pub struct MockSource {
    config: MockConfig,
}

impl MockSource {
    pub fn new(config: MockConfig) -> Self {
        Self { config }
    }

    fn generate_fleet_nodes(&self) -> Vec<FleetNode> {
        (0..self.config.count)
            .map(|i| {
                let status = match i % 7 {
                    0 | 1 => NodeStatus::Idle,
                    2..=4 => NodeStatus::Busy,
                    5 => NodeStatus::Saturated,
                    _ => NodeStatus::Degraded,
                };
                let gpu_count = match i % 3 {
                    0 => 8,
                    1 => 4,
                    _ => 1,
                };
                let gpu_allocated = match status {
                    NodeStatus::Idle => 0,
                    NodeStatus::Busy => gpu_count / 2,
                    NodeStatus::Saturated => gpu_count,
                    NodeStatus::Degraded => 0,
                };
                let utilization = match status {
                    NodeStatus::Idle => None,
                    NodeStatus::Busy => Some(40.0 + (i % 30) as f32),
                    NodeStatus::Saturated => Some(95.0 + (i % 5) as f32),
                    NodeStatus::Degraded => None,
                };

                FleetNode {
                    name: format!("ip-10-0-{}-{}", i / 256, i % 256),
                    instance_type: "p4d.24xlarge".to_string(),
                    gpu_model: Some("H100-80GB".to_string()),
                    gpu_count,
                    gpu_allocated,
                    utilization,
                    memory_pct: utilization.map(|u| u * 0.8),
                    temperature: utilization.map(|u| (35.0 + u * 0.4) as u32),
                    power_watts: utilization.map(|u| (100.0 + u * 3.0) as u32),
                    workload: match status {
                        NodeStatus::Idle => None,
                        NodeStatus::Busy => Some(format!("training-job-{}", i / 8)),
                        NodeStatus::Saturated => Some(format!("llm-finetune-{}", i / 8)),
                        NodeStatus::Degraded => None,
                    },
                    idle_duration: match status {
                        NodeStatus::Idle => Some(Duration::from_secs(3600 * (i as u64 % 24 + 1))),
                        _ => None,
                    },
                    status,
                    ready: i % 7 != 6,
                }
            })
            .collect()
    }

    fn generate_resource_objects(&self, gvr: &Gvr) -> Vec<ResourceObject> {
        (0..self.config.count)
            .map(|i| {
                let name = format!("{}-{}", gvr.resource.trim_end_matches('s'), i);
                let namespace = if gvr.resource == "nodes" {
                    None
                } else {
                    Some("default".to_string())
                };
                ResourceObject {
                    uid: format!("uid-{}", i),
                    name: name.clone(),
                    namespace: namespace.clone(),
                    raw: json!({
                        "metadata": {
                            "name": name,
                            "namespace": namespace,
                            "uid": format!("uid-{}", i),
                            "creationTimestamp": "2026-05-01T00:00:00Z",
                        },
                        "status": {
                            "phase": "Running",
                            "conditions": [{
                                "type": "Ready",
                                "status": "True"
                            }]
                        }
                    }),
                }
            })
            .collect()
    }
}

#[async_trait]
impl DataSource for MockSource {
    async fn fetch_fleet(&self) -> Result<(Vec<FleetNode>, DataHealth)> {
        let nodes = self.generate_fleet_nodes();
        let health = DataHealth {
            dcgm_status: SourceStatus::Connected,
            last_fetch_error: None,
        };
        Ok((nodes, health))
    }

    async fn list_resources(&self, gvr: &Gvr, _namespace: Option<&str>) -> Result<ResourceList> {
        Ok(ResourceList {
            gvr: gvr.clone(),
            objects: self.generate_resource_objects(gvr),
        })
    }

    fn context_name(&self) -> &str {
        "mock-cluster"
    }

    async fn server_version(&self) -> Result<String> {
        Ok("v1.35.0".to_string())
    }

    async fn discover_resources(&self) -> Result<HashMap<String, Gvr>> {
        let mut map = HashMap::new();
        map.insert("nodes".to_string(), Gvr::new("", "v1", "nodes"));
        map.insert("pods".to_string(), Gvr::new("", "v1", "pods"));
        map.insert("services".to_string(), Gvr::new("", "v1", "services"));
        map.insert(
            "deployments".to_string(),
            Gvr::new("apps", "v1", "deployments"),
        );
        Ok(map)
    }
}
