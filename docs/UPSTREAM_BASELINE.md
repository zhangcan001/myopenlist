# DEV-001 upstream baseline

Captured 2026-08-20 (Asia/Shanghai) from clean official clones. No upstream source was modified.

| Component | Repository | Remote | Default/current branch | HEAD | Latest tag observed | HEAD commit date |
|---|---|---|---|---|---|---|
| Core | `OpenListTeam/OpenList` | `https://github.com/OpenListTeam/OpenList.git` | `main` / `main` | `0b1e9d0943780fec5d48ffffb25bf2ce2076f09a` | `beta` | `2026-08-19T20:38:12+08:00` |
| Desktop | `OpenListTeam/OpenList-Desktop` | `https://github.com/OpenListTeam/OpenList-Desktop.git` | `main` / `main` | `7efcb538e7128a3281011604146137c3d178b369` | `v0.9.1` | `2026-07-26T06:00:48+08:00` |

`origin/HEAD` resolves to `origin/main` for both clones. `git status --short` was empty immediately after cloning.

## 115 SDK

- Module: `github.com/OpenListTeam/115-sdk-go v0.2.6` from `core/go.mod:153`.
- `go.sum` contains the v0.2.6 module checksum at `core/go.sum:48-49`.
- No active `replace` is present. The only SDK replacement is commented out at `core/go.mod:325`.
- Official tag `v0.2.6` resolves to commit `7799bb98e73949fc902c93c689677b1e640c365c` (checked out separately under the OS temp directory for read-only audit).
- SDK module declares Go `1.23.4` and `resty.dev/v3 v3.0.0-beta.1` in its own `go.mod`.

## Reproducibility commands

```text
git remote -v
git branch --show-current
git symbolic-ref --short refs/remotes/origin/HEAD
git rev-parse HEAD
git tag --sort=-creatordate
git describe --tags --abbrev=0
git show -s --format='%cI' HEAD
```

## Repository integration

- Core integrated as a squashed subtree at `core/`, pinned to `0b1e9d0943780fec5d48ffffb25bf2ce2076f09a`.
- Desktop integrated as a squashed subtree at `desktop/`, pinned to `7efcb538e7128a3281011604146137c3d178b369`.
- 115 SDK integrated as a squashed subtree at `third_party/115-sdk-go/`, pinned to tag `v0.2.6`, commit `7799bb98e73949fc902c93c689677b1e640c365c`.
- The parent repository no longer tracks `core` or `desktop` as gitlinks and no longer has `.gitmodules` entries for them.
