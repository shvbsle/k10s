use crossterm::event::{KeyCode, KeyEvent};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{
    Block, Borders, Row, Scrollbar, ScrollbarOrientation, ScrollbarState, Table, TableState,
};
use ratatui::Frame;

use crate::action::Action;
use crate::app::AppContext;
use crate::data::node::{FleetNode, NodeStatus};
use crate::msg::AppMsg;
use crate::ui::styles;
use crate::views::View;

const SCROLL_STEP: u16 = 4;
const COL_WIDTHS: [u16; 9] = [28, 14, 5, 7, 18, 5, 6, 7, 22];

#[derive(Default)]
pub struct FleetView {
    nodes: Vec<FleetNode>,
    cells: Vec<[String; 9]>,
    selected: usize,
    scroll_x: u16,
    error: Option<String>,
}

impl FleetView {
    pub fn new() -> Self {
        Self::default()
    }

    fn total_width() -> u16 {
        COL_WIDTHS.iter().sum()
    }

    fn visible_columns(scroll_x: u16) -> (usize, Vec<ratatui::layout::Constraint>) {
        let mut consumed: u16 = 0;
        let mut first_visible = 0;
        for (i, &w) in COL_WIDTHS.iter().enumerate() {
            if consumed + w + 1 > scroll_x {
                first_visible = i;
                break;
            }
            consumed += w + 1; // +1 for column spacing
        }
        let widths: Vec<ratatui::layout::Constraint> = COL_WIDTHS[first_visible..]
            .iter()
            .map(|&w| ratatui::layout::Constraint::Min(w))
            .collect();
        (first_visible, widths)
    }

    fn recompute_cells(&mut self) {
        self.cells = self
            .nodes
            .iter()
            .map(|node| {
                let workload_str = match &node.workload {
                    Some(w) => w.clone(),
                    None if node.status == NodeStatus::Idle => "IDLE".to_string(),
                    None => "—".to_string(),
                };
                [
                    node.name.clone(),
                    node.gpu_model
                        .clone()
                        .unwrap_or_else(|| "unknown".to_string()),
                    format!("{}x", node.gpu_count),
                    format!("{}/{}", node.gpu_allocated, node.gpu_count),
                    Self::render_util(node.utilization),
                    Self::render_optional_pct(node.memory_pct),
                    Self::render_optional_u32(node.temperature, "°C"),
                    Self::render_optional_u32(node.power_watts, "W"),
                    workload_str,
                ]
            })
            .collect();
    }

    fn sort_nodes(&mut self) {
        self.nodes.sort_by(|a, b| {
            let order = |s: &NodeStatus| match s {
                NodeStatus::Idle => 0,
                NodeStatus::Degraded => 1,
                NodeStatus::Busy => 2,
                NodeStatus::Saturated => 3,
            };
            let ord = order(&a.status).cmp(&order(&b.status));
            if ord != std::cmp::Ordering::Equal {
                return ord;
            }
            match (&a.status, &b.status) {
                (NodeStatus::Idle, NodeStatus::Idle) => b.idle_duration.cmp(&a.idle_duration),
                _ => a
                    .utilization
                    .partial_cmp(&b.utilization)
                    .unwrap_or(std::cmp::Ordering::Equal),
            }
        });
    }

    fn render_util(val: Option<f32>) -> String {
        match val {
            Some(v) => {
                let filled = ((v / 10.0).round() as usize).min(10);
                let bar: String = "#".repeat(filled) + &" ".repeat(10 - filled);
                format!("[{}] {:>3.0}%", bar, v)
            }
            None => "—".to_string(),
        }
    }

    fn render_optional_pct(val: Option<f32>) -> String {
        match val {
            Some(v) => format!("{:.0}%", v),
            None => "—".to_string(),
        }
    }

    fn render_optional_u32(val: Option<u32>, suffix: &str) -> String {
        match val {
            Some(v) => format!("{}{}", v, suffix),
            None => "—".to_string(),
        }
    }
}

impl View for FleetView {
    fn update(&mut self, msg: AppMsg, _ctx: &AppContext) -> Option<Action> {
        match msg {
            AppMsg::NodesUpdated(nodes) => {
                self.nodes = nodes;
                self.sort_nodes();
                self.recompute_cells();
                self.error = None;
                if self.selected >= self.nodes.len() {
                    self.selected = self.nodes.len().saturating_sub(1);
                }
            }
            AppMsg::Error(e) => {
                self.error = Some(e);
            }
            AppMsg::HealthUpdate(_)
            | AppMsg::ResourcesUpdated(_)
            | AppMsg::DiscoveryComplete(_) => {}
        }
        None
    }

    fn handle_key(&mut self, key: KeyEvent, _ctx: &AppContext) -> Option<Action> {
        match key.code {
            KeyCode::Char('j') | KeyCode::Down
                if self.selected < self.nodes.len().saturating_sub(1) =>
            {
                self.selected += 1;
            }
            KeyCode::Char('k') | KeyCode::Up => {
                self.selected = self.selected.saturating_sub(1);
            }
            KeyCode::Char('h') | KeyCode::Left => {
                self.scroll_x = self.scroll_x.saturating_sub(SCROLL_STEP);
            }
            KeyCode::Char('l') | KeyCode::Right => {
                let max = Self::total_width();
                self.scroll_x = (self.scroll_x + SCROLL_STEP).min(max);
            }
            KeyCode::Char('G') => {
                self.selected = self.nodes.len().saturating_sub(1);
            }
            KeyCode::Char('g') => {
                self.selected = 0;
            }
            KeyCode::Char('0') => {
                self.scroll_x = 0;
            }
            KeyCode::Char('?') => return Some(Action::ShowHelp),
            KeyCode::Char('q') => return Some(Action::Quit),
            _ => {}
        }
        None
    }

    fn render(&self, frame: &mut Frame, area: Rect, _ctx: &AppContext) {
        if let Some(ref err) = self.error {
            let block = Block::default()
                .borders(Borders::ALL)
                .title(Line::from(" k10s — GPU Fleet (error) "));
            let paragraph = ratatui::widgets::Paragraph::new(err.as_str())
                .style(styles::degraded_style())
                .block(block);
            frame.render_widget(paragraph, area);
            return;
        }

        if self.nodes.is_empty() {
            let block = Block::default()
                .borders(Borders::ALL)
                .title(Line::from(" k10s — GPU Fleet "));
            let paragraph =
                ratatui::widgets::Paragraph::new("Connecting to cluster...").block(block);
            frame.render_widget(paragraph, area);
            return;
        }

        let (first_col, widths) = Self::visible_columns(self.scroll_x);

        let all_headers = [
            "NODE", "MODEL", "GPUs", "ALLOC", "UTIL", "MEM", "TEMP", "POWER", "WORKLOAD",
        ];
        let header = Row::new(all_headers[first_col..].to_vec()).style(styles::header_style());

        let rows: Vec<Row> = self
            .nodes
            .iter()
            .zip(self.cells.iter())
            .enumerate()
            .map(|(i, (node, cells))| {
                let style = match node.status {
                    NodeStatus::Idle => styles::idle_style(),
                    NodeStatus::Busy => styles::busy_style(),
                    NodeStatus::Saturated => styles::saturated_style(),
                    NodeStatus::Degraded => styles::degraded_style(),
                };

                let visible_cells: Vec<&str> =
                    cells[first_col..].iter().map(|s| s.as_str()).collect();
                let row = Row::new(visible_cells);

                if i == self.selected {
                    row.style(style.patch(
                        ratatui::style::Style::default().bg(ratatui::style::Color::DarkGray),
                    ))
                } else {
                    row.style(style)
                }
            })
            .collect();

        let scroll_indicator = if self.scroll_x > 0 {
            format!(" ◄ scroll:{} ", self.scroll_x)
        } else {
            String::new()
        };

        let title = format!(
            " k10s — GPU Fleet ({} nodes){} ",
            self.nodes.len(),
            scroll_indicator
        );

        let mut table_state = TableState::default();
        table_state.select(Some(self.selected));

        let table = Table::new(rows, widths)
            .header(header)
            .column_spacing(1)
            .row_highlight_style(
                ratatui::style::Style::default().bg(ratatui::style::Color::DarkGray),
            )
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .title(Line::from(title)),
            );

        frame.render_stateful_widget(table, area, &mut table_state);

        if Self::total_width() > area.width {
            let mut scrollbar_state = ScrollbarState::new(Self::total_width() as usize)
                .position(self.scroll_x as usize)
                .viewport_content_length(area.width as usize);
            let scrollbar = Scrollbar::new(ScrollbarOrientation::HorizontalBottom)
                .begin_symbol(Some("◄"))
                .end_symbol(Some("►"));
            frame.render_stateful_widget(scrollbar, area, &mut scrollbar_state);
        }
    }

    fn on_enter(&mut self, _ctx: &AppContext) {
        self.sort_nodes();
    }

    fn on_leave(&mut self) {}
}
