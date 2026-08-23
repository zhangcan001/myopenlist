# OpenList 115 Client ID Research

Date: 2026-08-23

## Conclusion

OpenList users do not need to register their own 115 Open Platform application when they use the official OpenList token service. The OpenList-provided 115 application ID and secret are held by `api.oplist.org`, not embedded in the OpenList Core binary.

Therefore:

- "OpenList can authorize 115 without the user's own AppID" is correct.
- "The OpenList Core executable contains a built-in AppID" is not accurate for the current upstream source.

## Primary evidence

1. The official [115 Open driver documentation](https://github.com/OpenListTeam/OpenList-Docs/blob/main/pages/guide/drivers/115_open.md) says Open Platform application registration is optional when using the OpenList built-in key pair. It instructs users to visit `api.oplist.org`, select the 115 provider, enable "Use parameters provided by OpenList", leave Client ID and Secret empty, and authorize in the pop-up window.

2. The official [OpenList-APIPages 115 authorization implementation](https://github.com/OpenListTeam/OpenList-APIPages/blob/main/src/driver/115cloud_oa.ts) loads `client_id` and `client_secret` from the token service environment (`cloud115_uid` and `cloud115_key`) when `server_use=true`. The secret is used server-side for the authorization-code exchange and is not returned to the caller.

3. The official [OpenList-APIPages API documentation](https://github.com/OpenListTeam/OpenList-APIPages#%E6%8E%A5%E5%8F%A3%E6%96%87%E6%A1%A3) states that `server_use=true` removes the requirement for callers to provide AppID and Key.

4. The current upstream [115 Open driver metadata](https://raw.githubusercontent.com/OpenListTeam/OpenList/main/drivers/115_open/meta.go) contains only Access Token and Refresh Token fields. It has no Client ID or Client Secret field. The [driver initialization](https://raw.githubusercontent.com/OpenListTeam/OpenList/main/drivers/115_open/driver.go) consumes those tokens and refreshes them locally.

5. A live request to `https://api.oplist.org/115cloud/requests?driver_txt=115cloud_go&server_use=true` succeeded on 2026-08-23 and returned an HTTPS authorization URL hosted by `115.com` without the caller supplying an AppID or Secret.

## Web authorization versus QR PKCE

The official path that currently avoids a user-owned AppID is the hosted authorization-code web flow. The official 115 Open documentation still marks mobile QR PKCE authorization as "not yet implemented".

OpenList-APIPages also contains a generic [115 QR helper](https://github.com/OpenListTeam/OpenList-APIPages/blob/main/src/driver/115cloud_qr.ts), but it only obtains and polls a QR login token; it does not perform the 115 Open OAuth token exchange shown in the hosted authorization-code flow.

## Impact on this project

This project's custom `auth115` service currently implements direct local PKCE and deliberately leaves `BuiltinClientID` empty. That makes it return `CONFIG_REQUIRED`, even though the normal OpenList user experience can use the official hosted credentials.

The compatible default should be:

1. Open the official OpenList-hosted 115 authorization page.
2. Let the user authorize on `115.com`.
3. Import and persist the returned Access Token and Refresh Token through the existing secure token-import path.
4. Keep direct local PKCE with a user-owned AppID as an optional advanced mode.

The hosted flow means the authorization code and resulting tokens are processed by the official OpenList token service. A fully local flow still requires an approved application identity.
