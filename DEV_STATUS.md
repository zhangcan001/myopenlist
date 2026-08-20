# Development status

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

DEV-002.1:
COMPLETE

DEV-002.2:
COMPLETE

DEV-002 Series:
FROZEN

DEV-003:
COMPLETE

DEV-004:
COMPLETE

DEV-004 Production Verification:
REAL_115_NOT_YET_TESTED

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
- Added atomic Token Pair updates for refresh and CodeToToken without changing the existing QR callback behavior.
- Added deterministic concurrent refresh, stale-401, cancellation, snapshot, failure-recovery, and error-classification tests for the SDK auth path.
- Linearized the public `RefreshToken` entry so it joins or creates a flight inside the token lock.
- Added the explicit-refresh coordinator-gap regression test and froze the DEV-002 contract.
- Added stale refresh-response CAS using start generation and refresh token snapshots.
- Added conservative refresh error classification, bounded retry/backoff/jitter, `Retry-After` cooldown, and circuit states.
- Added context-aware refresh cancellation, `RefreshStatus`, Token Pair circuit reset rules, and offline resilience tests.
- Added Ubuntu GitHub Actions normal and Race jobs for the offline refresh test subset.
- Added Core 115 PKCE/device-code authorization sessions, explicit single-flight completion, token import fallback, managed `/115` provisioning, and admin-only authorization APIs.
- Added Addition-only Token Pair persistence with bounded retry, visible persistence state, latest-pair retry semantics, and DB narrow-write coverage.
- Verified SDK tests, repeated stress tests, and Core compatibility checks; Race validation is environment-blocked because the local Windows toolchain lacks `gcc`.

Blocked:

- The Desktop installer bundle remains unverified because the Windows signing step failed in DEV-001; Packaging DEV owns that issue.
- No real 115 account login, token refresh, WebDAV mount, WinFsp drive mount, VLC playback, or media acceptance run was performed. This is intentional for DEV-001.

Next:
DEV-005

Managed WebDAV + Localhost Service Profile
