pub mod nodes;
pub mod pods;

use std::collections::HashMap;
use std::sync::LazyLock;

use ratatui::layout::Constraint;
use serde_json::Value;

use crate::action::Action;
use crate::k8s::gvr::Gvr;

pub type Resolver = fn(&Value) -> String;
pub type DrillDownFn = fn(&Value) -> Action;
pub type SortFn = fn(&Value, &Value) -> std::cmp::Ordering;

pub struct ColumnDef {
    pub header: &'static str,
    pub width: Constraint,
    pub resolver: Resolver,
}

pub struct ResourceSchema {
    pub display_name: &'static str,
    pub columns: Vec<ColumnDef>,
    pub sort: Option<SortFn>,
    pub drill_down: Option<DrillDownFn>,
}

static SCHEMA_REGISTRY: LazyLock<HashMap<String, ResourceSchema>> = LazyLock::new(|| {
    let mut map = HashMap::new();
    map.insert("nodes".to_string(), nodes::schema());
    map.insert("pods".to_string(), pods::schema());
    map
});

pub fn lookup_schema(gvr: &Gvr) -> &'static ResourceSchema {
    static FALLBACK: LazyLock<ResourceSchema> = LazyLock::new(|| ResourceSchema {
        display_name: "Resources",
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
                header: "AGE",
                width: Constraint::Length(8),
                resolver: resolve_age,
            },
        ],
        sort: None,
        drill_down: None,
    });

    SCHEMA_REGISTRY.get(&gvr.resource).unwrap_or(&FALLBACK)
}

pub fn resolve_name(v: &Value) -> String {
    v.pointer("/metadata/name")
        .and_then(|v| v.as_str())
        .unwrap_or("<unknown>")
        .to_string()
}

pub fn resolve_namespace(v: &Value) -> String {
    v.pointer("/metadata/namespace")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string()
}

pub fn resolve_age(_v: &Value) -> String {
    // TODO: parse creationTimestamp and compute relative age
    "—".to_string()
}
