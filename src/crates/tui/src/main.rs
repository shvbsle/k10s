fn main() {
    if std::env::args().any(|arg| arg == "--version" || arg == "-V") {
        println!("k10s {}", env!("CARGO_PKG_VERSION"));
        return;
    }

    println!("k10s tui");
}
