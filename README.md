# OpenList 115 Media Drive

## Goal

115
→ OpenList
→ WebDAV
→ Rclone
→ WinFsp
→ Windows local drive
→ VLC / PotPlayer / MPC-BE / mpv

## V1

- Windows
- read-only
- local media drive

## Development status

- DEV-001 complete
- DEV-001.5 repository foundation
- DEV-002 generation-aware 115 token refresh coordination complete
- DEV-002.1 RefreshCoordinator completeness and concurrency coverage complete
- DEV-002.2 public RefreshToken linearization complete; DEV-002 series frozen
- DEV-003 refresh resilience complete: stale CAS, typed failures, bounded retry, cooldown, circuit states, cancellation, and offline Race CI

The vendored 115 SDK now coalesces concurrent auth-failure refreshes, rejects stale refresh responses, tracks token generation, commits Token Pairs atomically, applies bounded refresh resilience, and shares refresh results with waiting requests. The next development task is DEV-004 Authorization + Token Persistence Integration. QR login, real 115 account integration, and the media-drive runtime are not yet implemented.
