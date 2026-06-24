# 103 — The failure arm: capture from the failed dispatch

The failure arm is capture-only, and it is the one seam where the failed
dispatch's context is alive: the mock fails with window metadata configured, the
`onFailure` arm's `assign` captures `provider.metadata.requestId`, and the
handler the `catch` routes to returns the captured variable.

## Checks

- The `mock` exposes its `metadata` parameter as `provider.metadata` whatever
  the outcome.
- The `provider` window is in scope in the failure arm, and the arm's `assign`
  is the only way its members survive the call execution.
- The arm does not reshape the failure; the envelope ascends as produced and
  `catch` matches it.

Reference: The Call interface and Result § onFailure: capture only, § The target
windows.
