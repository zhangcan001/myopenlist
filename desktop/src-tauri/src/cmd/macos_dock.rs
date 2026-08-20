#[tauri::command]
pub async fn set_dock_icon_visibility(
    app_handle: tauri::AppHandle,
    visible: bool,
) -> Result<bool, String> {
    #[cfg(target_os = "macos")]
    {
        app_handle
            .set_activation_policy(if visible {
                tauri::ActivationPolicy::Regular
            } else {
                tauri::ActivationPolicy::Accessory
            })
            .map_err(|e| format!("Failed to set activation policy: {e}"))?;

        log::info!(
            "macOS dock icon visibility set to: {}",
            if visible { "visible" } else { "hidden" }
        );
        Ok(true)
    }

    #[cfg(not(target_os = "macos"))]
    {
        let _ = app_handle;
        let _ = visible;
        log::debug!("set_dock_icon_visibility is only supported on macOS");
        Ok(false)
    }
}
