# V3.1 layered health model

The following is a design, not an implementation. Checks are ordered from local process facts to remote storage facts.

| Level | Check | Allowed frequency / trigger | Success meaning | Failure action |
|---|---|---|---|---|
| L0 | Desktop alive, Core process state, Rclone process state, binary existence | continuous local supervisor once implemented; no 115 access | local process/binary prerequisites exist | recover process or mark dependency failure |
| L1 | `GET http://127.0.0.1:<port>/ping` | startup and state transitions; short timeout | Core HTTP listener is ready; no directory List | wait/retry or classify port/Core failure |
| L2 | local WebDAV `OPTIONS` or authenticated depth-0 `PROPFIND` | after L1, mount start/recovery, and low-frequency diagnosis | local DAV endpoint/auth path responds | rebuild/check remote config; do not call 115 directly from the probe |
| L3 | 115 storage/auth check using existing Core driver path (for example `UserInfo` or a minimal storage lookup) | startup, explicit degraded state, and low-frequency periodic check; never every few seconds | storage exists and credentials are usable | classify auth/storage/rate/server error; refresh/recover as policy allows |
| L4 | read mounted drive root / targeted file metadata | after mount, after recovery, or user-reported read error | actual drive is readable | unmount/recover or notify; later media probe is user-triggered |

`READY` requires all of: Core alive, localhost HTTP ready, storage usable, WebDAV usable, Rclone alive, and drive readable. A process PID alone is never READY. The current source provides L1 (`get_openlist_core_status`) and a basic L4-style mount read (`check_mount_status`); L2/L3 orchestration and L0 continuous supervision are future work.
