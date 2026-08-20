# DEV-007 User Ready Workflow

## Scope

DEV-007 adds one Core workflow manager that coordinates the existing 115
authorization status, managed localhost WebDAV service, and DEV-006 WinFsp
mount manager. It does not modify the 115 SDK, WebDAV protocol, or WinFsp
filesystem implementation.

The start sequence is:

```text
115 storage ready -> WebDAV running -> WinFsp mount -> RUNNING
```

If the mount step fails after this workflow started WebDAV, WebDAV is stopped
again. A later start can retry the sequence.

## State and health

The workflow reports `INIT`, `CHECKING`, `READY`, `STARTING`, `RUNNING`,
`DEGRADED`, `FAILED`, and `STOPPING`.

`GET /api/admin/media-drive/health` returns JSON for the workflow, 115, WebDAV,
and Mount components. A running workflow becomes `DEGRADED` when a component
later stops being ready. Diagnostics contain only a stable module, code,
reason, and suggested action. Access tokens, refresh tokens, and passwords are
never included.

## Admin API

All routes require the existing admin middleware:

- `GET /api/admin/media-drive/status`
- `POST /api/admin/media-drive/start`
- `POST /api/admin/media-drive/stop`
- `GET /api/admin/media-drive/health`

The start body is optional. When the managed WebDAV password is needed by the
runtime-only WinFsp client, send it as `webdav_password` (and optionally
`webdav_username`). The password is used in memory for the mount and is not
returned, logged, or persisted by the mount profile. The managed WebDAV
service itself continues to persist only its bcrypt password hash.

Example:

```json
{
  "webdav_password": "<managed-webdav-password>"
}
```

The response contains workflow status or a stable workflow error with a safe
diagnostic. It never contains the request body.

## Failure diagnosis

Typical diagnostics point to one of these actions:

- `AUTH_REQUIRED`: complete 115 authorization or import a token pair.
- `WEBDAV_PASSWORD_REQUIRED`: configure the managed WebDAV password.
- `WEBDAV_PORT_CONFLICT`: free the localhost WebDAV port.
- `WINFSP_UNAVAILABLE`: install WinFsp for the current Windows architecture.
- `MOUNT_CREDENTIALS_REQUIRED`: provide the managed WebDAV password to start.

DEV-008 is not started by this change.
