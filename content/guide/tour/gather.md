---
title: "Concurrency with Gather"
weight: 70
---

Within a Flow, Steps run one at a time. Concurrency is explicit, and it has one
primitive: **`Gather`**, the action that dispatches many calls at once and
collects every Result. Because each dispatch is an ordinary
[`call` object](../calls-and-results/), a fan-out can run providers, subflows,
or a mix, and everything you know about calls applies per dispatch.

## The iterate form: one dispatch per element

The common fan-out runs the same call once per element of a collection:

```json
"process-granules": {
  "action": "Gather",
  "over": "{{ step.input.features }}",
  "call": {
    "flow": "RegisterGranule",
    "with": { "collection": "modis-l1" }
  },
  "concurrency": 10,
  "next": "summarize",
  "catch": [{ "match": { "codes": ["*"] }, "next": "failed" }]
}
```

`over` is evaluated once and must produce an array; each element makes one
dispatch of the `call` template, the element arriving as that dispatch's
`call.input` and its position as `call.index`. The call's fields evaluate afresh
per dispatch, so a `with` reading `call.input` or `call.index` configures each
dispatch from its own element. `concurrency` caps how many dispatches are in
flight at once; absent (or `null`), the fan-out is unlimited, which is rarely
kind to whatever is on the other end.

## The scatter form: a fixed set of dispatches

The other shape runs different calls side by side: `calls`, a literal array of
call objects, one dispatch each, with the Step's received value arriving as
every dispatch's `call.input`:

```json
"enrich": {
  "action": "Gather",
  "calls": [
    {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "GET", "path": "/metadata" }
    },
    {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "GET", "path": "/thumbnails" }
    },
    { "flow": "BuildSummary" }
  ],
  "next": "combine"
}
```

A `Gather` carries exactly one form: `over` with `call`, or `calls`.

## `step.results`: the complete record

When the fan-out completes, `step.results` holds one Result per dispatch, flat,
in dispatch order — element order or `calls` order — whatever each outcome was:
settled successes and failures, dispatches the `Gather` cancelled, dispatches
that never started. Position is preserved, so results line up with inputs.

The Step's default `output` is the success projection,
`{{ step.results.filter(r, r.type == 'success').map(r, r.value) }}`: just the
succeeded values, in order. When you need more — pairing results back to inputs,
partitioning failures — shape `output` over the full record:

```json
"output": "{{ step.results.map(r, r.type == 'success' ? r.value : null) }}"
```

## `completion`: the completion policy

By default every dispatch must succeed: one failure makes the policy
unachievable and the `Gather` fails with `System.GatherCompletionUnmet` (its
`details` carrying the evidence). `completion` makes the policy explicit:

```json
"completion": { "successes": 3, "wait": false }
```

`successes` is how many dispatches must succeed; `wait` governs what happens to
in-flight work once the outcome is determined. With `wait: true` (the default)
everything runs to completion regardless; with `wait: false` the `Gather`
cancels what is in flight (those dispatches resolve
`System.GatherDispatchCancelled`) and abandons what hasn't started (resolved
`System.GatherDispatchSkipped`). "First three of ten win" and "all ten run,
three must succeed" are both one line.

A dispatch's failure is data the `Gather` observes — a slot in `step.results`,
counted by the policy — never something the `Gather`'s `catch` matches. The
`catch` matches only failures the `Gather` itself produces:
`System.GatherCompletionUnmet`, or a fault in its own fields. By the time a
clause runs, every dispatch has resolved and `step.results` is complete.

## The arms run at completion

Each dispatch's call has its arms, and under `Gather` they all run at fan-out
completion, one dispatch at a time, in dispatch order, with that dispatch's
context (`call.result`, the target window) alive. Deterministic order makes
cross-dispatch accumulation safe:

```json
"call": {
  "flow": "RegisterGranule",
  "onSuccess": {
    "assign": { "ids": "{{ vars.ids + [call.result.value.id] }}" }
  }
}
```

The list grows in dispatch order no matter what order dispatches finished in.
While the fan-out is in flight, nothing writes the frame's variables at all:
dispatches are isolated by construction and cannot observe each other.

## Wrapping a dispatch: target a Flow

A `Gather` carries no `middleware`; per-dispatch behavior belongs inside the
dispatch, which means targeting a Flow and putting the behavior in it. The idiom
for per-item retry is an inline flow target whose one `Call` Step carries the
retry stack — each element then retries independently, inside its own frame, and
the `Gather` counts one Result per element however many attempts it took. The
[middleware page](../middleware/) introduces the stack itself; bounding the
whole fan-out (a deadline over everything) is a Flow-level stack on a Flow
containing the `Gather`.

## Where the spec covers this

- [`Gather`](/reference/step-actions/#gather) — both forms, the dispatch model,
  `completion`, `step.results`, and the arms at completion.
- [Frames and sequential execution](/reference/execution-model/#frames-and-sequential-execution)
  — why dispatches are isolated and where concurrency lives.
- [The collected Results](/reference/step-actions/#the-collected-results-stepresults)
  — projections over the record.
