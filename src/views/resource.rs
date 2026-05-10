use crossterm::event::{KeyCode, KeyEvent};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Block, Borders, Row, Table, TableState};
use ratatui::Frame;

use crate::action::Action;
use crate::app::AppContext;
use crate::datasource::ResourceList;
use crate::k8s::gvr::Gvr;
use crate::msg::AppMsg;
use crate::resource_schema::{self, ResourceSchema};
use crate::ui::styles;
use crate::views::View;

pub struct ResourceView {
    gvr: Gvr,
    schema: &'static ResourceSchema,
    objects_count: usize,
    cells: Vec<Vec<String>>,
    selected: usize,
    error: Option<String>,
}

impl ResourceView {
    pub fn new(gvr: Gvr) -> Self {
        let schema = resource_schema::lookup_schema(&gvr);
        Self {
            gvr,
            schema,
            objects_count: 0,
            cells: Vec::new(),
            selected: 0,
            error: None,
        }
    }

    fn apply_resource_list(&mut self, list: ResourceList) {
        self.objects_count = list.objects.len();
        self.cells = list
            .objects
            .iter()
            .map(|obj| {
                self.schema
                    .columns
                    .iter()
                    .map(|col| (col.resolver)(&obj.raw))
                    .collect()
            })
            .collect();

        if self.selected >= self.objects_count {
            self.selected = self.objects_count.saturating_sub(1);
        }
        self.error = None;
    }
}

impl View for ResourceView {
    fn update(&mut self, msg: AppMsg, _ctx: &AppContext) -> Option<Action> {
        match msg {
            AppMsg::ResourcesUpdated(list) if list.gvr == self.gvr => {
                self.apply_resource_list(list);
            }
            AppMsg::Error(e) => {
                self.error = Some(e);
            }
            _ => {}
        }
        None
    }

    fn handle_key(&mut self, key: KeyEvent, _ctx: &AppContext) -> Option<Action> {
        match key.code {
            KeyCode::Char('j') | KeyCode::Down
                if self.selected < self.objects_count.saturating_sub(1) =>
            {
                self.selected += 1;
            }
            KeyCode::Char('k') | KeyCode::Up => {
                self.selected = self.selected.saturating_sub(1);
            }
            KeyCode::Char('G') => {
                self.selected = self.objects_count.saturating_sub(1);
            }
            KeyCode::Char('g') => {
                self.selected = 0;
            }
            KeyCode::Char('?') => return Some(Action::ShowHelp),
            KeyCode::Char('q') => return Some(Action::Quit),
            KeyCode::Esc => return Some(Action::NavigateBack),
            _ => {}
        }
        None
    }

    fn render(&self, frame: &mut Frame, area: Rect, _ctx: &AppContext) {
        if let Some(ref err) = self.error {
            let block = Block::default()
                .borders(Borders::ALL)
                .title(Line::from(format!(
                    " {} (error) ",
                    self.schema.display_name
                )));
            let paragraph = ratatui::widgets::Paragraph::new(err.as_str())
                .style(styles::degraded_style())
                .block(block);
            frame.render_widget(paragraph, area);
            return;
        }

        if self.cells.is_empty() {
            let block = Block::default()
                .borders(Borders::ALL)
                .title(Line::from(format!(" {} ", self.schema.display_name)));
            let paragraph = ratatui::widgets::Paragraph::new("Loading...").block(block);
            frame.render_widget(paragraph, area);
            return;
        }

        let headers: Vec<&str> = self.schema.columns.iter().map(|c| c.header).collect();
        let header_row = Row::new(headers).style(styles::header_style());

        let widths: Vec<ratatui::layout::Constraint> =
            self.schema.columns.iter().map(|c| c.width).collect();

        let rows: Vec<Row> = self
            .cells
            .iter()
            .enumerate()
            .map(|(i, cells)| {
                let visible: Vec<&str> = cells.iter().map(|s| s.as_str()).collect();
                let row = Row::new(visible);
                if i == self.selected {
                    row.style(ratatui::style::Style::default().bg(ratatui::style::Color::DarkGray))
                } else {
                    row
                }
            })
            .collect();

        let title = format!(
            " {} [all] ({}) ",
            self.schema.display_name, self.objects_count
        );

        let mut table_state = TableState::default();
        table_state.select(Some(self.selected));

        let table = Table::new(rows, widths)
            .header(header_row)
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
    }

    fn on_enter(&mut self, _ctx: &AppContext) {}
    fn on_leave(&mut self) {}
}
