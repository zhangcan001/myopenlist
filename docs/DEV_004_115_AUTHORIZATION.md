# DEV-004 — 115 Authorization + Token Persistence Integration

Project: OpenList 115 Media Drive
Architecture: V3.1
Status: DEV-004.1 audit complete; offline verification complete; REAL_115_NOT_YET_TESTED.

## 1. Scope

DEV-004 adds an admin-only Core authorization service for the existing 115 SDK and connects successful authorization or token import to one managed `/115` storage. It does not change Desktop, WebDAV, Rclone, WinFsp, VLC, or the frozen DEV-002/DEV-003 SDK refresh implementation.

## 2. Provider boundary

`core/internal/media_drive/auth115` owns the `AuthProvider` interface and the session state machine. `SDKAuthProvider` is the only production adapter and delegates to the existing SDK methods `AuthDeviceCode`, `QrCodeStatus`, and `CodeToToken`. Offline tests use fakes and do not call 115.

## 3. PKCE and session security

The service creates a 32-byte random URL-safe session ID and a 32-byte random URL-safe PKCE verifier using `crypto/rand` and `base64.RawURLEncoding`. The verifier, UID, and QR sign are retained only in the in-memory session map. The session TTL is 10 minutes and at most four non-terminal sessions are active at once.

The API never returns the verifier, UID, sign, access token, refresh token, client ID, or app secret. Session state is one of `CREATED`, `WAITING`, `EXCHANGING`, `PERSISTING`, `READY`, `CANCELED`, `EXPIRED`, or `FAILED`.

## 4. Client ID

`OPENLIST_115_CLIENT_ID` overrides the configured `BuiltinClientID`. The open-source default is empty, so a missing approved client ID returns `CONFIG_REQUIRED` before the provider is called. No third-party client ID is embedded in this repository.

## 5. QR status

`GetQRStatus` returns the provider’s numeric status, message, and version unchanged. Core does not guess or invent QR status meanings and does not auto-complete based on an undocumented integer.

## 6. Explicit completion and single exchange

`POST /auth/complete` is the only operation that calls `CodeToToken`. A session transitions through `EXCHANGING` and `PERSISTING`; concurrent completion calls wait on the same session completion channel. A successful session becomes `READY` and later completion calls return the stored storage result without another exchange.

## 7. Token import fallback

`POST /auth/import` accepts an access/refresh Token Pair for deployments where QR authorization is unavailable. The pair is passed directly to the storage provisioner and is never included in the response.

## 8. Managed storage provisioning

The product-managed target is:

- MountPath: `/115`
- Driver: `115 Open`
- RootID: `0`
- LimitRate: `1`
- PageSize: `200`

If `/115` does not exist, the service creates one using `_115_open.Addition`. If it exists with driver `115 Open`, only `access_token` and `refresh_token` are replaced in the existing Addition JSON. RootID, ordering, rate, page size, and unknown Addition keys are retained. Storage-layer fields such as remark, proxy, ordering, cache policy, and sign settings are not reconstructed.

If `/115` belongs to another driver, the service returns `STORAGE_CONFLICT` and does not delete, change, or overwrite that row. A create/init failure is returned as `STORAGE_INIT_FAILED`; the service does not retry creation and risk a duplicate row.

## 9. Token persistence

Refresh callbacks in `Open115` update the in-memory Token Pair under `tokenPersistenceMu`, increment `tokenPersistenceGeneration`, then persist a copied Addition through `SaveStorageAdditionSnapshot`. A non-empty pair loaded during initialization starts at generation `1`; Import and CodeToToken both use the same atomic Pair provisioning path. Each save records its generation and abandons stale work before another attempt. A write mutex serializes database writes so a newer Pair is the final writer. That path calls `db.UpdateStorageAddition`, which updates only the `addition` column. The existing `MustSaveDriverStorage` behavior remains available for unrelated driver lifecycle saves.

Persistence uses exactly three local attempts with delays of `0`, `100ms`, and `500ms`. The pair in memory remains authoritative if all attempts fail; the observable persistence state becomes `FAILED`, and the log emits only `115 token persistence failed` with `storage_id` and `state` fields. Token and authorization material is never included. A later retry saves the current pair, never an older snapshot. There is no claim of crash-safe transactional coordination across the SDK, database, and process boundary.

## 10. Failure and response safety

Stable `AuthError` codes include `CONFIG_REQUIRED`, `PROVIDER_ERROR`, `SESSION_EXPIRED`, `STATE_CONFLICT`, `EXCHANGE_FAILED`, `PERSISTENCE_FAILED`, `STORAGE_CONFLICT`, and `STORAGE_INIT_FAILED`. API error messages expose only the stable code; provider/database causes are not returned. No token-bearing request or provider response is logged by the new service.

## 11. API routes

All routes are nested below the existing `AuthAdmin` middleware:

```text
GET  /api/admin/media-drive/115/auth/capabilities
POST /api/admin/media-drive/115/auth/start
GET  /api/admin/media-drive/115/auth/status?session_id=...
POST /api/admin/media-drive/115/auth/complete
POST /api/admin/media-drive/115/auth/cancel
POST /api/admin/media-drive/115/auth/import
GET  /api/admin/media-drive/115/status
POST /api/admin/media-drive/115/persistence/retry
```

`start` returns only `session_id`, `state`, `qr_code`, and `expires_at`. `status` returns provider passthrough fields and, when ready, safe storage fields. `complete`, import, and retry return only safe storage state.

## 12. Offline verification

The tests cover configuration gating, random sessions, provider status passthrough, concurrent single exchange, idempotent completion, cancellation, expiry, token import, Addition preservation, retry scheduling, latest-pair persistence, and narrow DB updates. No real 115 account, token, or API call is used.

```text
go test ./internal/media_drive/auth115
go test ./drivers/115_open
go test ./internal/db
go test ./server/handles
go test ./internal/op
```

The full SDK regression remains required before handoff:

```text
go test ./...
```

## 13. Production verification requirements

Production acceptance is pending an approved 115 client ID and a controlled real-account test. The remaining checks are real QR start/status/complete, real Token Pair refresh and restart persistence, `/115` WebDAV behavior, Windows Core build, Rclone/WinFsp mounting, and VLC seek/playback acceptance. This task does not claim those checks passed.

`REAL_115_NOT_YET_TESTED` is an explicit release marker: no real 115 account, token, or production API call was used during this audit.
