# Localhost binding audit

## Current defaults

- Core `DefaultConfig` sets `scheme.address` to `0.0.0.0` and `scheme.http_port` to `5244` (`core/internal/conf/config.go:143-157`).
- Desktop defaults to port `5244` and starts Core with `server --data <data-dir>` (`desktop/src-tauri/src/conf/core.rs:12-21`; `cmd/openlist_core.rs:35-45`).
- Desktop does not currently expose or set `scheme.address`.

## Safe future change

The packaged data directory should be initialized or migrated before Core spawn with an explicit `scheme.address=127.0.0.1`; alternatively Core’s packaged default/config-generation path must be changed. Desktop health and WebDAV generation must use the same resolved address. This DEV-001 only identifies the locations; it does not change defaults.

## Port collision behavior

There is no automatic free-port search. `start_openlist_core` reports the `ProcessManager::register_and_start` result, which is spawn success, not bind success. If another process owns 5244, Core can exit after spawn; `get_openlist_core_status` later reports false because `GET /ping` fails. The configured port is persisted in Desktop settings and can also be synchronized from Core’s data config by `MergedSettings::load` (`conf/config.rs:46-61, 94-105`).

Risk: a collision can be logged as a successful startup and auto-mount can run before the failure is observed.
