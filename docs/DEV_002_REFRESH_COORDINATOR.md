# DEV-002 RefreshCoordinator

## Problem

The SDK previously allowed concurrent authenticated requests to do this:

```text
A -> 401 -> Refresh
B -> 401 -> Refresh
C -> 401 -> Refresh
```

That could rotate the same refresh token more than once and race token persistence.

## Solution

```text
A ─┐
B ─┼-> one Refresh flight
C ─┘
          ↓
   atomic Token Pair
          ↓
   Generation +1
          ↓
   requests retry once
```

## Components

- Token Snapshot reads access token, refresh token, and generation under one read lock.
- Token Pair updates replace both tokens and increment generation under one write lock.
- Token Generation identifies the authentication state used by each request.
- Refresh Flight has one leader and shared `resp`/`err` state for waiters.
- Public `RefreshToken` enters the refresh-flight coordinator directly; it does not snapshot generation before registration.
- A stale 401 from an older generation skips refresh and retries with the current token.
- Waiters may stop waiting through their own context without canceling the leader.

## Failure semantics

Refresh failure is shared only by callers that joined the current in-flight flight. The flight is cleared after completion, so a later request may start a new flight. DEV-003 adds the separate bounded retry, cooldown, and circuit policy without changing this DEV-002 singleflight contract.

## Not implemented

- QR login
- Token persistence integration

Backoff, jitter, cooldown, circuit breaker, and advanced refresh-error classification are implemented in [DEV-003 Refresh Resilience](DEV_003_REFRESH_RESILIENCE.md). QR login and token persistence integration remain later work.
