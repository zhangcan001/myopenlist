use std::sync::Arc;

use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use tauri::AppHandle;

use crate::cmd::os_operate::VersionCache;
use crate::conf::config::MergedSettings;

#[derive(Debug, Serialize, Clone)]
pub struct ServiceStatus {
    pub running: bool,
    pub pid: Option<u32>,
    pub port: Option<u16>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct RcloneMountInfo {
    pub name: String,
    pub process_id: String,
    pub remote_path: String,
    pub mount_point: String,
    pub status: String,
    pub error_msg: Option<String>,
}

pub struct AppState {
    pub app_settings: Arc<RwLock<Option<MergedSettings>>>,
    pub app_handle: Arc<RwLock<Option<AppHandle>>>,
    pub version_cache: Arc<RwLock<Option<VersionCache>>>,
}
