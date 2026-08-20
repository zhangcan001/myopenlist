# DEV-003 — 115 Refresh Resilience

## 1. Scope

DEV-003 hardens the vendored `115-sdk-go` refresh path. It covers stale-response protection, typed refresh failures, bounded retry, rate-limit cooldown, circuit state, cancellation, status inspection, and deterministic offline tests.

It does not implement QR login, token persistence, Desktop behavior, Core behavior, WebDAV, WinFsp, or real 115 account access.

## 2. Refresh contract

There is at most one refresh flight per SDK `Client`. Authenticated requests join that flight. Public `RefreshToken` also enters the coordinator while holding the token lock, so it cannot observe a snapshot-to-flight registration gap.

The raw Passport request is side-effect free. Only the coordinator may accept the returned Token Pair and invoke the existing callback.

## 3. Stale-response CAS

Each flight captures `startGeneration` and `refreshToken` under `tokenMu` before the network call. A response is accepted only when both values still match the current client state.

If either value changed, the response is marked superseded, the current Token Pair is preserved, no callback runs, and the caller receives `ErrRefreshSuperseded`. The auth-401 path treats that result as “no refresh required” and retries once with the current token state.

An explicit refresh arriving during a stale flight waits for that flight to finish, then starts a new flight with the current refresh token. Refresh requests are never concurrent for one client.

## 4. Error categories

Refresh errors expose one of: `CONTEXT`, `NETWORK`, `RATE_LIMIT`, `SERVER`, `AUTH_REQUIRED`, `PERMISSION`, `UNKNOWN`, or `SUPERSEDED`.

HTTP 401/403/429/5xx and Passport codes 401, exact `40140116`, 403, 429, and 5xx are classified conservatively. Context cancellation/deadline and wrapped `net.Error` values retain their categories. Ordinary authenticated-request 99/401-prefix handling remains `IsAuthFailureCode`.

`PassportError` also preserves HTTP status and `Retry-After` metadata for refresh policy decisions.

## 5. Retry policy

The default engineering policy is:

| Setting | Default |
|---|---:|
| Max attempts | 3 |
| Base backoff | 500 ms |
| Maximum backoff | 4 s |
| Jitter | ±20% |
| Server/network circuit open | 15 s |
| Rate-limit fallback cooldown | 30 s |

Only `NETWORK` and `SERVER` failures retry. Backoff is exponential, capped, jittered, and slept with context cancellation. These are local engineering defaults, not claims about official 115 limits.

## 6. Rate limiting

`RATE_LIMIT` never immediately retries. The circuit opens until `Retry-After` seconds or HTTP date when present; otherwise it uses the bounded fallback cooldown. The original response metadata remains available through the wrapped error.

## 7. Circuit states

The refresh circuit is `CLOSED`, `OPEN`, `HALF_OPEN`, or `AUTH_REQUIRED`.

- `CLOSED`: normal refresh and bounded transient retries.
- `OPEN`: fast-fail until `RetryAt`; after expiry exactly one flight becomes the half-open probe.
- `HALF_OPEN`: one probe flight; concurrent callers join it.
- `AUTH_REQUIRED`: permanent fast-fail for the current Token Pair.

Successful Token Pair replacement closes the circuit. A changed refresh token or `CodeToToken` pair replacement also closes it. Access-token-only replacement deliberately does not clear `AUTH_REQUIRED`.

## 8. Token replacement

Access and refresh tokens are written atomically under the existing token lock. An accepted refresh increments generation once and updates both values together. A stale response cannot mutate either value, generation, circuit state, or callback state.

## 9. Callback contract

The existing refresh callback runs once, after an accepted successful pair and after releasing `tokenMu`. It does not run for retryable failures, rate limits, circuit fast-fails, permanent auth failures, cancellations, or superseded responses.

## 10. Cancellation

Cancellation is checked before starting a flight, while waiting for a flight, during retry backoff, and by the HTTP request context. A canceled retry does not open the circuit and does not start another attempt.

## 11. Status

`Client.RefreshStatus()` returns only `State`, `LastErrorKind`, and `RetryAt`. It is read-only and thread-safe; token values and generation are intentionally excluded.

## 12. Offline test coverage

The SDK tests use in-memory local HTTP handlers and deterministic injected clock/sleeper/randomness. Coverage includes concurrent singleflight, stale CAS, explicit-refresh ordering, error classification, bounded backoff, retry exhaustion, cancellation, `Retry-After`, half-open single-probe behavior, auth-required fast-fail/reset, callback cardinality, token atomicity, and status redaction.

## 13. CI

`.github/workflows/115-sdk-tests.yml` runs the refresh-focused offline test subset on Ubuntu with both normal and `go test -race` modes. It does not call 115 and does not require secrets. The repository’s full SDK package also retains its existing unit tests.

## 14. Boundary and next task

DEV-003 is complete when the SDK tests, repository compatibility checks, and CI workflow are committed. DEV-002, DEV-002.1, and DEV-002.2 remain frozen.

The next task is DEV-004 — 115 Authorization + Token Persistence Integration. No DEV-004 implementation is included here.
