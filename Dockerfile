FROM rust:bookworm AS builder
WORKDIR /build
RUN rustup toolchain install nightly --component rust-src && cargo install bpf-linker
COPY Cargo.toml Cargo.lock ./
COPY src/ src/
RUN cd src/crates/kitty-ebpf && cargo +nightly build --release
RUN cargo build --release -p kitty

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/target/release/kitty /usr/local/bin/kitty
COPY --from=builder /build/src/crates/kitty-ebpf/target/bpfel-unknown-none/release/kitty-ebpf /opt/kitty/ebpf.o
ENTRYPOINT ["kitty"]
