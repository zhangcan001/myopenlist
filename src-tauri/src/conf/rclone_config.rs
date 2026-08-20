use std::collections::HashMap;
use std::fs;
use std::io::{BufRead, BufReader, Write};
use std::path::Path;
use std::process::Command;

use serde::{Deserialize, Serialize};
use tauri::State;

use crate::object::structs::AppState;
use crate::utils::path::{get_rclone_binary_path_with_custom, get_rclone_config_path_with_custom};

pub fn obscure_password(password: &str, state: State<'_, AppState>) -> Result<String, String> {
    if password.is_empty() {
        return Ok(String::new());
    }

    let rclone_bin = get_rclone_binary_path_with_custom(state)?;

    let mut cmd = Command::new(&rclone_bin);
    cmd.args(["obscure", password]);
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000); // CREATE_NO_WINDOW
    }
    let output = cmd
        .output()
        .map_err(|e| format!("Failed to run rclone obscure: {e}"))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("rclone obscure failed: {stderr}"));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

pub fn reveal_password(obscured: &str, state: State<'_, AppState>) -> Result<String, String> {
    if obscured.is_empty() {
        return Ok(String::new());
    }

    let rclone_bin = get_rclone_binary_path_with_custom(state)?;

    let mut cmd = Command::new(&rclone_bin);

    cmd.args(["reveal", obscured]);
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000); // CREATE_NO_WINDOW
    }
    let output = cmd
        .output()
        .map_err(|e| format!("Failed to run rclone reveal: {e}"))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("rclone reveal failed: {stderr}"));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

/// Represents a remote configuration entry in rclone.conf
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RcloneRemoteConfig {
    pub name: String,
    pub remote_type: String,
    #[serde(flatten)]
    pub options: HashMap<String, String>,
}

/// Represents the full rclone configuration file
#[derive(Debug, Clone, Default)]
pub struct RcloneConfigFile {
    pub remotes: HashMap<String, RcloneRemoteConfig>,
    custom_config_path: Option<String>,
}

impl RcloneConfigFile {
    pub fn load_with_custom(state: State<'_, AppState>) -> Result<Self, String> {
        let settings = state
            .app_settings
            .read()
            .clone()
            .ok_or("Failed to read app settings")?;
        let custom_path = settings.rclone.rclone_conf_path.as_deref();
        let config_path = get_rclone_config_path_with_custom(state)
            .map_err(|e| format!("Failed to get config path: {e}"))?;
        let mut config = Self::load_from_path(&config_path)?;
        config.custom_config_path = custom_path.map(|s| s.to_string());
        Ok(config)
    }

    pub fn load_from_path(path: &Path) -> Result<Self, String> {
        if !path.exists() {
            return Ok(Self::default());
        }

        let file =
            fs::File::open(path).map_err(|e| format!("Failed to open rclone config: {e}"))?;
        let reader = BufReader::new(file);

        let mut config = Self::default();
        let mut current_section: Option<String> = None;
        let mut current_options: HashMap<String, String> = HashMap::new();
        let mut current_type: Option<String> = None;

        for line in reader.lines() {
            let line = line.map_err(|e| format!("Failed to read line: {e}"))?;
            let trimmed = line.trim();

            if trimmed.is_empty() || trimmed.starts_with('#') || trimmed.starts_with(';') {
                continue;
            }
            if trimmed.starts_with('[') && trimmed.ends_with(']') {
                if let Some(section_name) = current_section.take() {
                    let remote = RcloneRemoteConfig {
                        name: section_name.clone(),
                        remote_type: current_type.take().unwrap_or_default(),
                        options: std::mem::take(&mut current_options),
                    };
                    config.remotes.insert(section_name, remote);
                }

                current_section = Some(trimmed[1..trimmed.len() - 1].to_string());
                current_options = HashMap::new();
                current_type = None;
                continue;
            }

            if let Some((key, value)) = trimmed.split_once('=') {
                let key = key.trim().to_string();
                let value = value.trim().to_string();

                if key == "type" {
                    current_type = Some(value);
                } else {
                    current_options.insert(key, value);
                }
            }
        }

        if let Some(section_name) = current_section.take() {
            let remote = RcloneRemoteConfig {
                name: section_name.clone(),
                remote_type: current_type.take().unwrap_or_default(),
                options: current_options,
            };
            config.remotes.insert(section_name, remote);
        }

        Ok(config)
    }

    pub fn save(&self, state: State<'_, AppState>) -> Result<(), String> {
        let config_path = get_rclone_config_path_with_custom(state)
            .map_err(|e| format!("Failed to get config path: {e}"))?;
        self.save_to_path(&config_path)
    }

    pub fn save_to_path(&self, path: &Path) -> Result<(), String> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| format!("Failed to create config directory: {e}"))?;
        }

        let mut content = String::new();

        for (name, remote) in &self.remotes {
            content.push_str(&format!("[{name}]\n"));
            content.push_str(&format!("type = {}\n", remote.remote_type));

            for (key, value) in &remote.options {
                content.push_str(&format!("{key} = {value}\n"));
            }

            content.push('\n');
        }

        let mut file =
            fs::File::create(path).map_err(|e| format!("Failed to create rclone config: {e}"))?;
        file.write_all(content.as_bytes())
            .map_err(|e| format!("Failed to write rclone config: {e}"))?;

        Ok(())
    }

    pub fn set_remote(&mut self, remote: RcloneRemoteConfig) {
        self.remotes.insert(remote.name.clone(), remote);
    }

    pub fn remove_remote(&mut self, name: &str) -> Option<RcloneRemoteConfig> {
        self.remotes.remove(name)
    }

    pub fn list_remotes(&self) -> Vec<String> {
        self.remotes.keys().cloned().collect()
    }

    pub fn has_remote(&self, name: &str) -> bool {
        self.remotes.contains_key(name)
    }
}

/// WebDAV specific configuration helper
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebDavRemoteConfig {
    pub name: String,
    pub url: String,
    pub vendor: Option<String>,
    pub user: String,
    pub pass: String,
}

impl WebDavRemoteConfig {
    pub fn to_rclone_config_with_obscured_pass(
        &self,
        state: State<'_, AppState>,
    ) -> Result<RcloneRemoteConfig, String> {
        let obscured_pass = obscure_password(&self.pass, state)?;

        let mut options = HashMap::new();
        options.insert("url".to_string(), self.url.clone());
        options.insert("user".to_string(), self.user.clone());
        options.insert("pass".to_string(), obscured_pass);

        if let Some(vendor) = &self.vendor
            && !vendor.is_empty()
        {
            options.insert("vendor".to_string(), vendor.clone());
        }

        Ok(RcloneRemoteConfig {
            name: self.name.clone(),
            remote_type: "webdav".to_string(),
            options,
        })
    }
}
