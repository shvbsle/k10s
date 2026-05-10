use std::collections::HashMap;

use anyhow::Result;
use k8s_openapi::api::core::v1::Service;
use kube::api::ListParams;
use kube::{Api, Client};

use crate::data::gpu_metrics::GpuNodeMetrics;

pub struct DcgmScraper {
    endpoint: Option<String>,
}

impl DcgmScraper {
    pub async fn discover(client: &Client) -> Self {
        let endpoint = find_dcgm_endpoint(client).await.ok().flatten();
        if let Some(ref ep) = endpoint {
            tracing::info!("DCGM exporter discovered at {}", ep);
        } else {
            tracing::warn!("DCGM exporter not found — GPU metrics will be unavailable");
        }
        Self { endpoint }
    }

    pub async fn scrape(&self) -> Result<HashMap<String, GpuNodeMetrics>> {
        let endpoint = match &self.endpoint {
            Some(ep) => ep,
            None => return Ok(HashMap::new()),
        };

        let body = reqwest::get(format!("{}/metrics", endpoint))
            .await?
            .text()
            .await?;

        Ok(parse_dcgm_metrics(&body))
    }
}

async fn find_dcgm_endpoint(client: &Client) -> Result<Option<String>> {
    let namespaces = [
        "gpu-operator",
        "monitoring",
        "dcgm-exporter",
        "nvidia-device-plugin",
        "default",
    ];

    for ns in &namespaces {
        let svc_api: Api<Service> = Api::namespaced(client.clone(), ns);
        let svcs = svc_api.list(&ListParams::default()).await;
        if let Ok(svc_list) = svcs {
            for svc in svc_list.items {
                let name = svc.metadata.name.as_deref().unwrap_or("");
                if name.contains("dcgm") || name.contains("gpu-metrics") {
                    let port = svc
                        .spec
                        .as_ref()
                        .and_then(|s| s.ports.as_ref())
                        .and_then(|ports| ports.first())
                        .map(|p| p.port)
                        .unwrap_or(9400);
                    let endpoint = format!("http://{}.{}.svc.cluster.local:{}", name, ns, port);
                    return Ok(Some(endpoint));
                }
            }
        }
    }

    Ok(None)
}

fn parse_dcgm_metrics(body: &str) -> HashMap<String, GpuNodeMetrics> {
    let mut node_gpus: HashMap<String, Vec<GpuDeviceMetrics>> = HashMap::new();

    for line in body.lines() {
        if line.starts_with('#') || line.is_empty() {
            continue;
        }

        let hostname = extract_label(line, "Hostname")
            .or_else(|| extract_label(line, "hostname"))
            .or_else(|| extract_label(line, "instance"));

        let hostname = match hostname {
            Some(h) => normalize_hostname(&h),
            None => continue,
        };

        let gpu_idx = extract_label(line, "gpu")
            .and_then(|g| g.parse::<usize>().ok())
            .unwrap_or(0);

        let entry = node_gpus.entry(hostname).or_default();
        while entry.len() <= gpu_idx {
            entry.push(GpuDeviceMetrics::default());
        }
        let device = &mut entry[gpu_idx];

        if let Some(value) = extract_metric_value(line) {
            if line.starts_with("DCGM_FI_DEV_GPU_UTIL") {
                device.utilization = value;
            } else if line.starts_with("DCGM_FI_DEV_FB_USED") {
                device.fb_used = value;
            } else if line.starts_with("DCGM_FI_DEV_FB_FREE") {
                device.fb_free = value;
            } else if line.starts_with("DCGM_FI_DEV_GPU_TEMP") {
                device.temperature = value;
            } else if line.starts_with("DCGM_FI_DEV_POWER_USAGE") {
                device.power_watts = value;
            }
        }
    }

    node_gpus
        .into_iter()
        .map(|(hostname, gpus)| {
            let count = gpus.len() as u32;
            let avg_util = gpus.iter().map(|g| g.utilization).sum::<f32>() / count as f32;
            let avg_mem = gpus
                .iter()
                .map(|g| {
                    let total = g.fb_used + g.fb_free;
                    if total > 0.0 {
                        (g.fb_used / total) * 100.0
                    } else {
                        0.0
                    }
                })
                .sum::<f32>()
                / count as f32;
            let max_temp = gpus.iter().map(|g| g.temperature as u32).max().unwrap_or(0);
            let total_power = gpus.iter().map(|g| g.power_watts as u32).sum();

            (
                hostname,
                GpuNodeMetrics {
                    avg_utilization: avg_util,
                    avg_memory_pct: avg_mem,
                    max_temperature: max_temp,
                    total_power_watts: total_power,
                    gpu_count: count,
                },
            )
        })
        .collect()
}

#[derive(Debug, Default)]
struct GpuDeviceMetrics {
    utilization: f32,
    fb_used: f32,
    fb_free: f32,
    temperature: f32,
    power_watts: f32,
}

fn normalize_hostname(h: &str) -> String {
    h.split('.').next().unwrap_or(h).to_string()
}

fn extract_label(line: &str, label_name: &str) -> Option<String> {
    let pattern = format!("{}=\"", label_name);
    let start = line.find(&pattern)? + pattern.len();
    let rest = &line[start..];
    let end = rest.find('"')?;
    Some(rest[..end].to_string())
}

fn extract_metric_value(line: &str) -> Option<f32> {
    let parts: Vec<&str> = line.rsplitn(2, [' ', '\t']).collect();
    if parts.len() == 2 {
        parts[0].trim().parse().ok()
    } else {
        None
    }
}
