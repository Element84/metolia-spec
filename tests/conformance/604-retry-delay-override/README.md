# 604 — Per-failure delay override

A flaky mock dispatch fails once, carrying a server-supplied retry-after hint in
`details`, then succeeds. The policy's backoff schedule says to wait an hour;
the phase's `delay` parameter reads the hint—`PT0S`—and the gap waits that
instead. The discriminating observable is the gap's duration: an implementation
that ignores the override waits the schedule's hour and times out any reasonable
harness bound.

## Checks

- The `onFailure` `delay` parameter evaluates per rising failure, with the
  failure in scope.
- A non-null `delay` is the consumed failure's gap, used as-is in place of the
  matched policy's backoff delay.

Reference: Middleware providers § The Retry middleware.
