# VLC / media acceptance plan

No real 115 media test was executed in DEV-001. Final acceptance is based on actual player reads, not merely the presence of `R:`.

## Matrix

- Windows cold boot; Desktop autostart; automatic mount.
- Large directory listing.
- Sequential playback of 20 GB MKV, 50 GB MKV, and 80 GB+ REMUX.
- Seek 5 min → 60 min, 60 min → 10 min, 10 min → near end.
- Pause/resume, subtitle switching, audio-track switching.
- Two simultaneous reads.
- Network disconnect/reconnect; Wi-Fi switch; sleep/resume.
- OpenList crash recovery and Rclone crash recovery.
- AccessToken refresh during listing and playback.
- Concurrent 401 refresh storm.
- 115 rate limiting and temporary server errors.
- Invalid RefreshToken and reauthorization UX.

## Evidence to capture later

For each case record player, file size/container, operation, startup latency, first-frame latency, seek latency, throughput, error text, Core/Rclone logs, health state, and whether recovery was automatic. Test both cold boot and warm recovery. Do not treat “drive exists” or “process exists” as pass criteria.
