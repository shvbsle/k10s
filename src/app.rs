use crossterm::event::{KeyCode, KeyEvent};
use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::style::{Color, Style};
use ratatui::text::Span;
use ratatui::widgets::Paragraph;
use ratatui::Frame;

use crate::action::{Action, ViewRequest};
use crate::command::parse::parse_command;
use crate::command::Command;
use crate::k8s::gvr::Gvr;
use crate::msg::AppMsg;
use crate::ui::header;
use crate::views::fleet::FleetView;
use crate::views::help::HelpView;
use crate::views::resource::ResourceView;
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

#[derive(Debug, Clone, PartialEq)]
pub enum InputMode {
    Normal,
    Command { buffer: String, cursor: usize },
}

/// Commands sent to the background data task.
#[derive(Debug, Clone)]
pub enum DataCommand {
    FetchResources { gvr: Gvr, namespace: Option<String> },
}

pub struct App {
    view_stack: Vec<Box<dyn View>>,
    help_visible: bool,
    help_view: HelpView,
    pub ctx: AppContext,
    pub should_quit: bool,
    input_mode: InputMode,
    status_message: Option<String>,
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
            input_mode: InputMode::Normal,
            status_message: None,
        };

        if let Some(view) = app.view_stack.last_mut() {
            view.on_enter(&app.ctx);
        }

        app
    }

    pub fn handle_msg(&mut self, msg: AppMsg) -> Option<DataCommand> {
        let msg = match msg {
            AppMsg::HealthUpdate(health) => {
                self.ctx.state.health = health;
                return None;
            }
            other => other,
        };

        if let Some(view) = self.view_stack.last_mut() {
            if let Some(action) = view.update(msg, &self.ctx) {
                return self.dispatch(action);
            }
        }
        None
    }

    pub fn handle_key(&mut self, key: KeyEvent) -> Option<DataCommand> {
        match &self.input_mode {
            InputMode::Normal => self.handle_key_normal(key),
            InputMode::Command { .. } => self.handle_key_command(key),
        }
    }

    fn handle_key_normal(&mut self, key: KeyEvent) -> Option<DataCommand> {
        if key.code == KeyCode::Char(':') {
            self.input_mode = InputMode::Command {
                buffer: String::new(),
                cursor: 0,
            };
            self.status_message = None;
            return None;
        }

        if self.help_visible {
            if let Some(action) = self.help_view.handle_key(key, &self.ctx) {
                return self.dispatch(action);
            }
            return None;
        }

        if let Some(view) = self.view_stack.last_mut() {
            if let Some(action) = view.handle_key(key, &self.ctx) {
                return self.dispatch(action);
            }
        }
        None
    }

    fn handle_key_command(&mut self, key: KeyEvent) -> Option<DataCommand> {
        match key.code {
            KeyCode::Esc => {
                self.input_mode = InputMode::Normal;
                self.status_message = None;
            }
            KeyCode::Enter => {
                let buffer = if let InputMode::Command { ref buffer, .. } = self.input_mode {
                    buffer.clone()
                } else {
                    return None;
                };
                self.input_mode = InputMode::Normal;
                return self.execute_command(&buffer);
            }
            KeyCode::Backspace => {
                if let InputMode::Command {
                    ref mut buffer,
                    ref mut cursor,
                } = self.input_mode
                {
                    if *cursor > 0 {
                        buffer.remove(*cursor - 1);
                        *cursor -= 1;
                    }
                    if buffer.is_empty() {
                        self.input_mode = InputMode::Normal;
                    }
                }
            }
            KeyCode::Char(c) => {
                if let InputMode::Command {
                    ref mut buffer,
                    ref mut cursor,
                } = self.input_mode
                {
                    buffer.insert(*cursor, c);
                    *cursor += 1;
                }
            }
            _ => {}
        }
        None
    }

    fn execute_command(&mut self, input: &str) -> Option<DataCommand> {
        match parse_command(input) {
            Ok(Command::Quit) => {
                self.should_quit = true;
                None
            }
            Ok(Command::Help) => {
                self.help_visible = true;
                None
            }
            Ok(Command::ResourceShow { gvr, namespace }) => {
                let view = ResourceView::new(gvr.clone());
                self.push_view(Box::new(view));
                Some(DataCommand::FetchResources { gvr, namespace })
            }
            Ok(Command::ContextSwitch) | Ok(Command::NamespaceSwitch) => {
                self.status_message = Some("not yet implemented".to_string());
                None
            }
            Err(e) => {
                self.status_message = Some(format!("{}", e));
                None
            }
        }
    }

    fn dispatch(&mut self, action: Action) -> Option<DataCommand> {
        match action {
            Action::Quit => {
                self.should_quit = true;
                None
            }
            Action::NavigateBack => {
                self.pop_view();
                None
            }
            Action::ShowHelp => {
                self.help_visible = true;
                None
            }
            Action::HideHelp => {
                self.help_visible = false;
                None
            }
            Action::PushView(ViewRequest::Resource {
                gvr,
                namespace,
                filter: _,
            }) => {
                let view = ResourceView::new(gvr.clone());
                self.push_view(Box::new(view));
                Some(DataCommand::FetchResources { gvr, namespace })
            }
            Action::ShowError(msg) => {
                self.status_message = Some(msg);
                None
            }
        }
    }

    fn push_view(&mut self, view: Box<dyn View>) {
        if let Some(current) = self.view_stack.last_mut() {
            current.on_leave();
        }
        self.view_stack.push(view);
        if let Some(new_view) = self.view_stack.last_mut() {
            new_view.on_enter(&self.ctx);
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

        let has_cmd_bar = self.input_mode != InputMode::Normal || self.status_message.is_some();

        let chunks = Layout::vertical([
            Constraint::Length(header::header_height()),
            Constraint::Min(0),
            Constraint::Length(if has_cmd_bar { 1 } else { 0 }),
        ])
        .split(area);

        header::render_header(frame, chunks[0], &self.ctx);

        if let Some(view) = self.view_stack.last() {
            view.render(frame, chunks[1], &self.ctx);
        }

        if self.help_visible {
            self.help_view.render(frame, area, &self.ctx);
        }

        if has_cmd_bar {
            self.render_command_bar(frame, chunks[2]);
        }
    }

    fn render_command_bar(&self, frame: &mut Frame, area: Rect) {
        let content = match &self.input_mode {
            InputMode::Command { buffer, .. } => {
                Span::styled(format!(":{}", buffer), Style::default().fg(Color::White))
            }
            InputMode::Normal => {
                if let Some(ref msg) = self.status_message {
                    Span::styled(msg.as_str(), Style::default().fg(Color::Red))
                } else {
                    Span::raw("")
                }
            }
        };

        let paragraph = Paragraph::new(content);
        frame.render_widget(paragraph, area);
    }
}
