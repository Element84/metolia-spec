---
title: "Middleware providers"
weight: 20
---

A **middleware provider** plugs into the phase model: it is the integration a
middleware entry names in its `provider` field, and its contract is what gives
the entry's phases their behavior. This page defines the middleware-provider
contract—what a middleware provider's catalog entry declares—and the four
middlewares this specification defines: [`Retry`](#the-retry-middleware),
[`Timeout`](#the-timeout-middleware), [`Loop`](#the-loop-middleware), and
[`Finally`](#the-finally-middleware). The phase model itself—entries, stacks,
phases, `when`, re-entry—is defined in
[Middleware mechanics](../../middleware-mechanics/).

## What a middleware provider declares

A middleware provider's catalog entry carries the declarations common to every
provider: its URI, of type `provider.middleware`; its `codePrefix` and failure
catalog (see [The provider catalog](../#the-provider-catalog)). The rest of the
entry is the contract
[Middleware mechanics](../../middleware-mechanics/#what-a-middleware-declares)
requires of every middleware, in these kind-specific forms:

- **its phases**—for each phase where the middleware acts, the action's kind and
  behavior, and the parameter schema for that phase's `with`
  ([Per-phase declarations](#per-phase-declarations));
- **structural parameters**—which `with` members, if any, are definitions rather
  than values ([Structural parameters](#structural-parameters));
- **its attachment**—whether entries may attach at the Step level, the Flow
  level, or both (see
  [Where middleware attaches](../../middleware-mechanics/#where-middleware-attaches));
- **its contributed metadata**—the members it adds to `middleware.metadata`
  ([Contributed metadata](#contributed-metadata)).

For a complete middleware-provider specification in its machine-consumable form,
see the `Retry` middleware's: [`retry.v1.json`](../retry.v1.json). Each
spec-defined middleware publishes one beside this page.

### Per-phase declarations

A middleware declares, for each phase it acts at, one action—its kind
(side-effect, control, or transform) and what it does (see
[Author shaping and the middleware action](../../middleware-mechanics/#author-shaping-and-the-middleware-action))—and
a parameter schema, the schema that phase's `with` is validated against when the
phase runs (see [Validation](../../middleware-mechanics/#validation) and
[Parameter validation](../#parameter-validation)). The phases where a middleware
declares an action are the phases where `when` has effect (see
[`when`: gating the action](../../middleware-mechanics/#when-gating-the-action)).
A concurrent control action's declaration also states its acceptance semantics:
the commitment point past which it can no longer substitute.

A phase for which a middleware declares no parameter schema accepts no `with`:
validation is closed by default, so only an absent or empty `with` is valid
there.

### Structural parameters

A middleware MAY declare a parameter **structural**: its value is a definition,
not a value to compute. An ordinary `with` member is evaluated per field at the
phase boundary; a structural member is taken as written. Its subtree is
validated statically, as the same structure is anywhere else in a definition
(see [Static checks](../../flow-object/#static-checks)), and the expressions
inside it evaluate under their own construct's rules when the middleware's
action executes it, not at the phase boundary. The device is what lets a
middleware hold executable structure: the [`Finally`](#the-finally-middleware)
middleware's cleanup call and stack are the catalog's instances.

### Contributed metadata

A middleware declares the members it contributes to the execution context as a
JSON Schema describing an object, the same declaration device as a call
provider's metadata schema (see
[The metadata schema](../call-providers/#the-metadata-schema)). Contributed
members are entry-level state: they persist across the entry's phases and
surface beside the engine's universal pair on `middleware.metadata`, readable in
the entry's own phase blocks only (see
[Middleware-contributed metadata](../../execution-context/#middleware-contributed-metadata)).

A middleware MUST NOT expose contributed members beyond its declared schema, and
a contributed member MUST NOT be named `enteredAt` or `exitedAt`: those names
are the records' universal pair (see
[Metadata records](../../execution-context/#metadata-records)).

## Spec-defined middleware providers

This specification defines four middlewares, and an implementation MUST provide
all four: they are the language's control-flow and cleanup vocabulary, and
workflows in this specification's own examples depend on them.

| Middleware | Acts at                | Action kind          |
| ---------- | ---------------------- | -------------------- |
| `Retry`    | `onEntry`, `onFailure` | control              |
| `Timeout`  | `onEntry`              | control (concurrent) |
| `Loop`     | `onEntry`, `onSuccess` | control              |
| `Finally`  | `onAlways`             | side-effect          |

All four attach at either level.

Platforms extend the catalog by defining middleware providers under their own
namespaces, with the same contract (see
[The provider catalog](../#the-provider-catalog)). The middlewares appearing in
this specification's examples under the `example` namespace—cache, notify,
stac-index, translate-error—are illustrations, not specifications (see
[Reserved namespaces](../#reserved-namespaces)).

### The `Retry` middleware

```
mwl:provider.middleware/mwl/retry/v1
```

`Retry` re-runs its inner scope on matching failures. It acts at two phases:
`onEntry` arms the entry, evaluating and capturing its policies once at first
entry (see [`onEntry`](../../middleware-mechanics/#onentry)); `onFailure`
matches each rising failure against the captured policies and re-enters the
inner scope while the matching policy's attempt budget lasts. Its specification
document: [`retry.v1.json`](../retry.v1.json).

#### Parameters

`onEntry`:

| Parameter  | Type                          | Required | Default | Description                                                                                 |
| ---------- | ----------------------------- | -------- | ------- | ------------------------------------------------------------------------------------------- |
| `policies` | array of policies (non-empty) | required | —       | Scanned in array order; the first policy whose `match` matches a rising failure handles it. |

Each policy:

| Member     | Type                     | Required | Default  | Description                                                                                                            |
| ---------- | ------------------------ | -------- | -------- | ---------------------------------------------------------------------------------------------------------------------- |
| `match`    | object (failure matcher) | required | —        | The failures the policy handles, in the matcher grammar of [Failure matching](../../step-mechanics/#failure-matching). |
| `attempts` | integer ≥ 1              | required | —        | The policy's attempt budget, counting the first run.                                                                   |
| `backoff`  | object                   | optional | no delay | Backoff timing, below. Absent, retries are immediate.                                                                  |

`backoff`:

| Member    | Type              | Required | Default  | Description                                                                                                              |
| --------- | ----------------- | -------- | -------- | ------------------------------------------------------------------------------------------------------------------------ |
| `initial` | string (duration) | required | —        | The delay before the first retry, as a duration (see [the temporal profile](../../data-model/#temporal-format-profile)). |
| `rate`    | number ≥ 1        | optional | `1`      | The multiplier applied to the delay between consecutive retries.                                                         |
| `max`     | string (duration) | optional | none     | A cap on the per-retry delay.                                                                                            |
| `jitter`  | string            | optional | `"none"` | One of `"none"`, `"full"`, `"equal"`, `"decorrelated"`.                                                                  |

`onFailure`:

| Parameter | Type                          | Required | Default | Description                                                                                                      |
| --------- | ----------------------------- | -------- | ------- | ---------------------------------------------------------------------------------------------------------------- |
| `delay`   | string (duration) &#124; null | optional | `null`  | A per-failure override of the gap's delay: non-null on a consumed failure, the wait before re-entry, used as-is. |

#### Behavior

The `onEntry` action arms the entry: the policies are evaluated and captured at
first entry, like all `onEntry` configuration, and persist across every re-run.
Gated off—the phase's `when` false—the entry is transparent: nothing is armed,
and failures pass through. A flag in `vars` disabling retries for a whole run is
the typical use (see
[`when`: gating the action](../../middleware-mechanics/#when-gating-the-action)).

Each failure rising at the entry's position runs its `onFailure` phase. The
phase's action scans the captured policies in array order; the first whose
`match` matches the failure handles it (see
[Failure matching](../../step-mechanics/#failure-matching)), and each policy
counts its attempts independently. While the matching policy's budget lasts, the
action waits the gap's delay and re-enters the inner scope; every attempt
receives the same input, the value the entry passed down at first entry. When
the budget is exhausted, the action emits
[`Provider.Middleware.Retry.Exhausted`](#providermiddlewareretryexhausted). A
failure no policy matches passes through. The phase's `when` gates the action
per failure—gated off, the failure passes through. That is enablement, not
matching; matching belongs to the policies.

The gap's delay is ordinarily the matched policy's `backoff` value for that
attempt. The phase's `delay` parameter is the per-failure override: like any
ascent-phase `with`, it evaluates as the failure rises, with the failure in
scope, so it can wait on what only the failure knows—typically a server-supplied
retry-after hint in `details`:

```json
"onFailure": {
  "with": {
    "delay": "{{ has(middleware.result.details.retryAfter) ? middleware.result.details.retryAfter : null }}"
  }
}
```

When the action consumes a failure and `delay` is non-null, the gap waits
exactly `delay`: the policy's schedule is not consulted for that gap and no
jitter is applied. The override is per-gap—the backoff schedule is a function of
the attempt number, so a later gap without an override waits the schedule's
value for its own position. On a failure the action does not consume, on
pass-through and on exhaustion, `delay` has no effect. Matching stays
declarative either way: `delay` decides how long a consumed failure waits, never
whether a failure is consumed.

A consumed failure is not an emission, so the block's envelope keys evaluate
only when the phase emits—on pass-through and on exhaustion—while its `assign`
evaluates on every run of the phase (see
[Re-execution and re-entry](../../middleware-mechanics/#re-execution-and-re-entry)).

Re-entry restores variables. When the action decides to re-enter, it restores
the frame's variables to their state immediately following the entry's `onEntry`
evaluation—each variable's value then, or unbound if it had none—so a re-run
begins from the variable state the first run began from: a retry repeats the
same work, not a drifted variant of it. The gap's `assign` is the explicit
carry, the one deliberate way to move state across attempts: its expressions
evaluate against the pre-restore state—the failed attempt's aftermath, failure
in hand, under the ordinary block discipline—and its bindings take effect on the
restored variables. An arithmetic carry therefore chains across attempts:
`"tries": "{{ vars.tries + 1 }}"` in the gap reads the binding the previous gap
applied and advances it. No restore occurs on the entry's final emission; the
surviving attempt's writes stand. The entry's own state and contributed metadata
are not variables and persist untouched, and the data plane needs no restoring:
each attempt's expressions evaluate afresh, against the same delivered input.

```json
{
  "provider": "mwl:provider.middleware/mwl/retry/v1",
  "onEntry": {
    "with": {
      "policies": [
        {
          "match": { "codes": ["Provider.Call.Container.PlatformError"] },
          "attempts": 3,
          "backoff": {
            "initial": "PT30S",
            "rate": 2,
            "max": "PT90S",
            "jitter": "full"
          }
        }
      ]
    }
  },
  "onFailure": {
    "assign": { "attemptsSoFar": "{{ middleware.metadata.attempt }}" }
  }
}
```

Here each failed attempt captures the entry's attempt count
([`Retry` metadata](#retry-metadata)) into a variable: the gap's binding takes
effect on the restored variables, so the next attempt's expressions, and the
rest of the frame after the final emission, read how many runs the work has
taken. `Loop` is the deliberate contrast: an iteration is progress, not a
replay, so `Loop` persists variables across iterations where `Retry` restores
them (see [The `Loop` middleware](#the-loop-middleware)).

On the final emission the entry behaves as any entry does: a success, whether
first-run or recovered, rises through `onSuccess` once, a genuine success; an
exhausted or unmatched failure emits once, with the block's envelope keys
applied to it.

Where the entry sits in the stack decides what a retry repeats: a duration bound
outside `Retry` budgets all attempts together, one inside it budgets each
attempt separately (see
[The stack: ordering and composition](../../middleware-mechanics/#the-stack-ordering-and-composition)).

#### `Provider.Middleware.Retry.Exhausted`

Type `error`. Emitted when the matching policy's attempt budget is exhausted.
The final attempt's failure is chained as `previous` (see
[Chaining](../../execution-context/#chaining)), and `details` carries:

- `attempts`: the number of runs of the inner scope the entry made in total;
- `policy`: the position, in the `policies` array, of the exhausted policy.

#### `Retry` metadata

`Retry` contributes one member: `attempt`, a number—the runs of the inner scope
so far, counting the first. It is `1` during the first attempt; in an
`onFailure` run after the Nth attempt it is N. Like all contributed metadata it
is readable in the entry's own phase blocks only; carrying the count to the rest
of the frame is an `assign`—`onSuccess`'s, for the count a success took.

### The `Timeout` middleware

```
mwl:provider.middleware/mwl/timeout/v1
```

`Timeout` bounds its inner scope's execution time. Its single action, at
`onEntry`, is a concurrent control action: it races the inner scope from
establishment, and when the bound elapses first it interrupts the scope and
emits
[`Provider.Middleware.Timeout.Exceeded`](#providermiddlewaretimeoutexceeded).
Its specification document: [`timeout.v1.json`](../timeout.v1.json).

#### Parameters

`onEntry`:

| Parameter  | Type              | Required | Default | Description                                                                                                                      |
| ---------- | ----------------- | -------- | ------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `duration` | string (duration) | required | —       | The maximum time the inner scope may run, as a duration (see [the temporal profile](../../data-model/#temporal-format-profile)). |

#### Behavior

The bound is captured at first entry and spans the entry's whole participation:
every re-run of the inner scope by an entry inside `Timeout` shares it, while a
`Timeout` inside a re-running entry is re-established per run and bounds each
run separately (see
[The stack: ordering and composition](../../middleware-mechanics/#the-stack-ordering-and-composition)).
Gated off—the phase's `when` false—no bound exists for that pass. An enablement
flag is the honest spelling where a sentinel duration is not:

```json
{
  "provider": "mwl:provider.middleware/mwl/timeout/v1",
  "onEntry": {
    "when": "{{ vars.enforceTimeout }}",
    "with": { "duration": "PT15M" }
  }
}
```

`Timeout` is a concurrent control action, so its contract states its acceptance
semantics (see
[Author shaping and the middleware action](../../middleware-mechanics/#author-shaping-and-the-middleware-action)):
a Result rising at the entry's position is accepted when the platform receives
it, and once accepted it is committed—the bound can no longer fire, however
little time remains. At the bottom of a Step-level stack, the dispatch-level
acceptance instant is the Call record's `acceptedAt` (see
[`Call` metadata](../../step-actions/#call-metadata)).

When the bound elapses before acceptance, the action interrupts the inner scope
under the cancellation rules of
[Execution model](../../execution-model/#cancellation): it imposes a
cancellation whose `previous` is the explanatory failure—the
`Provider.Middleware.Timeout.Exceeded` envelope—and the scope unwinds, running
established inner entries' `onAlways` phases (see
[The unwind](../../execution-model/#the-unwind)). At the entry's own seam the
cancellation pops: the explanatory failure continues onto the forward path
alone, emitted from the entry's position as an ordinary failure (see
[The conversion seam](../../execution-model/#the-conversion-seam)). A chain the
unwind has since wrapped—a cleanup failure superseded the cancellation—is no
longer the entry's own bare cancellation and ascends as-is.

#### `Provider.Middleware.Timeout.Exceeded`

Type `timeout`. Emitted when the inner scope does not produce an accepted Result
within `duration`.

#### `Timeout` metadata

`Timeout` contributes one member: `deadline`, the instant the bound fires—
establishment plus `duration`—as a timestamp (see
[the temporal profile](../../data-model/#temporal-format-profile)). Like all
contributed metadata it is readable in the entry's own phase blocks only;
publishing it inward is an `assign`:

```json
"onEntry": {
  "with": { "duration": "PT15M" },
  "assign": { "deadline": "{{ middleware.metadata.deadline }}" }
}
```

after which everything in the inner scope can read `vars.deadline`—a target can
be told how much budget remains.

### The `Loop` middleware

```
mwl:provider.middleware/mwl/loop/v1
```

`Loop` re-runs its inner scope while its continuation holds. It declares no
parameters at all: a loop is configured entirely by `when`, the enablement
channel, at its two action phases—`onEntry`, the entry gate, and `onSuccess`,
the continuation. Its specification document: [`loop.v1.json`](../loop.v1.json).

A `Loop` entry MUST write `onSuccess.when`. That key is the loop's only
terminator; absent, it defaults to `true`, and the loop cannot end.

#### Behavior

`Loop`'s action owns every run of the inner scope, including the first. At
`onEntry`, the action admits the loop: gated off—the phase's `when` false—no run
occurs, and the entry emits its `onEntry` `output` product as a success Result,
the carried value of a loop of zero iterations. (Contrast `Retry`, whose action
owns only re-runs: a gated-off `Retry` still sees its inner scope run once.)

Each success rising at the entry's position runs its `onSuccess` phase. The
phase's `when` is the continuation: true, and the action re-enters the inner
scope; false, and the entry emits. The phase's `value` evaluates on every
iteration—it is the **carried value**, fed to the re-entered scope as its input
or, on the final iteration, emitted upward as the entry's success value (see
[Re-execution and re-entry](../../middleware-mechanics/#re-execution-and-re-entry)).
Its default passes each iteration's produced value through unchanged, so by
default each run's output is the next run's input. `assign` likewise evaluates
per iteration.

Variables persist across iterations: an iteration is progress, not a repeat, and
accumulating in `vars`—appending results, advancing a cursor, setting a done
flag—is idiomatic loop state. (Contrast `Retry`, which restores.)

A failure on any iteration is not the continuation's concern: `Loop` declares no
`onFailure` action, so the failure ascends past the entry unhandled. Re-running
on failure is `Retry`'s.

The continuation typically reads the iteration's Result, the frame's variables,
or the entry's own [`iteration`](#loop-metadata) count:

- A do-while loop is `onSuccess.when` alone, reading the Result:
  `"{{ middleware.result.value.nextCursor != null }}"`.
- A while loop, where the first run may already be unwanted, is the same
  predicate at both keys: `onEntry.when` is its first check, the zero-run case,
  and `onSuccess.when` re-checks it after each run. When the first run is always
  valid, `onSuccess.when` alone is the whole loop.
- A bounded loop reads the iteration count:
  `"{{ middleware.metadata.iteration < 100 }}"`, alone or as a conjunct.

A paginating Call:

```json
{
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
        "assign": {
          "items": "{{ vars.items + middleware.result.value.items }}"
        }
      }
    }
  ],
  "output": "{{ vars.items }}",
  "next": "process-items"
}
```

Each response is carried into the next dispatch as its input—the default carried
value—with the cursor riding in it; each iteration's items accumulate in
`vars.items` (declared in the Flow's `parameters` with a default of `[]`); when
a response carries no `nextCursor`, the loop emits and the Step's `output` reads
the accumulated list.

#### `Loop` metadata

`Loop` contributes one member: `iteration`, a number—the run of the inner scope
currently executing or, in an `onSuccess` run, just completed. It is `1` on the
first run.

`Loop` emits no failure codes of its own: its catalog is empty. An iteration's
failure is the inner scope's own, and a loop that must fail when a bound is
exceeded is a different shape—attempts against a budget are `Retry`'s, and
routing on an unsatisfactory final value is the Step graph's.

### The `Finally` middleware

```
mwl:provider.middleware/mwl/finally/v1
```

`Finally` dispatches a cleanup call on every exit. Its single action, at
`onAlways`, is a side-effect: it runs work—an audit write, a resource release, a
notification—without touching the Result in flight, at the one phase that runs
on every outcome (see [`onAlways`](../../middleware-mechanics/#onalways)). Once
the entry is established, the cleanup runs exactly once on the way out,
including when the scope is being torn down (see
[The unwind](../../execution-model/#the-unwind)). Its specification document:
[`finally.v1.json`](../finally.v1.json).

#### Parameters

`onAlways`:

| Parameter    | Type                        | Required | Default | Description                                      |
| ------------ | --------------------------- | -------- | ------- | ------------------------------------------------ |
| `call`       | call object                 | required | —       | The cleanup dispatch. Structural.                |
| `middleware` | array of middleware entries | optional | `[]`    | A stack around the cleanup dispatch. Structural. |

Both parameters are [structural](#structural-parameters): the call object and
the stack are definitions, exactly as
[The Call interface and Result](../../call-interface/) and
[Middleware mechanics](../../middleware-mechanics/) define them, validated
statically and evaluated by their own rules when the cleanup dispatches.

#### Behavior

When the phase runs—its `when` gates the action as anywhere, making cleanup
conditional—the action dispatches `call`, wrapped by `middleware` as a Step's
stack wraps its call. The cleanup call's fields evaluate against the phase's
bindings: its `input` can read `middleware.result`, the Result in flight at the
entry's position. The value the call arrives with—what an unwritten `input`
passes through—is the entry's `middleware.input`.

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

The cleanup's Result is discarded: `onAlways` stands outside the data plane, and
the Result in flight continues to rise unchanged (see
[`onAlways`](../../middleware-mechanics/#onalways)). What the cleanup learned is
captured, where wanted, by the cleanup call's own arms, whose `assign` writes
`vars` like any call's (see
[The arms](../../call-interface/#the-arms-onsuccess-and-onfailure)). `Finally`
contributes no metadata of its own.

The exception to discard is the phase's own rule: if the cleanup fails—the
failure the cleanup call emits after its own arms and stack have had their
say—the phase fails, and that failure supersedes the Result in flight under the
`onAlways` rules, chaining a superseded failure via `previous`. The superseding
failure carries its originator's code—the cleanup target's, or its stack's—so
`Finally`'s own catalog is empty.

A cleanup's stack is an ordinary stack: any middleware may appear in it,
`Finally` included. Where the entry sits in its own stack decides what its
cleanup observes—`middleware.result` at an outer position has been shaped by
everything inside it (see
[The stack: ordering and composition](../../middleware-mechanics/#the-stack-ordering-and-composition)).
