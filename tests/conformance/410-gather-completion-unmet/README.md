# 410 — Gather: completion unmet

One dispatch of three fails. Under the default policy every dispatch must
succeed, so the policy is unachievable and the `Gather` fails with
`System.GatherCompletionUnmet`, carrying the evidence in `details`: the
non-success Results paired with their dispatch indexes, and the failure count.

## Checks

- A failing dispatch does not by itself fail the Step; the Step fails when the
  policy becomes unachievable.
- The default `completion` requires every dispatch to succeed, and `wait`
  defaults to `true`: every dispatch runs to completion.
- `details.failures` pairs each non-success dispatch's `index` with its
  `result`; `details.failureCount` counts them.

Reference: Step actions § completion: the completion policy, §
System.GatherCompletionUnmet.
