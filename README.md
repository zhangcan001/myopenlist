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

The vendored 115 SDK now coalesces concurrent auth-failure refreshes, tracks token generation, commits Token Pairs atomically, and shares refresh results with waiting requests. The next development task is DEV-003 Refresh Resilience. QR login, real 115 account integration, and the media-drive runtime are not yet implemented.
