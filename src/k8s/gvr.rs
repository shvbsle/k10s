/// GroupVersionResource identifies a Kubernetes API resource type.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct Gvr {
    pub group: String,
    pub version: String,
    pub resource: String,
}

impl Gvr {
    pub fn new(group: &str, version: &str, resource: &str) -> Self {
        Self {
            group: group.to_string(),
            version: version.to_string(),
            resource: resource.to_string(),
        }
    }

    /// Parse user input like "pods", "nodes/v1", "deployments.apps/v1"
    pub fn parse_user_input(input: &str) -> Result<Self, String> {
        let input = input.trim();
        if input.is_empty() {
            return Err("empty resource string".to_string());
        }

        let (resource_part, version) = match input.split_once('/') {
            Some((r, v)) => (r, v.to_string()),
            None => (input, "v1".to_string()),
        };

        let (resource, group) = match resource_part.split_once('.') {
            Some((r, g)) => (r.to_string(), g.to_string()),
            None => (resource_part.to_string(), String::new()),
        };

        if resource.is_empty() {
            return Err("resource name cannot be empty".to_string());
        }

        if version.is_empty() {
            return Err("version cannot be empty".to_string());
        }

        Ok(Self {
            group,
            version,
            resource,
        })
    }

    pub fn display_short(&self) -> String {
        if self.group.is_empty() {
            format!("{}/{}", self.resource, self.version)
        } else {
            format!("{}.{}/{}", self.resource, self.group, self.version)
        }
    }
}

impl std::fmt::Display for Gvr {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.display_short())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_simple_resource() {
        let gvr = Gvr::parse_user_input("pods").unwrap();
        assert_eq!(gvr.group, "");
        assert_eq!(gvr.version, "v1");
        assert_eq!(gvr.resource, "pods");
    }

    #[test]
    fn parse_resource_with_version() {
        let gvr = Gvr::parse_user_input("nodes/v1").unwrap();
        assert_eq!(gvr.group, "");
        assert_eq!(gvr.version, "v1");
        assert_eq!(gvr.resource, "nodes");
    }

    #[test]
    fn parse_grouped_resource() {
        let gvr = Gvr::parse_user_input("deployments.apps/v1").unwrap();
        assert_eq!(gvr.group, "apps");
        assert_eq!(gvr.version, "v1");
        assert_eq!(gvr.resource, "deployments");
    }

    #[test]
    fn parse_grouped_resource_no_version() {
        let gvr = Gvr::parse_user_input("pytorchjobs.kubeflow.org").unwrap();
        assert_eq!(gvr.group, "kubeflow.org");
        assert_eq!(gvr.version, "v1");
        assert_eq!(gvr.resource, "pytorchjobs");
    }

    #[test]
    fn parse_empty_fails() {
        assert!(Gvr::parse_user_input("").is_err());
    }

    #[test]
    fn parse_slash_only_fails() {
        assert!(Gvr::parse_user_input("/v1").is_err());
    }

    #[test]
    fn display_short_core() {
        let gvr = Gvr::new("", "v1", "pods");
        assert_eq!(gvr.display_short(), "pods/v1");
    }

    #[test]
    fn display_short_grouped() {
        let gvr = Gvr::new("apps", "v1", "deployments");
        assert_eq!(gvr.display_short(), "deployments.apps/v1");
    }
}
