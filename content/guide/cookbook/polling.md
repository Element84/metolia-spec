---
title: "Polling with timeout"
weight: 10
---

## Problem

A long-running external operation reports its status through a separate
endpoint. You want to check at a fixed interval until the operation reaches a
terminal state, and give up after a bounded wait.

## Pattern

Put the interval and the check in a small subflow — a `Sleep` Step and a `Call`
Step — and drive it with Flow-level middleware: `Loop` re-runs the pair while
the job is still running, and `Timeout` outside the `Loop` bounds the whole
polling cycle.

```json
"poll-job": {
  "action": "Call",
  "call": {
    "with": { "jobId": "{{ call.input.jobId }}" },
    "flow": {
      "middleware": [
        {
          "provider": "mwl:provider.middleware/mwl/timeout/v1",
          "onEntry": { "with": { "duration": "PT10M" } }
        },
        {
          "provider": "mwl:provider.middleware/mwl/loop/v1",
          "onSuccess": { "when": "{{ !vars.jobDone }}" }
        }
      ],
      "parameters": {
        "type": "object",
        "properties": {
          "jobId": { "type": "string" },
          "jobDone": { "type": "boolean", "default": false }
        },
        "required": ["jobId"]
      },
      "entrypoint": "wait",
      "steps": {
        "wait": { "action": "Sleep", "for": "PT5S", "next": "check" },
        "check": {
          "action": "Call",
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "with": {
              "method": "GET",
              "path": "{{ '/jobs/' + vars.jobId + '/status' }}"
            }
          },
          "assign": {
            "jobDone": "{{ step.result.value.status == 'complete' || step.result.value.status == 'failed' }}"
          },
          "next": "done"
        },
        "done": { "action": "Return" }
      }
    }
  },
  "next": "handle-result",
  "catch": [
    {
      "match": { "codes": ["Provider.Middleware.Timeout.Exceeded"] },
      "next": "job-took-too-long"
    }
  ]
}
```

## Why this shape

The interval lives in the graph. `Loop` re-runs its inner scope, and here the
inner scope is the whole Step graph: `wait` then `check`, every iteration. A
fixed cadence is a `Sleep` Step, not a property of any middleware.

The control plane carries both the identity and the signal. The job ID enters as
a parameter and is read from `vars` every iteration, so it doesn't depend on
what the loop's carried value happens to be (each iteration's input is the
previous iteration's output — the status response, which may not echo the ID).
Likewise `check` assigns `vars.jobDone` from each response and the `Loop`'s
continuation reads `{{ !vars.jobDone }}`; variables persist across `Loop`
iterations, which is exactly what makes them the natural loop state. The data
plane is left alone: whatever the status endpoint returns is the subflow's
result when the loop ends.

`Timeout` outside `Loop` bounds the cycle. The bound is captured when the entry
is established and spans every iteration; when it fires, the polling subflow is
preempted and the `Call` Step fails with `Provider.Middleware.Timeout.Exceeded`,
which the `catch` routes. Were the order reversed, each
five-second-plus-one-check iteration would get its own ten minutes, and the
workflow would never give up.

The subflow earns its keep. Middleware wraps one of two things, a single
dispatch or a Step graph; an interval-then-check sequence is two Steps, so the
loop must wrap a graph, and the inline `flow` target is how a `Call` Step
carries one. The `jobDone` parameter exists to give the variable a typed,
defaulted declaration.

## Variations

- **Poll as fast as the provider responds.** When no interval is wanted, drop
  the subflow: put `Loop` directly in the `Call` Step's `middleware` with
  `"when": "{{ middleware.result.value.status != 'complete' }}"` as its
  continuation. The carried value means each response is the next dispatch's
  input.
- **Bound by attempts instead of time.** Add a conjunct on the loop's
  continuation: `{{ !vars.jobDone && middleware.metadata.iteration < 120.0 }}`.
  When you want exceeding the count to be a distinct failure, follow the loop
  with a `Match` that routes a still-unfinished status to a `Raise`.
- **Backoff between polls.** Compute the `Sleep` duration from the iteration:
  assign a delay variable each pass and use `"for": "{{ vars.delay }}"`. If the
  reason for backoff is that the _check itself_ fails intermittently, that is
  `Retry`'s job, not `Loop`'s: retry on failure, loop on success.

## See also

- [The `Loop` middleware](/reference/providers/middleware-providers/#the-loop-middleware)
  — continuation, carried value, and iteration metadata.
- [The `Timeout` middleware](/reference/providers/middleware-providers/#the-timeout-middleware)
  — acceptance semantics and what preemption emits.
- [Ordering and composition](/reference/middleware-mechanics/#the-stack-ordering-and-composition)
  — why position decides what a bound covers.
- [Pagination accumulation](../pagination/) — `Loop` carrying a cursor instead
  of a flag.
