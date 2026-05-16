use async_trait::async_trait;
use nvml_wrapper::enum_wrappers::device::TemperatureSensor;
use nvml_wrapper::Nvml;

use crate::error::{KittyError, Result};
use crate::gpu::{
    GpuMemoryInfo, GpuProcess, GpuProvider, GpuUtilization, NvlinkThroughput, PcieThroughput,
};
use nvml_wrapper::enums::device::UsedGpuMemory;

pub struct NvidiaProvider {
    nvml: Nvml,
    device_count: u32,
}

impl NvidiaProvider {
    pub fn new() -> Result<Self> {
        let nvml = Nvml::init().map_err(|e| nvml_err(&e))?;
        let device_count = nvml.device_count().map_err(|e| nvml_err(&e))?;
        Ok(Self { nvml, device_count })
    }
}

#[async_trait]
impl GpuProvider for NvidiaProvider {
    fn vendor(&self) -> &'static str {
        "nvidia"
    }

    fn device_count(&self) -> Result<u32> {
        Ok(self.device_count)
    }

    async fn utilization(&self, device_idx: u32) -> Result<GpuUtilization> {
        let nvml = &self.nvml;
        let device = nvml.device_by_index(device_idx).map_err(|e| nvml_err(&e))?;
        let util = device.utilization_rates().map_err(|e| nvml_err(&e))?;
        Ok(GpuUtilization {
            sm: util.gpu as f64,
            memory: util.memory as f64,
        })
    }

    async fn memory_info(&self, device_idx: u32) -> Result<GpuMemoryInfo> {
        let device = self
            .nvml
            .device_by_index(device_idx)
            .map_err(|e| nvml_err(&e))?;
        let mem = device.memory_info().map_err(|e| nvml_err(&e))?;
        Ok(GpuMemoryInfo {
            used: mem.used,
            free: mem.free,
            total: mem.total,
        })
    }

    async fn power_usage(&self, device_idx: u32) -> Result<f64> {
        let device = self
            .nvml
            .device_by_index(device_idx)
            .map_err(|e| nvml_err(&e))?;
        let milliwatts = device.power_usage().map_err(|e| nvml_err(&e))?;
        Ok(milliwatts as f64 / 1000.0)
    }

    async fn temperature(&self, device_idx: u32) -> Result<f64> {
        let device = self
            .nvml
            .device_by_index(device_idx)
            .map_err(|e| nvml_err(&e))?;
        let temp = device
            .temperature(TemperatureSensor::Gpu)
            .map_err(|e| nvml_err(&e))?;
        Ok(temp as f64)
    }

    async fn pcie_throughput(&self, device_idx: u32) -> Result<PcieThroughput> {
        let device = self
            .nvml
            .device_by_index(device_idx)
            .map_err(|e| nvml_err(&e))?;
        let tx = device
            .pcie_throughput(nvml_wrapper::enum_wrappers::device::PcieUtilCounter::Send)
            .map_err(|e| nvml_err(&e))?;
        let rx = device
            .pcie_throughput(nvml_wrapper::enum_wrappers::device::PcieUtilCounter::Receive)
            .map_err(|e| nvml_err(&e))?;
        // NVML returns KB/s, convert to bytes
        Ok(PcieThroughput {
            tx_bytes: tx as u64 * 1024,
            rx_bytes: rx as u64 * 1024,
        })
    }

    async fn nvlink_throughput(&self, device_idx: u32) -> Result<Option<NvlinkThroughput>> {
        let device = self
            .nvml
            .device_by_index(device_idx)
            .map_err(|e| nvml_err(&e))?;

        let mut total_tx: u64 = 0;
        let mut total_rx: u64 = 0;
        let mut has_nvlink = false;

        for link_idx in 0..18 {
            let link = device.link_wrapper_for(link_idx);
            match link.is_active() {
                Ok(true) => {
                    has_nvlink = true;
                    if let Ok(counter) =
                        link.utilization_counter(nvml_wrapper::enums::nv_link::Counter::Zero)
                    {
                        total_tx += counter.send;
                        total_rx += counter.receive;
                    }
                }
                Ok(false) => {}
                Err(_) => break,
            }
        }

        if has_nvlink {
            Ok(Some(NvlinkThroughput {
                tx_bytes: total_tx,
                rx_bytes: total_rx,
            }))
        } else {
            Ok(None)
        }
    }

    async fn running_compute_processes(&self, device_idx: u32) -> Result<Vec<GpuProcess>> {
        let device = self
            .nvml
            .device_by_index(device_idx)
            .map_err(|e| nvml_err(&e))?;
        let procs = device
            .running_compute_processes()
            .map_err(|e| nvml_err(&e))?;
        Ok(procs
            .into_iter()
            .map(|p| GpuProcess {
                pid: p.pid,
                gpu_memory_used: match p.used_gpu_memory {
                    UsedGpuMemory::Used(bytes) => Some(bytes),
                    UsedGpuMemory::Unavailable => None,
                },
            })
            .collect())
    }
}

fn nvml_err(e: &dyn std::fmt::Display) -> KittyError {
    KittyError::Gpu {
        vendor: "nvidia".to_string(),
        msg: e.to_string(),
    }
}
