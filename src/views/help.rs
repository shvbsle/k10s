use crossterm::event::{KeyCode, KeyEvent};
use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Clear, Paragraph};
use ratatui::Frame;

use crate::action::Action;
use crate::app::AppContext;
use crate::msg::AppMsg;
use crate::views::View;

#[derive(Default)]
pub struct HelpView;

impl HelpView {
    pub fn new() -> Self {
        Self
    }
}

impl View for HelpView {
    fn update(&mut self, _msg: AppMsg, _ctx: &AppContext) -> Option<Action> {
        None
    }

    fn handle_key(&mut self, key: KeyEvent, _ctx: &AppContext) -> Option<Action> {
        match key.code {
            KeyCode::Esc | KeyCode::Char('?') | KeyCode::Char('q') => Some(Action::HideHelp),
            _ => None,
        }
    }

    fn render(&self, frame: &mut Frame, area: Rect, _ctx: &AppContext) {
        let popup_area = centered_rect(60, 70, area);
        frame.render_widget(Clear, popup_area);

        let bindings = [
            ("j / ↓", "Move down"),
            ("k / ↑", "Move up"),
            ("h / ←", "Scroll left"),
            ("l / →", "Scroll right"),
            ("Enter", "Drill down"),
            ("Esc", "Go back / Close"),
            ("?", "Toggle help"),
            ("q", "Quit"),
        ];

        let lines: Vec<Line> = bindings
            .iter()
            .map(|(key, desc)| {
                Line::from(vec![
                    Span::styled(
                        format!("  {:>10}  ", key),
                        Style::default()
                            .fg(Color::Cyan)
                            .add_modifier(Modifier::BOLD),
                    ),
                    Span::raw(*desc),
                ])
            })
            .collect();

        let help = Paragraph::new(lines).block(
            Block::default()
                .borders(Borders::ALL)
                .title(Line::from(" Help — Key Bindings ")),
        );

        frame.render_widget(help, popup_area);
    }

    fn on_enter(&mut self, _ctx: &AppContext) {}
    fn on_leave(&mut self) {}
}

fn centered_rect(percent_x: u16, percent_y: u16, area: Rect) -> Rect {
    let vertical = Layout::vertical([
        Constraint::Percentage((100 - percent_y) / 2),
        Constraint::Percentage(percent_y),
        Constraint::Percentage((100 - percent_y) / 2),
    ])
    .split(area);

    Layout::horizontal([
        Constraint::Percentage((100 - percent_x) / 2),
        Constraint::Percentage(percent_x),
        Constraint::Percentage((100 - percent_x) / 2),
    ])
    .split(vertical[1])[1]
}
