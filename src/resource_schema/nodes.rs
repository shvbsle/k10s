use ratatui::layout::Constraint;
use serde_json::Value;

use crate::action::{Action, ResourceFilter, ViewRequest};
use crate::k8s::gvr::Gvr;

use super::{resolve_age, resolve_name, ColumnDef, ResourceSchema};

pub fn schema() -> ResourceSchema {
    ResourceSchema {
        display_name: "Nodes",
        columns: vec![
            ColumnDef {
                header: "NAME",
                width: Constraint::Fill(2),
                resolver: resolve_name,
            },
            ColumnDef {
                header: "STATUS",
                width: Constraint::Length(10),
                resolver: resolve_node_status,
            },
            ColumnDef {
                header: "ROLES",
                width: Constraint::Length(14),
                resolver: resolve_roles,
            },
            ColumnDef {
                header: "VERSION",
                width: Constraint::Length(12),
                resolver: resolve_kubelet_version,
            },
            ColumnDef {
                header: "AGE",
                width: Constraint::Length(8),
                resolver: resolve_age,
            },
        ],
        sort: None,
        drill_down: Some(drill_to_pods_on_node),
    }
}

fn resolve_node_status(v: &Value) -> String {
    v.pointer("/status/conditions")
        .and_then(|v| v.as_array())
        .and_then(|conditions| {
            conditions
                .iter()
                .find(|c| c.get("type").and_then(|t| t.as_str()) == Some("Ready"))
        })
        .and_then(|ready| ready.get("status").and_then(|s| s.as_str()))
        .map(|s| if s == "True" { "Ready" } else { "NotReady" })
        .unwrap_or("Unknown")
        .to_string()
}

fn resolve_roles(v: &Value) -> String {
    v.pointer("/metadata/labels")
        .and_then(|v| v.as_object())
        .map(|labels| {
            let roles: Vec<&str> = labels
                .keys()
                .filter_map(|k| k.strip_prefix("node-role.kubernetes.io/"))
                .collect();
            if roles.is_empty() {
                "<none>".to_string()
            } else {
                roles.join(",")
            }
        })
        .unwrap_or_else(|| "<none>".to_string())
}

fn resolve_kubelet_version(v: &Value) -> String {
    v.pointer("/status/nodeInfo/kubeletVersion")
        .and_then(|v| v.as_str())
        .unwrap_or("—")
        .to_string()
}

fn drill_to_pods_on_node(v: &Value) -> Action {
    let node_name = v
        .pointer("/metadata/name")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();

    Action::PushView(ViewRequest::Resource {
        gvr: Gvr::new("", "v1", "pods"),
        namespace: None,
        filter: Some(ResourceFilter::FieldEquals {
            json_pointer: "/spec/nodeName".to_string(),
            value: node_name,
        }),
    })
}
