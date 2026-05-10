use std::collections::HashMap;

use crate::app::DataHealth;
use crate::data::node::FleetNode;
use crate::datasource::ResourceList;
use crate::k8s::gvr::Gvr;

#[derive(Debug, Clone)]
pub enum AppMsg {
    NodesUpdated(Vec<FleetNode>),
    HealthUpdate(DataHealth),
    ResourcesUpdated(ResourceList),
    DiscoveryComplete(HashMap<String, Gvr>),
    Error(String),
}
