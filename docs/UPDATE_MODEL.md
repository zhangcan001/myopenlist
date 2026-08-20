# Update model audit and V3.1 release model

## Current upstream behavior

- `scripts/prepare.js` starts with fallback OpenList `v4.2.4`, queries the latest OpenList release, and downloads the returned release; it queries Rclone `version.txt` and downloads the latest Rclone release for Windows/macOS (`prepare.js:31-59, 207-234`).
- Runtime `get_available_versions` queries the latest ten GitHub releases for OpenList or Rclone with a 30-minute in-memory cache (`src-tauri/src/cmd/os_operate.rs:100-240`).
- Runtime `update_tool_version` stops Core or mounts, downloads a selected archive, backs up the current binary, replaces it, and removes the backup after success (`os_operate.rs:243-387`).
- The Tauri updater updates the Desktop package from the signed `latest.json` endpoint in `tauri.conf.json`; a background app-update check runs after 300 seconds when enabled (`src-tauri/src/lib.rs:49-88`).

## Risk

The current build preparation and runtime version lists are “latest” driven. An Enhanced Desktop release must not silently pair with an arbitrary future Core or Rclone. The current source has no Media Drive release manifest and no binding between Desktop, Core, SDK, Rclone, and WinFsp versions.

## Future Media Drive Release Manifest

Every release should pin:

```text
Desktop version
Enhanced Core version/commit
OpenList upstream commit
115 SDK version/commit
Rclone version
WinFsp minimum compatible version
```

Update orchestration should validate the manifest before replacing sidecars, retain a recoverable previous binary until health checks pass, and never replace Enhanced Core with arbitrary upstream latest. This DEV-001 only defines the boundary.
