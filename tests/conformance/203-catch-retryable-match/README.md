# 203 — Catch matching on the retryable signal

A mock dispatch fails with the `retryable` signal taken from the input. Three
clauses select on it: one asserting `true`, one asserting `false`, and a
wildcard. Each scenario injects one of the signal's three values.

## Checks

- A matcher's `retryable: true` matches only a failure asserting
  `retryable: true`, and `retryable: false` only one asserting
  `retryable: false`.
- A failure whose `retryable` is unset (`null`) matches neither boolean member
  and falls through to the wildcard clause.

Reference: Steps and step mechanics § Failure matching; The Call interface and
Result § The failure envelope.
