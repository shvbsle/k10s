#![no_std]
#![no_main]

use aya_ebpf::{
    helpers::bpf_probe_read_kernel,
    macros::{kprobe, map},
    maps::RingBuf,
    programs::ProbeContext,
};

#[repr(C)]
pub struct RttEvent {
    pub sport: u16,
    pub dport: u16,
    pub rtt_us: u64,
}

#[map]
static EVENTS: RingBuf = RingBuf::with_byte_size(1024 * 1024, 0);

#[kprobe(function = "tcp_rtt_estimator")]
pub fn tcp_rtt_estimator(ctx: ProbeContext) -> u32 {
    let sk: *const u8 = match unsafe { ctx.arg(0) } {
        Some(v) => v,
        None => return 0,
    };

    let mrtt_us: u64 = match unsafe { ctx.arg(1) } {
        Some(v) => v,
        None => return 0,
    };

    // struct sock -> __sk_common.skc_num (local port, host byte order)
    // offset 14 bytes into struct sock on x86_64 kernel 6.x
    let sport: u16 = match unsafe { bpf_probe_read_kernel(sk.add(14) as *const u16) } {
        Ok(v) => v,
        Err(_) => return 0,
    };

    // struct sock -> __sk_common.skc_dport (remote port, network byte order)
    // offset 12 bytes into struct sock
    let dport_be: u16 = unsafe { bpf_probe_read_kernel(sk.add(12) as *const u16) }.unwrap_or(0);
    let dport = u16::from_be(dport_be);

    if let Some(mut buf) = EVENTS.reserve::<RttEvent>(0) {
        buf.write(RttEvent {
            sport,
            dport,
            rtt_us: mrtt_us,
        });
        buf.submit(0);
    }

    0
}

#[panic_handler]
fn panic(_info: &core::panic::PanicInfo) -> ! {
    loop {}
}
