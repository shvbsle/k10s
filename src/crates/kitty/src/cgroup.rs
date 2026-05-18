use tokio::fs;

#[derive(Debug, Clone)]
pub struct CgroupInfo {
    pub pod_uid: String,
    pub container_id: String,
}

pub async fn resolve(pid: u32) -> Option<CgroupInfo> {
    let content = fs::read_to_string(format!("/proc/{pid}/cgroup"))
        .await
        .ok()?;
    parse_cgroup(&content)
}

fn parse_cgroup(content: &str) -> Option<CgroupInfo> {
    for line in content.lines() {
        // cgroup lines are formatted as: hierarchy-ID:controller-list:cgroup-path
        let path = line.splitn(3, ':').nth(2)?;
        if let Some(info) = extract_from_path(path) {
            return Some(info);
        }
    }
    None
}

fn extract_from_path(path: &str) -> Option<CgroupInfo> {
    let segments: Vec<&str> = path.split('/').collect();

    let mut pod_uid: Option<String> = None;
    let mut container_id: Option<String> = None;

    for (i, seg) in segments.iter().enumerate() {
        // v1 cgroupfs: "pod<uid>" segment
        if let Some(uid) = seg.strip_prefix("pod") {
            if !uid.is_empty() {
                pod_uid = Some(normalize_uid(uid));
                // container ID is the next segment
                if let Some(cid) = segments.get(i + 1) {
                    container_id = Some(extract_container_id(cid));
                }
                break;
            }
        }

        // v1/v2 systemd: "kubepods-burstable-pod<uid>.slice" or "kubepods-besteffort-pod<uid>.slice"
        if seg.contains("-pod") && seg.ends_with(".slice") {
            if let Some(uid_part) = seg.split("-pod").nth(1) {
                if let Some(uid) = uid_part.strip_suffix(".slice") {
                    pod_uid = Some(normalize_uid(uid));
                    // container ID is the next segment (the .scope one)
                    if let Some(cid) = segments.get(i + 1) {
                        container_id = Some(extract_container_id(cid));
                    }
                    break;
                }
            }
        }
    }

    Some(CgroupInfo {
        pod_uid: pod_uid?,
        container_id: container_id.unwrap_or_default(),
    })
}

fn normalize_uid(uid: &str) -> String {
    uid.replace('_', "-")
}

fn extract_container_id(segment: &str) -> String {
    // Formats:
    //   cri-containerd-<id>.scope
    //   docker-<id>.scope
    //   just the raw container ID hash
    let s = segment.strip_suffix(".scope").unwrap_or(segment);
    if let Some(id) = s.strip_prefix("cri-containerd-") {
        return id.to_string();
    }
    if let Some(id) = s.strip_prefix("docker-") {
        return id.to_string();
    }
    s.to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_v1_cgroupfs() {
        let content =
            "12:memory:/kubepods/burstable/pod1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d/abc123def456\n";
        let info = parse_cgroup(content).unwrap();
        assert_eq!(info.pod_uid, "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d");
        assert_eq!(info.container_id, "abc123def456");
    }

    #[test]
    fn test_v1_systemd() {
        let content = "1:name=systemd:/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod1a2b3c4d_5e6f_7a8b_9c0d_1e2f3a4b5c6d.slice/cri-containerd-abc123def456.scope\n";
        let info = parse_cgroup(content).unwrap();
        assert_eq!(info.pod_uid, "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d");
        assert_eq!(info.container_id, "abc123def456");
    }

    #[test]
    fn test_v2() {
        let content = "0::/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-podaabbccdd_eeff_0011_2233_445566778899.slice/cri-containerd-deadbeef1234.scope\n";
        let info = parse_cgroup(content).unwrap();
        assert_eq!(info.pod_uid, "aabbccdd-eeff-0011-2233-445566778899");
        assert_eq!(info.container_id, "deadbeef1234");
    }

    #[test]
    fn test_non_k8s_process() {
        let content = "0::/user.slice/user-1000.slice/session-1.scope\n";
        assert!(parse_cgroup(content).is_none());
    }
}
