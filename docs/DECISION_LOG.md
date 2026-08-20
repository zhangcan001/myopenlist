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
