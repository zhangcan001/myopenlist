use tauri::State;

use crate::object::structs::AppState;
use crate::utils::path::get_rclone_binary_path_with_custom;

#[tauri::command]
pub async fn check_rclone_available(state: State<'_, AppState>) -> Result<bool, String> {
    get_rclone_binary_path_with_custom(state)
        .map(|_| true)
        .or(Ok(false))
}
