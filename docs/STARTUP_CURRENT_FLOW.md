# Current startup and auto-mount flow

```text
Windows login
  -> Tauri autostart plugin launches Desktop only if the user enabled it
  -> Tauri setup()
  -> AppState::init() -> MergedSettings::load()
  -> create tray
  -> spawn auto_start_openlist_core_on_login()
       if openlist.auto_launch:
         build config: openlist server --data <data-dir>
         ProcessManager::register_and_start()
         child process spawn returns ProcessInfo
  -> auto_mount_rclone_remotes_on_login()
       select autoMount + non-empty mountPoint + non-empty volumeName
       optionally start Core when a local OpenList remote is found
       mount_remote()
         build rclone mount --config <rclone.conf> <remote>:<volume> <mountPoint>
         insert Windows network-mode flag
         insert default --vfs-cache-mode=writes when absent
         ProcessManager::register_and_start()
```

Evidence: `desktop/src-tauri/src/lib.rs:105-221, 272-351`; `cmd/openlist_core.rs:13-60`; `cmd/rclone_mount.rs:306-370`.

## Readiness boundary

`start_openlist_core` means successful child-process spawn. It does not wait for HTTP readiness. The separate `get_openlist_core_status` path calls `GET http(s)://localhost:<port>/ping` and returns HTTP success as `running`, but the login auto-mount task does not call that health check before mounting (`cmd/openlist_core.rs:79-144`).

There is no pre-mount check in this chain for:

- OpenList HTTP readiness;
- 115 storage/auth usability;
- WebDAV response;
- Rclone mount readability.

Manual mount waits three seconds in the Vue store and then loads mount info; `MountView.vue` polls mount info every 15 seconds. The login path logs spawn success and proceeds immediately. A process that exits after spawn can therefore be reported as started while the subsequent mount fails or becomes an error later.

## Port and binding implications

Packaged Core defaults to `scheme.address=0.0.0.0`, `http_port=5244` in `core/internal/conf/config.go:143-157`. Desktop passes only `server --data`, so it does not override the address. Desktop health uses `localhost`, but that does not make the listener localhost-only.
