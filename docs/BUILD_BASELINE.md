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

## DEV-001.5 regression validation

| Command | Result | Notes |
|---|---|---|
| `scripts/bootstrap-dev.ps1` | PASS | Official `go1.26.4.windows-amd64.zip`; SHA-256 `3ca8fb4630b07c419cbdd51f754e31363cfcfb83b3a5354d9e895c90be2cc345`; installed under `.tools/go/` without system changes |
| `. .\scripts\use-dev-env.ps1; go version` | PASS | `go version go1.26.4 windows/amd64` |
| `go mod download` | PASS | Local SDK replacement and external modules resolved |
| `go list -m -json github.com/OpenListTeam/115-sdk-go` | PASS | `v0.2.6` resolves to `../third_party/115-sdk-go` |
| `go list -m all` | PASS | Listed `github.com/OpenListTeam/115-sdk-go v0.2.6 => ../third_party/115-sdk-go` |
| `go test ./...` | FAIL | Existing source/environment failures: Go 1.26 vet rejects non-constant format strings in several upstream packages; multipart Windows cleanup race; aria2 requires localhost:6800; MCP session tests fail; no business code changed |
| `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags=jsoniter ... .` | PASS | `core/build/openlist-windows-amd64.exe`, 118,105,088 bytes |
| `corepack yarn install --frozen-lockfile` after subtree import | PASS | Yarn Classic `1.22.22` |
| `corepack yarn web:build` after subtree import | PASS | `vue-tsc --noEmit` + Vite; same third-party PURE-comment warnings |
| `corepack yarn check:rust` after subtree import | PASS | `cargo check --all-targets --all-features` |

The known Tauri installer signing failure from DEV-001 was not re-run; it remains Packaging DEV scope.
