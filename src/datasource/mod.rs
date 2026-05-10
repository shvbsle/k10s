pub mod live;
pub mod mock;

use std::collections::HashMap;

use anyhow::Result;
use async_trait::async_trait;
use serde_json::Value;

use crate::app::DataHealth;
use crate::data::node::FleetNode;
use crate::k8s::gvr::Gvr;

#[derive(Debug, Clone)]
pub struct ResourceObject {
    pub uid: String,
    pub name: String,
    pub namespace: Option<String>,
    pub raw: Value,
}

#[derive(Debug, Clone)]
pub struct ResourceList {
    pub gvr: Gvr,
    pub objects: Vec<ResourceObject>,
}

#[async_trait]
pub trait DataSource: Send + Sync + 'static {
    async fn fetch_fleet(&self) -> Result<(Vec<FleetNode>, DataHealth)>;

    async fn list_resources(&self, gvr: &Gvr, namespace: Option<&str>) -> Result<ResourceList>;

    fn context_name(&self) -> &str;

    async fn server_version(&self) -> Result<String>;

    async fn discover_resources(&self) -> Result<HashMap<String, Gvr>>;
}
