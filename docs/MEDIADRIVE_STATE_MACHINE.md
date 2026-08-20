# MediaDriveController state machine (design only)

| State | Enter when | Exit when | Timeout | Retry policy | Forbidden behavior |
|---|---|---|---|---|---|
| `INIT` | app process starts | config/dependency checks queued | 10s | once | do not touch 115 |
| `CHECK_DEPENDENCIES` | controller created | WinFsp/Rclone/Core prerequisites known | 10s | low-rate local retry | do not claim READY from binaries alone |
| `WAIT_NETWORK` | no usable network | network probe succeeds | bounded, then `DEGRADED` | exponential backoff + jitter | no rapid 115 polling |
| `STARTING_CORE` | Core start requested | child spawned or start failed | 10s | bounded restart attempts | do not mount yet |
| `WAIT_CORE_READY` | child exists | `/ping` succeeds or timeout | 30s | bounded, backoff | spawn success is not readiness |
| `CHECKING_AUTH` | Core is ready | token valid, refresh succeeds, or auth required | 30s | one classified refresh attempt | no infinite RefreshToken retry |
| `CHECKING_STORAGE` | auth is usable | 115 storage check succeeds | 30s | low-frequency remote retry | no directory storm |
| `CHECKING_WEBDAV` | Core/storage are usable | local DAV check succeeds | 10s | rebuild/retry bounded | no user credential prompt for internal profile |
| `MOUNTING` | DAV profile ready | Rclone spawn + mount check | 30s | unmount stale process then bounded retry | do not call mount READY on PID alone |
| `VERIFYING` | mount process exists | drive read succeeds | 10s | one recovery attempt | no real large-file test at startup |
| `READY` | all six readiness conditions hold | any health loss | continuous | supervisor enters `RECOVERING` | no hidden config mutation |
| `DEGRADED` | nonfatal local/remote condition | recovery succeeds or auth/dependency fatality known | continuous | scheduled low-frequency recovery | no rapid remote polling |
| `RECOVERING` | a health transition is detected | returns to prior check or fatal state | per operation | backoff, jitter, crash-loop guard | do not overlap recovery attempts |
| `AUTH_REQUIRED` | RefreshToken invalid or no token | QR/device flow completes | user decision | no automatic retry loop | do not erase valid state blindly |
| `FATAL` | WinFsp/admin install, severe corruption, unrecoverable config | explicit user repair | none | no automatic destructive repair | do not silently reset database/tokens |

`READY` means: Core alive, localhost HTTP ready, 115 storage usable, local WebDAV usable, Rclone alive, and drive readable. `ProcessManager`’s current recovery is not this state machine; it only recovers still-alive processes on Desktop initialization.
