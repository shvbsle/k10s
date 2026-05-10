use ratatui::layout::Constraint;
use serde_json::Value;

use super::{resolve_age, resolve_name, resolve_namespace, ColumnDef, ResourceSchema};

pub fn schema() -> ResourceSchema {
    ResourceSchema {
        display_name: "Pods",
        columns: vec![
            ColumnDef {
                header: "NAME",
                width: Constraint::Fill(2),
                resolver: resolve_name,
            },
            ColumnDef {
                header: "NAMESPACE",
                width: Constraint::Fill(1),
                resolver: resolve_namespace,
            },
            ColumnDef {
                header: "STATUS",
                width: Constraint::Length(12),
                resolver: resolve_pod_status,
            },
            ColumnDef {
                header: "RESTARTS",
                width: Constraint::Length(9),
                resolver: resolve_restarts,
            },
            ColumnDef {
                header: "NODE",
                width: Constraint::Fill(1),
                resolver: resolve_node,
            },
            ColumnDef {
                header: "AGE",
                width: Constraint::Length(8),
                resolver: resolve_age,
            },
        ],
        sort: None,
        drill_down: None,
    }
}

fn resolve_pod_status(v: &Value) -> String {
    v.pointer("/status/phase")
        .and_then(|v| v.as_str())
        .unwrap_or("Unknown")
        .to_string()
}

fn resolve_restarts(v: &Value) -> String {
    v.pointer("/status/containerStatuses")
        .and_then(|v| v.as_array())
        .map(|statuses| {
            statuses
                .iter()
                .filter_map(|s| s.get("restartCount").and_then(|c| c.as_u64()))
                .sum::<u64>()
        })
        .map(|count| count.to_string())
        .unwrap_or_else(|| "0".to_string())
}

fn resolve_node(v: &Value) -> String {
    v.pointer("/spec/nodeName")
        .and_then(|v| v.as_str())
        .unwrap_or("—")
        .to_string()
}
