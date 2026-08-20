# DEV-001 build baseline

## Core

| Command | Result | Classification |
|---|---|---|
| `go mod download` | failed before execution: PowerShell could not resolve `go` | BLOCKED — missing Go |
| `go test ./...` | failed before package loading for the same reason | BLOCKED — missing Go |
| `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ... .` | failed before compilation for the same reason | BLOCKED — missing Go |
| official `build.sh` Windows release path | not runnable from this shell because `bash` resolves to WSL and no usable Bash distro is available; the exact Go build primitive was attempted separately | BLOCKED — missing Go/Bash environment |

The repository CI pins Go `1.26.4`. No Core binary or test result is claimed.

## Desktop

| Command | Result | Notes |
|---|---|---|
| `corepack yarn install --frozen-lockfile` | PASS | Yarn Classic `1.22.22`; 88.25s |
| `corepack yarn web:build` | PASS | `vue-tsc --noEmit` + Vite; 2076 modules; Rollup emitted two third-party PURE-comment warnings |
| `corepack yarn check:rust` before sidecar prep | FAIL | Tauri build script reported missing `src-tauri/binary/openlist-x86_64-pc-windows-msvc.exe` |
| `corepack yarn prebuild:dev` | PASS | OpenList `v4.2.4`, Rclone `v1.75.0`; SHA-256 values logged by script; NSIS helper plugins prepared under user AppData |
| `corepack yarn check:rust` after sidecar prep | PASS | `cargo check --all-targets --all-features` |
| `corepack yarn build` | FAIL after release compilation | Tauri produced `src-tauri/target/release/openlist-desktop.exe`, then bundling failed because `signtool.exe` found no signature for the bundled OpenList sidecar; no installer bundle is claimed |

No real 115 account, token, storage, WebDAV user, WinFsp install, drive mount, or media playback test was performed.
