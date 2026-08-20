use tauri::State;

use crate::cmd::config::save_settings;
use crate::object::structs::AppState;

#[tauri::command]
pub async fn get_current_version() -> Result<String, String> {
    Ok(env!("CARGO_PKG_VERSION").to_string())
}

#[tauri::command]
pub async fn set_auto_check_enabled(
    enabled: bool,
    state: State<'_, AppState>,
) -> Result<(), String> {
    log::info!("Setting auto-check updates preference to: {enabled}");

    let mut settings = state.get_settings().unwrap_or_else(|| {
        use crate::conf::config::MergedSettings;
        MergedSettings::default()
    });

    settings.app.auto_update_enabled = Some(enabled);
    state.update_settings(settings.clone());
    save_settings(settings, state)
        .await
        .map_err(|e| format!("Failed to save settings: {e}"))?;
    Ok(())
}

#[tauri::command]
pub async fn is_auto_check_enabled(state: State<'_, AppState>) -> Result<bool, String> {
    let settings = state.get_settings().unwrap_or_else(|| {
        use crate::conf::config::MergedSettings;
        MergedSettings::default()
    });

    Ok(settings.app.auto_update_enabled.unwrap_or(true))
}
