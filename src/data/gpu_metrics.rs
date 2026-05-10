#[derive(Debug, Clone, Default)]
#[allow(dead_code)]
pub struct GpuNodeMetrics {
    pub avg_utilization: f32,
    pub avg_memory_pct: f32,
    pub max_temperature: u32,
    pub total_power_watts: u32,
    pub gpu_count: u32,
}
