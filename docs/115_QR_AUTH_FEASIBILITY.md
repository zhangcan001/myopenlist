# 115 QR / device-code authorization feasibility

This is a static interface audit only. No 115 login, real token, or live API call was performed.

## Answers

1. **`client_id` source:** caller supplied; `AuthDeviceCode(clientID, codeVerifier)` sends it as `client_id`. The local source does not identify where an approved production value is obtained.
2. **`code_verifier` generation:** caller supplied. The SDK computes the SHA-256/base64 code challenge but does not generate, retain, or restore the verifier.
3. **Existing helper:** challenge computation exists in `auth.go`; no verifier generator, polling loop, timeout, or status mapper exists.
4. **Core usage today:** no. `115_open` currently accepts AccessToken/RefreshToken and initializes a token-authenticated client only.
5. **Desktop QR UI today:** no 115-specific QR/device-code UI was found.
6. **Own 115 Open app/client ID:** required in the practical design, but the local source cannot prove 115’s registration policy. Marked `UNKNOWN — requires official 115 API verification` for policy details.
7. **Installer configuration:** technically possible because the SDK method accepts a string; whether a client ID may be redistributed in an installer is `UNKNOWN — requires official 115 API verification` and release-policy review.
8. **Client secret:** no `client_secret` parameter or use appears in the audited SDK methods. Whether a separate official flow requires one is `UNKNOWN — requires official 115 API verification`.
9. **Successful exchange result:** `CodeToToken` returns `AccessToken`, `RefreshToken`, and `ExpiresIn`.
10. **Direct storage use:** the pair matches the existing `115_open` Addition fields, but `CodeToToken` does not invoke `WithOnRefreshToken`; explicit Core storage persistence is required. Live acceptance of the pair is `UNKNOWN — requires official 115 API verification`.

## Current interface chain

```text
AuthDeviceCode(client_id, code_verifier)
  -> uid/time/qrcode/sign
  -> caller polls QrCodeStatus(uid,time,sign)
  -> caller detects approved status
  -> CodeToToken(uid, code_verifier)
  -> access_token + refresh_token + expires_in
  -> explicit OpenList storage persistence required
```

Evidence: SDK `auth.go:11-75`, `const.go:8-14`; Core `core/drivers/115_open/meta.go:8-18` and `driver.go:43-50`.

## Conclusion

**FEASIBLE WITH OWN CLIENT_ID.** The SDK exposes enough request primitives, but a future implementation must own the client ID, generate and retain a verifier, map QR terminal states, persist `CodeToToken` results, and verify live 115 protocol semantics through official API documentation before shipping.
