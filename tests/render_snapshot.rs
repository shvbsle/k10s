use std::time::Duration;

use crossterm::event::{KeyCode, KeyEvent};
use ratatui::backend::TestBackend;
use ratatui::layout::{Constraint, Layout};
use ratatui::Terminal;

use k10s::app::{AppContext, AppState, ClusterInfo, DataHealth};
use k10s::data::node::{FleetNode, NodeStatus};
use k10s::msg::AppMsg;
use k10s::ui::header;
use k10s::views::fleet::FleetView;
use k10s::views::View;

fn make_context() -> AppContext {
    AppContext {
        cluster: ClusterInfo {
            context: "arn:aws:eks:us-west-2:123456789:cluster/gpu-research".to_string(),
            k8s_version: "v1.35.4".to_string(),
            k10s_version: "v0.1.0".to_string(),
        },
        state: AppState {
            health: DataHealth::default(),
        },
    }
}

fn make_nodes(count: usize) -> Vec<FleetNode> {
    (0..count)
        .map(|i| FleetNode {
            name: format!("ip-172-31-{}-{}", i / 10, i % 10),
            instance_type: "p4d.24xlarge".to_string(),
            gpu_model: Some("Tesla-T4".to_string()),
            gpu_count: 8,
            gpu_allocated: 0,
            utilization: None,
            memory_pct: None,
            temperature: None,
            power_watts: None,
            workload: None,
            idle_duration: Some(Duration::from_secs(3600)),
            status: NodeStatus::Idle,
            ready: true,
        })
        .collect()
}

fn render_fleet(width: u16, height: u16, nodes: Vec<FleetNode>) -> String {
    let backend = TestBackend::new(width, height);
    let mut terminal = Terminal::new(backend).unwrap();
    let ctx = make_context();

    let mut fleet = FleetView::new();
    fleet.update(AppMsg::NodesUpdated(nodes), &ctx);

    terminal
        .draw(|frame| {
            let area = frame.area();
            let chunks = Layout::vertical([
                Constraint::Length(header::header_height()),
                Constraint::Min(0),
            ])
            .split(area);

            header::render_header(frame, chunks[0], &ctx);
            fleet.render(frame, chunks[1], &ctx);
        })
        .unwrap();

    let backend = terminal.backend();
    buffer_to_string(backend)
}

fn buffer_to_string(backend: &TestBackend) -> String {
    let buffer = backend.buffer();
    let mut result = String::new();
    for y in 0..buffer.area.height {
        for x in 0..buffer.area.width {
            let cell = &buffer[(x, y)];
            result.push_str(cell.symbol());
        }
        result.push('\n');
    }
    result
}

// --- Render Tests ---

#[test]
fn fleet_renders_at_80x24_without_clipping_headers() {
    let output = render_fleet(80, 24, make_nodes(10));
    println!("=== 80x24 render ===\n{}", output);

    assert!(output.contains("Context:"));
    assert!(output.contains("k10s:"));
    assert!(output.contains("NODE"));
    assert!(output.contains("MODEL"));
    assert!(output.contains("GPUs"));
    assert!(output.contains("WORKLOAD"));
}

#[test]
fn fleet_renders_at_120x40() {
    let output = render_fleet(120, 40, make_nodes(10));
    println!("=== 120x40 render ===\n{}", output);

    assert!(output.contains("NODE"));
    assert!(output.contains("WORKLOAD"));
    assert!(output.contains("ip-172-31-0-0"));
}

#[test]
fn fleet_renders_at_200x50() {
    let output = render_fleet(200, 50, make_nodes(10));
    println!("=== 200x50 render ===\n{}", output);

    assert!(output.contains("ip-172-31-0-0"));
    assert!(output.contains("Tesla-T4"));
    assert!(output.contains("IDLE"));
}

#[test]
fn fleet_renders_empty_state() {
    let output = render_fleet(80, 24, vec![]);
    println!("=== empty state ===\n{}", output);
    assert!(output.contains("Connecting to cluster"));
}

#[test]
fn fleet_renders_at_minimum_width() {
    let output = render_fleet(60, 24, make_nodes(5));
    println!("=== 60x24 render ===\n{}", output);
    assert!(output.contains("NODE"));
}

#[test]
fn header_shows_kitten_art() {
    let output = render_fleet(120, 24, make_nodes(1));
    println!("=== header kitten check ===\n{}", output);
    assert!(output.contains("o.o"));
}

#[test]
fn node_names_visible_at_default_scroll() {
    let output = render_fleet(80, 24, make_nodes(3));
    println!("=== node name visibility ===\n{}", output);
    assert!(output.contains("ip-172-31"));
}

// --- Issue 10: State Machine Tests ---

#[test]
fn fleet_handles_error_then_recovery() {
    let ctx = make_context();
    let mut fleet = FleetView::new();

    fleet.update(AppMsg::Error("connection lost".to_string()), &ctx);

    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
    let error_output = buffer_to_string(terminal.backend());
    assert!(error_output.contains("connection lost"));

    // Recovery
    fleet.update(AppMsg::NodesUpdated(make_nodes(3)), &ctx);
    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
    let recovered_output = buffer_to_string(terminal.backend());
    assert!(recovered_output.contains("ip-172-31"));
    assert!(!recovered_output.contains("connection lost"));
}

#[test]
fn fleet_key_bounds_at_last_node() {
    let ctx = make_context();
    let mut fleet = FleetView::new();
    fleet.update(AppMsg::NodesUpdated(make_nodes(5)), &ctx);

    // Move to last node
    for _ in 0..10 {
        fleet.handle_key(KeyEvent::from(KeyCode::Char('j')), &ctx);
    }
    // Should be clamped at 4 (index of last node)
    // Pressing j again shouldn't panic or go past
    fleet.handle_key(KeyEvent::from(KeyCode::Char('j')), &ctx);

    // Verify render doesn't panic
    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
}

#[test]
fn fleet_key_bounds_at_first_node() {
    let ctx = make_context();
    let mut fleet = FleetView::new();
    fleet.update(AppMsg::NodesUpdated(make_nodes(5)), &ctx);

    // Press k multiple times at the start
    for _ in 0..5 {
        fleet.handle_key(KeyEvent::from(KeyCode::Char('k')), &ctx);
    }

    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
}

#[test]
fn fleet_scroll_bounds() {
    let ctx = make_context();
    let mut fleet = FleetView::new();
    fleet.update(AppMsg::NodesUpdated(make_nodes(3)), &ctx);

    // Scroll left at 0 — should stay at 0
    fleet.handle_key(KeyEvent::from(KeyCode::Char('h')), &ctx);

    // Scroll right many times — should cap
    for _ in 0..100 {
        fleet.handle_key(KeyEvent::from(KeyCode::Char('l')), &ctx);
    }

    // Render shouldn't panic
    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
}

#[test]
fn fleet_scroll_on_empty_list() {
    let ctx = make_context();
    let mut fleet = FleetView::new();

    fleet.handle_key(KeyEvent::from(KeyCode::Char('l')), &ctx);
    fleet.handle_key(KeyEvent::from(KeyCode::Char('h')), &ctx);
    fleet.handle_key(KeyEvent::from(KeyCode::Char('j')), &ctx);
    fleet.handle_key(KeyEvent::from(KeyCode::Char('k')), &ctx);

    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
}

#[test]
fn fleet_selection_tracks_when_nodes_exceed_viewport() {
    let ctx = make_context();
    let mut fleet = FleetView::new();
    fleet.update(AppMsg::NodesUpdated(make_nodes(50)), &ctx);

    // Move selection to node 30
    for _ in 0..30 {
        fleet.handle_key(KeyEvent::from(KeyCode::Char('j')), &ctx);
    }

    // Render in 24-row terminal — shouldn't panic
    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
}

#[test]
fn fleet_update_with_fewer_nodes_adjusts_selection() {
    let ctx = make_context();
    let mut fleet = FleetView::new();
    fleet.update(AppMsg::NodesUpdated(make_nodes(10)), &ctx);

    // Select node 9 (last)
    for _ in 0..9 {
        fleet.handle_key(KeyEvent::from(KeyCode::Char('j')), &ctx);
    }

    // Now update with only 5 nodes — selection should clamp to 4
    fleet.update(AppMsg::NodesUpdated(make_nodes(5)), &ctx);

    let backend = TestBackend::new(80, 24);
    let mut terminal = Terminal::new(backend).unwrap();
    terminal
        .draw(|frame| fleet.render(frame, frame.area(), &ctx))
        .unwrap();
}
