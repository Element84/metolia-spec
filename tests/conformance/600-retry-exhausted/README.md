# 600 — Retry: budget exhaustion

A dispatch fails on every attempt under a `Retry` policy with a budget of two.
The policy's budget exhausts and the entry emits
`Provider.Middleware.Retry.Exhausted`, chaining the final attempt's failure as
`previous` and reporting the run count and the exhausted policy's position in
`details`.

## Checks

- `Retry`'s `onFailure` action re-enters the inner scope while the matching
  policy's attempt budget lasts, counting the first run.
- On exhaustion the entry emits `Provider.Middleware.Retry.Exhausted`, type
  `error`, with `previous` the final failure and `details` carrying `attempts`
  and `policy`.

Reference: Middleware providers § The Retry middleware, §
Provider.Middleware.Retry.Exhausted.
