# DEV-001 build environment

Checked 2026-08-20 on Windows 10 Pro x64, build `19045`.

| Tool | Required | Installed | Compatible | Missing / mismatch |
|---|---|---|---|---|
| Git | repository operations | `2.52.0.windows.1` | YES | — |
| Go | Core `go 1.25.0`, toolchain `go1.26.4` from `core/go.mod` and CI | not found (`go` command unavailable) | NO | Core cannot download, test, or build |
| Node.js | Desktop README: v22+ | `v24.14.0` | YES | — |
| Yarn | Desktop `yarn.lock v1`, README uses Yarn | global command not found; Corepack provides Yarn `1.22.22` | YES via Corepack | no standalone `yarn.exe` on PATH |
| npm | not selected by repository | `11.9.0` | N/A | deliberately not used |
| pnpm | not selected by repository | `10.33.0` | N/A | deliberately not used |
| Rust | Cargo project, Rust 2024 | `rustc 1.94.1` stable | YES for check/build | README says latest nightly, but source built with installed stable |
| Cargo | Rust build | `cargo 1.94.1` | YES for Desktop | — |
| MSVC `cl`/`link` | native Windows/Tauri toolchain | not resolvable in current shell | UNKNOWN/blocked for native installer | Visual Studio developer shell not active |
| NSIS `makensis` | installer packaging | not resolvable in current shell | not needed for completed `cargo check`; installer availability not proven | package tool not verified |
| WebView2 | Tauri runtime | not independently queried | UNKNOWN | runtime availability not proven by static checks |

## Notes

- Corepack was used only to provide the repository-selected Yarn Classic version; no npm/pnpm migration occurred.
- `corepack yarn install --frozen-lockfile` completed successfully without changing `yarn.lock`.
- `prebuild:dev` downloaded build-local OpenList/Rclone sidecars and NSIS helper plugins; it did not install Rclone, WinFsp, a service, a scheduled task, or a registry entry.
