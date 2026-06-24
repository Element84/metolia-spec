---
title: "Fan-out and partial success"
weight: 20
---

## Problem

You have a collection — files to process, IDs to fetch, records to transform —
and want to run an operation on each element concurrently, then route on the
outcome: sometimes "all must succeed", sometimes "enough succeeded", and either
way downstream Steps need to know what happened to which element.

## Pattern

A `Gather` in the iterate form, with the outcome policy stated in `completion`
and the downstream view shaped from `step.results`:

```json
"check-inventory": {
  "action": "Gather",
  "over": "{{ step.input.lineItems }}",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "POST", "path": "/inventory/check" }
  },
  "concurrency": 10,
  "completion": { "successes": "{{ step.metadata.dispatchCount }}", "wait": true },
  "output": "{{ step.results.map(r, r.type == 'success' ? r.value : null) }}",
  "next": "summarize",
  "catch": [
    { "match": { "codes": ["System.GatherCompletionUnmet"] }, "next": "report-shortfall" }
  ]
}
```

Each line item becomes one dispatch of the call template, arriving as that
dispatch's `call.input`; at most ten run at once; and when the fan-out
completes, `step.results` holds one Result per item, in item order, whatever
each outcome was.

## Why this shape

The policy is explicit, and it is the only failure path. A dispatch's failure
never trips the `Gather` directly; it makes the policy harder to meet. The
`completion` above restates the default (every dispatch must succeed) to make
the contract visible; when it becomes unachievable, the `Gather` fails with
`System.GatherCompletionUnmet`, whose `details` carry each failed dispatch's
index and Result — which is what the `catch` clause routes on.

`step.results` is the honest record. The default `output` projects just the
succeeded values, which is right when success is all-or-nothing. Under partial
success it silently drops the failed positions, so this pattern shapes its own
`output`: mapping to `value`-or-`null` keeps the output positionally aligned
with the input collection, letting `summarize` pair outcomes back to line items.

`wait: true` runs everything to completion. With the default policy a single
failure already determines the outcome; waiting anyway means `step.results`
records every element's true result, which matters when the downstream step
reports or compensates per element.

## Variations

- **Enough is enough.** `"completion": { "successes": 3, "wait": false }`
  succeeds on the third success and cancels the rest: cancelled dispatches
  resolve `System.GatherDispatchCancelled` and never-started ones
  `System.GatherDispatchSkipped`, each still holding its slot in `step.results`.
  Use it when the dispatches are safe to interrupt — and don't when they aren't.
- **Partition instead of aligning.** When downstream wants the failures as a
  collection, shape both projections:
  `{{ {'ok': step.results.filter(r, r.type == 'success').map(r, r.value), 'failed': step.results.filter(r, r.type != 'success')} }}`.
- **Accumulate across dispatches.** Arms run at fan-out completion in dispatch
  order, so an `onSuccess` arm can build state deterministically:
  `"assign": { "ids": "{{ vars.ids + [call.result.value.id] }}" }`.
- **Fixed set instead of a collection.** The scatter form (`calls`) runs
  different targets side by side under the same policy and record; everything
  above applies unchanged.
- **Per-element retry.** Wrap the operation in an inline flow target carrying
  the retry stack; see [Wrapping work in a subflow](../subflow-wrapper/).

## See also

- [`Gather`](/reference/step-actions/#gather) — the forms, the dispatch model,
  and the resolution guarantee.
- [`completion`: the completion policy](/reference/step-actions/#completion-the-completion-policy)
  — `successes`, `wait`, and the cancellation mechanics.
- [The collected Results](/reference/step-actions/#the-collected-results-stepresults)
  — projections over `step.results`.
- [Saga-style compensation](../saga/) — when partial failure needs explicit
  undo.
