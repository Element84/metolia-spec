---
title: "Middleware mechanics"
weight: 80
---

Middleware wraps an operation: it attaches behavior around a unit of work
(retrying it, bounding its duration, caching its result, observing its outcome)
without being part of the work itself. The wrapped operation is one of two
things: the Call a `Call` Step dispatches, or a Flow's Step graph. In both cases
a `middleware` array forms a stack of wrappers around the operation: data
descends through the stack into the operation, and the operation's Result
ascends back out. Each entry in the stack participates at four phases, one on
the way down (`onEntry`) and up to two on the way back out (`onSuccess` or
`onFailure`, then `onAlways`).

This section defines the middleware entry, the phase model, and how a stack
composes, threads data, and handles failure. The concrete middlewares available
to a workflow, each with its configuration schemas and failure codes, are a
catalog concern: the specification's own middlewares are documented in
[Middleware providers](../providers/middleware-providers/), and platforms extend
the catalog through the same mechanism (see [Providers](../providers/)).

## The middleware entry

A middleware entry names a middleware by provider URI and configures its
participation phase by phase:

```json
{
  "provider": "mwl:provider.middleware/example/cache/v1",
  "onEntry": {
    "when": "{{ vars.replace == false }}",
    "with": {
      "key": "{{ middleware.input.id }}",
      "ttl": "{{ vars.cacheTTL }}"
    }
  }
}
```

| Field                                           | Type                  | Required | Default | Expression                              |
| ----------------------------------------------- | --------------------- | -------- | ------- | --------------------------------------- |
| `provider`                                      | string (provider URI) | required | —       | no (structural)                         |
| `onEntry`, `onSuccess`, `onFailure`, `onAlways` | object (phase block)  | optional | —       | see [The phase block](#the-phase-block) |

Middleware are providers: `provider` holds a middleware-provider URI, resolved
against the platform's catalog the way a Call's `provider` is. URI namespacing
is defined in [Providers](../providers/).

An entry may also carry a [`comment`](../definition-format/#the-comment-field)
for documentation.

Each of the four phase keys holds a [phase block](#the-phase-block): the
author's configuration of that middleware at that phase. An absent block is
equivalent to an empty one — every key in it defaults, and what the middleware
does at that phase still happens. What a middleware does at each phase is fixed
by its contract, not by the presence of a block (see
[What a middleware declares](#what-a-middleware-declares)).

## Where middleware attaches

Middleware attaches at two levels. The model is the same at both; only the
wrapped operation differs.

At the Step level, the `middleware` array on a `Call` Step wraps its dispatch:
the wrapped operation is the dispatch to the call's target, and the stack sits
strictly inside the Step's own fields. The Step's shaped `input` is what enters
the stack (see
[Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output)),
the Result the stack emits is the Step's Result, and the Step's `catch` matches
that Result from outside the stack (see
[Where `catch` sits](../step-mechanics/#where-catch-sits)). `Call` is the only
action that carries the field. A `Gather` carries no `middleware`: wrapping one
of its dispatches is a flow target's own flow-level stack, inside the dispatch's
frame, and wrapping the fan-out as a whole is the stack of a Flow whose graph
contains the `Gather`, attached at the Flow level below (see
[Step actions](../step-actions/)).

At the Flow level, the `middleware` array on a Flow object (see
[`middleware`](../flow-object/#middleware)) wraps the Flow's Step graph: the
wrapped operation is the execution of the graph from `entrypoint` to terminal
completion, including every Step's own routing. A failure that a Step's `catch`
routes to a terminal `Raise` reaches Flow-level middleware as the graph's
Result, not as something to intercept mid-graph. The Result the outermost entry
emits is the basis of the frame's Result; the completion contract is defined in
[Execution model](../execution-model/).

## The stack: ordering and composition

The `middleware` array is ordered outside-in: the first entry is the outermost
wrapper, each later entry nests inside the one before it, and the last entry
wraps the operation directly. Reading the array top to bottom reads the layers
from the outside inward.

Execution descends the stack and ascends back out. On the way down, each entry's
`onEntry` phase runs in array order; an entry is **established** once its
`onEntry` phase has completed. At the bottom, the wrapped operation runs and
produces a Result. On the way up, in reverse array order, each entry's
`onSuccess` or `onFailure` phase runs, selected by the Result rising at that
position, followed by its `onAlways`. An entry occupies one stack position: all
four of its phases run there, one on the descent and two at most on the ascent.

```json
"middleware": [
  {
    "provider": "mwl:provider.middleware/example/translate-error/v1",
    "onFailure": { "...": "..." }
  },
  {
    "provider": "mwl:provider.middleware/example/notify/v1",
    "onAlways": { "...": "..." }
  },
  {
    "provider": "mwl:provider.middleware/mwl/retry/v1",
    "onEntry": { "...": "..." }
  }
]
```

In this Step-level stack the retry middleware is last, so it wraps the Call
directly and re-runs just the Call (see
[Re-execution and re-entry](#re-execution-and-re-entry)). If the Call ultimately
fails, the failure ascends: past the retry entry once its policies are
exhausted; past the notify entry, whose `onAlways` sends its notification; and
through the translate-error entry last, whose `onFailure` rewrites the failure.
The rewritten failure is what the Step's `catch` matches — the outermost entry
has the final word on the envelope precisely because it runs last on the way
out.

Because each entry wraps everything below it, the same entries in a different
order are a different composition, sometimes a meaningfully different one. A
duration bound placed outside a retrying middleware budgets all attempts
together; placed inside it, each attempt gets its own budget. Both are
legitimate for some workload: ordering is the author's to choose, and the
language does not invalidate a composition for being unusual.

## The phase model

A phase is a crossing of the entry's boundary:

- `onEntry` runs on the way down, before anything inside the entry.
- `onSuccess` runs on the way up, when the Result rising at this position is a
  success.
- `onFailure` runs on the way up, when the rising Result is a failure. Failure
  here means any non-success Result; see the
  [terminology note](../call-interface/#result-types) in The Call interface and
  Result.
- `onAlways` runs on the way up after `onSuccess` or `onFailure`, whatever the
  outcome.

Exactly one of `onSuccess` and `onFailure` runs per ascent, selected by the
rising Result's `type` (see [Result types](../call-interface/#result-types)). No
phase crosses the boundary between the two: `onFailure` shapes a failure into
another failure and cannot synthesize a success, and a middleware that recovers
from a failure does not convert it — recovery means re-running the wrapped
scope, and the success it surfaces is the re-run's own (see
[Re-execution and re-entry](#re-execution-and-re-entry)).

### Author shaping and the middleware action

A phase carries at most two kinds of work, and the distinction between them
organizes everything else in this section.

The first is _author shaping_: the expressions the author writes into the phase
block — `output` on `onEntry`, `value` on `onSuccess`, the envelope fields on
`onFailure`, and `assign` on any phase. These are the author's own data-flow
code, carrying the same authority a Step's `input`, `output`, and `assign`
carry; the same shaping discipline recurs at the Step, the middleware phase, and
the call's arms (see
[Steps and step mechanics](../step-mechanics/#data-flow-input-and-output)).
Author shaping evaluates whenever its phase's data flow occurs. It is not part
of the middleware's behavior, and no middleware can prevent it.

The second is _the middleware's action_: what the middleware itself does at the
phase. A middleware has at most one action per phase, implementation-defined and
declared in its contract. The action is configured by the phase's `with` and
gated by the phase's `when` (see
[`when`: gating the action](#when-gating-the-action)). Every action is of one of
three kinds:

- A **side-effect** action acts without touching the data flow: send a
  notification, publish to an index, record an audit entry. The value in flight
  passes through unchanged.
- A **control** action passes the in-flight Result through or substitutes a
  whole Result of its own: serve a cached success instead of running the
  operation, preempt the operation with a timeout failure, re-run the wrapped
  scope and emit the re-run's Result. A control action never transforms a value
  in flight.
- A **transform** action performs an implementation-defined transformation of
  the value itself: decrypt a payload, decompress an archive, redact fields —
  work the author could not express as a shaping expression.

The line between the two slots is who authored the work, not what the work looks
like. An envelope rewrite written as expressions in an `onFailure` block is
author shaping; a middleware whose implementation maps failure codes performs a
transform action, gateable like any action, even though the two produce similar
effects.

A control action that acts concurrently with the wrapped operation — racing it,
as a timeout does, rather than acting strictly before or after it — MUST define
its acceptance semantics: the point at which the rising Result is committed and
the action can no longer substitute for it. A middleware with such an action
documents its acceptance semantics in its contract (see
[What a middleware declares](#what-a-middleware-declares)).

When an action substitutes a Result, ascent proceeds from that entry's position.
Whether a Result the action itself produced — a substituted Result, or a
re-run's — passes through that entry's own `onSuccess` or `onFailure` first, or
is emitted directly, is a property of the middleware, documented in its
contract. The specification fixes only what the rest of the stack sees: a single
Result rising from that position.

## The phase block

A phase block configures one phase of one entry. Three keys are accepted on
every phase; the remaining keys belong to specific phases:

| Field                                             | Phases      | Type                           | Required | Default                         | Expression                      |
| ------------------------------------------------- | ----------- | ------------------------------ | -------- | ------------------------------- | ------------------------------- |
| `when`                                            | any         | boolean                        | optional | `true`                          | yes (predicate)                 |
| `with`                                            | any         | object                         | optional | `{}`                            | yes (per field, or whole-value) |
| `assign`                                          | any         | object                         | optional | `{}`                            | yes (per value)                 |
| `output`                                          | `onEntry`   | any                            | optional | `{{ middleware.input }}`        | yes                             |
| `value`                                           | `onSuccess` | any                            | optional | `{{ middleware.result.value }}` | yes                             |
| `type`, `code`, `message`, `details`, `retryable` | `onFailure` | see [`onFailure`](#onfailure)  | optional | pass-through                    | yes                             |
| `previous`                                        | `onFailure` | non-success Result &#124; null | optional | the superseded failure          | yes                             |

`when` and `with` serve the action: `when` decides whether it runs, and `with`
configures it. `with` is the middleware's parameter namespace, validated against
the schema its contract declares for the phase (see [Validation](#validation)),
exactly as a Call's `with` is validated against its target's `parameters` (see
[The three axes](../call-interface/#the-three-axes-parameters-with-and-input)).

When a phase runs, its keys resolve in a fixed order: `when` first; then, if
`when` is true, `with` and the action; then the phase's shaping key; then
`assign`. When `when` is false, the action does not run and the phase's `with`
is not evaluated. The shaping key evaluates after the action, so a transform
action's effect is visible to it. The shaping keys are author shaping and
evaluate whenever their phase's data flow occurs, regardless of `when`; the
absent-field defaults in the table are the passthrough rule of
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough).

`assign` captures values into the frame's `vars`, evaluated against the phase's
bindings. It follows the same discipline as a Step's `assign` — every expression
in the block is evaluated against the variable state from before the block, and
the bindings take effect after it (see
[Variables: `assign`](../step-mechanics/#variables-assign)). The `vars` model
itself is defined in [The Flow object](../flow-object/#the-vars-model).

### `onEntry`

`onEntry` runs before anything inside the entry, and it runs once: its `when`,
`with`, and action are evaluated at first entry, and what they capture and
decide persists for the entry's lifetime, including across re-runs of the inner
scope (see [Re-execution and re-entry](#re-execution-and-re-entry)). Setup
configuration — retry policies, an iteration bound, a duration — is captured
here.

`output` shapes the value passed down to the next inner entry or, at the last
entry, into the wrapped operation. Its default passes the entry's input through
unchanged.

### `onSuccess`

`onSuccess` runs on ascent when the rising Result is a success. `value` produces
the success value the entry emits upward; the next outer entry sees it as its
`middleware.result.value`. The phase shapes a value, not a whole Result: a
success Result carries only its `value` (see
[Success Result](../call-interface/#success-result)), so emitting the value is
emitting the Result. The default passes the rising value through unchanged.

### `onFailure`

`onFailure` runs on ascent when the rising Result is a failure. Its shaping keys
are the authorable fields of the failure envelope — `type`, `code`, `message`,
`details`, `retryable`, and `previous` (see
[The failure envelope](../call-interface/#the-failure-envelope)) — and writing
any of them constructs a **new** failure, in the manner of a `Raise` (see
[Step actions](../step-actions/)). The new failure supersedes the rising one
and, unless the block writes `previous` itself, the engine links the superseded
failure as the new one's `previous`. Reshaping a failure is never mutation: the
original survives intact, one link down the chain. The chain's lifecycle is
defined in [Execution context](../execution-context/).

Each envelope field other than `previous` that the block does not write is taken
from the superseded failure, so a translation that changes only what it must is
exactly that small:

```json
"onFailure": {
  "code": "Pipeline.GranuleProcessingFailed",
  "details": { "stage": "l0-to-l1" }
}
```

This block emits a failure whose `code` and `details` are as written and whose
`type` and `message` are the rising failure's own, with the rising failure
chained as `previous`. A block that writes no envelope fields constructs
nothing: the rising failure passes through unchanged, and no chain link is
added.

Writing `previous` overrides the engine's link. The override is uncommon, and
its typical use is not forming a chain but severing one: a block that sets
`previous` to `null` emits its failure with the superseded failure's history
deliberately dropped, for the case where carrying it onward is unwanted.

One constraint follows from the envelope's own rules: the phase cannot cross
into success. The constructed failure's `type` MUST be a non-success type, and a
`type` expression that yields `success` is a validation failure (see
[Validation](#validation)).

### `onAlways`

`onAlways` is the cleanup phase. It runs on ascent after `onSuccess` or
`onFailure`, whatever the outcome — success, error, cancellation, or any other
non-success type — and it runs even when the same entry's `onSuccess` or
`onFailure` phase itself failed. Every established entry's `onAlways` runs
exactly once on the way out. That guarantee is what the phase is for: releasing
what `onEntry` acquired, sending a notification that must go out either way,
recording that the operation ran at all.

The guarantee extends to interruption. When the frame is being torn down —
cancelled from outside, or preempted by a control action such as a timeout — the
engine still ascends the stack, and established entries' `onAlways` phases run
on the way out. The normative account of cancellation and teardown belongs to
[Execution model](../execution-model/); what matters here is that `onAlways` is
the one phase an author can rely on in every exit.

In exchange, the phase stands outside
[the data plane](../concepts/#the-data-plane). It has no shaping key: the Result
rising past the entry continues to rise unchanged, and whatever the phase's
action and expressions produce is discarded. An `onAlways` phase passes the
in-flight Result through or supersedes it entirely; it can never
inspect-and-modify.

Supersession is the one exception to discard. If the `onAlways` phase itself
fails — its action fails, or one of its expressions faults — that failure
becomes the rising Result, and if a failure was already in flight the engine
chains it via `previous` (see [Execution context](../execution-context/)). A
failed cleanup is real and must surface; swallowing it would falsify the Result.

`when` is accepted on `onAlways` as on any phase and gates its action. "Always"
names the outcomes the phase runs on; whether its action is enabled is
orthogonal (see [`when`: gating the action](#when-gating-the-action)). `assign`
likewise works as on any phase.

## How values thread the stack

Threading is a full onion, with a defined value at every layer.

On the way down, the outermost entry receives the operation input: at the Step
level the `Call` Step's shaped `input`, and at the Flow level the frame's input.
Each entry's `onEntry` `output` becomes the next inner entry's input, and the
last entry's `output` becomes the input of the wrapped operation itself.

On the way up, the operation's Result ascends. Where the rising Result is a
success, each position's `onSuccess` `value` becomes the value the next outer
entry sees; where it is a failure, the envelope ascends unchanged unless a
position's `onFailure` constructs a successor. The Result the outermost entry
emits is the wrapped operation's yield: on a `Call` Step, the Step's Result —
what its `catch` matches and its `output` reads (see
[Steps and step mechanics](../step-mechanics/)); at the Flow level, the basis of
the frame's Result (see [Execution model](../execution-model/)).

Every shaping default passes through, so a stack whose blocks shape nothing is
transparent: the operation receives the input as the Step or frame shaped it,
and the Result emerges as the operation produced it.

### When a phase fails

A phase can itself fail: an expression faults (see
[Evaluation errors](../expressions/#evaluation-errors)), a `with` fails
validation (see [Validation](#validation)), or the action fails. The failure is
emitted from that entry's position and ascends from there, exactly as a failure
rising from below would: outer entries see it in their `onFailure` phases, and
the entry that produced it does not handle its own failure.

What the failure displaces depends on the crossing. On the descent, an `onEntry`
failure means the wrapped operation never runs and the entries below are never
established; the failing entry itself is not established either, so its own
`onAlways` does not run. On the ascent, the failure supersedes the rising
Result: a superseded failure is chained via `previous` (see
[Execution context](../execution-context/)); a superseded success is simply
displaced — the chain records failures, not the success they displaced. A
failure in an ascent phase leaves the entry established, so its `onAlways` still
runs; a failure in `onAlways` itself is the supersession case defined with
[that phase](#onalways).

## Bindings

Expressions in a phase block read the frame's `vars`, like expressions anywhere
in the frame (see [The `vars` model](../flow-object/#the-vars-model)), and the
`middleware` root, whose members are position- and phase-relative:

| Binding               | `onEntry` | `onSuccess` | `onFailure` | `onAlways` |
| --------------------- | --------- | ----------- | ----------- | ---------- |
| `middleware.input`    | yes       | yes         | yes         | yes        |
| `middleware.result`   | —         | yes         | yes         | yes        |
| `middleware.metadata` | yes       | yes         | yes         | yes        |

`middleware.input` is the input this entry received: the value the enclosing
entry's `onEntry` `output` produced or, at the outermost position, the operation
input. It is stable across all four phases of the entry, so an ascent phase can
still read the input that led to the result.

`middleware.result` is the full Result rising at this position: read
`middleware.result.value` on a success, the envelope fields on a failure (see
[The Result](../call-interface/#the-result)). It does not exist in `onEntry`,
where the operation has not run. In `onAlways` it is the in-flight Result,
success or failure.

`middleware.metadata` is the current phase's metadata record: the phase's own
timing, beside the members the middleware contributes, which persist across the
entry's phases (see [What a middleware declares](#what-a-middleware-declares)).
It is valid in every phase and, like the other members, position-relative.

The remaining roots in scope are those of the contexts that enclose the stack.
The ambient roots are in scope in a phase block as in any expression the frame
evaluates: `execution`, `frame`, and, while a failure is being handled,
`failure`. In a Step-level stack, `step` is in scope as well: the enclosing
Step's identity, received input, and entry instant are settled before the stack
establishes, and a phase reads them as the Step's other fields do. What the Step
has not yet produced is not readable there: `step.result` is the Result the
outermost entry emits, settled only after every phase has run, and
`step.metadata.exitedAt` is set at that same settlement (see
[`step`](../execution-context/#step)). In a Flow-level stack no Step is
executing when a phase runs (the descent precedes the entry Step; the ascent
follows the graph's completion), so `step` is not in scope.

No context interior to the wrapped operation is in scope: the `call` root
belongs to the dispatch inside the stack, and the target windows to its arms
(see [`call`](../execution-context/#call)). A phase reads the value in flight at
its own position through its `middleware` root.

This table states where the bindings are valid. Their shapes, with the rest of
the runtime data model, are defined in
[Execution context](../execution-context/).

## `when`: gating the action

Every phase accepts `when`: an expression evaluated as a predicate (see
[Predicates and `when`](../expressions/#predicates-and-when)) that gates the
phase's [action](#author-shaping-and-the-middleware-action). It is optional and
defaults to `true`; an absent `when` means the action fires whenever its phase
runs.

`when` is evaluated at the phase's boundary, against the phase's full
[bindings](#bindings), before the phase's `with`. When it is false, the action
does not run and `with` is not evaluated. It gates the action only: author
shaping always evaluates, whatever `when` decides.

`when` is accepted on any phase, but it bites only where the middleware declares
an action: on a phase with no action there is nothing to gate, and the key is a
no-op. A middleware's contract MUST document which phases expose a gateable
action, so that authors know where `when` has effect (see
[What a middleware declares](#what-a-middleware-declares)).

`when` is the enablement channel, uniform across all middleware and distinct
from configuration: `with` describes how the action behaves, `when` decides
whether it runs.

```json
"middleware": [
  {
    "provider": "mwl:provider.middleware/example/cache/v1",
    "onEntry": {
      "when": "{{ vars.replace == false }}",
      "with": {
        "key": "{{ middleware.input.id }}",
        "ttl": "{{ vars.cacheTTL }}"
      }
    }
  },
  {
    "provider": "mwl:provider.middleware/example/stac-index/v1",
    "onSuccess": {
      "when": "{{ vars.publish == true }}",
      "with": {
        "items": "{{ middleware.result.value.features }}"
      }
    }
  }
]
```

Here one Flow parameter disables the cache lookup for a run that must recompute,
and another gates publication: feature flags, the typical use. Keeping
enablement in `when` rather than encoding it into a configuration value — a
duration set to zero to disable a bound, an empty recipient list to disable a
send — keeps the two channels honest: configuration describes the action, and
`when` decides it.

> [!WARNING]
> Gating a transform action makes the downstream shape conditional
>
> All three action kinds are gateable, including transforms. A gated transform —
> a decryption that runs only when a flag is set — leaves the payload in a
> different shape on each branch, and everything downstream must cope with both.
> This is the same footgun as any conditional shaping expression, such as a
> ternary `value` whose branches produce different shapes. It is the author's to
> manage; the language does not restrict it.

A `Match` clause's `when` is the other site of this field name. The two share
the predicate contract and nothing else; the `Match` use is defined with that
action (see [Step actions](../step-actions/)).

## Re-execution and re-entry

A control action may re-run its inner scope: the wrapped operation together with
every entry below the re-running one. A retrying middleware re-runs on a
matching failure; an iterating middleware re-runs while its continuation holds.
`Retry` and `Loop`, the catalog's canonical pair, are documented in
[Middleware providers](../providers/middleware-providers/).

Re-entry has exactly one target: the inner scope. The re-running entry's own
`onEntry` is not part of it. `onEntry` evaluates exactly once, at first entry,
and the configuration it captured persists across every re-run; there is no
re-entry path that re-evaluates an entry's own setup.

Each re-run re-establishes the inner scope from scratch. Inner entries'
`onEntry` phases run afresh — a fresh `when` decision, a fresh `with` — and
whatever state they keep restarts. A re-run therefore resets everything below
it: an iteration count kept by an inner entry starts over, and a duration bound
inside the re-running entry budgets each run separately, while one outside it
spans all runs (see
[The stack: ordering and composition](#the-stack-ordering-and-composition)).

The Result a re-run produces ascends as itself. A success obtained on the third
attempt is a genuine success — outer entries' `onSuccess` phases fire, and
nothing has converted a failure into a success. Whether that Result passes
through the re-running entry's own `onSuccess` or `onFailure` first is, as with
substitution generally, a property of the middleware documented in its contract.
Where an iterating middleware carries a value from one run into the next, the
carried value is the one its `onSuccess` emitted; how it feeds the next run is
the middleware's contract to define.

A Result the re-running entry's action consumes is not emitted: it rises to the
entry's position and runs the phase there, but ascent stops where the action
re-enters. On such a run the phase's `assign` still evaluates—`assign` evaluates
on every run of its phase—while the shaping key evaluates only where its product
has a destination: an emission upward, or a feed the contract defines, as with a
carried value.

## Validation

A phase's `with` is validated against the parameter schema the middleware's
contract declares for that phase, at the time the phase runs: an `onEntry`
`with` at first entry, an ascent phase's `with` as the Result rises past the
entry. A value that fails the schema produces
`System.ParameterValidationFailed`, whose meaning is defined in
[The Flow object](../flow-object/#systemparametervalidationfailed); the failure
is emitted from the entry's position as described under
[When a phase fails](#when-a-phase-fails).

Several constraints on middleware usage are statically checkable: that each
`provider` URI resolves against the platform's catalog, and that the middleware
is used at a level its contract declares applicable. These follow the
static-checking policy of [The Flow object](../flow-object/#static-checks).

## What a middleware declares

Every middleware, spec-defined or platform-defined, has a contract in the
platform's catalog; the catalog format and the spec-defined middlewares are
documented in [Providers](../providers/). For the mechanics in this section, the
contract is where a middleware declares:

- its provider URI (see [Providers](../providers/));
- the action it performs at each phase, and each action's kind (see
  [Author shaping and the middleware action](#author-shaping-and-the-middleware-action));
- which phases expose a gateable action (see
  [`when`: gating the action](#when-gating-the-action));
- for a concurrent control action, its acceptance semantics;
- the parameter schema for each phase's `with` (see [Validation](#validation));
- the attachment levels it supports (see
  [Where middleware attaches](#where-middleware-attaches));
- the metadata it contributes to the execution context (see
  [Execution context](../execution-context/));
- the failure codes it can emit, reported under its
  `Provider.Middleware.<codePrefix>.` prefix (see
  [Code namespaces](../call-interface/#code-namespaces)).

The concrete middlewares — each one's per-phase schemas, behavior, and codes —
are the catalog's to define; the phase model in this section is what every one
of them plugs into.
