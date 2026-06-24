# 510 — Subflow failure propagation

A subflow completes with a failure it raised, and the failure becomes the
calling Step's Result: a caller consumes a Flow target's failure exactly as a
provider's, and the caller's `catch` matches the propagated code.

## Checks

- An unhandled `Raise` in the subflow completes that frame with the constructed
  failure.
- The failure propagates to the caller unchanged and is matched by the caller's
  `catch` on the authored code (Flow-Call Result parity).

Reference: The Flow object § How a Flow completes; The Call interface and Result
§ Flow-Call Result parity.
