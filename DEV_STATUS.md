# DEV-001 status

Project:
OpenList 115 Media Drive

Architecture:
V3.1

DEV-001:
COMPLETE

DEV-001.5:
COMPLETE

DEV-002:
COMPLETE

Repository:
single-repository subtree layout

Core:
vendored/subtree

Desktop:
vendored/subtree

115 SDK:
vendored/subtree v0.2.6

Development Go:
portable local toolchain

Product Goal:
115 → local Windows drive → VLC

V1 Scope:
read-only media drive

## Completed

- Cloned and audited official OpenList Core and OpenList Desktop repositories.
- Audited the official `115-sdk-go` v0.2.6 source before the DEV-002 SDK change.
- Verified the Desktop frontend build and Rust checks.
- Attempted the Core dependency download, tests, Windows build, and official release path.
- Attempted the complete Desktop Tauri build; the release executable compiled, but bundling stopped at Windows code-signing because the downloaded OpenList sidecar has no signature.
- Recorded current startup, authentication, QR feasibility, local binding, WebDAV, drive-letter, media-profile, health, failure, update, acceptance, and UX findings in `docs/`.
- Converted Core, Desktop, and the pinned 115 SDK from parent-repository gitlinks to ordinary subtree-managed source trees.
- Added local SDK resolution through `core/go.mod` without changing SDK Go source.
- Added idempotent portable Go bootstrap, temporary environment setup, and dry-run upstream sync helper.
- Added generation-aware singleflight token refresh coordination to the vendored 115 SDK.
- Added deterministic concurrent refresh and shared-failure tests for the SDK auth path.

Blocked:

- `go test ./...` still fails on upstream vet checks and tests requiring external/local services; no business logic was changed to force a pass.
- The Desktop installer bundle remains unverified because the Windows signing step failed in DEV-001; Packaging DEV owns that issue.
- No real 115 account login, token refresh, WebDAV mount, WinFsp drive mount, VLC playback, or media acceptance run was performed. This is intentional for DEV-001.

Next:
DEV-003 QR/login and token persistence integration
