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

These are architecture decisions only; no DEV-002 token coordinator or QR login was implemented.

- **ADR-015:** Use a single-repository source layout. OpenList Core, Desktop, and the 115 SDK enter `myopenlist` as squashed subtree/vendor source. This supports one clone, branch, CI, release, and coordinated review.
- **ADR-016:** Prefer a project-level portable Go toolchain. It is reproducible, requires no administrator permission, and avoids permanent system `PATH` or registry changes.
