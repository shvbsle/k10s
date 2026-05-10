use std::collections::HashMap;

use anyhow::Result;
use async_trait::async_trait;
use serde_json::Value;

use crate::app::DataHealth;
use crate::data::node::FleetNode;
use crate::k8s::cluster::ClusterDataSource;
use crate::k8s::gvr::Gvr;

use super::{DataSource, ResourceList, ResourceObject};

pub struct LiveSource {
    cluster: ClusterDataSource,
}

impl LiveSource {
    pub async fn new() -> Result<Self> {
        let cluster = ClusterDataSource::new().await?;
        Ok(Self { cluster })
    }
}

#[async_trait]
impl DataSource for LiveSource {
    async fn fetch_fleet(&self) -> Result<(Vec<FleetNode>, DataHealth)> {
        self.cluster.fetch_fleet().await
    }

    async fn list_resources(&self, gvr: &Gvr, namespace: Option<&str>) -> Result<ResourceList> {
        use kube::api::{DynamicObject, ListParams};
        use kube::discovery::ApiResource;
        use kube::{Api, Client};

        let config = kube::Config::infer().await?;
        let client = Client::try_from(config)?;

        let ar = ApiResource {
            group: gvr.group.clone(),
            version: gvr.version.clone(),
            api_version: if gvr.group.is_empty() {
                gvr.version.clone()
            } else {
                format!("{}/{}", gvr.group, gvr.version)
            },
            kind: String::new(),
            plural: gvr.resource.clone(),
        };

        let api: Api<DynamicObject> = match namespace {
            Some(ns) => Api::namespaced_with(client, ns, &ar),
            None => Api::all_with(client, &ar),
        };

        let list = api.list(&ListParams::default()).await?;

        let objects = list
            .items
            .into_iter()
            .map(|obj| {
                let raw: Value = serde_json::to_value(&obj).unwrap_or_default();
                ResourceObject {
                    uid: obj.metadata.uid.unwrap_or_default(),
                    name: obj.metadata.name.unwrap_or_default(),
                    namespace: obj.metadata.namespace,
                    raw,
                }
            })
            .collect();

        Ok(ResourceList {
            gvr: gvr.clone(),
            objects,
        })
    }

    fn context_name(&self) -> &str {
        self.cluster.context_name()
    }

    async fn server_version(&self) -> Result<String> {
        self.cluster.server_version().await
    }

    async fn discover_resources(&self) -> Result<HashMap<String, Gvr>> {
        let config = kube::Config::infer().await?;
        let client = Client::try_from(config)?;
        let discovery = kube::discovery::Discovery::new(client).run().await?;

        let mut map = HashMap::new();
        for group in discovery.groups() {
            for (ar, _caps) in group.recommended_resources() {
                map.insert(
                    ar.plural.clone(),
                    Gvr::new(&ar.group, &ar.version, &ar.plural),
                );
            }
        }
        Ok(map)
    }
}

use kube::Client;
