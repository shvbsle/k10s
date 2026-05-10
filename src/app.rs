use crossterm::event::KeyEvent;
use ratatui::layout::{Constraint, Layout};
use ratatui::Frame;

use crate::action::Action;
use crate::msg::AppMsg;
use crate::ui::header;
use crate::views::fleet::FleetView;
use crate::views::help::HelpView;
use crate::views::View;

#[derive(Debug, Clone)]
pub enum SourceStatus {
    Connected,
    Degraded(String),
    Unavailable,
}

#[derive(Debug, Clone)]
pub struct DataHealth {
    pub dcgm_status: SourceStatus,
    pub last_fetch_error: Option<String>,
}

impl Default for DataHealth {
    fn default() -> Self {
        Self {
            dcgm_status: SourceStatus::Unavailable,
            last_fetch_error: None,
        }
    }
}

pub struct ClusterInfo {
    pub context: String,
    pub k8s_version: String,
    pub k10s_version: String,
}

pub struct AppState {
    pub health: DataHealth,
}

pub struct AppContext {
    pub cluster: ClusterInfo,
    pub state: AppState,
}

pub struct App {
    view_stack: Vec<Box<dyn View>>,
    help_visible: bool,
    help_view: HelpView,
    pub ctx: AppContext,
    pub should_quit: bool,
}

impl App {
    pub fn new(cluster_context: String, k8s_version: String) -> Self {
        let fleet = FleetView::new();
        let ctx = AppContext {
            cluster: ClusterInfo {
                context: cluster_context,
                k8s_version,
                k10s_version: format!("v{}", env!("CARGO_PKG_VERSION")),
            },
            state: AppState {
                health: DataHealth::default(),
            },
        };

        let mut app = Self {
            view_stack: vec![Box::new(fleet)],
            help_visible: false,
            help_view: HelpView::new(),
            ctx,
            should_quit: false,
        };

        if let Some(view) = app.view_stack.last_mut() {
            view.on_enter(&app.ctx);
        }

        app
    }

    pub fn handle_msg(&mut self, msg: AppMsg) {
        let msg = match msg {
            AppMsg::HealthUpdate(health) => {
                self.ctx.state.health = health;
                return;
            }
            other => other,
        };

        if let Some(view) = self.view_stack.last_mut() {
            if let Some(action) = view.update(msg, &self.ctx) {
                self.dispatch(action);
            }
        }
    }

    pub fn handle_key(&mut self, key: KeyEvent) {
        if self.help_visible {
            if let Some(action) = self.help_view.handle_key(key, &self.ctx) {
                self.dispatch(action);
            }
            return;
        }

        if let Some(view) = self.view_stack.last_mut() {
            if let Some(action) = view.handle_key(key, &self.ctx) {
                self.dispatch(action);
            }
        }
    }

    fn dispatch(&mut self, action: Action) {
        match action {
            Action::Quit => self.should_quit = true,
            Action::NavigateBack => self.pop_view(),
            Action::ShowHelp => self.help_visible = true,
            Action::HideHelp => self.help_visible = false,
        }
    }

    fn pop_view(&mut self) {
        if self.view_stack.len() > 1 {
            if let Some(mut view) = self.view_stack.pop() {
                view.on_leave();
            }
            if let Some(view) = self.view_stack.last_mut() {
                view.on_enter(&self.ctx);
            }
        }
    }

    pub fn render(&self, frame: &mut Frame) {
        let area = frame.area();

        let chunks = Layout::vertical([
            Constraint::Length(header::header_height()),
            Constraint::Min(0),
        ])
        .split(area);

        header::render_header(frame, chunks[0], &self.ctx);

        if let Some(view) = self.view_stack.last() {
            view.render(frame, chunks[1], &self.ctx);
        }

        if self.help_visible {
            self.help_view.render(frame, area, &self.ctx);
        }
    }
}
