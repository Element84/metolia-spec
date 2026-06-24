# 621 — Loop: variables persist

The same bounded loop as case 620, with an `assign` on the continuation phase:
each iteration appends to a variable, and the writes accumulate. An iteration is
progress, not a repeat — `Loop` persists variables where `Retry` restores them
(case 601's contrast).

## Checks

- Variables persist across `Loop` iterations; accumulating in `vars` is
  well-defined loop state.
- A phase's `assign` evaluates on every run of its phase, including the final,
  emitting one.
- The Step's `output` evaluates after the stack emits, observing the accumulated
  state.

Reference: Middleware providers § The Loop middleware (contrast with Retry's
restore).
