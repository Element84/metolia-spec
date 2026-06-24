---
title: "Cleanup with Finally"
weight: 60
---

## Problem

Some work must happen on _every_ exit — success, failure, cancellation, a
timeout firing overhead: write the audit record, release the lock, tell the
tracker the run is over. Putting it at the end of the graph misses every path
that doesn't reach the end.

## Pattern

A `Finally` middleware entry. Its cleanup is an ordinary `call` object,
dispatched at `onAlways`, the one phase guaranteed to run on every outcome once
the entry is established:

```json
"charge-payment": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "POST", "path": "/billing/charge" }
  },
  "middleware": [
    {
      "provider": "mwl:provider.middleware/mwl/finally/v1",
      "onAlways": {
        "with": {
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "input": "{{ {'step': step.name, 'execution': execution.id, 'result': middleware.result} }}",
            "with": { "method": "POST", "path": "/audit" }
          }
        }
      }
    },
    {
      "provider": "mwl:provider.middleware/mwl/retry/v1",
      "onEntry": {
        "with": {
          "policies": [{ "match": { "codes": ["Provider.Call.*"] }, "attempts": 3 }]
        }
      }
    }
  ],
  "next": "done",
  "catch": [{ "match": { "codes": ["*"] }, "next": "failed" }]
}
```

## Why this shape

`onAlways` is the everything phase. It runs after `onSuccess` or `onFailure`,
whatever the Result's type — and it runs during teardown too: when an enclosing
timeout preempts the scope or the execution is cancelled from outside,
established entries' `onAlways` phases still run on the way out. That teardown
guarantee is what no Step in the graph can offer, and it is the difference
between cleanup and a step you hope runs.

The cleanup sees the outcome but cannot touch it. The cleanup call's `input`
reads `middleware.result`, the full Result in flight at the entry's position —
success value or failure envelope, whichever it is. The cleanup's own Result is
discarded: `onAlways` stands outside the data plane, so an audit write can never
corrupt the value passing through. The one exception is honest: a cleanup that
_fails_ supersedes the Result in flight, with what it displaced chained as
`previous` — a failed release is real and surfaces rather than being swallowed.

Position decides what the cleanup observes. `Finally` outermost (as above) sees
the final outcome: one audit record after `Retry` has done its worst. Innermost,
it would run per attempt. The same position rule as every stack.

## Variations

- **Flow-level cleanup.** The same entry on a Flow's `middleware` runs once per
  frame, on every way the Flow can end. This is the natural home for
  per-workflow audit records and resource release tied to the run rather than
  one Step.
- **Cleanup for `Gather` work.** A dispatch that needs teardown cleanup targets
  a Flow whose own stack carries the `Finally`; the cleanup then runs inside the
  dispatch's frame on every exit, including when the `Gather` cancels the
  dispatch under `wait: false`.
- **Capture what the cleanup learned.** The cleanup call's arms work as always:
  `"onSuccess": { "assign": { "auditId": "..." } }` inside the cleanup `call`
  records its receipt without touching the data plane.
- **Wrap the cleanup itself.** `Finally` accepts a `middleware` parameter, a
  stack around the cleanup dispatch: a timeout so a slow audit endpoint can't
  stall teardown, a retry so a blip doesn't become a superseding failure.
- **Conditional cleanup.** Gate it:
  `"onAlways": { "when": "{{ vars.auditEnabled }}", ... }`.

## See also

- [The `Finally` middleware](/reference/providers/middleware-providers/#the-finally-middleware)
  — parameters and behavior.
- [`onAlways`](/reference/middleware-mechanics/#onalways) — the guarantee, the
  discard rule, and supersession.
- [The unwind](/reference/execution-model/#the-unwind) — what runs during
  teardown.
- [Saga-style compensation](../saga/) — when "undo" depends on which failure
  happened; compensation routes, cleanup always runs.
