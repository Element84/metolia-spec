# 210 — Raise constructs a failure

A terminal `Raise` constructs the frame's failure Result from its `result`
fields. With no failure active, nothing is chained: the envelope is exactly what
the author wrote.

## Checks

- `Raise` completes the Flow with the constructed failure envelope.
- Every authorable envelope field (`type`, `code`, `message`, `details`,
  `retryable`) carries through as written.
- No `previous` is linked when no failure is active.

Reference: Step actions § Raise; The Call interface and Result § The failure
envelope.
