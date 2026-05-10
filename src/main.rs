use std::io;
use std::time::Duration;

use anyhow::Result;
use crossterm::event::{self, Event, KeyEventKind};
use crossterm::execute;
use crossterm::terminal::{
    disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen,
};
use ratatui::prelude::CrosstermBackend;
use ratatui::Terminal;
use tokio::sync::mpsc;

use k10s::app::App;
use k10s::k8s::cluster::ClusterDataSource;
use k10s::msg::AppMsg;

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

    let cluster = ClusterDataSource::new().await?;
    let cluster_context = cluster.context_name().to_string();
    let k8s_version = cluster
        .server_version()
        .await
        .unwrap_or_else(|_| "unknown".to_string());

    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;

    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let result = run(&mut terminal, cluster, cluster_context, k8s_version).await;

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
    cluster: ClusterDataSource,
    cluster_context: String,
    k8s_version: String,
) -> Result<()> {
    let mut app = App::new(cluster_context, k8s_version);

    let (msg_tx, mut msg_rx) = mpsc::channel::<AppMsg>(64);

    let fetch_tx = msg_tx.clone();
    tokio::spawn(async move {
        loop {
            match cluster.fetch_fleet().await {
                Ok((nodes, health)) => {
                    if let Err(e) = fetch_tx.send(AppMsg::NodesUpdated(nodes)).await {
                        tracing::warn!("channel send failed: {}", e);
                        break;
                    }
                    if let Err(e) = fetch_tx.send(AppMsg::HealthUpdate(health)).await {
                        tracing::warn!("channel send failed: {}", e);
                        break;
                    }
                }
                Err(e) => {
                    if let Err(send_err) = fetch_tx.send(AppMsg::Error(format!("{:#}", e))).await {
                        tracing::warn!("channel send failed: {}", send_err);
                        break;
                    }
                }
            }
            tokio::time::sleep(Duration::from_secs(5)).await;
        }
    });

    loop {
        // Drain pending messages first (Issue 6: process data before render)
        while let Ok(msg) = msg_rx.try_recv() {
            app.handle_msg(msg);
        }

        terminal.draw(|frame| app.render(frame))?;

        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if key.kind == KeyEventKind::Press {
                    app.handle_key(key);
                }
            }
        }

        if app.should_quit {
            break;
        }
    }

    Ok(())
}
