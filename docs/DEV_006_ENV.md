# DEV-006 Windows Mount Environment

Checked 2026-08-20 on Windows `amd64`.

| Component | Result | Evidence |
|---|---|---|
| Windows target | PASS | `windows/amd64` |
| Go toolchain | PASS | portable `go1.26.4` |
| Core Windows build | PASS | `go -C core build` |
| WinFsp | PASS | installed under `C:\Program Files (x86)\WinFsp\` |
| WinFsp x64 DLL | PASS | `bin\winfsp-x64.dll` exists |
| WinFsp driver | PASS | x64 driver and `fsptool-x64.exe` exist |
| WinFsp version | PASS | `fsptool-x64.exe ver` reports `2.2` |

The mount package uses `github.com/winfsp/cgofuse` on Windows. Its Windows
backend loads the installed WinFsp DLL; no installer, service, registry
provisioning, or PATH mutation is added by DEV-006. Non-Windows builds use an
unsupported backend so CI can run the lifecycle tests without a real mount.

The automated verification uses a mock backend only. No drive was mounted and
no real 115 account or media was used. Actual Windows mounting remains a
manual verification step on a machine with WinFsp installed.
