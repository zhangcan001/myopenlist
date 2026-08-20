use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct OpenListCoreConfig {
    pub port: u16,
    pub data_dir: String,
    pub binary_path: Option<String>,
    pub auto_launch: bool,
    pub ssl_enabled: bool,
}

impl OpenListCoreConfig {
    pub fn new() -> Self {
        Self {
            port: 5244,
            data_dir: "".to_string(),
            binary_path: None,
            auto_launch: false,
            ssl_enabled: false,
        }
    }
}
