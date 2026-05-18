use std::result;

#[derive(Debug, thiserror::Error)]
pub enum KittyError {
    #[error("gpu ({vendor}): {msg}")]
    Gpu { vendor: String, msg: String },

    #[error("io: {0}")]
    Io(#[from] std::io::Error),

    #[error("serialization: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("config: {0}")]
    Config(String),
}

pub type Result<T> = result::Result<T, KittyError>;
