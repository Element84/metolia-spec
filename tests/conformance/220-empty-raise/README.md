# 220 — Empty raise

A bare `Raise` reached with no active failure has nothing to re-emit. The frame
completes with `System.EmptyRaise`, a concrete and matchable failure rather than
a silent no-op.

## Checks

- A bare `Raise` outside any handler path produces a failure Result of type
  `error` and code `System.EmptyRaise`.

Reference: Step actions § System.EmptyRaise.
