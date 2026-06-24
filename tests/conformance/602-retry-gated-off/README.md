# 602 — Retry: gated off

The `Retry` entry's `onEntry.when` is false, so nothing is armed: the entry is
transparent, the dispatch runs exactly once, and its failure passes through
unchanged rather than being retried or rewrapped.

## Checks

- `when` gates the phase's action; gated off, the entry is transparent and
  failures pass through.
- A gated-off phase's `with` is not evaluated (the policies are never captured).
- The emerging failure is the dispatch's own, not
  `Provider.Middleware.Retry.Exhausted`.

Reference: Middleware mechanics § when: gating the action; Middleware providers
§ The Retry middleware.
