---
title: "Step actions"
weight: 110
---

Every Step names an `action`: the verb the Step performs. This section is the
catalog of the seven actions — `Call`, `Gather`, `Match`, `Pass`, `Sleep`,
`Return`, and `Raise` — each with its complete field set. The mechanics every
Step shares — routing, data flow, variable capture, and `catch` — are defined in
[Steps and step mechanics](../step-mechanics/); each action's entry here states
how those shared fields apply to it and defines the fields that are its own.

`action` is a structural discriminator and does not accept an expression (see
[Where expressions may appear](../expressions/#where-expressions-may-appear)).
The table below refines the shared-field table of
[Steps and step mechanics](../step-mechanics/#what-a-step-is) per action:

| Field        | `Call` | `Gather` | `Match` | `Pass` | `Sleep` | `Return` | `Raise` |
| ------------ | ------ | -------- | ------- | ------ | ------- | -------- | ------- |
| `input`      | ✓      | —        | ✓       | —      | —       | —        | —       |
| `output`     | ✓      | ✓        | (c)     | ✓      | —       | —        | —       |
| `assign`     | ✓      | ✓        | (c)     | ✓      | —       | —        | —       |
| `middleware` | ✓      | —        | —       | —      | —       | —        | —       |
| `catch`      | ✓      | ✓        | —       | —      | —       | —        | —       |
| `next`       | ✓      | ✓        | (c)     | ✓      | ✓       | —        | —       |
| `comment`    | ✓      | ✓        | ✓       | ✓      | ✓       | ✓        | ✓       |

(c): on the matched clause, not at Step level.

Each action's own fields — `Call`'s `call`; `Gather`'s `over`, `call`, `calls`,
`completion`, and `concurrency`; `Match`'s `cases` and `default`; `Sleep`'s
`for` and `until`; `Return`'s `value`; `Raise`'s `result` — are defined with the
action below. Every action accepts a
[`comment`](../definition-format/#the-comment-field). Two actions define
action-specific metadata, given with each: a `Call`'s dispatch instants (kept on
the Call's own record, not the Step's) and a `Gather`'s dispatch count. The rest
record only the universal members of their metadata record (see
[Metadata records](../execution-context/#metadata-records)).

## `Call`

A `Call` Step dispatches a single `call` object and routes on the Result it
yields. The call itself — its target, `input`, `with`, and its
`onSuccess`/`onFailure` arms — is the dispatch unit defined in
[The Call interface and Result](../call-interface/); the Step contributes
routing (`next`, `catch`), boundary shaping (`input`, `output`, `assign`), and
the middleware stack that wraps the dispatch.

```json
{
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/container/v1",
    "with": {
      "image": "repo/modis-l0-to-l1-stac-task:latest",
      "command": ["run"]
    }
  },
  "next": "process-granules",
  "catch": [{ "match": { "codes": ["*"] }, "next": "failed" }]
}
```

| Field        | Type               | Required | Default                   | Expression      |
| ------------ | ------------------ | -------- | ------------------------- | --------------- |
| `call`       | object             | required | —                         | no (structural) |
| `input`      | any                | optional | `{{ step.input }}`        | yes             |
| `output`     | any                | optional | `{{ step.result.value }}` | yes             |
| `assign`     | object             | optional | `{}`                      | yes (per value) |
| `middleware` | array              | optional | `[]`                      | no (structural) |
| `catch`      | array              | optional | `[]`                      | no (structural) |
| `next`       | string (Step name) | required | —                         | no (structural) |

`call` names exactly one target, a provider or a Flow, and carries the
dispatch's own fields, each of which accepts an expression per its definition in
[The call object](../call-interface/#the-call-object).

The Step's `input` shapes the value entering the Step's machinery. That value
descends the `middleware` stack, and what the innermost entry emits arrives at
the dispatch as `call.input` (see
[How values thread the stack](../middleware-mechanics/#how-values-thread-the-stack));
with no middleware, the shaped input arrives directly. The Result the call
yields ascends the same stack, and the Result the outermost entry emits is the
Step's `step.result` (see
[Where `catch` sits](../step-mechanics/#where-catch-sits)). On success, `output`
shapes the emitted value from it and `assign` captures into `vars`; on failure,
`catch` matches it (see
[Failures and `catch`](../step-mechanics/#failures-and-catch)).

The Step's `middleware` wraps the dispatch: the stack establishes once per Step
pass, wraps the single Call, and a retrying entry re-runs the Call inside it
(see
[Where middleware attaches](../middleware-mechanics/#where-middleware-attaches)).
`Call` is the only action that carries the field; wrapping a `Gather`'s
dispatches is a flow target's work (see
[Wrapping a dispatch](#wrapping-a-dispatch-flows-not-middleware)).

A flow-targeted call additionally exposes the completed frame to the call's
arms, as the `flow` window; capturing out of it happens in an arm's `assign`
(see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)).

### `Call` metadata

A `Call` Step's own record carries no action-specific members; the dispatch
timing belongs to the Call's record (see [`call`](../execution-context/#call)),
where two context-specific members sit beside the universal pair:

| Member         | Type               | Description                                            |
| -------------- | ------------------ | ------------------------------------------------------ |
| `dispatchedAt` | string (timestamp) | The instant dispatch to the target was initiated.      |
| `acceptedAt`   | string (timestamp) | The instant the platform accepted the target's Result. |

`dispatchedAt` follows the record's `enteredAt`: the call's fields have
evaluated and the request leaves for the target. `acceptedAt` is the instant the
platform commits the target's Result — the acceptance instant that a control
action racing the dispatch defines its semantics against (see
[Author shaping and the middleware action](../middleware-mechanics/#author-shaping-and-the-middleware-action)).
Both are readable in the call's arms and are carried onward with an arm's
`assign`. A re-dispatched Call is a new Call execution with a fresh record (see
[Re-execution evaluates afresh](../execution-model/#re-execution-evaluates-afresh)).

## `Gather`

A `Gather` Step fans work out and gathers the Results back in. It makes one or
more concurrent **dispatches**, each one execution of a `call` object — every
dispatch is a call, with everything that entails (see
[The Call interface and Result](../call-interface/)) — collects every dispatch's
Result into `step.results`, and routes on a completion policy. The `Gather`
contributes the fan-out, in one of two forms; the completion policy
(`completion`); and the collected record.

```json
{
  "action": "Gather",
  "over": "{{ step.input.features }}",
  "call": { "flow": "ProcessGranule", "with": { "collection": "modis-l1" } },
  "concurrency": 10,
  "next": "summarize",
  "catch": [{ "match": { "codes": ["*"] }, "next": "failed" }]
}
```

This `Gather` is in the iterate form: one dispatch per feature, each running
`ProcessGranule` in a frame of its own, at most ten in flight at once.

| Field         | Type                  | Required                  | Default                                                                           | Expression                                                        |
| ------------- | --------------------- | ------------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| `over`        | array                 | iterate form, with `call` | —                                                                                 | yes                                                               |
| `call`        | object                | iterate form, with `over` | —                                                                                 | no (structural)                                                   |
| `calls`       | array of call objects | scatter form              | —                                                                                 | no (structural)                                                   |
| `completion`  | object                | optional                  | all dispatches succeed                                                            | per field (see [`completion`](#completion-the-completion-policy)) |
| `concurrency` | integer ≥ 1 \| null   | optional                  | unlimited                                                                         | no (literal)                                                      |
| `output`      | any                   | optional                  | the success projection (see [`step.results`](#the-collected-results-stepresults)) | yes                                                               |
| `assign`      | object                | optional                  | `{}`                                                                              | yes (per value)                                                   |
| `catch`       | array                 | optional                  | `[]`                                                                              | no (structural)                                                   |
| `next`        | string (Step name)    | required                  | —                                                                                 | no (structural)                                                   |

A `Gather` MUST carry exactly one of its two forms: `over` together with `call`,
the **iterate form**, or `calls`, the **scatter form** — never both and never
neither. `calls` MUST be a non-empty array: an empty `calls` is a fan-out,
written literally, that dispatches nothing — almost certainly an authoring
error, statically detectable, and the definition is ill-formed (see
[Static checks](../flow-object/#static-checks)). An `over` that produces an
empty array is data-dependent and legal; it makes zero dispatches (below).

A `Gather` carries no step-level `input`: the value the Step received is in
scope as `step.input` for the Step's own fields, and what each dispatch receives
is its call's to shape.

`concurrency` caps the number of dispatches active at once. A dispatch is active
from its entry until its Result settles; the order in which pending dispatches
start under a cap is implementation-defined. A present cap MUST be a positive
integer; a value below 1 is ill-formed, statically detectable like the
non-emptiness of `calls` (see [Static checks](../flow-object/#static-checks)).
The fan-out is unlimited — every dispatch may be active at once — when
`concurrency` is absent or `null`; the explicit `null` is the in-band way to
write unlimited where omitting the field is inconvenient.

`concurrency` caps the dispatches active at once, not the fan-out's size. An
implementation MAY bound the number of dispatches a single fan-out may
enumerate, as a platform-defined limit; the bound and the failure produced on
exceeding it are the implementation's to document — a fan-out limit is a
platform constraint, not an engine semantic, so it does not mint a `System.`
code (see [Code namespaces](../call-interface/#code-namespaces)). An
implementation that imposes a bound MUST fail the `Gather` rather than truncate
the fan-out, and SHOULD fail it at enumeration, where the dispatch count is
known before any dispatch starts.

### The iterate form: `over` and `call`

The iterate form makes one dispatch per element of a collection. `over` is
evaluated once, when the `Gather`'s action begins, against the Step's bindings.
Its result MUST be an array; a non-array result is a value of the wrong type for
a field whose type is known, and the `Gather` MUST fail with
`System.ParameterValidationFailed` (see
[Evaluation errors](../expressions/#evaluation-errors)). An empty array makes
zero dispatches: the action completes with `step.results` empty.

`call` is the form's one call template. Each element of the `over` result makes
one dispatch of it: the element enters the dispatch as `call.input`, and
`call.index` is the element's 0-based position. The call's fields evaluate
afresh per dispatch, so a `with` that reads `call.input` or `call.index`
configures each dispatch from its own element and position.

### The scatter form: `calls`

The scatter form makes one dispatch per entry of `calls`, a literal array of
call objects:

```json
{
  "action": "Gather",
  "calls": [
    {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "GET", "path": "/a" }
    },
    {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "GET", "path": "/b" }
    },
    { "flow": "Summarize" }
  ],
  "next": "combine"
}
```

Each entry is a complete `call` object, independently targeted and configured.
The value the Step received enters every dispatch — arriving at the call
boundary as `call.input` — and `call.index` is the entry's 0-based position in
`calls`.

### Wrapping a dispatch: flows, not middleware

A `Gather` carries no `middleware`: among the actions, only `Call` accepts the
field (see [Steps and step mechanics](../step-mechanics/#what-a-step-is)). The
reason is the frame. A middleware that re-runs its inner scope, such as `Retry`,
restores the frame's variables on each re-entry, which is well-defined only
because a frame evaluates serially, one block at a time (see
[Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).
Concurrent dispatches sharing one frame's variables have no such serial point to
restore to — a per-dispatch re-run would race every sibling's writes — so a
re-runnable dispatch needs a variable scope of its own, which is to say a frame
of its own. While the fan-out is in flight, then, nothing in the parent frame
gates, re-runs, or observes a dispatch; a dispatch that needs wrapped behavior
targets a Flow and carries the wrapper inside it: the target Flow's own
`middleware`, or a stack on a `Call` Step within it, wraps the work inside the
dispatch's frame (see
[Where middleware attaches](../middleware-mechanics/#where-middleware-attaches)).
A need for middleware around a dispatch is a need for a frame around it, and the
frame is exactly what gives the re-run its own variables to restore. The cost is
that a dispatch's variables stay within its frame: a value the dispatch must
surface is captured across the boundary by the call's arm, like any flow target
(see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)).
Wrapping the fan-out as a whole is the same composition one level up: a stack
around the entire `Gather` — a duration bound over the whole fan-out, say —
wraps a Flow whose graph contains the `Gather`, attached at the Flow level.

Per-dispatch retry is the common case, and an inline flow target is its idiom:

```json
{
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
                    {
                      "match": { "codes": ["Provider.Call.*"] },
                      "attempts": 3
                    }
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
  "next": "summarize"
}
```

Each feature's dispatch runs the wrapper Flow in a frame of its own; inside it,
the retrying stack re-runs that feature's registration, so every dispatch
retries independently, and `completion` counts each dispatch's one emerging
Result: a dispatch that succeeds on its third attempt is one success. The
element flows through on the default passthroughs — the dispatch's `call.input`
seeds the frame, the frame's input reaches the inner `Call` Step, and the inner
call delivers it to the provider — so the wrapper adds no data plumbing. Cleanup
that must run on a dispatch's teardown lives the same way: a `Finally` entry in
the target Flow's stack runs `onAlways` within the dispatch's frame on every
exit, including when the `Gather` cancels the dispatch (see
[The `Finally` middleware](../providers/middleware-providers/#the-finally-middleware)).

### The dispatch model

Every dispatch is one execution of a `call` object: a fresh evaluation of its
fields against that dispatch's bindings, with its own metadata record and clock
pin (see
[Expression evaluation timing](../execution-model/#expression-evaluation-timing)).
The call MAY target a provider or a Flow, exactly as a `Call` Step's call does;
a flow-targeted dispatch runs in a frame of its own (see
[Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).

Both forms present one surface to the call. `call.input` is the dispatch's
inbound payload — the element, in the iterate form; the value the Step received,
in the scatter form — and `call.index` its 0-based position, available on
`Gather`-dispatched calls in both forms (see
[`call`](../execution-context/#call)). A call template therefore moves between
the forms unchanged.

The dispatch's Result is consumed where every call's is: in the call's own arms
(see [The arms](../call-interface/#the-arms-onsuccess-and-onfailure)).
`onSuccess.value` shapes the value the dispatch's success Result carries into
`step.results`, and either arm's `assign` captures into the frame's variables at
the seam where the dispatch's context — `call.result`, the target window — is
alive.

Dispatches run their work concurrently; the frame's variables hold still beneath
them (see
[Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).
A dispatch's call fields evaluate as it starts and only read, against the
variable state at the action's start — every dispatch reads the same state,
however `concurrency` staggers the starts — and the arms, the fan-out's only
write-capable expressions, are deferred to fan-out completion
([below](#the-arms-at-fan-out-completion)). Dispatches are therefore isolated by
construction: a dispatch cannot observe a sibling's writes, progress, or
outcome, and nothing inside the fan-out reads `step.results`. Coordination among
dispatches is not expressible; what the fan-out must achieve is `completion`'s
to say, and anything more is graph structure — separate Steps, or a flow target
composing its own.

A dispatched Flow does not see the `Gather`'s frame: data crosses into its frame
only through the call's `input` and `with`, so a subflow that needs the element
or the index is given it explicitly.

### The arms at fan-out completion

Under `Gather`, the arms do not run as each Result settles: arm evaluation is
deferred to fan-out completion. Once every dispatch has resolved, the arms run
one dispatch at a time, in dispatch order, each arm one block under the
discipline of [Variables: `assign`](../step-mechanics/#variables-assign), each
reading the variable state every lower-indexed dispatch's arm left. Each settled
dispatch's arm runs exactly once. An interrupted dispatch — one the `Gather`
cancelled — evaluates no arms, and a skipped dispatch evaluates nothing at all
(see [When the arms run](../call-interface/#when-the-arms-run)).

Deferral splits the dispatch's Result into two stages. Settlement yields the
target's Result: what the `completion` arithmetic counts as Results arrive, and
what stamps the call record's `exitedAt` and `acceptedAt`. The arm then
finalizes the Result that fills the dispatch's slot in `step.results`:
`onSuccess.value` shapes the success value, and an arm that faults makes the
fault's failure that dispatch's Result (see
[Faults in the call object's fields](../call-interface/#faults-in-the-call-objects-fields)).
Wherever `step.results` is read, a dispatch's Result is the arm-finalized one.
The dispatch's context is still alive when its arm runs: `call.result`, the
completed `call.metadata` record, and the target window are in scope at fan-out
completion exactly as at settlement (see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)).
What an arm never reads is `step.results` itself: when it runs, its own slot is
not yet final.

Dispatch order makes cross-dispatch accumulation deterministic. An
`onSuccess.assign` of `"ids": "{{ vars.ids + [call.result.value.id] }}"` grows
the list in dispatch order — element order, `calls` order — reproducibly,
whatever order the dispatches completed in.

### The collected Results: `step.results`

`step.results` is the complete record of the fan-out: one Result per dispatch,
flat, in dispatch order — element order in the iterate form, `calls` order in
the scatter form. Every dispatch holds its true position whatever its outcome:
settled, cancelled, and skipped dispatches alike, a skipped dispatch resolving
[`System.GatherDispatchSkipped`](#systemgatherdispatchskipped) without
evaluating anything.

`step.results` is available after the action, like `step.result`: the Step's
`output`, `assign`, and `catch` clauses read it, and no expression inside the
fan-out can. By the time any expression reads it, every slot is a Result — the
**resolution guarantee**: the `Gather`'s action does not complete until every
dispatch has resolved, settled, cancelled, or skipped. The guarantee covers the
`Gather` failing from its own machinery — a `completion.successes` or `over`
fault with dispatches in flight: the `Gather` cancels the in-flight dispatches
and resolves all of them, by the `wait: false` mechanics, before constructing
its own failure Result.

Because the record is flat, uniform, and position-faithful, its consumers are
one-line projections. The success projection

```
{{ step.results.filter(r, r.type == 'success').map(r, r.value) }}
```

produces the succeeded dispatches' values, in order. It is also the `output`
default: a `Gather` under the default `completion`, where every dispatch must
succeed, emits exactly its dispatches' values.

Under a non-default `completion` the default still projects successes only. A
`Gather` that succeeds with some dispatches failed — `completion` met without
every dispatch succeeding — therefore emits just the succeeded values by
default; the failed dispatches' Results are not in the emitted value, and their
positions are not held. A partial-success fan-out that must carry the failures
forward shapes its own `output` over `step.results`, where every dispatch is
still present in its true position. To keep one slot per input element, success
or failure, emit the whole record:

```json
"output": "{{ step.results }}"
```

To emit each element's value where it succeeded and a placeholder where it did
not, map over the record by outcome:

```json
"output": "{{ step.results.map(r, r.type == 'success' ? r.value : null) }}"
```

The successor then receives a list positionally aligned with the input, the
failed positions distinguishable. Its failure dual

```
{{ step.results.filter(r, r.type != 'success') }}
```

collects the non-success Results alone, and the same shape serves any projection
— partitioning by `code`, counting by type. Like every field default, the
`output` default is stated as an expression as precise notation for the
behavior, and realizing it requires no expression evaluator (see
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough)).

### `completion`: the completion policy

`completion` defines what the fan-out must achieve for the `Gather` to succeed,
and it is the only path by which dispatch failures affect the `Gather`: the
`Gather` observes its dispatches' Results, it does not catch them. A failing
dispatch does not by itself fail the Step; the Step fails when the policy
becomes unachievable.

| Field       | Type             | Required | Default | Expression   |
| ----------- | ---------------- | -------- | ------- | ------------ |
| `successes` | number (integer) | required | —       | yes          |
| `wait`      | boolean          | optional | `true`  | no (literal) |

An absent `completion` requires every dispatch to succeed, equivalent to
`{ "successes": <the dispatch count>, "wait": true }`, so under the default a
single failure makes the policy unachievable and fails the `Gather`.

`successes` is the number of dispatches that must succeed. It is evaluated once,
when the action begins and after the dispatches are enumerated, so the Step's
dispatch count is available to it: `{{ step.metadata.dispatchCount }}` restates
the default explicitly, and `1` succeeds on the first success.

`wait` governs the dispatches still pending once the outcome is determined —
`successes` reached, or unreachable. When `true`, every dispatch runs to
completion regardless. When `false`, the `Gather` cancels what is in flight and
abandons what has not started: a cancelled dispatch resolves with a Result of
type `cancellation` and code `System.GatherDispatchCancelled` (see
[External cancellation](../execution-model/#external-cancellation)), and a
never-started dispatch resolves skipped, below. This cancellation is the
`Gather` interrupting its own dispatches, not a Result arriving from a target:
the dispatch's scope unwinds (see [The unwind](../execution-model/#the-unwind)),
no arm of its call runs — a flow-targeted dispatch's frame is torn down, its
established entries running `onAlways` only; a provider-targeted dispatch is
cancelled through the platform — and the `System.GatherDispatchCancelled` Result
exists for the outside observers, `step.results` and the `completion`
arithmetic, without any seam inside the dispatch having evaluated (see
[When the arms run](../call-interface/#when-the-arms-run)).

Cancellation races completion: a dispatch whose Result the platform has already
accepted — the `acceptedAt` instant of its call's record (see
[`Call` metadata](#call-metadata)) — resolves as that Result, not as cancelled;
the cancellation takes only the dispatches not yet accepted.

`completion` reads settled Results; the `Gather`'s own outcome is determined
only after the arms run, from the final `step.results` (see
[The arms at fan-out completion](#the-arms-at-fan-out-completion)). The two
determinations can differ only by tightening: an arm fault turns a success into
that dispatch's failure, never the reverse, so a fan-out that met its policy at
settlement can still fail with
[`System.GatherCompletionUnmet`](#systemgathercompletionunmet) — and under
`wait: false` the dispatches cancelled once the policy was met keep their
cancelled Results. The arms run whatever the determination will be: when the
`Gather` fails from its own machinery, the settled dispatches' arms still
evaluate at fan-out completion, before the failure Result is constructed.

#### `System.GatherDispatchSkipped`

A dispatch never started when the `Gather` completed — held behind
`concurrency`, or moot once the outcome was determined under `wait: false` —
resolves with a Result of type `skipped` and code
`System.GatherDispatchSkipped`, without ever executing: none of its fields
evaluate and no record is kept. The Result exists so an unrun dispatch is a
concrete, countable outcome rather than an absence; it holds the dispatch's slot
in `step.results` like any other.

#### `System.GatherCompletionUnmet`

When `successes` can no longer be reached, the `Gather` fails with a Result of
type `error` and code `System.GatherCompletionUnmet`. The envelope's `details`
carry the evidence:

- `failures` — the fan-out's non-success Results, each entry pairing the
  dispatch's `index` with its `result`.
- `failureCount` — the total number of non-success dispatches.

The list deliberately duplicates the non-success subset of `step.results`: the
envelope propagates to parent frames, which can never read the failed Step's
`step.results`.

Under `wait: false`, the list holds both the failures that made the policy
unachievable and the cancelled and skipped Results imposed in consequence; each
entry's `result` carries its `type`, so causes are told apart from aftermath.

An implementation MAY cap the `failures` list at a platform-defined size;
`failureCount` is not subject to the cap, so a capped list is always
distinguishable from a complete one. The collected failures ride `details`, not
`previous`: they are evidence the policy was unmet, not failures this one
superseded (see [Chaining](../execution-context/#chaining)).

### Failures and `catch`

A `Gather`'s `catch` matches failures the `Gather` itself produces, and only
those: `System.GatherCompletionUnmet`, and faults in its own fields, such as an
`over` or `successes` that fails to evaluate (see
[Evaluation errors](../expressions/#evaluation-errors)) or yields the wrong
type. A dispatch's failure is never matched by the `Gather`'s `catch`: it is
data the `Gather` observes — a slot in `step.results`, evidence in
`System.GatherCompletionUnmet`'s `details` — and it does not set the frame's
failure context; the `Gather`'s own failure does (see
[Lifecycle](../execution-context/#lifecycle)).

A dispatch's `with` is validated against its target's `parameters` like any
call's; a validation failure is that dispatch's Result,
`System.ParameterValidationFailed` (see
[`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed)),
counted by `completion` like any other dispatch failure.

When the `Gather` fails, every dispatch has already resolved — the resolution
guarantee — so a matching `catch` clause's expressions read `step.results`
complete.

### `Gather` metadata

`step.metadata` carries one `Gather`-specific member beside the universal pair:

| Member          | Type   | Description                                                                     |
| --------------- | ------ | ------------------------------------------------------------------------------- |
| `dispatchCount` | number | The total number of dispatches: the length of the `over` result, or of `calls`. |

`dispatchCount` settles at enumeration, before `successes` evaluates, and the
record holds nothing that accrues: an expression reading `step.metadata` during
the fan-out observes nothing unsettled, since `dispatchCount` is fixed and the
universal pair are instants. Per-outcome counts are projections over
`step.results` — `{{ size(step.results.filter(r, r.type == 'error')) }}` — like
any other partition of the record.

Each dispatch keeps its own record, `call.metadata`, bracketing that call
execution: `enteredAt` as its fields evaluate, `exitedAt` when its Result
settles, both readable in the call's arms, beside the dispatch instants
(`dispatchedAt`, `acceptedAt`; see [`Call` metadata](#call-metadata)). A
flow-targeted dispatch's inner frame keeps its own record besides, observable
through the `flow` window (see [`flow`](../execution-context/#flow)). A skipped
dispatch keeps no record.

## `Match`

A `Match` Step routes to one of several successors by testing predicates against
a single shaped value. It performs no work beyond selection: clauses are tried
in order, the first whose `when` holds is selected, and that clause routes,
shapes, and captures.

```json
{
  "action": "Match",
  "input": "{{ step.input.order }}",
  "cases": [
    {
      "when": "{{ match.input.status == 'approved' && match.input.amount > 1000.0 }}",
      "next": "manual-review"
    },
    {
      "when": "{{ match.input.status == 'approved' }}",
      "next": "auto-approve"
    }
  ],
  "default": { "next": "reject" }
}
```

| Field     | Type                    | Required | Default            | Expression      |
| --------- | ----------------------- | -------- | ------------------ | --------------- |
| `input`   | any                     | optional | `{{ step.input }}` | yes             |
| `cases`   | array of clause objects | required | —                  | no (structural) |
| `default` | clause (no `when`)      | required | —                  | no (structural) |

`input` produces the value the predicates test. It is evaluated once, at input
shaping (see [The Step lifecycle](../execution-model/#the-step-lifecycle));
every clause reads it as `match.input` (see
[`match`](../execution-context/#match)), and `step.input` still recovers the
value the Step received. A `Match` carries no step-level `output`, `assign`, or
`next` — shaping and routing belong to the selected clause, which may shape
differently per target — and no `catch` (see
[Failures and `catch`](../step-mechanics/#failures-and-catch)). The match
context is coextensive with its Step: its record's instants equal the Step's
(see [`match`](../execution-context/#match)). `Match` is the one action with no
expression-free form: every `cases` clause requires a `when` expression, so a
definition that uses `Match` requires an expression evaluator (see
[Conformance](../conformance/)).

### Clauses

| Field     | Type               | Required                                       | Default             | Expression      |
| --------- | ------------------ | ---------------------------------------------- | ------------------- | --------------- |
| `when`    | boolean            | required in `cases`; not accepted on `default` | —                   | yes (predicate) |
| `output`  | any                | optional                                       | `{{ match.input }}` | yes             |
| `assign`  | object             | optional                                       | `{}`                | yes (per value) |
| `next`    | string (Step name) | required                                       | —                   | no (structural) |
| `comment` | string             | optional                                       | —                   | no (literal)    |

`cases` is ordered and the first match wins: predicates evaluate one clause at a
time until one holds, and no later predicate evaluates. `default` is the
required fallback — the same clause shape, without `when` — selected when no
case matches. The selected clause's `output` shapes the value its `next`
receives, and its `assign` captures into `vars`, each evaluated once, under the
Step's disciplines (see
[Variables: `assign`](../step-mechanics/#variables-assign)).

### Predicates and failure

`when` holds an expression evaluated as a predicate (see
[Predicates and `when`](../expressions/#predicates-and-when)).

A `when` that fails to evaluate is an evaluation error like any other: it fails
the frame with `System.ExpressionEvaluationError` (see
[Evaluation errors](../expressions/#evaluation-errors)). Failure is not absorbed
into a non-match, and evaluation does not fall through to the next clause. An
author routing on data whose shape is uncertain must write predicates that
tolerate that uncertainty rather than relying on a failed predicate to skip its
clause.

A clause's `output` and `assign` are likewise ordinary expressions: a fault in
either fails the frame the same way. `default` protects against fallthrough — no
case matching — not against an evaluation fault in any clause.

Predicates can be opaque to static analysis. Validators MAY warn about apparent
tautologies, contradictions, or unreachable clauses, but MUST NOT reject a
definition on that basis.

## `Pass`

A `Pass` Step performs no action work: it exists for the shaping phases around
it (see [The Step lifecycle](../execution-model/#the-step-lifecycle)). It emits
a value and captures variables, then transitions.

```json
{
  "action": "Pass",
  "output": "{{ {'wrapped': step.input} }}",
  "next": "deliver"
}
```

| Field    | Type               | Required | Default            | Expression      |
| -------- | ------------------ | -------- | ------------------ | --------------- |
| `output` | any                | optional | `{{ step.input }}` | yes             |
| `assign` | object             | optional | `{}`               | yes (per value) |
| `next`   | string (Step name) | required | —                  | no (structural) |

Typical uses: naming a point in the graph, reshaping a value between two Steps,
staging variable bindings. A `Pass` carries no `input` field; with no action to
feed, its one shaping seam is `output`, which reads the received value directly.

## `Sleep`

A `Sleep` Step pauses the frame: for a duration, or until an instant. Exactly
one of `for` and `until` MUST be present.

```json
{ "action": "Sleep", "for": "PT30S", "next": "poll-status" }
```

```json
{ "action": "Sleep", "until": "{{ vars.resumeAt }}", "next": "send-reminder" }
```

| Field   | Type               | Required             | Default | Expression      |
| ------- | ------------------ | -------------------- | ------- | --------------- |
| `for`   | string (duration)  | one of `for`/`until` | —       | yes             |
| `until` | string (timestamp) | one of `for`/`until` | —       | yes             |
| `next`  | string (Step name) | required             | —       | no (structural) |

`for` is an ISO 8601 duration (see
[Temporal format profile](../data-model/#temporal-format-profile)): the Step
completes at `step.metadata.enteredAt` plus the duration, immediately if that
duration is zero or negative. `until` is an RFC 3339 timestamp: the Step
completes at that instant, immediately if it is already past. Either way a
moment already reached completes the Step at once rather than failing it.

`Sleep` is a pure pause. It carries no `input`, `output`, or `assign` — the
value it received passes through unchanged (see
[Field defaults and passthrough](../execution-context/#field-defaults-and-passthrough))
— and no `middleware` or `catch`. It cannot fail on its own; its only failure
surface is its one field: an expression in it that fails to evaluate (see
[Evaluation errors](../expressions/#evaluation-errors)), or a value, literal or
computed, that is not a valid duration or timestamp (see
[`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed)).
An interruption during the pause is frame-level, like any interruption: the Step
is abandoned, not failed (see [Cancellation](../execution-model/#cancellation)).

## `Return`

`Return` is the terminal success: the frame completes with a success Result, and
`value` shapes the value that Result carries (see
[Success Result](../call-interface/#success-result)). How the completion is
consumed — by a calling Step, a `Gather`, or the platform at the root — is the
frame contract (see
[How a Flow completes](../flow-object/#how-a-flow-completes)).

```json
{ "action": "Return", "value": "{{ step.input.summary }}" }
```

| Field   | Type | Required | Default            | Expression |
| ------- | ---- | -------- | ------------------ | ---------- |
| `value` | any  | optional | `{{ step.input }}` | yes        |

The default passes the Step's received value through: a bare
`{ "action": "Return" }` returns what reached it. `Return` carries no `next`, no
`output` — there is no successor to emit to; the frame's product is the Result's
`value` — and no `assign`, since the frame ends with the Step, leaving no later
reader (see
[Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output)).

## `Raise`

`Raise` is the terminal failure: the frame completes with a failure Result. With
a `result`, it constructs the failure; bare, it re-raises the failure being
handled.

```json
{
  "action": "Raise",
  "result": {
    "type": "error",
    "code": "Pipeline.ManualReject",
    "message": "Order flagged for manual review"
  }
}
```

| Field    | Type              | Required | Default                                        | Expression      |
| -------- | ----------------- | -------- | ---------------------------------------------- | --------------- |
| `result` | object (envelope) | optional | re-raise (see [Bare re-raise](#bare-re-raise)) | yes (per field) |

`result`'s members are the authorable fields of the failure envelope (see
[The failure envelope](../call-interface/#the-failure-envelope)): `type`,
`code`, `message`, `details`, `retryable`, and `previous`. `code` is required.
`type` is optional, defaults to `"error"`, and MUST be a non-success type: a
`Raise` cannot complete a frame with success. Each field accepts an expression,
evaluated against the Step's bindings; an active failure is readable as
`failure` (see [`failure`](../execution-context/#failure)).

Constructing a failure while one is active supersedes it: unless `result` writes
`previous` itself, the engine links the active failure as the new one's
`previous`, and writing `previous` overrides the link — typically severing it
with `null` (see [Chaining](../execution-context/#chaining)). These are the same
construct-a-new-failure semantics as a middleware `onFailure` block (see
[`onFailure`](../middleware-mechanics/#onfailure)), with one difference: an
`onFailure` block transforms a rising failure, so its unwritten fields inherit
from the failure it supersedes, while a `Raise` constructs from nothing, so its
unwritten fields are simply absent — which is why `code` is required here and
inheritable there.

A `Raise` MAY use any `code`. The namespace convention, including why an author
SHOULD NOT mint `System.` codes, is defined with the envelope (see
[Code namespaces](../call-interface/#code-namespaces)). Validators MAY warn
where a `result`'s code trespasses the convention — a `System.` code, or a
`Provider.` code minted by workflow logic — but MUST NOT reject the definition.

### Bare re-raise

A `Raise` without `result` re-emits the frame's active failure unchanged: no new
failure is constructed and no `previous` link is added (see
[Chaining](../execution-context/#chaining)). It is the natural terminal for a
handler path that deals with what it can and propagates the rest:

```json
"failed": { "action": "Raise" }
```

#### `System.EmptyRaise`

A bare `Raise` reached with no active failure — outside any handler path, or
after a success cleared the context (see
[Lifecycle](../execution-context/#lifecycle)) — has nothing to re-emit. The
frame completes with a failure Result of type `error` and code
`System.EmptyRaise`, so the situation surfaces as a concrete, matchable failure
rather than a silent no-op.
