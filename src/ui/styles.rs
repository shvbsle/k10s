use ratatui::style::{Color, Modifier, Style};

pub fn idle_style() -> Style {
    Style::default()
        .fg(Color::Yellow)
        .add_modifier(Modifier::BOLD)
}

pub fn busy_style() -> Style {
    Style::default().fg(Color::Green)
}

pub fn saturated_style() -> Style {
    Style::default().fg(Color::Red)
}

pub fn degraded_style() -> Style {
    Style::default().fg(Color::Magenta)
}

pub fn header_style() -> Style {
    Style::default()
        .fg(Color::Cyan)
        .add_modifier(Modifier::BOLD)
}

#[allow(dead_code)]
pub fn absent_style() -> Style {
    Style::default().fg(Color::DarkGray)
}
