use crate::app::DataHealth;
use crate::data::node::FleetNode;

#[derive(Debug, Clone)]
pub enum AppMsg {
    NodesUpdated(Vec<FleetNode>),
    HealthUpdate(DataHealth),
    Error(String),
}
