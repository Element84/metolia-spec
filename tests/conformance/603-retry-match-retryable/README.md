# 603 — Retry policy matching on the retryable signal

A flaky mock dispatch fails twice before succeeding, with the `retryable` signal
taken from a Flow parameter. The single retry policy matches `Conformance.*`
codes that also assert `retryable: true`: when the failures assert it, the
policy consumes them and the third attempt's success emerges; when the signal is
unset, no policy matches and the first failure passes through as the Flow's
Result.

## Checks

- A policy's `match` is a failure matcher: every member present must match for
  the policy to handle a rising failure.
- A failure whose `retryable` is unset does not match a policy asserting
  `retryable: true`; an unmatched failure passes through.

Reference: Middleware providers § The Retry middleware; Steps and step mechanics
§ Failure matching.
