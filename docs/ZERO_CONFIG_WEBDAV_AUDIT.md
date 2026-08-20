# Zero-config WebDAV audit

## Current endpoint and authentication

- Core mounts the WebDAV router at `/dav` (`core/server/router.go:39-44`). The handler uses `path.Join(conf.URL.Path, "/dav")` (`core/server/webdav.go:23-31`).
- WebDAV accepts HTTP Basic Auth and also has a bearer-token path for the configured Core token (`core/server/webdav.go:66-94`). Basic credentials are checked by `tryLogin`, and the user must have `CanWebdavRead`.
- `PUT`, `MKCOL`, `MOVE`, `COPY`, `DELETE`, and `PROPPATCH` additionally require `CanWebdavManage` (`core/server/webdav.go:108-151`).
- Admin API exposes `POST /api/admin/user/create` and `POST /api/admin/user/update` (`core/server/router.go:127-134`). `CreateUser` hashes the password and writes the user (`core/server/handles/user.go:32-49`).

## Read-only internal account

The user permission model has separate WebDAV-read bit 8 and WebDAV-manage bit 9 (`core/internal/model/user.go:167-181`). A dedicated account with read bit set and manage bit clear is technically expressible. Its BasePath can be limited to the selected storage path. The current Desktop has no command that creates this account or obtains its password; this is a future Core/desktop orchestration change.

## Rclone generation

Desktop already accepts a WebDAV config and writes an `rclone.conf` remote. `WebDavRemoteConfig::to_rclone_config_with_obscured_pass` runs `rclone obscure` and stores `url`, `user`, and obscured `pass` (`desktop/src-tauri/src/conf/rclone_config.rs:13-36, 208-241`). The config is saved under the app data directory (`utils/path.rs:168-181`) and can be overridden by a custom path.

## Feasibility

**FEASIBLE WITH DESKTOP/CORE ORCHESTRATION.** Future setup needs to: create or provision the internal read-only user through an authenticated Core path, resolve the actual localhost URL/port, generate the remote using the existing obscure helper, persist the remote and drive profile, and never expose those values in the simple UI. No real user or remote was created in DEV-001.
