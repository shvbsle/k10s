use std::env;
use std::time::Duration;

use crate::error::{KittyError, Result};

#[derive(Debug, Clone)]
pub struct Config {
    pub collectors: CollectorsConfig,
    pub network_interfaces: Vec<String>,
    pub prometheus_port: u16,
    pub node_name: String,
}

#[derive(Debug, Clone)]
pub struct CollectorsConfig {
    pub gpu: CollectorConfig,
    pub network: CollectorConfig,
    pub process: CollectorConfig,
    pub system: CollectorConfig,
}

#[derive(Debug, Clone)]
pub struct CollectorConfig {
    pub enabled: bool,
    pub interval: Duration,
}

impl Config {
    pub fn from_env() -> Result<Self> {
        let node_name = env::var("K10S_NODE_NAME")
            .or_else(|_| env::var("HOSTNAME"))
            .unwrap_or_else(|_| "unknown".into());

        Ok(Self {
            collectors: CollectorsConfig::from_env()?,
            network_interfaces: parse_csv("KITTY_NETWORK_INTERFACES"),
            prometheus_port: parse_u16("K10S_PROMETHEUS_PORT", 9100)?,
            node_name,
        })
    }
}

impl CollectorsConfig {
    fn from_env() -> Result<Self> {
        Ok(Self {
            gpu: CollectorConfig::from_env("KITTY_GPU", true, 1000)?,
            network: CollectorConfig::from_env("KITTY_NETWORK", true, 500)?,
            process: CollectorConfig::from_env("KITTY_PROCESS", true, 2000)?,
            system: CollectorConfig::from_env("KITTY_SYSTEM", true, 5000)?,
        })
    }
}

impl CollectorConfig {
    fn from_env(prefix: &str, default_enabled: bool, default_interval_ms: u64) -> Result<Self> {
        let enabled = parse_bool(&format!("{prefix}_ENABLED"), default_enabled)?;
        let interval_ms = parse_u64(&format!("{prefix}_INTERVAL_MS"), default_interval_ms)?;

        if interval_ms == 0 {
            return Err(KittyError::Config(format!(
                "{prefix}_INTERVAL_MS must be > 0"
            )));
        }

        Ok(Self {
            enabled,
            interval: Duration::from_millis(interval_ms),
        })
    }
}

fn parse_bool(key: &str, default: bool) -> Result<bool> {
    match env::var(key) {
        Ok(val) => match val.to_lowercase().as_str() {
            "true" | "1" | "yes" => Ok(true),
            "false" | "0" | "no" => Ok(false),
            _ => Err(KittyError::Config(format!(
                "{key}={val} is not a valid boolean"
            ))),
        },
        Err(env::VarError::NotPresent) => Ok(default),
        Err(e) => Err(KittyError::Config(format!("{key}: {e}"))),
    }
}

fn parse_u64(key: &str, default: u64) -> Result<u64> {
    match env::var(key) {
        Ok(val) => val
            .parse::<u64>()
            .map_err(|e| KittyError::Config(format!("{key}={val} is not a valid u64: {e}"))),
        Err(env::VarError::NotPresent) => Ok(default),
        Err(e) => Err(KittyError::Config(format!("{key}: {e}"))),
    }
}

fn parse_u16(key: &str, default: u16) -> Result<u16> {
    match env::var(key) {
        Ok(val) => val
            .parse::<u16>()
            .map_err(|e| KittyError::Config(format!("{key}={val} is not a valid port: {e}"))),
        Err(env::VarError::NotPresent) => Ok(default),
        Err(e) => Err(KittyError::Config(format!("{key}: {e}"))),
    }
}

fn parse_csv(key: &str) -> Vec<String> {
    match env::var(key) {
        Ok(val) if !val.is_empty() => val.split(',').map(|s| s.trim().to_string()).collect(),
        _ => Vec::new(),
    }
}
