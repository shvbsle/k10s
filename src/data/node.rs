use std::time::Duration;

#[derive(Debug, Clone, PartialEq)]
pub enum NodeStatus {
    Idle,
    Busy,
    Saturated,
    Degraded,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct FleetNode {
    pub name: String,
    pub instance_type: String,
    pub gpu_model: Option<String>,
    pub gpu_count: u32,
    pub gpu_allocated: u32,
    pub utilization: Option<f32>,
    pub memory_pct: Option<f32>,
    pub temperature: Option<u32>,
    pub power_watts: Option<u32>,
    pub workload: Option<String>,
    pub idle_duration: Option<Duration>,
    pub status: NodeStatus,
    pub ready: bool,
}
