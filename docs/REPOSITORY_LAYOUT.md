# Repository layout and upstream workflow

## Layout

```text
core/
desktop/
third_party/115-sdk-go/
docs/
scripts/
```

`core/`, `desktop/`, and `third_party/115-sdk-go/` are ordinary files in the `myopenlist` repository. They are integrated from official upstream repositories with squashed `git subtree` imports, not long-term Git submodules.

## Why a single repository

This project may need coordinated changes across OpenList Core, OpenList Desktop, and `115-sdk-go`. A single clone, branch, build, CI, and release surface lets those changes be reviewed and committed together. A submodule-only layout would leave source changes in separate repositories and make the project difficult to build and release as one unit.

## Upstream

- Core URL: `https://github.com/OpenListTeam/OpenList.git`
- Core pinned commit: `0b1e9d0943780fec5d48ffffb25bf2ce2076f09a`

- Desktop URL: `https://github.com/OpenListTeam/OpenList-Desktop.git`
- Desktop pinned commit: `7efcb538e7128a3281011604146137c3d178b369`

- SDK URL: `https://github.com/OpenListTeam/115-sdk-go.git`
- SDK pinned tag/commit: `v0.2.6` / `7799bb98e73949fc902c93c689677b1e640c365c`

## Development dependency

`core/go.mod` retains:

```text
require github.com/OpenListTeam/115-sdk-go v0.2.6
replace github.com/OpenListTeam/115-sdk-go => ../third_party/115-sdk-go
```

The `require` records the audited upstream version. The `replace` makes local development resolve the same pinned source tree without changing SDK behavior.

## Update policy

Upstream updates require fetch, review, pinned subtree pull, Core/Desktop build and test, and later media acceptance. Do not follow arbitrary `latest` versions automatically. The helper at `scripts/sync-upstream.ps1` is dry-run by default and only applies pulls with explicit `-Apply`.

## Local SDK resolution

Validated with portable Go `go1.26.4`:

```text
github.com/OpenListTeam/115-sdk-go v0.2.6 => ../third_party/115-sdk-go
```

`go list -m -json github.com/OpenListTeam/115-sdk-go` reported:

```text
Path:     github.com/OpenListTeam/115-sdk-go
Version:  v0.2.6
Replace:  ../third_party/115-sdk-go
Dir:      C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\third_party\115-sdk-go
GoMod:    C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\third_party\115-sdk-go\go.mod
```

The module resolves to the local replacement directory, not the Go module cache.
