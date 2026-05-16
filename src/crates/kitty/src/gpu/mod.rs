pub mod nvidia;

use async_trait::async_trait;

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct GpuUtilization {
    pub sm: f64,
    pub memory: f64,
}

#[derive(Debug, Clone)]
pub struct GpuMemoryInfo {
    pub used: u64,
    pub free: u64,
    pub total: u64,
}

#[derive(Debug, Clone)]
pub struct PcieThroughput {
    pub tx_bytes: u64,
    pub rx_bytes: u64,
}

#[derive(Debug, Clone)]
pub struct NvlinkThroughput {
    pub tx_bytes: u64,
    pub rx_bytes: u64,
}

#[derive(Debug, Clone)]
pub struct GpuProcess {
    pub pid: u32,
    pub gpu_memory_used: Option<u64>,
}

#[async_trait]
pub trait GpuProvider: Send + Sync {
    fn vendor(&self) -> &'static str;
    fn device_count(&self) -> Result<u32>;

    async fn utilization(&self, device_idx: u32) -> Result<GpuUtilization>;
    async fn memory_info(&self, device_idx: u32) -> Result<GpuMemoryInfo>;
    async fn power_usage(&self, device_idx: u32) -> Result<f64>;
    async fn temperature(&self, device_idx: u32) -> Result<f64>;
    async fn pcie_throughput(&self, device_idx: u32) -> Result<PcieThroughput>;
    async fn nvlink_throughput(&self, device_idx: u32) -> Result<Option<NvlinkThroughput>>;
    async fn running_compute_processes(&self, device_idx: u32) -> Result<Vec<GpuProcess>>;
}
