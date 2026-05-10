use std::io;
use std::sync::Arc;
use std::time::Duration;

use anyhow::Result;
use clap::Parser;
use crossterm::event::{self, Event, KeyEventKind};
use crossterm::execute;
use crossterm::terminal::{
    disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen,
};
use ratatui::prelude::CrosstermBackend;
use ratatui::Terminal;
use tokio::sync::mpsc;

use k10s::app::{App, DataCommand};
use k10s::datasource::live::LiveSource;
use k10s::datasource::mock::{MockConfig, MockSource};
use k10s::datasource::DataSource;
use k10s::msg::AppMsg;

#[derive(Parser)]
#[command(name = "k10s", about = "GPU-aware Kubernetes TUI")]
struct Cli {
    /// Run with mock data (e.g., "fleet:10000", "nodes:100")
    #[arg(long)]
    mock: Option<String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_writer(|| {
            std::fs::OpenOptions::new()
                .create(true)
                .append(true)
                .open("k10s.log")
                .unwrap()
        })
        .with_env_filter("k10s=debug")
        .init();

    let cli = Cli::parse();

    let data_source: Arc<dyn DataSource> = match cli.mock {
        Some(ref mock_arg) => {
            let config = MockConfig::parse(mock_arg)?;
            Arc::new(MockSource::new(config))
        }
        None => Arc::new(LiveSource::new().await?),
    };

    let context_name = data_source.context_name().to_string();
    let k8s_version = data_source
        .server_version()
        .await
        .unwrap_or_else(|_| "unknown".to_string());

    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;

    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let result = run(&mut terminal, data_source, context_name, k8s_version).await;

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    if let Err(ref e) = result {
        eprintln!("Error: {:#}", e);
    }

    result
}

async fn run(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    data_source: Arc<dyn DataSource>,
    context_name: String,
    k8s_version: String,
) -> Result<()> {
    let mut app = App::new(context_name, k8s_version);

    let (msg_tx, mut msg_rx) = mpsc::channel::<AppMsg>(64);
    let (cmd_tx, mut cmd_rx) = mpsc::channel::<DataCommand>(16);

    // Fleet data fetch loop
    let fleet_ds = data_source.clone();
    let fleet_tx = msg_tx.clone();
    tokio::spawn(async move {
        loop {
            match fleet_ds.fetch_fleet().await {
                Ok((nodes, health)) => {
                    if let Err(e) = fleet_tx.send(AppMsg::NodesUpdated(nodes)).await {
                        tracing::warn!("channel send failed: {}", e);
                        break;
                    }
                    if let Err(e) = fleet_tx.send(AppMsg::HealthUpdate(health)).await {
                        tracing::warn!("channel send failed: {}", e);
                        break;
                    }
                }
                Err(e) => {
                    if let Err(send_err) = fleet_tx.send(AppMsg::Error(format!("{:#}", e))).await {
                        tracing::warn!("channel send failed: {}", send_err);
                        break;
                    }
                }
            }
            tokio::time::sleep(Duration::from_secs(5)).await;
        }
    });

    // Generic resource fetch task — responds to DataCommands
    let resource_ds = data_source.clone();
    let resource_tx = msg_tx.clone();
    tokio::spawn(async move {
        while let Some(cmd) = cmd_rx.recv().await {
            match cmd {
                DataCommand::FetchResources { gvr, namespace } => {
                    match resource_ds.list_resources(&gvr, namespace.as_deref()).await {
                        Ok(list) => {
                            let _ = resource_tx.send(AppMsg::ResourcesUpdated(list)).await;
                        }
                        Err(e) => {
                            let _ = resource_tx.send(AppMsg::Error(format!("{:#}", e))).await;
                        }
                    }
                }
            }
        }
    });

    loop {
        while let Ok(msg) = msg_rx.try_recv() {
            if let Some(data_cmd) = app.handle_msg(msg) {
                let _ = cmd_tx.send(data_cmd).await;
            }
        }

        terminal.draw(|frame| app.render(frame))?;

        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if key.kind == KeyEventKind::Press {
                    if let Some(data_cmd) = app.handle_key(key) {
                        let _ = cmd_tx.send(data_cmd).await;
                    }
                }
            }
        }

        if app.should_quit {
            break;
        }
    }

    Ok(())
}
