# V1 simple UI

The default screen exposes only:

- 115 account state: authorized / authorization required / recovering;
- local drive letter, for example `R:`;
- one connection state: Starting, Ready, Degraded, Reconnecting, or Needs authorization;
- **Open 115 drive**;
- **Reconnect 115**.

The simple mode does not expose WebDAV URL, WebDAV username/password, Rclone remote name, Rclone flags, OpenList port, `LimitRate`, AccessToken, or RefreshToken. Advanced mode may retain the official Desktop configuration surface, but it is hidden by default.

User-facing messages should describe the next action (“Waiting for network”, “WinFsp must be installed”, “115 authorization required”) rather than exposing process IDs, raw flags, or tokens.
