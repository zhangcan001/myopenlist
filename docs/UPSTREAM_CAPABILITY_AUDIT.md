# Upstream Desktop capability audit

Status is based on the checked-out source, not the README feature list.

| Capability | Status | Evidence |
|---|---|---|
| OpenList Core start/stop | SUPPORTED | `src-tauri/src/cmd/openlist_core.rs`: `start_openlist_core`, `stop_openlist_core` |
| OpenList auto launch | PARTIAL | `src-tauri/src/lib.rs`: `auto_start_openlist_core_on_login`; gated by `settings.openlist.auto_launch`, and only after Desktop itself starts |
| Rclone bundled/downloaded | SUPPORTED | `scripts/prepare.js`: `resolveSidecar`, Windows/macOS bundle path |
| Rclone availability check | PARTIAL | `cmd/rclone_core.rs::check_rclone_available` only resolves an existing path; it does not run `rclone version` |
| WebDAV remote creation | SUPPORTED | `cmd/rclone_mount.rs::rclone_create_remote`; `conf/rclone_config.rs::WebDavRemoteConfig` |
| Rclone mount | SUPPORTED | `cmd/rclone_mount.rs::mount_remote` |
| Rclone unmount | SUPPORTED | `cmd/rclone_mount.rs::unmount_remote` |
| `autoMount` | SUPPORTED | `lib.rs::auto_mount_rclone_remotes_on_login` filters and starts configured remotes |
| Windows `networkMode` | SUPPORTED | `cmd/rclone_mount.rs::insert_network_mode`; persisted in `conf/rclone.rs` |
| Mount status check | SUPPORTED | `cmd/rclone_mount.rs::check_mount_status` reads the mount path with a two-second timeout |
| ProcessManager | SUPPORTED | `core/process_manager.rs`: register/start/stop/status/list/remove |
| Process state persistence | SUPPORTED | `ProcessManager::persist_state` writes `process_state.json` |
| Process state recovery | SUPPORTED | `ProcessManager::recover_persisted_state` runs in `ProcessManager::new` |
| System tray | SUPPORTED | `src-tauri/src/tray.rs`, tray creation in `lib.rs::setup` |
| OS autostart | PARTIAL | Tauri autostart plugin and SettingsView `enable/disable`; it is user-controlled and not the same as Core `auto_launch` |
| App update mechanism | SUPPORTED | Tauri updater endpoint in `tauri.conf.json`, background check after 300s in `lib.rs` |
| OpenList Core update mechanism | SUPPORTED | `cmd/os_operate.rs::get_available_versions` and `update_tool_version` download/replace a chosen release |
| Rclone update mechanism | SUPPORTED | same two commands with `tool == "rclone"` |
| Continuous process monitoring | NOT FOUND | no supervisor task, exit watcher, restart loop, backoff, or crash-loop guard in the audited source |

The important boundary is that `ProcessManager` can recover a still-alive process after Desktop restarts and can detect termination when `status`/`list` is called, but it does not continuously supervise or restart a crashed process.
