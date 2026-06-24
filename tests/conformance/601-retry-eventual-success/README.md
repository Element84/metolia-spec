# 601 — Retry: eventual success via the assign carry

The "fail twice, then succeed" pattern. The mock computes its `failure` from a
counter variable, and the counter survives across attempts only because the
`Retry` entry's `onFailure` `assign` is the deliberate carry: re-entry restores
the frame's variables to their post-`onEntry` state, then applies the gap's
bindings.

The counter is a string accumulator (`"x"` per failed attempt) so the case needs
no numeric coercion rules, only `size()` and string concatenation.

## Checks

- A `vars` value seeds from a root `parameters` default without caller
  arguments.
- The call's fields re-evaluate per attempt against the carried variables.
- Re-entry restores variables; the gap's `assign` evaluates against the failed
  attempt's state and its bindings take effect on the restored variables,
  chaining across attempts.
- A success obtained on the final attempt is a genuine success: the Step routes
  through `next`.

Reference: Middleware providers § The Retry middleware (re-entry restores
variables); Call providers § The mock provider (the fail-then-succeed pattern).
