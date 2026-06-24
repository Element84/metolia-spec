---
title: "The Call interface and Result"
weight: 50
---

This section defines two concepts that the rest of the specification builds on:
the **`call` object** — the dispatch unit that names a target, supplies it data,
and yields a result — and the **Result** — the discriminated value that every
Call (and, per the completion contract, every frame) produces. They are defined
together because a Call's purpose is to produce a Result, and because most later
sections refer back to one or both.

## The call object

A `call` is a request to run a **target** and obtain its Result. A single shape
serves both kinds of target: a **provider** (an external service or platform
capability, addressed by URI) and a **Flow** (a named or inline workflow). The
Call supplies the target with arguments and an input payload, and consumes the
target's settled Result through a pair of arms (see
[The arms](#the-arms-onsuccess-and-onfailure)).

| Field       | Type                  | Required                 | Default            | Expression                                        | Description                                                                                                                                       |
| ----------- | --------------------- | ------------------------ | ------------------ | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `provider`  | string (provider URI) | one of `provider`/`flow` | —                  | no (structural)                                   | A call-provider URI, e.g. `mwl:provider.call/example/http/v1`. URI namespacing is defined in [Providers](../providers/).                          |
| `flow`      | string \| Flow object | one of `provider`/`flow` | —                  | no (structural)                                   | A named Flow declared in an enclosing `flows` map (see [Flow-name scoping](../flow-object/#flow-name-scoping)), or an inline Flow object.         |
| `input`     | any                   | optional                 | `{{ call.input }}` | yes                                               | A data payload threaded into the target. A separate channel from `with` (see [The three axes](#the-three-axes-parameters-with-and-input)).        |
| `with`      | object                | optional                 | `{}`               | yes (per field, or whole-value)                   | Arguments supplied to the target, validated against the target's `parameters` schema.                                                             |
| `onSuccess` | object (arm)          | optional                 | —                  | see [The arms](#the-arms-onsuccess-and-onfailure) | The success arm: shapes the value placed in the Call's success Result and captures variables (see [`onSuccess`](#onsuccess-shaping-and-capture)). |
| `onFailure` | object (arm)          | optional                 | —                  | see [The arms](#the-arms-onsuccess-and-onfailure) | The failure arm: captures variables from the failed dispatch's live context (see [`onFailure`](#onfailure-capture-only)).                         |
| `comment`   | string                | optional                 | —                  | no (literal)                                      | Human-readable documentation. See [`comment`](../definition-format/#the-comment-field).                                                           |

A Call MUST name exactly one target: either `provider` or `flow`, never both and
never neither. This is a structural constraint; the validation behavior that
enforces it is defined in [The Flow object](../flow-object/).

### The three axes: `parameters`, `with`, and `input`

A Call interacts with its target along three distinct axes. They are never
interchangeable, and keeping them separate is what lets one Call shape serve
every target uniformly.

- **`parameters`** is a parameter _schema_. It is declared by the **target**,
  not the call site: a Flow declares its `parameters` in its own definition (see
  [The Flow object](../flow-object/)), and a provider declares its `parameters`
  through the specification or the platform (see [Providers](../providers/)).
  Both kinds of target declare a schema for the arguments they accept; the
  relationship is symmetric.

- **`with`** is the set of _arguments_ supplied by the Call. Its shape is
  validated against the target's `parameters` schema — `with` is to `parameters`
  as arguments are to a signature. Its fields each accept an expression; or one
  whole-value expression may produce the entire arguments object, as `over` and
  `completion.successes` produce a structured value (see
  [Where expressions may appear](../expressions/#where-expressions-may-appear)).
  Either way the produced object is what the schema validates, at dispatch.

- **`input`** is a _separate data channel_, analogous to standard input. It
  carries a payload the target may or may not consume, orthogonal to `with`. A
  target reads its configuration from `with` and its working data from `input`.
  The field shares its name with a member of the `call` binding: `call.input` is
  the inbound payload arriving at the call boundary, and the `input` field
  shapes what the target receives from it, which is why the field's default,
  `{{ call.input }}`, is a passthrough.

### The arms: `onSuccess` and `onFailure`

A Call consumes its target's settled Result through two **arms**. When the
Result settles, exactly one arm runs, selected by the Result's `type`
([Result types](#result-types)): `onSuccess` on a success, `onFailure` on any
non-success type. Each arm is an object whose members are the author's shaping
and capture expressions:

| Field    | Arm         | Type   | Required | Default                   | Expression      |
| -------- | ----------- | ------ | -------- | ------------------------- | --------------- |
| `value`  | `onSuccess` | any    | optional | `{{ call.result.value }}` | yes             |
| `assign` | both        | object | optional | `{}`                      | yes (per value) |

```json
{
  "provider": "mwl:provider.call/example/http/v1",
  "with": { "method": "POST", "path": "/granules" },
  "onSuccess": {
    "value": "{{ call.result.value.body }}",
    "assign": { "requestId": "{{ provider.metadata.requestId }}" }
  },
  "onFailure": {
    "assign": { "failedRequestId": "{{ provider.metadata.requestId }}" }
  }
}
```

The arms mirror the same-named middleware phase blocks (see
[The phase model](../middleware-mechanics/#the-phase-model)), and the kinship is
deliberate: a middleware entry and a call object are both dispatch units,
selected the same way by the Result rising at their position, and each arm's
members are a strict subset of its namesake phase block's. The subset is
principled: an arm carries no `when` and no `with`, because those keys serve a
middleware's action, and the call is itself the action.

Both arms carry `assign`, an instance of the general rule: `assign` is available
in every Result-consuming arm or tail — a Step's tail, a middleware phase, a
call's arms — except the terminal actions `Return` and `Raise`, which produce a
Result no later expression reads (see [Step actions](../step-actions/)).

#### `onSuccess`: shaping and capture

A Call yields a Result. On success, the Result carries a `value`
([Success Result](#success-result)), and the arm's `value` member is the
expression that produces that carried value from whatever the target returned.
Its default, `{{ call.result.value }}`, passes the target's produced value
through unchanged; supplying an expression reshapes it at the call boundary. An
absent `onSuccess` is equivalent to `{ "value": "{{ call.result.value }}" }` —
the same passthrough contract every absent field follows (see
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough)).

The success Result's field is also named `value`: the arm's `value` member is
the expression that writes it. Naming both `value` keeps a single, consistent
answer to "what did this Call yield" — the Call shapes a `value`, and the Result
carries that `value`.

`assign` is `value`'s capture companion: a map from name to expression, written
into the frame's variables after `value`, under the discipline of
[Variables: `assign`](../step-mechanics/#variables-assign). It is where
call-boundary data is carried forward — the target's raw Result, the Call's
timing record, a completed inner Flow's variables — because the `call` binding
is in scope in the `call` object's fields and nowhere else, and the target
windows narrower still: only in the arms (see
[The target windows](#the-target-windows-flow-and-provider) and
[Execution context](../execution-context/)).

#### `onFailure`: capture only

The failure arm carries `assign` alone. It does not reshape the failure: the
envelope ascends exactly as the target produced it, and transforming a failure
belongs to the seams built for that — a middleware `onFailure` phase (in-stack,
composable, reusable; see [`onFailure`](../middleware-mechanics/#onfailure)) or
a `Raise` conditioned by the `catch` clause that routed to it (see
[Step actions](../step-actions/)).

What the arm provides is capture at the only seam where the failed dispatch's
context is alive: the `provider.metadata` of a provider whose envelope the
author does not control, the timing in `call.metadata`, the `flow` window of a
failed frame. A Step's `catch` can never reach that context — it sits outside
the middleware stack, and under a retrying middleware there is no single attempt
its expressions could mean — so what the failure arm does not capture into
variables is gone with the call execution.

`call.result` is in scope in both arms, and it is how the failure arm reads the
envelope: `call.result.code`, `call.result.message`, and the rest (see
[The failure envelope](#the-failure-envelope)). The frame's `failure` context is
not set in the arm: it sets only when the Step resolves to a failure Result (see
[Execution context](../execution-context/)).

#### When the arms run

An arm belongs to the call execution's own completion, and when it runs follows
the consuming action. On a `Call` Step, an arm runs when the target's Result
settles — before the Result, or the failure envelope, ascends the middleware
stack around the dispatch — and under a retrying middleware, each attempt is its
own call execution, its arm running per attempt. On a `Gather` dispatch, arm
evaluation is deferred: every arm runs at fan-out completion, in dispatch order,
exactly once per dispatch (see
[The arms at fan-out completion](../step-actions/#the-arms-at-fan-out-completion)).
In either position the arm consumes the settled Result with the dispatch's
context alive: `call.result`, the call's completed metadata record, and the
target window are in scope when the arm runs, however long the fan-out ran on
(see [The target windows](#the-target-windows-flow-and-provider)). A dispatch
that is skipped (see [Result types](#result-types)) evaluates nothing: no field,
no arm.

The line between an arm running and not running is settlement, not outcome. A
cancellation that arrives as the target's completed Result — a subflow cancelled
from outside, completing with `System.Cancelled` — is settled data: the failure
arm runs on it, exactly as a `catch` clause can match it. A call execution that
is interrupted — its enclosing scope unwinding, or a `Gather` cancelling its
in-flight dispatches — never settles at this seam: no arm runs, and the unwind
rules of [Execution model](../execution-model/) govern what happens instead.

### Faults in the call object's fields

A fault in any of the `call` object's fields fails the Call execution itself —
an expression that fails to evaluate (see
[Evaluation errors](../expressions/#evaluation-errors)), a `with` that fails
validation (see
[`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed))
— and the resulting failure is the Call's Result, taking the same path a failed
dispatch takes. On a `Call` Step it is emitted from the Call's position, the
innermost point of the Step's machinery, and ascends the middleware stack, where
outer entries see it in their `onFailure` phases and the Step's `catch` matches
what emerges (see [Where `catch` sits](../step-mechanics/#where-catch-sits)); on
a `Gather` dispatch it is that dispatch's Result, observed by the `Gather` (see
[Step actions](../step-actions/)). The arms are no exception: a `value` or
`assign` that faults fails the Call even where the target itself succeeded. A
fault in the failure arm composes with the failure already in hand: the fault's
failure supersedes it, and the engine chains the superseded failure as its
`previous` (see [Chaining](../execution-context/#chaining)) — the Call's Result
is the fault's failure, the target's own one link beneath it.

### Unified targets

A provider Call and a Flow Call share this one shape, and they share the Result
contract ([The Result](#the-result)): either kind of target yields the same kind
of Result. The two are therefore interchangeable wherever a Call is consumed —
the consumer does not branch on target kind.

Two consumers use the `call` object identically: the `Call` action and the
`Gather` action (both defined in [Step actions](../step-actions/)). A `Gather`
dispatches a `call` for every element of a collection, or one per entry of its
`calls` array, exactly as a `Call` Step dispatches a single `call`.

A Flow target may be named or inline. A named target references an entry
declared in the `flows` map of the Flow containing the call or of any enclosing
Flow (see [Flow-name scoping](../flow-object/#flow-name-scoping)):

```json
"call": {
  "flow": "ProcessGranule",
  "with": { "collection": "modis-l1" }
}
```

An inline target embeds the Flow object directly, and omits `with` when the Flow
declares no parameters:

```json
"call": {
  "flow": {
    "entrypoint": "build-summary",
    "steps": { "build-summary": { "...": "..." }, "done": { "action": "Return" } }
  },
  "input": "{{ call.input }}"
}
```

The `input` line restates the field's default, `{{ call.input }}`: written or
omitted, the same inbound payload reaches the target. Any other value, literal
or computed, would replace it. The line appears here to show the field in place;
examples elsewhere in this specification omit a field whose value would restate
its default.

## The Result

A Result is the value a Call produces. It is a discriminated union on its `type`
field: `type` names the outcome and determines which other fields the Result
carries. Every frame likewise completes by producing exactly one Result — the
completion contract that ties frames to Results is defined in
[Execution model](../execution-model/).

### Result types

The specification defines five Result types:

- `success` — the target completed normally, carrying a produced `value`.
- `error` — a recoverable failure from a provider, a middleware, or Flow logic.
- `cancellation` — the work was stopped by an external directive.
- `timeout` — a duration bound was exceeded.
- `skipped` — the target was never started (e.g. a `Gather` dispatch still
  waiting behind a concurrency limit when the `Gather` completed early).

The first is the _success_ type; the remaining four are _non-success_ types. All
four non-success types share one envelope shape
([The failure envelope](#the-failure-envelope)) and are handled uniformly by the
matching and propagation machinery.

> [!IMPORTANT]
> A note on terminology
>
> This specification also calls a non-success Result a **failure Result**, and
> refers to the non-success outcome generally as a _failure_. _Non-success_ is
> the more precise term: not every non-success outcome is a failure in the
> ordinary sense — a `cancellation` reflects an external directive, and a
> `skipped` Result records work that was never attempted, neither of which is a
> fault. But referring to "a non-success Result" at every turn is cumbersome,
> and "failure" is short, familiar, and reads naturally. The specification
> therefore uses _failure_ as the concise term for the whole non-success space,
> with no implied claim that every such outcome is an error. Where the
> distinction matters — for example, that `cancellation` is not produced the way
> an `error` is — the specification says so explicitly.

The set of non-success types is open for extension. A platform MAY define
additional non-success types for domain-specific outcomes, and a user MAY
construct one directly — the `Raise` action (see
[Step actions](../step-actions/)) produces a non-success Result whose `type` and
other fields are whatever the author specifies. Any such type shares the same
envelope and the same machinery.

The five spec-defined types are lowercase, and the lowercase space is reserved
to this specification: an extension type SHOULD be PascalCase
(`ProcessingError`). The convention keeps the two provenances distinguishable at
sight, and it leaves a future version of this specification free to define a new
lowercase type without colliding with deployed extensions — the same
producer-side discipline as the `System.` code namespace (see
[Code namespaces](#code-namespaces)).

Terminal states (`Succeeded`, `Failed`, `Cancelled`, and the like) are a
platform concern, not part of the language. A platform MAY expose terminal
states for operational purposes — alerting, dashboards — by mapping Result types
onto its own states.

### Success Result

A success Result has the shape:

```json
{ "type": "success", "value": <data> }
```

The `value` is any JSON value. It is what the Call's success arm produced
([`onSuccess`](#onsuccess-shaping-and-capture)) and what an expression reads to
consume a successful Call's data.

### The failure envelope

A Result whose `type` is one of the four non-success values carries structured
failure information:

```json
{
  "type": "<non-success-type>",
  "code": "<dotted-code>",
  "message": "<string>",
  "details": {/* arbitrary */},
  "retryable": true,
  "previous": {/* nested non-success Result, or null */}
}
```

| Field       | Type                           | Required | Description                                                                                 |
| ----------- | ------------------------------ | -------- | ------------------------------------------------------------------------------------------- |
| `type`      | string                         | yes      | One of the non-success Result types (see [Result types](#result-types)).                    |
| `code`      | string (dotted)                | yes      | The specific failure identifier, a dotted string (see [Code namespaces](#code-namespaces)). |
| `message`   | string                         | no       | A human-readable description.                                                               |
| `details`   | any                            | no       | Arbitrary structured context.                                                               |
| `retryable` | boolean &#124; null            | no       | An advisory retry signal.                                                                   |
| `previous`  | non-success Result &#124; null | no       | A chained prior failure that this one supersedes.                                           |

`type` and `code` together identify a failure. `type` places it in one of the
non-success categories; `code` is the dotted, specific identifier — the value
expressions and failure-handling constructs key on.

`retryable` is an advisory, three-valued signal: `true` asserts the failed work
could meaningfully be retried, `false` asserts it could not, and `null` makes no
assertion. An absent `retryable` and an explicit `null` are equivalent; a
consumer MUST treat the two alike. A failure matcher can select on the signal —
a `catch` clause or `Retry` policy matching `retryable: true` is the designed
consumption (see [Failure matching](../step-mechanics/#failure-matching));
whether and how any other consumer acts on it is its own to define.

#### Code namespaces

A `code` is a dotted string. It is matched by a Step's `catch` (see
[Steps and step mechanics](../step-mechanics/#failure-matching)) and by `Retry`
middleware, both of which treat it as opaque and match over it lexically.
Nothing constrains the value mechanically; a `Raise` produces a failure with
whatever `code` the author writes (see [Step actions](../step-actions/)).

The first segment is a **namespace** that, by convention, signals where the code
came from. The convention is not a partition the runtime enforces; it is a
shared discipline that lets a reader, and a failure matcher's `codes` prefix
pattern, tell origins apart:

- `System.` is reserved for the engine. A conformant engine emits, under
  `System.`, only the codes this specification enumerates, and an author SHOULD
  NOT mint a `System.` code, since doing so misrepresents an authored failure as
  an engine one.
- `Provider.` belongs to providers, and its codes follow the provider taxonomy
  (see [Providers](../providers/)): the second segment names the provider kind,
  `Call` or `Middleware`, and the third names the specific provider, by the
  `codePrefix` its catalog declares (`Provider.Call.Http.ConnectionFailed`,
  `Provider.Middleware.Retry.Exhausted`). Each level is a meaningful prefix to
  match on: any provider failure, either kind as a whole, or one provider's
  codes.
- Any other code is author space.

The complete list of codes this specification defines is in the
[Failure code reference](../failure-code-reference/).

`previous` chains an earlier failure that this one supersedes. The engine
ordinarily populates it; a failure-constructing site, such as a `Raise` (see
[Step actions](../step-actions/)) or a middleware `onFailure` phase (see
[Middleware mechanics](../middleware-mechanics/)), MAY set it explicitly,
overriding the engine's handling. Authors typically override `previous` not for
forming a failure chain but severing one: setting `previous` to `null`
deliberately drops the failure history where carrying it onward is unwanted.
When and why the engine forms a chain, and the binding that exposes a failure to
expressions while it is handled, are defined in
[Execution context](../execution-context/).

This shape is the **failure envelope**. Every non-success Result carries it,
regardless of `type`.

#### An example failure

When an expression cannot be evaluated, the engine produces a non-success Result
of type `error` with the engine-defined code `System.ExpressionEvaluationError`:

```json
{
  "type": "error",
  "code": "System.ExpressionEvaluationError",
  "message": "no such key: features"
}
```

`type` and `code` are drawn from defined catalogs: the engine codes, such as
this one, are listed in the [Failure code reference](../failure-code-reference/)
beside the spec-defined provider codes, and every provider's codes, call and
middleware alike, are documented with their source catalogs in
[Providers](../providers/).

## Flow-Call Result parity

A Flow that completes with a `Return` produces a **success** Result whose
`value` is the returned data — the same shape a provider Call's success Result
has. A caller therefore consumes a Flow target's Result exactly as it consumes a
provider target's: through the same `type` discrimination and the same `value`.
This parity is the payoff of unifying the two targets behind one Call shape.

### The target windows: `flow` and `provider`

A Call's arms consume the settled target through a **window**: a binding that
bears the name of the target field the call wrote. A flow target exposes `flow`,
the completed frame the call ran; a provider target exposes `provider`, the
provider's completed execution. The window is in scope in both arms, uniformly —
`onFailure` reads the same binding `onSuccess` does — and each call execution's
arms see that execution's own window: nothing of the window outlives the call
object.

- `flow` is the completed frame: its `result` and `vars`, with the frame's
  `input` and `metadata` beside them. The frame shape is defined in
  [Execution context](../execution-context/#flow).
- `provider` carries the provider's `input` (the value the provider received),
  its `result`, and the `metadata` its catalog declares (see
  [Execution context](../execution-context/#provider) and
  [Providers](../providers/)).

`call.result` is defined by reference to the window: it denotes `flow.result` or
`provider.result`, per the target the call names. That one name is what lets a
single default, `{{ call.result.value }}`, serve both target kinds.

Nothing crosses outward on its own. Promotion is explicit and immediate: an
arm's `assign` captures what later fields need — `flow.vars.<name>` to take an
inner Flow's variable, `flow.metadata` or `provider.metadata` to keep a record —
into the frame's variables, where the rest of the Step reads it back. A failed
target communicates through its failure Result plus what the failure arm
deliberately captures at this boundary; past the call object, the Result is the
only channel — an inner Flow that must surface state beyond what its caller
captures places it in the failure envelope it raises
([The failure envelope](#the-failure-envelope)).
