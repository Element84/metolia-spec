---
title: "Retry composition"
weight: 30
---

## Problem

An external call fails in more than one way, and the ways deserve different
treatment: rate limiting wants patient, jittered backoff; transient connectivity
wants a couple of quick attempts; a declined card wants no retry at all. And
whatever the policy, something has to bound how long the whole affair may take.

## Pattern

One `Retry` entry with multiple policies, composed with `Timeout` at the
position that matches the bound you mean:

```json
"charge-payment": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "POST", "path": "/billing/charge" }
  },
  "middleware": [
    {
      "provider": "mwl:provider.middleware/mwl/retry/v1",
      "onEntry": {
        "with": {
          "policies": [
            {
              "match": { "codes": ["Provider.Call.Http.Throttled"] },
              "attempts": 5,
              "backoff": { "initial": "PT10S", "rate": 2, "max": "PT2M", "jitter": "full" }
            },
            {
              "match": { "codes": ["Provider.Call.Http.ConnectionFailed"] },
              "attempts": 3,
              "backoff": { "initial": "PT1S", "rate": 2 }
            }
          ]
        }
      }
    },
    {
      "provider": "mwl:provider.middleware/mwl/timeout/v1",
      "onEntry": { "with": { "duration": "PT30S" } }
    }
  ],
  "next": "done",
  "catch": [
    {
      "match": { "codes": ["Provider.Call.Payments.CardDeclined"] },
      "next": "notify-customer"
    },
    {
      "match": {
        "codes": [
          "Provider.Middleware.Retry.Exhausted",
          "Provider.Middleware.Timeout.Exceeded"
        ]
      },
      "next": "escalate"
    }
  ]
}
```

## Why this shape

Policies separate failure classes. Policies are scanned in order and the first
match handles the failure, each policy counting its own attempts. Throttling
gets five patient, jittered attempts capped at two minutes apart; connection
blips get three quick ones. A declined card matches no policy, so it passes
through immediately to the `catch` — not retrying is expressed by not matching.

`Retry` outside `Timeout` is a per-attempt budget. The bound is established
afresh for each re-run of the inner scope, so every attempt gets its own 30
seconds; a hung attempt times out, `Provider.Middleware.Timeout.Exceeded` rises,
and if a policy matched it, `Retry` would re-run. (This stack retries only
HTTP-class codes, so a timeout passes through; add a policy on
`Provider.Middleware.Timeout.Exceeded` to retry hangs deliberately.)

`Timeout` outside `Retry` is a total budget. Swap the entries and one 30-second
clock spans every attempt and every backoff delay; when it elapses, the whole
retrying scope is preempted, however many attempts remained. Choose by which
bound you can actually promise: per-attempt cost, or the caller's total
patience.

Exhaustion is a real failure. When the matching policy's budget runs out,
`Retry` emits `Provider.Middleware.Retry.Exhausted` with the final attempt's
failure chained as `previous` — the `catch` routes on it, and the chain
preserves what kept failing.

## Variations

- **Three layers.** A `Finally` entry outermost — `Finally` over `Retry` over
  `Timeout` — audits the final outcome exactly once, after the retrying is done;
  placed innermost it would audit every attempt. Same reasoning as the bound:
  position is meaning.
- **Carry the attempt count.** The entry's metadata is readable only in its own
  phases; to know afterward how many runs it took, capture it:
  `"onSuccess": { "assign": { "attempts": "{{ middleware.metadata.attempt }}" } }`.
- **Disable retries per run.** Gate the entry,
  `"onEntry": { "when": "{{ vars.retriesEnabled }}", "with": ... }` — gated off,
  the entry is transparent; see
  [Conditional middleware](../conditional-middleware/).
- **Respect the advisory signal.** Providers may set `retryable` on failures,
  and a matcher selects on it directly:
  `{ "match": { "codes": ["Provider.Call.*"], "retryable": true }, "attempts": 3 }`
  retries only what the provider asserts is worth retrying. A failure that is
  silent about `retryable` matches neither `true` nor `false`.
- **Honor a server's retry-after.** The `onFailure` block's `delay` parameter
  evaluates per failure, with the failure in scope, and overrides the gap's
  backoff delay when non-null:
  `"onFailure": { "with": { "delay": "{{ has(middleware.result.details.retryAfter) ? middleware.result.details.retryAfter : null }}" } }`.
  The schedule stays the fallback for failures that carry no hint.
- **Select a policy by predicate.** Matchers are deliberately closed over the
  envelope's contract fields; when one failure class is distinguishable only by
  a predicate, gate a single-policy entry with the phase's `when` —
  `"onFailure": { "when": "{{ middleware.result.details.status >= 500 }}" }` —
  and stack a second `Retry` entry for a second predicated class. Each entry is
  one (predicate, policy) pair, evaluated outside-in. For a distinction worth
  naming, reclassify instead: an inner entry's `onFailure` block constructs a
  successor failure with a new code, and the policy matches the name.
- **Whole-subflow retry.** The same entry on a Flow's `middleware` re-runs its
  entire Step graph; variables restore to their post-entry state on each re-run,
  so a retried graph repeats the same work rather than a drifted variant.

## See also

- [The `Retry` middleware](/reference/providers/middleware-providers/#the-retry-middleware)
  — policies, backoff, variable restore, and metadata.
- [The `Timeout` middleware](/reference/providers/middleware-providers/#the-timeout-middleware)
  — acceptance semantics.
- [Ordering and composition](/reference/middleware-mechanics/#the-stack-ordering-and-composition)
  — the general position-is-meaning rule.
- [Polling with timeout](../polling/) — when "try again" is about waiting for
  the world, not surviving a failure.
