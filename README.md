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
- DEV-004 authorization and token persistence integration complete; real 115 acceptance pending approved client ID and controlled account test

The vendored 115 SDK now coalesces concurrent auth-failure refreshes, rejects stale refresh responses, tracks token generation, commits Token Pairs atomically, and applies bounded refresh resilience. Core now provides admin-only PKCE authorization, token import, managed `/115` provisioning, and narrow Token Pair persistence. The next development task is DEV-005 Managed WebDAV + Localhost Service Profile. Real 115 account integration and the media-drive runtime remain pending production acceptance.
