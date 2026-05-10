use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::Frame;

use crate::app::{AppContext, SourceStatus};

const KITTEN_ART: &[&str] = &[r"/\_/\  /\_/\", r"( o.o )( o.o )", r"> Y <  > Y <"];

pub fn header_height() -> u16 {
    4
}

pub fn render_header(frame: &mut Frame, area: Rect, ctx: &AppContext) {
    let chunks = Layout::horizontal([
        Constraint::Percentage(45),
        Constraint::Percentage(25),
        Constraint::Percentage(30),
    ])
    .split(area);

    render_cluster_info(frame, chunks[0], ctx);
    render_help_hints(frame, chunks[1]);
    render_kitten_art(frame, chunks[2]);
}

fn render_cluster_info(frame: &mut Frame, area: Rect, ctx: &AppContext) {
    let green_dot = Span::styled("● ", Style::default().fg(Color::Green));
    let context_label = Span::styled("Context: ", Style::default().fg(Color::Green));
    let context_value = Span::styled(&ctx.cluster.context, Style::default().fg(Color::White));

    let k8s_label = Span::styled("  K8s:    ", Style::default().fg(Color::Green));
    let k8s_value = Span::styled(&ctx.cluster.k8s_version, Style::default().fg(Color::White));

    let k10s_label = Span::styled("  k10s:   ", Style::default().fg(Color::Green));
    let k10s_value = Span::styled(&ctx.cluster.k10s_version, Style::default().fg(Color::White));

    let health_indicator = match &ctx.state.health.dcgm_status {
        SourceStatus::Connected => Span::styled("", Style::default()),
        SourceStatus::Degraded(reason) => Span::styled(
            format!(" [DCGM: {}]", reason),
            Style::default().fg(Color::Yellow),
        ),
        SourceStatus::Unavailable => {
            Span::styled(" [DCGM: unavailable]", Style::default().fg(Color::DarkGray))
        }
    };

    let lines = vec![
        Line::from(vec![green_dot, context_label, context_value]),
        Line::from(vec![k8s_label, k8s_value]),
        Line::from(vec![k10s_label, k10s_value, health_indicator]),
    ];

    let paragraph = ratatui::widgets::Paragraph::new(lines);
    frame.render_widget(paragraph, area);
}

fn render_help_hints(frame: &mut Frame, area: Rect) {
    let lines = vec![
        Line::from(vec![
            Span::styled(
                "? ",
                Style::default()
                    .fg(Color::Yellow)
                    .add_modifier(Modifier::BOLD),
            ),
            Span::styled("help", Style::default().fg(Color::Gray)),
        ]),
        Line::from(vec![
            Span::styled(
                ": ",
                Style::default()
                    .fg(Color::Yellow)
                    .add_modifier(Modifier::BOLD),
            ),
            Span::styled("command", Style::default().fg(Color::Gray)),
        ]),
        Line::from(vec![
            Span::styled(
                "esc ",
                Style::default()
                    .fg(Color::Yellow)
                    .add_modifier(Modifier::BOLD),
            ),
            Span::styled("go back", Style::default().fg(Color::Gray)),
        ]),
    ];

    let paragraph = ratatui::widgets::Paragraph::new(lines);
    frame.render_widget(paragraph, area);
}

fn render_kitten_art(frame: &mut Frame, area: Rect) {
    let lines: Vec<Line> = KITTEN_ART
        .iter()
        .map(|line| Line::from(Span::styled(*line, Style::default().fg(Color::Magenta))))
        .collect();

    let paragraph =
        ratatui::widgets::Paragraph::new(lines).alignment(ratatui::layout::Alignment::Right);
    frame.render_widget(paragraph, area);
}
