# 115 Open driver audit

This is a static audit of the unmodified upstream Core source. No 115 account, token, storage, or live API was used.

## Driver surface

| Surface | Current behavior | Evidence |
|---|---|---|
| `Init` | Builds one SDK client with `RefreshToken`, `AccessToken`, and `OnRefreshToken`; calls `UserInfo`; creates a limiter when configured; defaults and clamps page size | `core/drivers/115_open/driver.go:43-65` |
| `List` | Waits on the driver limiter before each page, calls `GetFiles`, and continues by offset until the reported count is collected | `core/drivers/115_open/driver.go:103-132` |
| `Get` | Resolves a path with `GetFolderInfoByPath`; on object-not-found, falls back to parent listing and matching | `core/drivers/115_open/driver.go:167-227` |
| `Link` | Waits on the limiter, derives the user agent, calls `DownURL`, and returns the URL plus user-agent header | `core/drivers/115_open/driver.go:135-164` |
| `WaitLimit` | Calls `rate.Limiter.Wait(ctx)` when a limiter exists; otherwise returns immediately | `core/drivers/115_open/driver.go:92-97` |
| `LimitRate` | Default is `1` in the driver metadata; non-positive values disable the limiter | `core/drivers/115_open/meta.go:8-18`; `driver.go:58-60` |
| `PageSize` | Default is `200`; values above `1150` are clamped to `1150` | `core/drivers/115_open/meta.go:8-18`; `driver.go:61-65` |
| `LinkCacheMode` | Uses `driver.LinkCacheUA` | `core/drivers/115_open/meta.go:20-24` |
| `AccessToken` | Required Addition field and passed to SDK client initialization | `core/drivers/115_open/meta.go:8-18`; `driver.go:43-45` |
| `RefreshToken` | Required Addition field and passed to SDK client initialization | `core/drivers/115_open/meta.go:8-18`; `driver.go:43-45` |
| `WithOnRefreshToken` | Callback writes both new token strings back into Addition and calls `op.MustSaveDriverStorage` | `core/drivers/115_open/driver.go:44-50` |
| Storage save | Serializes the current Addition and calls the database update path; save errors are logged by `MustSaveDriverStorage` rather than returned | `core/internal/op/storage.go:288-308` |

## Current configuration

The driver metadata exposes `OrderBy`, `OrderDirection`, `LimitRate`, `PageSize`, `AccessToken`, and `RefreshToken`. There is no QR/device-code or `client_id` field in the current `115_open` Addition. The driver root is `0`, and the link cache mode is `LinkCacheUA` (`core/drivers/115_open/meta.go:8-24`).

## Boundary and risks

- The limiter is a request-rate control mechanism, not a token refresh coordinator.
- The driver assigns refreshed access and refresh tokens and persists them through the existing callback, but the audited path shows no mutex, generation check, or compare-and-swap around these assignments and saves.
- `Init` performs `UserInfo`, so driver initialization depends on a usable token pair and remote API reachability.
- QR/device-code authorization is not integrated here; the current driver expects tokens already present in storage.

No driver source was modified in DEV-001.
