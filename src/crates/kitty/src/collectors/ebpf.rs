use std::collections::HashMap;
use std::time::SystemTime;

use async_trait::async_trait;
use aya::maps::{MapData, RingBuf};
use aya::programs::KProbe;
use aya::Ebpf;
use tracing::info;

use crate::collector::Collector;
use crate::error::Result;
use crate::process_state::Snapshot;
use crate::sample::{MetricValue, Sample};

const BPF_OBJ_PATH: &str = "/opt/kitty/ebpf.o";
const BPF_OBJ_EMBEDDED: &[u8] = include_bytes!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../kitty-ebpf/target/bpfel-unknown-none/release/kitty-ebpf"
));

#[repr(C)]
struct RttEvent {
    sport: u16,
    dport: u16,
    rtt_us: u64,
}

pub struct EbpfCollector {
    ring_buf: RingBuf<MapData>,
    snapshot: Snapshot,
    #[allow(dead_code)]
    bpf: Ebpf,
}

impl EbpfCollector {
    pub fn new(snapshot: Snapshot) -> Result<Self> {
        let bpf_bytes = std::fs::read(BPF_OBJ_PATH).unwrap_or_else(|_| BPF_OBJ_EMBEDDED.to_vec());
        let mut bpf =
            Ebpf::load(&bpf_bytes).map_err(|e| crate::error::KittyError::Ebpf(e.to_string()))?;

        attach_kprobe(&mut bpf, "tcp_rtt_estimator", "tcp_rtt_estimator")?;

        let ring_buf = RingBuf::try_from(
            bpf.take_map("EVENTS")
                .ok_or_else(|| crate::error::KittyError::Ebpf("EVENTS map not found".into()))?,
        )
        .map_err(|e| crate::error::KittyError::Ebpf(e.to_string()))?;

        info!("ebpf probe attached: tcp_rtt_estimator");

        Ok(Self {
            ring_buf,
            snapshot,
            bpf,
        })
    }
}

#[async_trait]
impl Collector for EbpfCollector {
    fn name(&self) -> &'static str {
        "ebpf"
    }

    async fn collect(&mut self) -> Result<Vec<Sample>> {
        let now = SystemTime::now();
        let processes = self.snapshot.read().await;

        if processes.is_empty() {
            while self.ring_buf.next().is_some() {}
            return Ok(Vec::new());
        }

        let port_to_pid: HashMap<u16, u32> = processes
            .iter()
            .flat_map(|p| p.local_ports.iter().map(move |&port| (port, p.pid)))
            .collect();

        let mut rtt_per_pid: HashMap<u32, u64> = HashMap::new();

        while let Some(item) = self.ring_buf.next() {
            if item.len() < std::mem::size_of::<RttEvent>() {
                continue;
            }

            let event: RttEvent =
                unsafe { std::ptr::read_unaligned(item.as_ptr() as *const RttEvent) };

            if let Some(&pid) = port_to_pid.get(&event.sport) {
                let entry = rtt_per_pid.entry(pid).or_insert(0);
                *entry = (*entry).max(event.rtt_us);
            }
        }

        let samples = processes
            .iter()
            .map(|proc| {
                let rtt = rtt_per_pid.get(&proc.pid).copied().unwrap_or(0);
                Sample {
                    timestamp: now,
                    metric: "net.tcp_rtt_us".into(),
                    value: MetricValue::Gauge(rtt as f64),
                    labels: proc.labels.clone(),
                }
            })
            .collect();

        Ok(samples)
    }
}

fn attach_kprobe(bpf: &mut Ebpf, prog_name: &str, fn_name: &str) -> Result<()> {
    let prog: &mut KProbe = bpf
        .program_mut(prog_name)
        .ok_or_else(|| crate::error::KittyError::Ebpf(format!("{prog_name} not found")))?
        .try_into()
        .map_err(|e: aya::programs::ProgramError| crate::error::KittyError::Ebpf(e.to_string()))?;
    prog.load()
        .map_err(|e| crate::error::KittyError::Ebpf(e.to_string()))?;
    prog.attach(fn_name, 0)
        .map_err(|e| crate::error::KittyError::Ebpf(e.to_string()))?;
    Ok(())
}
