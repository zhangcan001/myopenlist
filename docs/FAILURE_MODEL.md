# V3.1 failure model

Policy: recover locally and remotely when safe; require a decision only for reauthorization, WinFsp/admin installation, or severe database/config corruption.

| ID / failure | Detection | State transition | Automatic action | Retry policy | User notification | User action required | Risk |
|---|---|---|---|---|---|---|---|
| F001 no network at Windows start | local network probe fails | `WAIT_NETWORK` | keep Desktop alive; defer Core/DAV/mount | exponential backoff + jitter | Waiting for network | No | M |
| F002 network appears slowly | probe remains pending | `WAIT_NETWORK` | extend bounded startup window | backoff, no 1s loop | Still waiting for network | No | M |
| F003 Core starts slowly | child alive but `/ping` fails | `WAIT_CORE_READY` | wait; keep mount blocked | bounded 30s attempts | Starting local service | No | M |
| F004 Core start fails | spawn error or repeated `/ping` failure | `DEGRADED`/`RECOVERING` | inspect log; restart if safe | bounded 3 attempts with backoff | Local service unavailable | No unless persistent | H |
| F005 Core crashes while running | child exit watcher/status transition | `RECOVERING` | restart Core, then recheck L1-L4 | backoff + crash-loop guard | Reconnecting local service | No unless loop persists | H |
| F006 Rclone start fails | spawn failure or immediate exit | `RECOVERING` | validate binary/WinFsp/profile; retry mount | bounded | Drive reconnecting | No unless dependency missing | H |
| F007 Rclone crashes while running | child exit watcher/status transition | `RECOVERING` | stop stale mount and remount | backoff + crash-loop guard | Drive reconnecting | No unless loop persists | H |
| F008 WinFsp missing | dependency probe / Rclone error | `FATAL` | do not loop mount attempts | none | WinFsp installation required | Yes, install official WinFsp/admin | H |
| F009 preferred drive occupied | drive enumeration sees R: used | `MOUNTING` | choose next usable letter and persist | once per startup/profile | Using S: because R: is occupied | No, inform | M |
| F010 Core port occupied | `/ping` absent and bind error/log or port probe mismatch | `RECOVERING` | choose a free loopback port in packaged profile or preserve old config for diagnosis | bounded | Local port changed / service unavailable | No unless policy cannot resolve | H |
| F011 WebDAV unavailable | L2 OPTIONS/PROPFIND fails | `CHECKING_WEBDAV` -> `RECOVERING` | recheck Core, rebuild internal remote, remount | bounded backoff | Reconnecting drive | No | H |
| F012 storage missing | Core storage lookup returns not found | `CHECKING_STORAGE` -> `DEGRADED` | stop mount; retain profile; do not create fake storage | low-frequency | 115 storage is unavailable | Yes only if user must choose storage | H |
| F013 storage config damaged | parse/validation failure | `RECOVERING` or `FATAL` | use validated backup; never overwrite valid token with empty data | one recovery path | Storage configuration needs repair | Yes if no valid backup | H |
| F014 AccessToken expired | SDK auth code 99/401-prefix | `CHECKING_AUTH` | run coordinated refresh | one refresh attempt per generation | Reconnecting 115 | No | H |
| F015 concurrent 401 storm | multiple requests enter refresh path | `RECOVERING` | future RefreshCoordinator coalesces refresh and shares result | one in-flight refresh + backoff | Reconnecting 115 | No | H |
| F016 RefreshToken refresh succeeds | refresh response returns token pair | return to prior check | atomically persist pair; retry original request once | no duplicate refresh | Connected | No | M |
| F017 RefreshToken invalid | refresh endpoint rejects/invalid pair | `AUTH_REQUIRED` | stop remote calls and mount; preserve diagnostic state | no automatic infinite retry | 115 authorization required | Yes, scan/authorize again | H |
| F018 115 API rate limit | typed rate-limit response or repeated limiter errors | `DEGRADED` | slow requests; keep local service available | exponential backoff + jitter | 115 is rate-limiting requests | No | H |
| F019 temporary 115 server error | 5xx/typed temporary error | `DEGRADED` | retain mount state, retry safe operation | bounded low-frequency | 115 temporarily unavailable | No | M |
| F020 local disconnect | local network probe or DAV/read failure | `WAIT_NETWORK`/`DEGRADED` | pause remote work; keep UI/tray | backoff | Network disconnected | No | H |
| F021 Wi-Fi switch | interface/network identity changes | `RECOVERING` | re-run L1-L4; remount if stale | one transition recovery | Network changed; reconnecting | No | M |
| F022 Windows sleep | power/session event | `DEGRADED` | pause supervisor timers | resume-aware | Paused during sleep | No | M |
| F023 Windows wake | resume event | `RECOVERING` | run L0-L4; replace stale mount | bounded | Restoring drive | No | H |
| F024 drive exists but unreadable | L4 `read_dir`/read probe fails | `RECOVERING` | stop stale Rclone and remount | one remount then backoff | Drive is recovering | No unless persistent | H |
| F025 stale Rclone mount | process/PID and drive state disagree | `RECOVERING` | unmount/terminate stale process, recreate mount | bounded, crash-loop guarded | Cleaning up stale drive | No | H |
| F026 VLC first large-file open fails | player report plus L4/link logs | `DEGRADED` | refresh link/health, retry one open; keep read-only | one operation retry | First open failed; retrying | No | H |
| F027 VLC large seek fails | player seek test/report | `DEGRADED` | capture diagnostics; do not invent tuning values | no background loop | Seek failed; diagnostics saved | No, unless user chooses profile | M |
| F028 playback URL expires | Core link/read error during playback | `RECOVERING` | refresh URL/token through Core and retry range once | one retry | Reconnecting media stream | No | H |
| F029 software update available | signed Tauri manifest or pinned release manifest | stay `READY` until approved | stage/verify manifest; update during safe window | one staged attempt | Update available | No, approval policy may ask | M |
| F030 update fails | checksum/install/health failure | `RECOVERING`/previous version | restore validated backup; recheck L0-L4 | one rollback | Update failed; previous version retained | No unless rollback impossible | H |
| F031 database corrupt | SQLite/open/migration failure | `FATAL` | preserve copy; validate backup; never destructive reset automatically | one safe recovery | Database repair required | Yes if no valid backup | H |
| F032 app config corrupt | JSON parse/schema failure | `RECOVERING` | preserve bad file, load safe defaults, re-save only after validation | one migration | Settings reset to safe defaults | Only if repair cannot be automatic | M |
