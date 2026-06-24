---
title: "Wrapping work in a subflow"
weight: 80
---

## Problem

Some behavior wants to wrap a _unit of work_ that isn't a single dispatch: a
retry around each element of a fan-out, one deadline over a three-Step sequence,
cleanup tied to a logical operation rather than one call. Middleware wraps
exactly two things — one dispatch, or one Flow's Step graph — so the move is
always the same: give the work a Flow of its own.

## Pattern

The canonical case is per-dispatch retry in a `Gather`. A `Gather` carries no
`middleware`; instead, each dispatch targets a Flow that carries the behavior
inside:

```json
"register-granules": {
  "action": "Gather",
  "over": "{{ step.input.features }}",
  "call": {
    "flow": {
      "entrypoint": "register",
      "steps": {
        "register": {
          "action": "Call",
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "with": { "method": "POST", "path": "/granules" }
          },
          "middleware": [
            {
              "provider": "mwl:provider.middleware/mwl/retry/v1",
              "onEntry": {
                "with": {
                  "policies": [
                    { "match": { "codes": ["Provider.Call.*"] }, "attempts": 3,
                      "backoff": { "initial": "PT1S", "rate": 2 } }
                  ]
                }
              }
            }
          ],
          "next": "done"
        },
        "done": { "action": "Return" }
      }
    }
  },
  "concurrency": 10,
  "next": "summarize"
}
```

Each feature's dispatch runs the wrapper Flow in its own frame; the retry
re-runs that feature's registration independently of its siblings; and the
`Gather` counts one Result per feature, however many attempts it took.

## Why this shape

A re-runnable dispatch needs a frame. Re-running middleware restores variables
on re-entry, which only means something on a serial evaluator; concurrent
dispatches sharing the parent's variables would have no consistent state to
restore. A frame gives each dispatch its own variables, which is exactly what
makes its re-runs well-defined. "I need middleware around a dispatch" and "I
need a frame around a dispatch" are the same sentence.

The wrapper adds no data plumbing. Passthrough defaults carry the element from
`call.input` into the frame, through the inner `Call`, to the provider, and the
inner Flow's Result back out as the dispatch's Result. The wrapper Flow is pure
structure; only the middleware is new.

The same move solves the non-`Gather` cases. One deadline over a multi-Step
sequence is a `Timeout` on a Flow that contains those Steps; cleanup tied to an
operation is a `Finally` on the Flow that _is_ the operation; bounding a whole
fan-out wraps a Flow containing the `Gather` itself.

## Variations

- **Name it when it recurs.** The inline Flow above is single-use. Used twice,
  it moves to the `flows` map with `parameters` for its knobs, and the dispatch
  becomes `"call": { "flow": "RegisterGranule", "with": {...} }`.
- **Group Steps for shared behavior.** A subflow whose graph is just
  `step-a → step-b → step-c` exists legitimately to give those Steps one
  middleware stack, one parameter surface, and one Result.
- **The refactoring seam.** Because providers and Flows are interchangeable
  targets, wrapping is non-disruptive in both directions: a provider call grows
  into a wrapper Flow, a wrapper Flow shrinks back to a provider, and no call
  site changes shape.

## See also

- [Wrapping a dispatch: flows, not middleware](/reference/step-actions/#wrapping-a-dispatch-flows-not-middleware)
  — the normative account and the frame argument.
- [`flows`](/reference/flow-object/#flows) — naming and scoping.
- [Flow-Call Result parity](/reference/call-interface/#flow-call-result-parity)
  — why the seam is free.
- [Cleanup with `Finally`](../finally-cleanup/) — cleanup inside a dispatch's
  frame.
