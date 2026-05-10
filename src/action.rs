use crate::k8s::gvr::Gvr;

#[derive(Debug, Clone)]
pub enum Action {
    Quit,
    NavigateBack,
    ShowHelp,
    HideHelp,
    PushView(ViewRequest),
    ShowError(String),
}

#[derive(Debug, Clone)]
pub enum ViewRequest {
    Resource {
        gvr: Gvr,
        namespace: Option<String>,
        filter: Option<ResourceFilter>,
    },
}

#[derive(Debug, Clone)]
pub enum ResourceFilter {
    FieldEquals { json_pointer: String, value: String },
}
