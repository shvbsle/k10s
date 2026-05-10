pub mod fleet;
pub mod help;

use crossterm::event::KeyEvent;
use ratatui::layout::Rect;
use ratatui::Frame;

use crate::action::Action;
use crate::app::AppContext;
use crate::msg::AppMsg;

pub trait View {
    fn update(&mut self, msg: AppMsg, ctx: &AppContext) -> Option<Action>;
    fn handle_key(&mut self, key: KeyEvent, ctx: &AppContext) -> Option<Action>;
    fn render(&self, frame: &mut Frame, area: Rect, ctx: &AppContext);
    fn on_enter(&mut self, ctx: &AppContext);
    fn on_leave(&mut self);
}
