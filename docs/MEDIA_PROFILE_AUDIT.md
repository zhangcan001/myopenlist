# Read-only media profile audit

## Current argument construction

The store builds `[remote:volume, mountPoint, ...extraFlags]` (`desktop/src/stores/app.ts:246-257`). Rust splits the first two positional arguments, removes legacy network-mode flags, inserts the Windows network-mode option at position 2, and inserts `--vfs-cache-mode=writes` at position 2 only when no explicit cache-mode option occurs before `--` (`cmd/rclone_mount.rs:39-67, 318-341`).

Current behavior:

- `--vfs-cache-mode=writes` is added by the program by default.
- An explicit user `--vfs-cache-mode=...` or two-argument `--vfs-cache-mode ...` is preserved.
- A cache-mode token after the option terminator `--` is not considered by the default detector.
- `--read-only` is not added automatically.
- User flags are persisted as `extraFlags` and are appended after the two positionals before the program-owned defaults are inserted.

## Future read-only placement

`--read-only` belongs in the program-owned media profile, adjacent to the other generated mount flags and before user extras. The implementation must validate or reject conflicting user flags rather than silently create ambiguous duplicate settings. This audit does not choose final cache/performance values.

## Performance parameters intentionally not decided

The current source only supplies the default `--vfs-cache-mode=writes`; it does not define final values for `buffer-size`, `vfs-read-chunk-size`, `vfs-read-ahead`, `dir-cache-time`, `vfs-cache-max-size`, `vfs-cache-max-age`, or `network-mode`. Those values require real VLC/PotPlayer tests against large files and must not be guessed in DEV-001.
