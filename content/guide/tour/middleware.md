---
title: "Middleware"
weight: 80
---

Middleware wraps work without being part of it: retrying it, bounding its time,
looping it, observing its outcome. A `middleware` array forms an ordered stack
of wrappers around one of two operations — the dispatch of a `Call` Step, or a
whole Flow's Step graph — and the same mechanism carries the spec's control-flow
vocabulary and platform-defined behaviors like caching and audit logging alike.

## The stack and the phases

```json
"middleware": [
  {
    "provider": "mwl:provider.middleware/mwl/retry/v1",
    "onEntry": {
      "with": {
        "policies": [
          { "match": { "codes": ["Provider.Call.*"] }, "attempts": 3,
            "backoff": { "initial": "PT2S", "rate": 2, "jitter": "full" } }
        ]
      }
    }
  },
  {
    "provider": "mwl:provider.middleware/mwl/timeout/v1",
    "onEntry": { "with": { "duration": "PT30S" } }
  }
]
```

The array is ordered outside-in: the first entry is the outermost wrapper and
the last wraps the operation directly. Data descends the stack into the
operation, and the operation's Result ascends back out. Each entry participates
at four **phases**: `onEntry` on the way down, then on the way up `onSuccess` or
`onFailure` (selected by the rising Result) followed by `onAlways`, whatever the
outcome.

Each phase key holds a block with up to three general keys — `when` gates
whether the middleware's action runs, `with` configures it, `assign` captures
into `vars` — plus a shaping key where the phase has data to shape. Inside a
block, the `middleware` binding is in scope: `middleware.input` (what this entry
received) and, on the way up, `middleware.result` (the Result rising at this
position).

## `Retry`: re-run on failure

`Retry` re-runs everything inside it when a rising failure matches one of its
policies. Policies select failures with the same failure matcher as `catch`,
each with its own attempt budget and optional backoff; exhausting the matching
policy emits `Provider.Middleware.Retry.Exhausted`, with the final attempt's
failure chained as `previous`. A failure no policy matches passes through
untouched.

Retries repeat, they don't drift: on each re-entry the frame's variables are
restored to their post-`onEntry` state, so every attempt starts from the same
world. State that should survive across attempts is carried deliberately, by the
`onFailure` block's `assign`.

## `Timeout`: bound the time, and ordering is meaning

`Timeout` races its inner scope and, when the bound elapses first, interrupts it
and emits `Provider.Middleware.Timeout.Exceeded` (a Result of type `timeout`,
matchable like any failure). Which scope it bounds is purely a matter of
position, and the example above is one of two meaningfully different
compositions:

- **`Retry` outside `Timeout`** (as above): the bound is inside the re-run, so
  each attempt gets a fresh 30 seconds. A hung attempt times out and `Retry`
  tries again.
- **`Timeout` outside `Retry`**: one 30-second budget spans all attempts and
  their backoff delays; when it expires, the whole retrying scope is preempted.

Both are legitimate; per-attempt bounds suit "this call sometimes hangs",
total-time bounds suit "the caller can only wait so long". The language never
second-guesses an ordering — composition is the author's instrument.

## `Loop`: repeat while

`Loop` re-runs its inner scope while its continuation holds. It has no
parameters at all: the loop is written entirely with `when`. Its `onSuccess`
`when` is the continuation, re-entering while true; its `value` is the carried
value, each iteration's output feeding the next iteration as input. Variables
persist across iterations (an iteration is progress, not a retry), which makes
accumulation idiomatic. Pagination in one Step:

```json
"fetch-all-items": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "GET", "path": "/items" }
  },
  "middleware": [
    {
      "provider": "mwl:provider.middleware/mwl/loop/v1",
      "onSuccess": {
        "when": "{{ middleware.result.value.nextCursor != null }}",
        "assign": { "items": "{{ vars.items + middleware.result.value.items }}" }
      }
    }
  ],
  "output": "{{ vars.items }}",
  "next": "process-items"
}
```

Each response rides into the next dispatch (cursor and all) as the carried
value; each iteration's items accumulate in `vars.items` (declare it in
`parameters` with a default of `[]`); when a response has no `nextCursor`, the
loop emits and the Step's `output` reads the accumulated list. A bound is one
more conjunct: `{{ ... && middleware.metadata.iteration < 100.0 }}`.

## `Finally`: cleanup on every exit

`Finally` dispatches a cleanup call at `onAlways`, the one phase that runs on
every outcome — success, failure, cancellation, even mid-teardown when an
enclosing timeout fires. The cleanup is an ordinary `call` object; its `input`
can read the Result in flight:

```json
{
  "provider": "mwl:provider.middleware/mwl/finally/v1",
  "onAlways": {
    "with": {
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "input": "{{ middleware.result }}",
        "with": { "method": "POST", "path": "/audit" }
      }
    }
  }
}
```

The cleanup's Result is discarded — `onAlways` stands outside the data plane —
so the audit write can never corrupt the value in flight. A cleanup that _fails_
does surface, superseding the Result in flight with the original chained beneath
it: failed cleanup is real and is never swallowed.

## Two attachment levels, one `when`

Everything above attaches identically at the Flow level: a `middleware` array on
a Flow object wraps its whole Step graph, so a deadline over an entire workflow,
a retry around a whole subflow, or an audit of a Flow's outcome are the same
entries at the outer level. This is also how `Gather` work gets wrapped: a
dispatch targets a Flow that carries the stack
([previous page](../gather/#wrapping-a-dispatch-target-a-flow)).

And every phase block's `when` gates its middleware's action at runtime:

```json
{
  "provider": "mwl:provider.middleware/mwl/timeout/v1",
  "onEntry": {
    "when": "{{ vars.enforceTimeout }}",
    "with": { "duration": "PT15M" }
  }
}
```

Enablement is a parameter, not a second copy of the workflow: retries off for a
backfill run, publication only in production, a bound only when the caller asks
for one.

## Where the spec covers this

- [Middleware mechanics](/reference/middleware-mechanics/) — entries, the stack,
  the phase model, threading, and `when`.
- [Middleware providers](/reference/providers/middleware-providers/) — the four
  spec-defined middlewares in full: parameters, behavior, metadata, and failure
  codes.
- [The unwind](/reference/execution-model/#the-unwind) — what runs during
  teardown, and why `onAlways` is the phase you can rely on.
