# Windows drive-letter audit

## Current behavior

`RcloneMountConfig` persists `mountPoint`, `volumeName`, `autoMount`, `networkMode`, and `extraFlags` (`desktop/src-tauri/src/conf/rclone.rs:8-24`). The Vue form asks the user to type the mount point and volume name (`src/views/MountView.vue:390-420`), then the store passes the mount point as the second positional mount argument (`src/stores/app.ts:236-258`).

`mount_remote` creates the path if it does not exist, starts Rclone, and later `check_mount_status` reads the Windows path with a two-second timeout (`cmd/rclone_mount.rs:322-333, 391-449`). There is no current `R:` preference, drive enumeration, collision fallback, or automatic selection.

## V1 strategy to implement later

1. Enumerate usable letters from `R:` through `Z:` using Windows drive APIs or an equivalent read-only query.
2. Prefer the first unused letter; if `R:` is occupied, try `S:`, then `T:` and so on.
3. Persist the selected letter in the media-drive profile and reuse it on later launches.
4. If the saved letter becomes occupied, enter a recoverable “letter conflict” state and choose a new letter only with an explicit policy/notification.
5. Pass the selected letter to Rclone and verify it by opening the root directory; merely seeing a process is insufficient.

No actual drive enumeration or mount was performed in DEV-001.
