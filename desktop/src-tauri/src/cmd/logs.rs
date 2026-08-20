use std::path::PathBuf;

use tauri::State;

use crate::object::structs::AppState;
use crate::utils::path::{get_app_logs_dir, get_default_openlist_data_dir};

#[tauri::command]
pub async fn get_logs(
    source: Option<String>,
    state: State<'_, AppState>,
) -> Result<Vec<String>, String> {
    let data_dir = state
        .get_settings()
        .map(|s| s.openlist.data_dir)
        .filter(|d| !d.is_empty());

    let paths = resolve_log_paths(source.as_deref(), data_dir.as_deref())?;
    let mut logs = Vec::new();

    for path in paths {
        if path.exists() {
            let content = std::fs::read_to_string(&path)
                .map_err(|e| format!("Failed to read {path:?}: {e}"))?;
            logs.extend(content.lines().map(str::to_string));
        } else {
            log::info!("Log file does not exist: {path:?}");
        }
    }
    Ok(logs)
}

#[tauri::command]
pub async fn clear_logs(
    source: Option<String>,
    state: State<'_, AppState>,
) -> Result<bool, String> {
    let data_dir = state
        .get_settings()
        .map(|s| s.openlist.data_dir)
        .filter(|d| !d.is_empty());

    let paths = resolve_log_paths(source.as_deref(), data_dir.as_deref())?;
    let mut cleared_count = 0;

    for path in paths {
        if path.exists() {
            std::fs::write(&path, "").map_err(|e| format!("Failed to clear {path:?}: {e}"))?;
            cleared_count += 1;
        }
    }

    if cleared_count == 0 {
        Err("No log files found to clear".into())
    } else {
        Ok(true)
    }
}

fn resolve_log_paths(source: Option<&str>, data_dir: Option<&str>) -> Result<Vec<PathBuf>, String> {
    let logs_dir = get_app_logs_dir()?;

    let openlist_log_base = if let Some(dir) = data_dir.filter(|d| !d.is_empty()) {
        PathBuf::from(dir)
    } else {
        get_default_openlist_data_dir()
            .map_err(|e| format!("Failed to get default data directory: {e}"))?
    };

    let mut paths = Vec::new();
    match source {
        Some("openlist") => paths.push(openlist_log_base.join("log/log.log")),
        Some("app") => paths.push(logs_dir.join("app.log")),
        Some("rclone") => paths.push(logs_dir.join("process_rclone.log")),
        Some("all") => {
            paths.push(openlist_log_base.join("log/log.log"));
            paths.push(logs_dir.join("app.log"));
            paths.push(logs_dir.join("process_rclone.log"));
        }
        _ => return Err("Invalid log source".into()),
    }
    Ok(paths)
}
