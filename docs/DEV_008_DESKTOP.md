# DEV-008 Desktop User Experience

## User flow

The desktop path is intentionally small:

```text
Open desktop app
  -> check WinFsp and Core health
  -> check 115 authorization
  -> click Start media drive
  -> start managed WebDAV
  -> mount R:
  -> show “媒体盘已就绪”
  -> click Open VLC
```

The VLC action opens the configured drive folder with `--no-playlist-autostart`.
It does not select a file or start playback.

## Boundary

The desktop control layer calls the Core Admin HTTP API only. It does not
import or call the 115 driver, Auth115, WebDAV, Workflow, or WinFsp packages.
The Core API client logs in to the local Admin API using the existing desktop
admin credential, keeps the returned token in memory, and discards it with the
client instance.

Core routes used by the desktop are:

- `GET /api/admin/media-drive/health`
- `GET /api/admin/media-drive/status`
- `POST /api/admin/media-drive/start`
- `POST /api/admin/media-drive/stop`
- `GET /api/admin/media-drive/mount/profile`
- `POST /api/admin/media-drive/mount/profile`

All routes are behind the existing Admin middleware. The mount-profile API
persists only drive letter, localhost WebDAV URL, enabled, and reconnect
settings. Credentials are not part of the response or persisted profile.

## Desktop states

The UI maps the workflow to:

`APP_STARTING`, `CHECKING_ENV`, `WAITING_AUTH`, `READY_TO_START`, `STARTING`,
`RUNNING`, `DEGRADED`, and `ERROR`.

The home card shows separate 115, WebDAV, and drive health indicators. Errors
are translated from stable diagnostic codes into user-facing guidance; raw
Core error strings are not rendered.

## First-run setup

The setup card checks WinFsp, accepts a drive letter (default `R:`), accepts a
WebDAV password for the current start, and stores only the drive letter and
startup option in browser-local desktop configuration. The WebDAV password,
115 tokens, and Core Admin bearer token are never stored by this layer.

If the app is restarted, the WebDAV password must be entered again when the
mount needs it. This is the deliberate consequence of not saving a plaintext
password.

## Requirements and manual acceptance

- Windows 10 or later
- WinFsp installed and permitted to mount a drive
- OpenList Core running locally
- A valid 115 authorization
- VLC installed

`REAL_USER_ACCEPTANCE_REQUIRED` remains until the following are manually
verified on Windows: real 115 account, WinFsp R: mount, a 4K MKV, and VLC
playback. Automated tests use mocked HTTP responses and do not create a real
drive mount.
