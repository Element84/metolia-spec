---
title: "Execution context"
weight: 100
---

Expressions read the state of a running workflow through a fixed set of named
bindings: the binding roots enumerated in
[Expressions](../expressions/#evaluation-context-the-binding-roots). This
section defines the runtime data model behind those roots — the members under
each root, their types, when each is populated, and how long it lives — and
consolidates the [field defaults](#field-defaults-and-passthrough) that the
passthrough rule of
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough)
refers to.

Three roots are ambient: `vars`, `execution`, and `frame` are in scope in every
expression a frame evaluates. The rest are site-specific: each is in scope where
its construct evaluates expressions — `call` within a `call` object's fields,
`middleware` within a phase block — and each field's own documentation states
the bindings its expressions see. The tables in this section describe runtime
values, not definition fields: each lists a binding's members, with each
member's type and availability.

## `vars`

`vars` is the frame's variable namespace: an object with one member per bound
variable, holding the value most recently bound to that name. An expression
reads a variable as a member of the root — `vars.collection`, `vars.replace` —
and the namespace is flat: there is no nesting beyond what a variable's own
value carries.

The variable model is defined in
[The `vars` model](../flow-object/#the-vars-model): how a Flow's `parameters`
seed the namespace at frame entry, how `assign` writes it during execution, and
its scoping to the frame. Reading a name with no binding is an evaluation error
(see [Evaluation errors](../expressions/#evaluation-errors)); a parameter that
is neither required nor defaulted may be unbound, and an expression that must
tolerate that guards the read (see
[Defensive constructs](../expressions/#defensive-constructs)).

## `execution`

An execution is one run of a root Flow. The `execution` root exposes the
execution's identity and timing, and it is the one binding whose values are the
same in every frame: a subflow or a `Gather`-dispatched frame sees exactly what
the root frame sees.

| Member     | Type   | Description                                                                  |
| ---------- | ------ | ---------------------------------------------------------------------------- |
| `id`       | string | A platform-assigned identifier for the execution.                            |
| `metadata` | object | The execution's metadata record (see [Metadata records](#metadata-records)). |
| `platform` | object | Platform-defined members; empty when the platform adds none.                 |

`id` identifies the execution to the platform running it. Its format is
platform-defined, and its value is stable for the life of the execution. It is
the value a workflow includes where an external system must be able to refer
back to the run: a notification message, an audit record, a correlation key.

`execution.metadata` is the execution's record. The execution begins and ends
with its root frame — within the model there is no execution behavior outside
that frame's run — so the record's `enteredAt` and `exitedAt` are equal to the
root frame's (see [Metadata records](#metadata-records)). This is what makes the
run's start reachable from anywhere: a subflow cannot reach the root frame, but
`execution.metadata.enteredAt` is in scope in every frame. `exitedAt` is set
when the root frame completes, after the last expression of the run has
evaluated, so it is recorded but reachable from no expression. What precedes the
root frame — submission, queueing, scheduling — is outside the model; a platform
that tracks such instants MAY expose them under `execution.platform`.

`platform` is the platform's extension surface. A platform MAY expose additional
runtime data as members of `execution.platform`, with whatever names and
meanings it defines; a workflow that reads them is portable only across
platforms that expose the same members. This is the only ad-hoc extension
surface in the execution context: elsewhere, an implementation MUST NOT expose
members beyond those this specification defines and those declared through its
defined extension points — an action's metadata members, a middleware's
contributed members (both parts of the [metadata records](#metadata-records)
described next), and a call provider's window metadata (see
[`provider`](#provider)).

## Metadata records

An execution, a frame, a Step, a middleware phase, a Call, and a `Match`'s match
context are each a tracked context: a unit of execution that begins, runs, and
ends. The engine keeps a metadata record for every one, and the records share
two universal members:

| Member      | Type               | Description                                                                                                                   |
| ----------- | ------------------ | ----------------------------------------------------------------------------------------------------------------------------- |
| `enteredAt` | string (timestamp) | The instant the context began, an RFC 3339 timestamp (see [Temporal format profile](../data-model/#temporal-format-profile)). |
| `exitedAt`  | string (timestamp) | The instant the context ended, likewise.                                                                                      |

`enteredAt` is set when the context begins and is readable from within it. For
the contexts whose own fields evaluate expressions — a Step, a middleware phase,
a Call — it is the same instant the clock pin fixes, so `now()` within the
context returns it (see [The clock pin](../execution-model/#the-clock-pin)).

`exitedAt` is set when the context exits — the moment its work settles and its
product is in hand. The fields that shape and capture that product are the
context's _tail_, and they evaluate after the exit instant, while the context
remains in scope: a Step's `output`, `assign`, and `catch` clauses, and a Call's
arms all read a completed context, its `exitedAt` included (see
[The Step lifecycle](../execution-model/#the-step-lifecycle)). Beyond its own
tail, a completed context is observable wherever another context exposes it: a
completed frame through the [`flow`](#flow) window. Two members are reachable
from no expression: the execution's `exitedAt`, set after the run's last
expression has evaluated ([`execution`](#execution)), and a middleware phase's,
since a phase's shaping and capture are part of the phase itself, leaving no
tail to read it. Both are recorded regardless — the record is uniform whatever a
given vantage can see.

Beside the universal members, each context's record carries its own: a Step's
action-specific members, a Call's dispatch instants, and, on a middleware
phase's record, the members the middleware contributes — entry-level state that
persists across the entry's phases (see
[Middleware-contributed metadata](#middleware-contributed-metadata)). Each is
given with its context below.

The two remaining roots carry no record, because they view no execution: `vars`
is a namespace of author-bound names, and `failure` is the failure envelope
exactly (see [`failure`](#failure)).

## `frame`

A frame is the execution-time instantiation of a Flow (see
[Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).
Two bindings expose a frame, at two points in its life: the `frame` root is the
current frame — the one whose Flow contains the expression being evaluated —
observed from within while it executes, and the [`flow`](#flow) window is a
completed frame, observed from the call site that ran it. They are the same kind
of object; what differs is which members hold a value yet.

| Member     | Type   | Available                 | Description                                        |
| ---------- | ------ | ------------------------- | -------------------------------------------------- |
| `input`    | any    | always                    | The frame's input, set at creation and immutable.  |
| `metadata` | object | always                    | The frame's execution metadata.                    |
| `vars`     | object | completed frames (`flow`) | The frame's variables as they stood at completion. |
| `result`   | Result | completed frames (`flow`) | The Result the frame produced.                     |

The completion members, `vars` and `result`, settle when the frame completes. By
then no expression of the frame itself remains to run, so they are read through
the `flow` window; within the executing frame, the variables are live and are
read through the ambient `vars` root, and the frame's Result does not yet exist
(see [The frame lifecycle](../execution-model/#the-frame-lifecycle)).

`frame.input` is the value the frame was created with: the execution input for
the root frame, or the call's evaluated `input` for a called Flow (see
[Where Flows appear](../flow-object/#where-flows-appear)). It never changes.
Flow-level middleware reshapes the value delivered to the Step graph on the way
down (see
[How values thread the stack](../middleware-mechanics/#how-values-thread-the-stack)),
not the frame's input, so an expression anywhere in the frame can always recover
the value the frame was given.

`frame.metadata` is the frame's [metadata record](#metadata-records):

| Member      | Type               | Available                 | Description                        |
| ----------- | ------------------ | ------------------------- | ---------------------------------- |
| `enteredAt` | string (timestamp) | always                    | The instant the frame was created. |
| `exitedAt`  | string (timestamp) | completed frames (`flow`) | The instant the frame completed.   |

The frame's entry instant has its own member because no clock pin supplies it:
`now()` is pinned to the entry instant of the construct execution evaluating it
(a Step, a middleware phase, or a Call), never to the frame (see
[The clock pin](../execution-model/#the-clock-pin)). The frame's elapsed time is
therefore the difference between the clock and `frame.metadata.enteredAt`.
`exitedAt` follows the completion members' rule: it is recorded at completion
(see [The frame lifecycle](../execution-model/#the-frame-lifecycle)), so it is
read on a completed frame, through the `flow` window.

The `frame` root reaches the current frame only; frames are isolated, and data
crosses a frame boundary only explicitly — into a called frame through its
call's `input` and `with`, and back out through its Result and the `flow` window
(see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)).

## `step`

The `step` root exposes the Step currently executing in the current frame. Every
Step execution has its own binding: `step` always refers to the Step being
executed, and nothing carries into a successor — a value needed beyond the Step
is captured into `vars` with `assign` (see
[Variables: `assign`](../step-mechanics/#variables-assign)).

`step` is in scope throughout the Step's definition: its shared fields, the
fields of its action — a `call` object's fields, a `Call` Step's or a `Gather`
dispatch's alike, and a `Match` clause's — the phase blocks of its `middleware`
stack, and its `catch` clauses. A Flow-level middleware phase block belongs to
no Step's definition and runs while no Step is executing, so `step` is not in
scope there (see [Bindings](../middleware-mechanics/#bindings)).

| Member     | Type   | Available                       | Description                                                                                |
| ---------- | ------ | ------------------------------- | ------------------------------------------------------------------------------------------ |
| `name`     | string | always                          | The Step's key in its Flow's `steps` map.                                                  |
| `id`       | string | always                          | A platform-assigned identifier for this Step execution.                                    |
| `action`   | string | always                          | The Step's `action` discriminator.                                                         |
| `input`    | any    | always                          | The value the Step received.                                                               |
| `result`   | Result | after the action; `Call` only   | The Step's Result, as the outermost middleware entry emitted it.                           |
| `results`  | array  | after the action; `Gather` only | The `Gather`'s collected Results, one per dispatch (see [Step actions](../step-actions/)). |
| `metadata` | object | always                          | Engine-tracked execution metadata (below).                                                 |

"Always" means from Step entry, for the whole execution; "after the action"
means from the Result phase of the Step lifecycle on (see
[The Step lifecycle](../execution-model/#the-step-lifecycle)), where the
action's product becomes readable by `output`, `assign`, and the `catch`
clauses.

`name` and `id` together identify a Step execution. A Step name recurs wherever
an enclosing construct re-runs its scope — a retried Call, an iterated graph —
and `id` is unique to each execution where `name` is not. That uniqueness is
what makes `id` the building block for an idempotency key: a key constructed
from `execution.id` and `step.id` and passed to a provider lets work dispatched
more than once be deduplicated to exactly-once effect, and it ties a Step
execution to the platform's own records of it.

`step.input` is the value the Step received: the previous Step's output (on a
failure edge, the matched `catch` clause's output) or, for the entry Step, the
value the frame's descent delivered (see
[The frame lifecycle](../execution-model/#the-frame-lifecycle)). It is the
received value, before the Step's own `input` shaping; the shaped value is in
scope where it lands — `call.input` in a Call's fields, `match.input` in a
Match's clauses, `middleware.input` in the stack.

`step.result` is the single Result the Step resolved to, whatever its type (see
[Where `catch` sits](../step-mechanics/#where-catch-sits)). On a success,
`step.result.value` is what the Step's `output` reads by default; on a failure
it is the same envelope the [failure context](#failure) exposes.

### Step metadata

`step.metadata` is the Step's [metadata record](#metadata-records):

| Member      | Type               | Description                               |
| ----------- | ------------------ | ----------------------------------------- |
| `enteredAt` | string (timestamp) | The instant the Step execution began.     |
| `exitedAt`  | string (timestamp) | The instant the Step execution completed. |

`enteredAt` records the instant the clock pin fixes for the expressions the
Step's own lifecycle evaluates — its `input`, `output`, and `assign`, and its
clauses' fields: there, `now()` returns this same instant (see
[The clock pin](../execution-model/#the-clock-pin)). A context nested inside the
Step, such as a middleware phase or a Call, pins its own entry instant instead.
`exitedAt` is set when the Step's work settles — its action's product in hand,
before the tail runs (see
[The Step lifecycle](../execution-model/#the-step-lifecycle)) — and is read in
that tail: `output`, `assign`, and the `catch` clauses see the completed Step,
so an `assign` can capture the Step's span,
`durationToIso8601(timestamp(step.metadata.exitedAt) - timestamp(step.metadata.enteredAt))`.

The remaining members are action-specific: each action defines what its
execution records, with the action's reference (see
[Step actions](../step-actions/)). A `Call` Step's dispatch timing is not among
them — the Call is its own tracked context, and the dispatch instants belong to
its record (see [`call`](#call)).

### Middleware-contributed metadata

A middleware may contribute metadata: what it measured, decided, or did.
Contributed members are declared in the middleware's contract (see
[What a middleware declares](../middleware-mechanics/#what-a-middleware-declares))
and defined per middleware in its catalog entry. Unlike the phase-specific
universal members they ride beside, contributed members are entry-level state:
they persist across the entry's phases, so an ascent phase reads what descent
contributed. They share the record with the universal members and cannot take
their names, a constraint of the catalog format (see
[Providers](../providers/)).

Contributions surface in one place: `middleware.metadata`, beside the current
phase's universal members, in the entry's own phase blocks (see
[`middleware`](#middleware)). They are readable nowhere else. A contribution
needed beyond the entry — by the Step's tail, a `catch` clause, a later Step —
is carried forward explicitly: a phase's `assign` captures it into `vars`, and
later expressions read the variable (see
[Variables: `assign`](../step-mechanics/#variables-assign)).

An entry that must time its whole wrap captures the start the same way: its
`onEntry`'s `assign` records that phase's entry instant into `vars`, and a later
phase computes against it —
`"assign": { "wrapStart": "{{ middleware.metadata.enteredAt }}" }` at `onEntry`,
then `durationToIso8601(now() - timestamp(vars.wrapStart))` at `onAlways`. A
middleware that tracks such spans itself exposes them as contributed members.

## `call`

The `call` root is in scope within a `call` object's expression fields —
`input`, `with`, and its arms' members — and nowhere outside the `call` object
(see [The call object](../call-interface/#the-call-object)). The target windows
[`flow`](#flow) and [`provider`](#provider) share the arms' scope.

| Member     | Type   | Available                                         | Description                                                                                       |
| ---------- | ------ | ------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `input`    | any    | all `call` fields                                 | The data payload arriving at the Call's position.                                                 |
| `index`    | number | all `call` fields; `Gather`-dispatched calls only | The dispatch's 0-based position in its `Gather`'s fan-out (see [Step actions](../step-actions/)). |
| `result`   | Result | the arms (`onSuccess`, `onFailure`), post-exit    | The Result the target produced, before `value` shaping: the target window's `result`.             |
| `metadata` | object | all `call` fields                                 | The Call's metadata record (see [Metadata records](#metadata-records)).                           |

`call.input` is the payload in flight at the call site. On a `Call` Step it is
the value the Step's middleware stack delivered: the Step's shaped input threads
down the stack, and what the innermost entry emits arrives as `call.input` (see
[How values thread the stack](../middleware-mechanics/#how-values-thread-the-stack));
with no middleware it is the Step's shaped input itself. On a `Gather` dispatch
it is the dispatch's inbound payload, arriving at the call boundary directly —
the element, in the iterate form; the value the Step received, in the scatter
form (see [Step actions](../step-actions/)). The call's `input` field defaults
to `{{ call.input }}`: passthrough of this binding into the target.

`call.index` exists only on a call a `Gather` dispatched: the element's 0-based
position in the iterate form, the call's position in `calls` in the scatter
form, uniformly available across the dispatch's fields in both forms (see
[Step actions](../step-actions/)). A `Call` Step's call carries no `index`.

`call.result` is the target's own Result, in scope in both arms. On a success,
the `onSuccess` arm's `value` reads it to shape the value the Call's success
Result carries (see
[`onSuccess`](../call-interface/#onsuccess-shaping-and-capture)); on a failure,
it is how the failure arm reads the envelope — `call.result.code` and the rest —
the frame's `failure` context being unset there (see [Lifecycle](#lifecycle)).
It is not a member of its own: it denotes the target window's `result` —
`flow.result` for a flow target, `provider.result` for a provider target — so
one name serves both target kinds (see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)).
It is the Result at the call boundary, before the middleware stack processes the
ascent; what emerges from the outermost entry is the Step's `step.result`.

`call.metadata` is the Call's record. `enteredAt` is set when the Call begins:
the stack has delivered its input and the call's fields evaluate. `exitedAt` is
set when the target's Result arrives — the Call's exit — and the arm that
follows reads the completed Call. An arm's `assign` is therefore where
call-boundary data is carried into `vars`; its latency, for one:
`durationToIso8601(timestamp(call.metadata.exitedAt) - timestamp(call.metadata.enteredAt))`.
The Call's context-specific members — among them the instants at which the
request was dispatched to and accepted by the target — are defined with the
action (see [Step actions](../step-actions/)). A Call re-dispatched by a
retrying middleware is a new Call execution with a fresh record (see
[Re-execution evaluates afresh](../execution-model/#re-execution-evaluates-afresh)).

## `flow`

The `flow` root is one of the two target windows (see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)):
it exposes the completed frame a flow-targeted call ran, to that call's arms —
the `onSuccess` and `onFailure` of the `call` object that dispatched it (see
[The arms](../call-interface/#the-arms-onsuccess-and-onfailure)). The window is
in scope in both arms, and each call execution's arms see their own execution's
window: a retried Call reads each attempt's frame in that attempt's arm, and
nothing of the window outlives the call object. Past the call object, a failed
subflow communicates through its failure Result plus what the failure arm
captured — an inner Flow that must surface state beyond what its caller captures
places it in the failure envelope it raises (see
[The failure envelope](../call-interface/#the-failure-envelope)).

`flow` is a frame, with every member of [the frame shape](#frame) populated: the
frame has completed, so alongside its `input` and `metadata`, `exitedAt` now
among it, the completion members are settled. `result` is the Result the frame
produced, the same Result `call.result` denotes at this site, and `vars` is its
variable namespace at the moment it completed.

Nothing is promoted on its own: an arm's `assign` captures `flow.result.value`,
`flow.vars.<name>`, or whatever else later fields need into the frame's
variables (see [Variables: `assign`](../step-mechanics/#variables-assign)).

## `provider`

The `provider` root is the other target window: it exposes a provider target's
completed execution to that call's arms, under the same scope rules as `flow` —
both arms, each call execution's own, nothing outside the call object.

| Member     | Type   | Description                                                                 |
| ---------- | ------ | --------------------------------------------------------------------------- |
| `input`    | any    | The value the provider received: the product of the call's `input` field.   |
| `result`   | Result | The Result the provider produced — what `call.result` denotes at this site. |
| `metadata` | object | The members the provider's catalog declares; empty when it declares none.   |

`provider.metadata` is the provider's own reporting surface: what it measured,
decided, or did, beyond the Result itself. Its members are declared per provider
in the catalog (see [Providers](../providers/)), and the namespace is wholly the
provider's: it carries no engine universals — the Call's own record (see
[`call`](#call)) owns the engine's timing of the dispatch — and this
specification will never define a member under it, so a provider's declared
names cannot collide with a future version. It is a delegated namespace in the
manner of `execution.platform` (see [`execution`](#execution)).

## `match`

The `match` root is in scope in a `Match` Step's clause expressions: each
clause's `when`, `output`, and `assign` (the clause grammar is defined with the
action; see [Step actions](../step-actions/)).

| Member     | Type   | Description                                                                      |
| ---------- | ------ | -------------------------------------------------------------------------------- |
| `input`    | any    | The value the clause predicates test: the Step's shaped input.                   |
| `metadata` | object | The match context's metadata record (see [Metadata records](#metadata-records)). |

`match.input` is the product of the Step's `input` field, evaluated once at
input shaping (see
[The Step lifecycle](../execution-model/#the-step-lifecycle)); every clause's
expressions read that same value, and `step.input` still recovers the value the
Step received. A clause's `output` defaults to `{{ match.input }}` (see the
[defaults table](#field-defaults-and-passthrough)).

The match context begins and ends with its containing Step: there is no timing
difference between the Step's entry and exit and the match context's, so
`match.metadata`'s `enteredAt` and `exitedAt` are equal to the Step's (see
[Step metadata](#step-metadata)). Its span coincides with its container's, as
the execution's coincides with its root frame's (see [`execution`](#execution));
every other nested context — a Call, a middleware phase — has timing of its own.

## `middleware`

The `middleware` root is in scope within a middleware entry's phase blocks. Its
members are position-relative: every entry sees its own input and the Result
rising at its own position, wherever the entry sits in the stack.

| Member     | Type   | Description                                                                                                                |
| ---------- | ------ | -------------------------------------------------------------------------------------------------------------------------- |
| `input`    | any    | The input this entry received: the enclosing entry's `onEntry` `output`, or the operation input at the outermost position. |
| `result`   | Result | The Result rising at this entry's position. Ascent phases only.                                                            |
| `metadata` | object | The current phase's metadata record (see [Metadata records](#metadata-records)).                                           |

`middleware.input` is stable across an entry's phases; `middleware.result`
exists once there is a Result to rise, in `onSuccess`, `onFailure`, and
`onAlways`. `middleware.metadata` is the current phase's record, like every
other context root's: its `enteredAt` is the instant the phase began — the same
instant `now()` returns there — while the contributed members it carries are
entry-level state, persisting across the entry's phases (see
[Middleware-contributed metadata](#middleware-contributed-metadata)). The
per-phase validity of each member, and what each means at its phase, are given
in [Bindings](../middleware-mechanics/#bindings); the Result shape is defined in
[The Result](../call-interface/#the-result).

## `failure`

`failure` is the frame's failure context: the failure currently being handled in
the frame. Its value is the failure Result itself — the envelope defined in
[The failure envelope](../call-interface/#the-failure-envelope), with no members
added or removed — or null when no failure is active. It is in scope frame-wide:
while set, every expression the frame evaluates can read it, not only those on
the handler path.

### Lifecycle

`failure` is null when the frame begins. It is set when a Step resolves to a
failure Result — the outermost Result the Step's machinery emits (see
[Where `catch` sits](../step-mechanics/#where-catch-sits)) — whether or not a
`catch` clause matches it. It is not set at the seams inside the Step's
machinery, where a failure is still in flight: a call's failure arm reads the
envelope as `call.result` instead (see [`call`](#call)). A `Gather` dispatch's
failure does not set it at all: that failure is the dispatch's Result, observed
by the `Gather` as data (see [Step actions](../step-actions/)), and the `Gather`
Step sets the failure context only when it resolves to a failure itself.

The context stays set across the handler path. The matched `catch` clause's
`output` and `assign`, and every Step from the clause's `next` onward, read the
same envelope (see [`catch` clauses](../step-mechanics/#catch-clauses)). It is
cleared by the first successful Step completion after it was set; a handler that
needs the failure beyond that point captures what it needs into `vars` first:

```json
"catch": [
  {
    "match": { "codes": ["*"] },
    "assign": {
      "failedStep": "{{ step.name }}",
      "failureCode": "{{ failure.code }}"
    },
    "next": "notify-failure"
  }
]
```

The clause's expressions evaluate at the failing Step, so `step` is that Step:
capturing `step.name` beside `failure.code` records where the failure arose
along with what it was.

`failure` is frame-scoped. A failure that propagates out of the frame sets
nothing in the parent directly; it arrives there as the calling Step's failure
Result, and the parent's own context is set by that Step's failure under the
same rules.

### Chaining

A failure's `previous` member records the failure it superseded (see
[The failure envelope](../call-interface/#the-failure-envelope)). The engine
links it whenever one failure supersedes another:

- **A failed recovery.** A new failure arises in the frame while `failure` is
  set: a handler Step failed before any success cleared the context. The new
  failure's `previous` is the failure that was being handled.
- **A constructed successor.** A middleware `onFailure` block writes envelope
  fields, constructing a new failure that supersedes the rising one (see
  [`onFailure`](../middleware-mechanics/#onfailure)), or a `Raise` constructs a
  new failure while one is active (see [Step actions](../step-actions/)). The
  superseded failure is linked as the new one's `previous`.
- **A failed cleanup.** An `onAlways` phase fails with a failure already in
  flight; the cleanup failure supersedes it, chaining what it displaced (see
  [`onAlways`](../middleware-mechanics/#onalways)).
- **An imposed cancellation.** A construct interrupting work it owns imposes a
  cancellation that chains its explanatory failure, and a cleanup failure during
  the unwind stacks on top of that chain (see
  [The unwind](../execution-model/#the-unwind)).

In every case supersession preserves the superseded failure intact, one link
down the chain; and only failures chain — a superseded success is displaced, not
recorded (see
[When a phase fails](../middleware-mechanics/#when-a-phase-fails)).

A failure-constructing site MAY write `previous` itself, overriding the engine's
link. The typical override severs rather than builds: `previous` set to null
emits the new failure with its history deliberately dropped (see
[The failure envelope](../call-interface/#the-failure-envelope)). And a bare
`Raise` — one that constructs no envelope — re-emits the active failure
unchanged: no new failure exists, so no link is added (see
[Step actions](../step-actions/)).

An implementation MAY truncate a `previous` chain at a platform-defined depth.
When it truncates, it MUST replace the deepest retained `previous` with a
failure Result of type `error` and code `System.FailureChainTruncated`, so a
reader can tell discarded history from history that never existed. The code is
listed in the [Failure code reference](../failure-code-reference/).

## Field defaults and passthrough

Every data-flow field has a defined value when absent, stated below as the
expression an author would write to restate it (see
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough)).
All but one are passthrough: the value of the binding path the expression names,
flowing through unchanged. The `Gather` `output` default is the one defined
computation — the success projection over `step.results`, defined with the
action (see [Step actions](../step-actions/)). Realizing any of them requires no
expression evaluator. The table below consolidates the defaults across the
specification; each field's owning section is the authoritative definition, and
the actions' complete field sets are given in [Step actions](../step-actions/).

| Construct             | Field             | Absent value                                                        |
| --------------------- | ----------------- | ------------------------------------------------------------------- |
| `Call` / `Match` Step | `input`           | `{{ step.input }}`                                                  |
| `Call` Step           | `output`          | `{{ step.result.value }}`                                           |
| `Gather` Step         | `output`          | `{{ step.results.filter(r, r.type == 'success').map(r, r.value) }}` |
| `Pass` Step           | `output`          | `{{ step.input }}`                                                  |
| `Return` Step         | `value`           | `{{ step.input }}`                                                  |
| `Match` clause        | `output`          | `{{ match.input }}`                                                 |
| `catch` clause        | `output`          | `{{ step.input }}`                                                  |
| `call` object         | `input`           | `{{ call.input }}`                                                  |
| `call` object         | `onSuccess.value` | `{{ call.result.value }}`                                           |

An action that defines no `output` field passes its value through without one: a
`Sleep` emits the value it received, unchanged. A middleware phase block's
shaping defaults are given with
[The phase block](../middleware-mechanics/#the-phase-block).
