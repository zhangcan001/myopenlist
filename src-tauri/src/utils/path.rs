use std::path::{Path, PathBuf};
use std::{env, fs};

use dunce::canonicalize;
use tauri::State;

use crate::object::structs::AppState;

pub static APP_ID: &str = "io.github.openlistteam.openlist.desktop";

fn normalize_path(path: &Path) -> Result<PathBuf, String> {
    canonicalize(path).map_err(|e| format!("Failed to canonicalize path: {e}"))
}

pub fn get_app_dir() -> Result<PathBuf, String> {
    let app_dir = env::current_exe()
        .map_err(|e| format!("Failed to get current exe path: {e}"))?
        .parent()
        .ok_or("Failed to get parent directory")?
        .to_path_buf();
    if !app_dir.exists() {
        return Err(format!("Application directory does not exist: {app_dir:?}"));
    }

    Ok(app_dir)
}

pub fn get_user_data_dir() -> Result<PathBuf, String> {
    let data_dir = {
        #[cfg(target_os = "macos")]
        {
            let home = env::var("HOME").map_err(|_| "Failed to get HOME environment variable")?;
            PathBuf::from(home)
                .join("Library")
                .join("Application Support")
                .join("OpenList Desktop")
        }

        #[cfg(target_os = "linux")]
        {
            let home = env::var("HOME").map_err(|_| "Failed to get HOME environment variable")?;
            PathBuf::from(home)
                .join(".local")
                .join("share")
                .join("OpenList Desktop")
        }

        #[cfg(target_os = "windows")]
        {
            let appdata =
                env::var("APPDATA").map_err(|_| "Failed to get APPDATA environment variable")?;
            PathBuf::from(appdata).join("OpenList Desktop")
        }
    };

    fs::create_dir_all(&data_dir).map_err(|e| format!("Failed to create data directory: {e}"))?;

    normalize_path(&data_dir)
}

fn get_binary_path(binary: &str, service_name: &str) -> Result<PathBuf, String> {
    let mut name = binary.to_string();
    if cfg!(target_os = "windows") {
        name.push_str(".exe");
    }

    let path = get_app_dir()?.join(&name);
    if !path.exists() {
        return Err(format!(
            "{service_name} service binary not found at: {path:?}"
        ));
    }
    Ok(path)
}

pub fn get_openlist_binary_path() -> Result<PathBuf, String> {
    get_binary_path("openlist", "OpenList")
}

pub fn get_openlist_binary_path_with_custom(state: State<'_, AppState>) -> Result<PathBuf, String> {
    let settings = state
        .app_settings
        .read()
        .clone()
        .ok_or("Failed to read app settings")?;
    let custom_path = settings.openlist.binary_path;
    if let Some(path) = custom_path.filter(|p| !p.is_empty()) {
        let custom = PathBuf::from(path);
        if custom.exists() {
            return Ok(custom);
        }
        return Err(format!("Custom OpenList binary not found at: {custom:?}"));
    }
    get_openlist_binary_path()
}

pub fn get_rclone_binary_path() -> Result<PathBuf, String> {
    #[cfg(not(target_os = "linux"))]
    {
        get_binary_path("rclone", "Rclone")
    }

    #[cfg(target_os = "linux")]
    {
        use std::process::Command;
        if let Ok(output) = Command::new("which").arg("rclone").output()
            && output.status.success()
        {
            let path_str = String::from_utf8_lossy(&output.stdout).trim().to_string();
            if !path_str.is_empty() {
                return Ok(PathBuf::from(path_str));
            }
        }
        Err(
            "Rclone not found. Please install it via your package manager (e.g., apt install \
             rclone)"
                .to_string(),
        )
    }
}

pub fn get_rclone_binary_path_with_custom(state: State<'_, AppState>) -> Result<PathBuf, String> {
    let settings = state
        .app_settings
        .read()
        .clone()
        .ok_or("Failed to read app settings")?;
    let custom_path = settings.rclone.binary_path;
    if let Some(path) = custom_path.filter(|p| !p.is_empty()) {
        let custom = PathBuf::from(path);
        if custom.exists() {
            return Ok(custom);
        }
        return Err(format!("Custom Rclone binary not found at: {custom:?}"));
    }
    get_rclone_binary_path()
}

pub fn get_app_config_dir() -> Result<PathBuf, String> {
    get_user_data_dir()
}

pub fn app_config_file_path() -> Result<PathBuf, String> {
    Ok(get_app_config_dir()?.join("settings.json"))
}

pub fn get_app_logs_dir() -> Result<PathBuf, String> {
    let logs_dir = {
        #[cfg(target_os = "macos")]
        {
            let home = env::var("HOME").map_err(|_| "Failed to get HOME environment variable")?;
            PathBuf::from(home)
                .join("Library")
                .join("Logs")
                .join("OpenList Desktop")
        }

        #[cfg(not(target_os = "macos"))]
        {
            get_user_data_dir()?.join("logs")
        }
    };

    fs::create_dir_all(&logs_dir).map_err(|e| format!("Failed to create logs directory: {e}"))?;
    normalize_path(&logs_dir)
}

pub fn get_rclone_config_path() -> Result<PathBuf, String> {
    let rclone_config_path = (get_user_data_dir()?).join("rclone.conf");
    let _ = fs::create_dir_all(
        rclone_config_path
            .parent()
            .ok_or("Failed to get rclone config parent directory")?,
    );
    if !rclone_config_path.exists() {
        fs::File::create(&rclone_config_path)
            .map_err(|e| format!("Failed to create rclone config file: {e}"))?;
        fs::write(&rclone_config_path, "")
            .map_err(|e| format!("Failed to write to rclone config file: {e}"))?;
    }
    Ok(rclone_config_path)
}

pub fn get_rclone_config_path_with_custom(state: State<'_, AppState>) -> Result<PathBuf, String> {
    let settings = state
        .app_settings
        .read()
        .clone()
        .ok_or("Failed to read app settings")?;
    let custom_path = settings.rclone.rclone_conf_path;
    if let Some(path) = custom_path.filter(|p| !p.is_empty()) {
        let custom = PathBuf::from(path);
        if !custom.exists() {
            if let Some(parent) = custom.parent() {
                let _ = fs::create_dir_all(parent);
            }
            fs::File::create(&custom)
                .map_err(|e| format!("Failed to create custom rclone config file: {e}"))?;
            fs::write(&custom, "")
                .map_err(|e| format!("Failed to write to custom rclone config file: {e}"))?;
        }
        return Ok(custom);
    }
    get_rclone_config_path()
}

pub fn get_default_openlist_data_dir() -> Result<PathBuf, String> {
    Ok(get_user_data_dir()?.join("data"))
}
