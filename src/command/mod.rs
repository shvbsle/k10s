pub mod parse;

use crate::k8s::gvr::Gvr;

#[derive(Debug, Clone)]
pub enum Command {
    Quit,
    ResourceShow { gvr: Gvr, namespace: Option<String> },
    ContextSwitch,
    NamespaceSwitch,
    Help,
}

#[derive(Debug, Clone)]
pub enum ParseError {
    EmptyInput,
    UnknownCommand(String),
    InvalidResource(String),
    MissingArgument { command: String, expected: String },
}

impl std::fmt::Display for ParseError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ParseError::EmptyInput => write!(f, "empty command"),
            ParseError::UnknownCommand(cmd) => write!(f, "unknown command: {}", cmd),
            ParseError::InvalidResource(r) => write!(f, "invalid resource: {}", r),
            ParseError::MissingArgument { command, expected } => {
                write!(f, ":{} requires {}", command, expected)
            }
        }
    }
}
