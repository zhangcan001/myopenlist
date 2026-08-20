# 115 authentication flow audit

This document preserves the DEV-001 baseline audit. The vendored SDK implementation now includes the DEV-002 coordination delta described below.

## Scope and evidence

This is a static audit of the local core source and the local SDK source for `github.com/OpenListTeam/115-sdk-go v0.2.6`. No 115 login, real token, network API call, source modification, or system-component installation was used.

- Core module: `C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core`
- SDK source audited: `C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6`
- Source citations below use absolute Windows paths and line numbers.

## DEV-002 implementation delta

- `Client` protects token state with a mutex and tracks a token generation.
- Authenticated requests capture the generation used for their request.
- Concurrent auth failures share one in-flight refresh; requests that observe a newer generation skip a second refresh and retry with the new access token.
- Waiters receive the same refresh success or failure, and the refresh callback runs once per committed token pair.

The remaining capability table and source citations describe the pre-DEV-002 v0.2.6 baseline.

## Dependency status

`core/go.mod` requires `github.com/OpenListTeam/115-sdk-go v0.2.6` at line 153. The only 115 SDK replacement is commented out at line 325:

```text
// replace github.com/OpenListTeam/115-sdk-go => ../../OpenListTeam/115-sdk-go
```

Therefore the effective status is: **SDK v0.2.6, no active replace**. The SDK source's own `go.mod` identifies module `github.com/OpenListTeam/115-sdk-go` and requires Go 1.23.4 plus `resty.dev/v3 v3.0.0-beta.1` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\go.mod:1-6`).

## Core `115_open` implementation

`Addition` contains `LimitRate`, `PageSize`, `AccessToken`, and `RefreshToken`; both tokens are required configuration fields (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\meta.go:8-18`). The driver config selects `driver.LinkCacheUA` as `LinkCacheMode` and uses root `0` (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\meta.go:20-24`).

`Init` constructs one SDK client with the stored refresh token, access token, and `sdk.WithOnRefreshToken`. The callback replaces `d.Addition.AccessToken` and `d.Addition.RefreshToken`, then calls `op.MustSaveDriverStorage(d)` (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:43-50`). `Init` then calls `UserInfo`; any error aborts initialization (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:51-57`). It also creates a per-driver `rate.Limiter` only when `LimitRate > 0` (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:58-60`), defaults `PageSize` to 200, and clamps values above 1150 (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:61-65`).

The request-facing methods use that limiter as follows:

- `WaitLimit` calls `d.limiter.Wait(ctx)` when a limiter exists; otherwise it returns immediately (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:92-97`). This is a token-bucket rate limit, not auth-refresh coordination.
- `List` waits before each `GetFiles` page, passes `Limit=d.PageSize` and an offset, and keeps paging until the accumulated result count reaches the response count (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:103-132`).
- `Get` waits, resolves the path with `GetFolderInfoByPath`, and falls back to parent listing on `sdk.ErrObjectNotFound` (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:167-192`). Its recursive parent fallback also waits before parent lookups (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:195-220`).
- `Link` waits, derives the user agent, calls `DownURL`, and returns the selected URL plus the user-agent header (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:135-164`).

The refresh callback's storage behavior is not transactional at the driver layer. `MustSaveDriverStorage` calls `saveDriverStorage`, logs a save error, and returns no error to the caller (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\internal\op\storage.go:288-294`). The save marshals `GetAddition()` into `storage.Addition` and calls `db.UpdateStorage` (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\internal\op\storage.go:296-308`). No lock or compare-and-swap is visible around the two token assignments or the database update.

## SDK auth API and behavior

### Client token state and options

`Client` stores `accessToken`, `refreshToken`, and one `onRefreshToken` callback as ordinary string/function fields; no synchronization field is present (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\client.go:10-16`). `SetAccessToken` and `SetRefreshToken` unconditionally assign those fields (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\client.go:59-67`). `WithAccessToken`, `WithRefreshToken`, and `WithOnRefreshToken` are option functions that call the corresponding setters (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\option.go:31-46`).

### Authenticated request and refresh

`AuthRequest` delegates to `authRequest(..., extractData=true, retry=false)` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:71-73`). `authRequest`:

1. Builds a request and sets the access token only when it is non-empty (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:37-43`).
2. Returns transport/request errors directly (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:45-47`).
3. Treats `Resp.State == false` with code `99` or any code whose decimal representation starts with `401` as an auth failure when this is the first attempt (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:48-50`; `Is401Started` is just a string-prefix test at `C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\utils.go:24-27`).
4. Calls `RefreshToken`; if refresh succeeds, recursively retries the original request once with `retry=true` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:49-55`).
5. Does not refresh on a second failure; it returns `&Error{Code, Message}` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:54-57`). Successful responses unmarshal either `resp.Data` or the raw response according to `extractData` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:58-68`).

`RefreshToken` posts the current refresh token to `https://passportapi.115.com/open/refreshToken`, replaces both in-memory token fields, and invokes the callback with the new pair only after the passport response succeeds (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:78-93`). Passport errors are checked by `passportRequest`: transport errors return directly; nonzero `Code` or nonempty `Error` become formatted errors (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:19-35`).

### Device-code / QR primitives

The SDK defines these endpoints: `authDeviceCode`, `qrcodeapi.115.com/get/status/`, `deviceCodeToToken`, and `refreshToken` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\const.go:8-14`).

- `AuthDeviceCode(clientID, codeVerifier)` posts `client_id`, a SHA-256/base64 code challenge, and `code_challenge_method=sha256`; it returns `uid`, `time`, `qrcode`, and `sign` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:11-35`). The caller must supply both `clientID` and the verifier; the SDK does not generate or retain them.
- `QrCodeStatus(uid, time, sign)` makes one GET request with those three query parameters and returns `msg`, `status`, and `version` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:38-55`). There is no polling loop, interval, timeout policy, or status-state mapping in this SDK source.
- `CodeToToken(uid, codeVerifier)` posts the UID and verifier and returns access token, refresh token, and expiry (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:57-71`). It sets the SDK's in-memory access/refresh fields (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:72-75`) but does **not** invoke `onRefreshToken`; the adjacent `// TODO: set token?` comment is present in source.

## Current request → auth failure → refresh → callback → storage → retry flow

```text
Open115.Init
  -> sdk.New(RefreshToken, AccessToken, WithOnRefreshToken)
  -> client.UserInfo
  -> sdk.AuthRequest / authRequest
  -> request carries current access token
  -> API returns State=false with code 99 or 401...
  -> one call to client.RefreshToken using current refresh token
  -> set new access + refresh tokens in SDK
  -> invoke Open115 callback
  -> copy tokens into d.Addition
  -> marshal Addition and db.UpdateStorage
  -> retry the original request once with the new access token
```

The same SDK path is used by the file APIs called by `List`, `Get`, and `Link`: `GetFiles` uses `AuthRequestRaw` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\fs.go:117-135`), `GetFolderInfoByPath` uses `AuthRequest` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\fs.go:200-209`), and `DownURL` uses `AuthRequest` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\fs.go:309-318`). `UserInfo` also uses `AuthRequest` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\user_info.go:31-38`). A refresh failure returns the refresh error and does not retry the original request. A non-auth API failure returns the SDK `Error` without refresh. A transport error returns immediately without refresh.

## Concurrency and resilience audit

| Capability | Result | Evidence / meaning |
|---|---|---|
| Mutex | **NO** | No mutex or synchronization field exists in `Client` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\client.go:10-16`). |
| Singleflight | **NO** | Every first-attempt auth failure can call `RefreshToken`; no singleflight primitive exists in the audited SDK. |
| Refresh state | **NO** | No explicit idle/in-flight/succeeded/failed refresh state is stored. |
| Token generation/version | **NO** | Tokens are plain strings; no generation/version field exists. |
| CAS | **NO** | Setters overwrite unconditionally (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\client.go:59-67`); driver assignments are unconditional (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:46-49`). |
| Cooldown | **NO** | No refresh cooldown or minimum refresh interval exists. |
| Backoff | **NO** | No retry delay or backoff is implemented. |
| Jitter | **NO** | No randomized delay is implemented. |
| Circuit breaker | **NO** | No open/closed/half-open failure gate exists. |
| 401 storm protection | **PARTIAL** | `retry` prevents an individual request from refreshing more than once (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:49-55`), but concurrent requests are not coalesced or suppressed. |
| Failure classification | **PARTIAL** | Auth refresh is triggered only by code `99` or a `401` prefix; transport errors and other API codes follow separate direct-error paths, with no typed expiry/invalid-refresh distinction (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\request.go:45-57`, `C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\utils.go:24-27`). |

The core `LimitRate` limiter can reduce request rate for methods that call `WaitLimit`, but it is not a refresh-state mechanism and does not provide the missing storm protection (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:58-60,92-97`).

## QR/device-code feasibility: ten questions

These answers are limited to what the local source proves. Anything requiring live 115 protocol behavior is explicitly marked as requiring official verification.

| # | Question | Answer |
|---:|---|---|
| 1 | Is there a device-code request method? | **YES** — `AuthDeviceCode` posts to the local SDK constant `ApiAuthDeviceCode` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:25-35`, `C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\const.go:8-12`). |
| 2 | Can an integrator provide a client ID? | **YES** — `clientID` is a required method argument and is sent as `client_id` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:25-31`). |
| 3 | Is PKCE challenge computation available? | **YES** — the SDK computes SHA-256 then standard base64 and sends `code_challenge_method=sha256` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:18-23,27-30`). |
| 4 | Does the device-code response expose QR/session material? | **YES** — it exposes `uid`, `time`, `qrcode`, and `sign` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:11-16`). |
| 5 | Can the caller query QR status? | **YES, one request at a time** — `QrCodeStatus` exists (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:44-55`). Polling, timing, and terminal-state behavior are absent. |
| 6 | Are the status request parameters wired? | **YES** — `uid`, `time`, and `sign` are sent as query parameters (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:44-50`). |
| 7 | Can the caller exchange an approved device code for tokens? | **YES** — `CodeToToken` posts `uid` and `code_verifier` and returns both tokens plus `expires_in` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:57-71`). |
| 8 | Does the QR exchange use the existing refresh callback for persistence? | **NO** — `CodeToToken` sets in-memory fields but does not call `onRefreshToken` (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:72-75`); only `RefreshToken` invokes that callback (`C:\Users\ADMIN\AppData\Local\Temp\openlist-115-sdk-go-v0.2.6\auth.go:88-92`). |
| 9 | Does core `115_open` expose a QR/device-code login flow or client-ID field? | **NO** — its addition fields expose only access/refresh tokens, and `Init` only constructs the token-authenticated client (`C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\meta.go:8-18`, `C:\Users\ADMIN\Documents\OpenList-115-MediaDrive\core\drivers\115_open\driver.go:43-55`). |
| 10 | Are the 115 client-ID rules, QR status meanings, expiry, and production endpoint contract proven by this local source alone? | **UNKNOWN — requires official 115 API verification.** The source shows request shapes, not authoritative live semantics. |

### Conclusion

**FEASIBLE WITH OWN CLIENT_ID.** The local SDK contains the request, QR-status, and code-to-token primitives, but the caller must own/provide a valid `client_id` and preserve the verifier. Core `115_open` does not integrate this flow, does not persist `CodeToToken` results through its existing callback, and would need explicit orchestration plus storage handling. Production feasibility and status/expiry semantics remain **UNKNOWN — requires official 115 API verification**.
