# OpenList 115 Media Drive — V3.1 responsibility boundary

```text
Simple UI
  -> MediaDriveController / Startup Orchestrator / Supervisor
       -> Core health + storage/auth state
       -> WebDAV profile + Rclone mount
       -> Windows drive verification
```

## 115 SDK layer

SDK-side work owns `RefreshCoordinator`, `SingleFlight`, token generation/versioning, backoff, jitter, circuit breaker, and typed auth-error classification. DEV-002 implements the first three for concurrent access-token refreshes: one in-flight refresh per client, generation-aware retry suppression, and shared success/failure results. Backoff, jitter, circuit breaker, and typed auth-error classification remain future work.

## OpenList Core layer

Owns 115 driver integration, token persistence callback, storage, WebDAV, localhost-only packaged defaults, and health exposure. The current Core provides the driver, token callback, WebDAV, and basic `/ping`, but not the V3.1 localhost/profile orchestration.

## Desktop layer

Owns `MediaDriveController`, startup orchestration, a continuous Supervisor, recovery, QR/login UI, automatic WebDAV account/remote setup, automatic drive-letter selection, Rclone media profile, diagnostics, update orchestration, and the simple UI.

## External components

- Rclone remains the official Rclone; no fork.
- WinFsp remains the official WinFsp; no fork.
- VLC is not embedded and no VLC SDK is developed. The drive contract is validated through real player reads.

## V1 safety boundary

The default profile is Windows-only and read-only. Tokens and internal WebDAV credentials remain app-managed. `READY` is a compound health state, never a PID state.
