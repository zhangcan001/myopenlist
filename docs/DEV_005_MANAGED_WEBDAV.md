# DEV-005 Managed WebDAV

## Architecture

DEV-005 adds a managed localhost service profile around the existing Core WebDAV
implementation. `core/internal/media_drive/webdav` owns profile validation,
private-setting persistence, Basic Auth, and the listener lifecycle. After
authentication, the service delegates the request to
`core/server/webdav.Handler`; it does not implement a second WebDAV protocol.

The existing `/dav` route and its configuration are unchanged. The managed
service uses its own profile and listener, with `127.0.0.1:19080` as the
default address. Port `0` remains available for automated tests and the status
response reports the actual listener address.

## Why localhost only

The media endpoint is intended for a player on the same Windows machine. The
listener binds only to `127.0.0.1`, and the request guard accepts loopback
clients only (`127.0.0.1` and `::1`). Non-loopback requests are rejected before
Basic Auth. Public binding, `0.0.0.0`, reverse proxying, HTTPS certificates,
and public WebDAV are outside this DEV.

## Player compatibility

VLC, MPC-HC, and Kodi can use the standard HTTP WebDAV surface with independent
Basic Auth credentials. The existing Core handler already routes media reads
through Core's range-aware HTTP serving path, which provides
`Accept-Ranges: bytes`, `206 Partial Content`, and bounded readers for seek
requests.

The automated tests use fake readers for a 50 GB logical file and several byte
offsets. They verify that only the requested range is read; no complete video
or temporary file is created. Real player playback is not claimed by DEV-005.

## Security model

Managed WebDAV credentials are independent from 115 access and refresh tokens.
The profile stores only a bcrypt password hash. Profile and status APIs return
`password_configured`, never the hash or plaintext password. The service does
not log credentials, and its error responses use stable service errors.

The profile is stored as one private Core setting. It persists enabled state,
bind policy, port, username, password hash, and timestamps. Runtime listener
state is not persisted.

## Profile lifecycle

The service exposes `STOPPED`, `STARTING`, `RUNNING`, `STOPPING`, and `FAILED`.
`Start` validates the profile, binds the loopback listener, and starts the
existing handler. `Stop` closes the listener and releases the port. A bind
failure on an occupied port is reported as `PORT_CONFLICT`.

Admin routes are protected by the existing `AuthAdmin` group:

- `GET /api/admin/media-drive/webdav/profile`
- `POST /api/admin/media-drive/webdav/profile`
- `POST /api/admin/media-drive/webdav/start`
- `POST /api/admin/media-drive/webdav/stop`
- `GET /api/admin/media-drive/webdav/status`

## Future WinFsp integration

DEV-005 does not provide drive mounting, a drive letter, WinFsp, Rclone,
autostart, Desktop UI, or VLC integration. Windows mount integration is
deferred to DEV-006.
