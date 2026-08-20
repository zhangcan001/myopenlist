# DEV-006 Windows Mount Integration

## Scope

DEV-006 adds the Windows mount boundary between the managed DEV-005 localhost
WebDAV service and a Windows drive letter. It does not add Desktop UI,
autostart, an installer, a Windows service, or a scheduling subsystem.

The flow is:

```text
managed localhost WebDAV -> cgofuse -> WinFsp -> Windows drive letter
```

The existing WebDAV implementation and `core/pkg/gowebdav` client are reused;
the mount package does not implement another WebDAV protocol.

## Profile

`core/internal/media_drive/mount` stores one private Core setting with this
shape:

| Field | Default |
|---|---|
| DriveLetter | `R:` |
| WebDAVURL | `http://127.0.0.1:19080` |
| Enabled | `false` |
| AutoReconnect | `true` |

Only `127.0.0.1` HTTP URLs with an explicit port are accepted. Public hosts,
userinfo, HTTPS, and malformed drive letters are rejected. WebDAV credentials
are runtime-only and are excluded from profile persistence and JSON output.

## Lifecycle and recovery

The manager reports `UNMOUNTED`, `MOUNTING`, `MOUNTED`, `UNMOUNTING`, and
`FAILED`. `Mount`, `Unmount`, and `Status` are serialized against state changes.

When a mounted backend exits unexpectedly and `AutoReconnect` is enabled, the
manager waits 100 ms and performs one replacement mount attempt. A failed
replacement moves the manager to `FAILED`; this is intentionally a basic
reconnect path, not a scheduler or an infinite retry loop.

The Windows backend is read-only. Directory metadata and listings use the
existing WebDAV client, while file reads use bounded HTTP Range requests. Write,
delete, rename, and truncate operations return read-only errors.

## WinFsp and permissions

WinFsp must be installed for the target architecture and its driver must be
running. The installed `cgofuse` backend loads WinFsp at runtime. The process
must also have permission to claim the selected drive letter; if Windows or
WinFsp rejects the mount, the manager reports `FAILED` and preserves the error
class without logging credentials.

Automated tests use a mock backend and never create a real drive. Manual
verification should confirm WinFsp installation, an unused `R:` letter, a
running DEV-005 localhost WebDAV service, directory listing, bounded media
reads, unmount, and reconnect behavior.

DEV-007 is not started by this change.
