use crate::k8s::gvr::Gvr;

use super::{Command, ParseError};

pub fn parse_command(input: &str) -> Result<Command, ParseError> {
    let input = input.trim();
    if input.is_empty() {
        return Err(ParseError::EmptyInput);
    }

    let mut parts = input.split_whitespace();
    let cmd = parts.next().unwrap();

    match cmd.to_lowercase().as_str() {
        "q" | "quit" => Ok(Command::Quit),
        "help" => Ok(Command::Help),
        "ctx" => Ok(Command::ContextSwitch),
        "ns" => Ok(Command::NamespaceSwitch),
        "rs" | "resource" => parse_rs_args(parts),
        _ => Err(ParseError::UnknownCommand(cmd.to_string())),
    }
}

fn parse_rs_args<'a>(mut parts: impl Iterator<Item = &'a str>) -> Result<Command, ParseError> {
    let resource_str = parts.next().ok_or(ParseError::MissingArgument {
        command: "rs".to_string(),
        expected: "resource type (e.g., nodes/v1, pods)".to_string(),
    })?;

    let mut namespace = None;

    while let Some(flag) = parts.next() {
        match flag {
            "-n" | "in" => {
                namespace = Some(
                    parts
                        .next()
                        .ok_or(ParseError::MissingArgument {
                            command: "rs".to_string(),
                            expected: "namespace after -n".to_string(),
                        })?
                        .to_string(),
                );
            }
            "all" => {
                namespace = None;
            }
            other => {
                namespace = Some(other.to_string());
            }
        }
    }

    let gvr = Gvr::parse_user_input(resource_str).map_err(ParseError::InvalidResource)?;

    Ok(Command::ResourceShow { gvr, namespace })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_quit() {
        assert!(matches!(parse_command("q"), Ok(Command::Quit)));
        assert!(matches!(parse_command("quit"), Ok(Command::Quit)));
    }

    #[test]
    fn parse_help() {
        assert!(matches!(parse_command("help"), Ok(Command::Help)));
    }

    #[test]
    fn parse_rs_nodes() {
        let cmd = parse_command("rs nodes/v1").unwrap();
        match cmd {
            Command::ResourceShow { gvr, namespace } => {
                assert_eq!(gvr.resource, "nodes");
                assert_eq!(gvr.version, "v1");
                assert!(namespace.is_none());
            }
            _ => panic!("expected ResourceShow"),
        }
    }

    #[test]
    fn parse_rs_pods_with_namespace() {
        let cmd = parse_command("rs pods -n kube-system").unwrap();
        match cmd {
            Command::ResourceShow { gvr, namespace } => {
                assert_eq!(gvr.resource, "pods");
                assert_eq!(namespace, Some("kube-system".to_string()));
            }
            _ => panic!("expected ResourceShow"),
        }
    }

    #[test]
    fn parse_rs_no_arg() {
        assert!(matches!(
            parse_command("rs"),
            Err(ParseError::MissingArgument { .. })
        ));
    }

    #[test]
    fn parse_unknown() {
        assert!(matches!(
            parse_command("foobar"),
            Err(ParseError::UnknownCommand(_))
        ));
    }

    #[test]
    fn parse_empty() {
        assert!(matches!(parse_command(""), Err(ParseError::EmptyInput)));
    }

    #[test]
    fn parse_resource_long_form() {
        let cmd = parse_command("resource pods").unwrap();
        assert!(matches!(cmd, Command::ResourceShow { .. }));
    }

    #[test]
    fn parse_rs_with_in_namespace() {
        let cmd = parse_command("rs pods in monitoring").unwrap();
        match cmd {
            Command::ResourceShow { namespace, .. } => {
                assert_eq!(namespace, Some("monitoring".to_string()));
            }
            _ => panic!("expected ResourceShow"),
        }
    }
}
