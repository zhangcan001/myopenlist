# DEV-001 decision log

- **ADR-001:** V1 only targets a Windows local media drive.
- **ADR-002:** V1 defaults to read-only media access.
- **ADR-003:** The packaged Core default must be localhost-only.
- **ADR-004:** Reuse the official Desktop Rclone体系; do not fork Rclone.
- **ADR-005:** Do not reimplement download URL caching.
- **ADR-006:** Do not reimplement the 115 API limiter; place refresh coordination at the layer that actually refreshes.
- **ADR-007:** Token concurrency control belongs in the true Refresh execution layer, not in a UI retry loop.
- **ADR-008:** Health checks are layered L0–L4.
- **ADR-009:** Invalid RefreshToken must not cause infinite retry.
- **ADR-010:** Database/token persistence must not silently roll back to an older token.
- **ADR-011:** V1 does not develop a player.
- **ADR-012:** V1 acceptance uses real VLC playback and seek behavior.
- **ADR-013:** Enhanced Core and Desktop versions are bound by a release manifest.
- **ADR-014:** Ordinary users do not see WebDAV/Rclone technical configuration.

DEV-001 architecture decisions are recorded above. DEV-002 and DEV-002.1 implementation decisions are recorded below; QR login remains future work.

- **ADR-015:** Use a single-repository source layout. OpenList Core, Desktop, and the 115 SDK enter `myopenlist` as squashed subtree/vendor source. This supports one clone, branch, CI, release, and coordinated review.
- **ADR-016:** Prefer a project-level portable Go toolchain. It is reproducible, requires no administrator permission, and avoids permanent system `PATH` or registry changes.
- **ADR-017:** Token Pair is atomic authentication state. Internal flows that rotate access and refresh tokens update both under one token lock and increment generation once.
- **ADR-018:** Concurrent token refresh uses one in-flight refresh per SDK Client. Waiters share the leader's result and the refresh callback runs once per successful pair rotation.
- **ADR-019:** Each authenticated request records the Token Generation used for its request.
- **ADR-020:** A stale 401 from an older Generation must never cause another refresh.
- **ADR-021:** Refresh callback executes once per successful Token Pair rotation and runs after the token lock is released.
- **ADR-022:** Refresh failure is shared only by the current in-flight cohort. Retry policy, cooldown, and resilience escalation belong to DEV-003.
- **ADR-023:** Public explicit `RefreshToken` participates directly in RefreshCoordinator. This removes the token snapshot-to-flight-registration TOCTOU window.

DEV-003 refresh resilience decisions:

- **ADR-024:** A refresh flight captures Token Generation and refresh token before network I/O; the response may commit only when both still match. Stale responses return `ErrRefreshSuperseded` and cannot mutate tokens, generation, circuit state, or callbacks.
- **ADR-025:** Refresh failures use a small conservative taxonomy: context, network, rate limit, server, auth required, permission, unknown, and superseded. Existing authenticated-request 99/401-prefix classification remains unchanged.
- **ADR-026:** Only network and server refresh failures use bounded exponential backoff with jitter. Defaults are three attempts, 500 ms base, 4 s cap, and ±20% jitter; these are engineering defaults rather than official 115 limits.
- **ADR-027:** A rate-limited refresh opens the circuit until `Retry-After` or a bounded fallback cooldown. It is never retried immediately.
- **ADR-028:** Refresh circuit state is `CLOSED`, `OPEN`, `HALF_OPEN`, or `AUTH_REQUIRED`; only one half-open probe is allowed, and permanent auth failure fast-fails until a new Token Pair is installed.
- **ADR-029:** A changed refresh token, `CodeToToken`, or an accepted refresh Token Pair resets the circuit. Access-token-only replacement does not clear `AUTH_REQUIRED`.
- **ADR-030:** `RefreshStatus` exposes state, last error kind, and retry time only. It is thread-safe and must not expose access tokens, refresh tokens, or generation.
