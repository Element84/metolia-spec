---
title: "Steps and step mechanics"
weight: 70
---

A **Step** is a named entry in a Flow's `steps` map: the unit of execution
inside a frame. Each Step names an `action`, the verb it performs, and carries a
small set of shared mechanical fields. The actions themselves, and the fields
specific to each, are cataloged in [Step actions](../step-actions/); this
section defines the shared mechanics.

Those shared mechanics are the three roles a Step plays apart from its action: a
Step **routes**, it **shapes data** crossing its boundary on the way in and on
the way out, and it **captures values** into the frame's variables. Its fields
are confined to those roles: routing, shaping
[data-plane](../concepts/#the-data-plane) values, and writing
[control-plane](../concepts/#the-control-plane) state. The work itself, the Call
a Step dispatches and the middleware that wraps it, lives in the Call object
(see [The Call interface and Result](../call-interface/)) and the middleware
stack (see [Middleware mechanics](../middleware-mechanics/)), not in the Step's
own fields.

## What a Step is

A Step definition is a JSON object pairing an `action` with the shared fields
below and whatever fields that action defines. The shared fields are:

| Field        | Type               | Required                   | Default                                       | Expression                                             |
| ------------ | ------------------ | -------------------------- | --------------------------------------------- | ------------------------------------------------------ |
| `action`     | string             | required                   | —                                             | no (structural)                                        |
| `input`      | any                | optional (`Call`/`Match`)  | `{{ step.input }}`                            | yes                                                    |
| `output`     | any                | optional                   | per action ([Step actions](../step-actions/)) | yes                                                    |
| `assign`     | object             | optional                   | `{}`                                          | yes (per value)                                        |
| `middleware` | array              | optional (`Call`)          | `[]`                                          | no (structural)                                        |
| `next`       | string (Step name) | optional                   | —                                             | no (structural)                                        |
| `catch`      | array              | optional (`Call`/`Gather`) | `[]`                                          | no (structural; see [`catch` clauses](#catch-clauses)) |

Not every shared field applies to every action: `input` is carried only by
`Call` and `Match`; `catch` only by the call-dispatching actions, `Call` and
`Gather` (see [Failures and `catch`](#failures-and-catch)); `middleware` only by
`Call` — a stack wraps a single dispatch, and `Call` is the action that makes
exactly one, where a `Gather`'s dispatches carry their wrappers inside a flow
target (see [Step actions](../step-actions/)); and the applicability and
passthrough default of `output` and `assign` vary by action. Each action's
complete field set is given in [Step actions](../step-actions/).

`action` is a PascalCase discriminator naming the Step's action; its permitted
values and their semantics are defined in [Step actions](../step-actions/). A
Step may additionally carry a
[`comment`](../definition-format/#the-comment-field) for documentation.

`next` is optional because not every Step transitions; a terminal Step ends the
Flow instead (see [Routing](#routing-next-and-terminal-steps)). The remaining
fields — `input`, `output`, `assign`, and `catch` — are defined in the sections
that follow.

## Routing: `next` and terminal Steps

A Step is either **transitioning** or **terminal**. A transitioning Step names a
successor and, on success, passes control to it. A terminal Step ends the Flow,
producing the Flow's Result.

A transitioning Step names its successor in `next`. The value MUST be a key of
the same `steps` map; routing is scoped to that map, and a Step cannot
transition to a Step in a parent, child, or sibling Flow (see
[Step-name scoping](../flow-object/#step-name-scoping)).

A terminal Step carries no `next`. Two actions end a Flow: `Return` completes it
with a success Result, and `Raise` completes it with a failure Result. Both are
defined in [Step actions](../step-actions/); the Result they produce is defined
in [The Call interface and Result](../call-interface/).

Every Step MUST have a defined exit path. A transitioning Step satisfies this
with `next`; a terminal Step satisfies it by ending the Flow; and a Step's
`catch` clauses supply exit paths for its failure routing (see
[`catch` clauses](#catch-clauses)). A Step that can neither transition nor
terminate — a non-terminal action with no `next` — is ill-formed. This is a
structural property of the definition, checkable without running the Flow; it is
one of the constraints described under
[Static checks](../flow-object/#static-checks).

## Data flow: `input` and `output`

Two fields shape the data crossing a Step's boundary. `input` shapes the value
entering the Step; `output` shapes the value leaving it on success.

`input` produces the value the Step works on. Its default, `{{ step.input }}`,
passes through the value the Step received unchanged. On a `Call` Step, the
shaped input is what threads into the Call and its middleware stack on the way
down (see [Middleware mechanics](../middleware-mechanics/)); on a `Match` Step,
it is the value the clause predicates test (see
[Step actions](../step-actions/)).

`output` produces the value the Step emits on success — the value its successor
receives through `next`. What it reads when absent is the action's default: a
`Call`'s default, `{{ step.result.value }}`, reads the value the Step's Result
carries (see [Success Result](../call-interface/#success-result)); a `Gather`'s
projects its collected Results' successes; each action's default is given with
the action (see [Step actions](../step-actions/)). A terminal Step has no
successor to hand an output to: a `Return` shapes the value its success Result
carries with its own `value` field, and a `Raise` constructs a failure envelope
instead (see [Step actions](../step-actions/)).

What each field yields when it is absent, and the rule that a present expression
replaces it, are a property of the field defined under
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough).

The pairing of `output` (shape the value emitted) with `assign` (capture into
variables) is a uniform discipline: wherever a construct shapes what it emits —
an `output`, a `value`, a constructed failure envelope — an `assign` sits beside
it. It recurs at the Step, the middleware phase (see
[Middleware mechanics](../middleware-mechanics/)), and the call's arms (see
[The arms](../call-interface/#the-arms-onsuccess-and-onfailure)). The terminal
actions `Return` and `Raise` are the exception: they end the frame, leaving no
later Step to read a captured variable (see [Step actions](../step-actions/)).

## Variables: `assign`

`assign` is the Step's writer into the frame's variables. It is a map from name
to an expression; each expression's value is bound to that name in `vars`. The
variable model — what `vars` is, how it is scoped to the frame, how it is seeded
from `parameters`, and how a written name replaces an existing binding — is
defined in [The Flow object](../flow-object/#the-vars-model). This section
defines only `assign`'s timing relative to the Step.

```json
"assign": { "registeredCount": "{{ size(step.result.value.features) }}" }
```

Bindings declared in `assign` take effect on successful Step exit. `assign` runs
after `output`, so an assignment is not visible to the Step's `output`
expression. Within a single `assign` block, every expression is evaluated
against the variable state as it existed _before_ the block ran: assignments in
the same block are not visible to one another, and an expression that reads a
name being reassigned sees the prior value, not the new one. This follows from
the data model: an `assign` block is a JSON object, and object key order is not
significant ([The JSON data model](../data-model/#the-json-data-model)), so the
block defines no order in which one entry could see another's result. Evaluating
every entry against the prior state makes the outcome the same in whatever order
an implementation evaluates them. Subsequent Steps in the frame see the new
bindings.

To carry a value out of a completed inner Flow, the dispatching call's `assign`
captures `flow.vars.<name>` at the call boundary, where the window is in scope,
and later fields read the captured variable back from `vars`; nothing crosses
the boundary on its own (see
[The target windows](../call-interface/#the-target-windows-flow-and-provider)).

## Failures and `catch`

A Result-consuming Step's fields encode a two-armed match on its Result.
`output`, `assign`, and `next` are the success arm; `catch` is the failure arm;
exactly one arm applies, selected by the Result's type. Actions that consume no
Result have no arms, only sequencing. The same two-armed match appears on the
language's stack constructs — middleware entries and call objects — encoded
there as `onSuccess`/`onFailure` blocks (see
[The phase model](../middleware-mechanics/#the-phase-model) and
[The arms](../call-interface/#the-arms-onsuccess-and-onfailure)); a routing
construct encodes it as flat success fields beside a `catch`, whose distinct
name marks the difference in kind: a clause list that matches codes and routes
the graph, not a shaping block.

`catch` is a Step's failure-path routing: the failure analogue of `next`. Where
`next` routes on success, a `catch` clause routes when the Step produces a
failure Result whose code matches the clause. The failure Result itself — its
`type`, `code`, and the rest of the envelope `catch` matches against — is
defined in
[The Call interface and Result](../call-interface/#the-failure-envelope). This
section owns the matching grammar and the clause mechanics.

`catch` is a field of the call-dispatching actions, `Call` and `Gather` (see
[Step actions](../step-actions/)): the actions whose ordinary work is to run a
Call and route on the Result it yields, including its recoverable failures. Any
Step can fail regardless of its action, since any Step may evaluate an
expression and an evaluation that faults produces a `System.*` failure (see
[Evaluation errors](../expressions/#evaluation-errors)). A failure on a Step
that carries no `catch` clause has no local edge to match it, and so propagates
out of the frame.

### Where `catch` sits

A Step resolves to a single Result, and `catch` matches that Result. For a
`Call` Step the Result emerges from the bottom of the Step's machinery and
rises: the inner Call runs, the middleware stack processes it on the way out
(`onEntry` on the way down; `onSuccess` or `onFailure`, then `onAlways`, on the
way up), and the Result the outermost middleware emits is what `catch` sees.
`catch` sits _outside_ that stack; it does not participate in it. The ordering
and semantics of the stack are defined in
[Middleware mechanics](../middleware-mechanics/); here it is enough that `catch`
matches the Step's outermost emitted Result.

When that Result is a success, `next` and `output` apply and `catch` does not
run. When it is a failure, the Step's own `output` and `assign` do not run,
since they shape and capture a successful exit. Instead, `catch` is consulted
before the failure would propagate out of the frame. On the failure path the
only shaping that runs is the matched clause's own `output` and `assign` (see
[`catch` clauses](#catch-clauses)).

### Failure matching

A `catch` clause selects failures with a **failure matcher**: an object whose
members each constrain one contract field of the failure envelope (see
[The failure envelope](../call-interface/#the-failure-envelope)). The matcher is
the language's one failure-selection grammar: `catch` clauses match with it
here, and the `Retry` middleware's policies match with it against rising
failures (see
[The `Retry` middleware](../providers/middleware-providers/#the-retry-middleware)).

```json
"match": {
  "codes": ["Provider.Call.Payments.CardDeclined", "System.*"],
  "types": ["error", "timeout"],
  "retryable": true
}
```

| Member      | Type                          | Required | Matches when                                               |
| ----------- | ----------------------------- | -------- | ---------------------------------------------------------- |
| `codes`     | array of patterns (non-empty) | optional | the failure's `code` matches any pattern in the list       |
| `types`     | array of strings (non-empty)  | optional | the failure's `type` is any of the named non-success types |
| `retryable` | boolean                       | optional | the failure's `retryable` is the same explicit value       |

A matcher MUST have at least one member, and it matches a failure when every
member present matches. Each member constrains its field independently; the only
cross-field grammar is the conjunction. The match-any matcher is spelled
`{ "codes": ["*"] }`.

`codes` matches over the failure's `code` with a closed pattern grammar:

- `"Prefix.Code"` — exact match on a fully-qualified code.
- `"Prefix.*"` — match any code with the given prefix.
- `"*"` — match any failure code.

A pattern's prefix is one or more leading segments. The first segment is a code
namespace, defined with the failure envelope in
[The Call interface and Result](../call-interface/#code-namespaces); a deeper
prefix matches a finer grain of the taxonomy: `Provider.*` matches any provider
failure, `Provider.Call.*` any called target's failure, and
`Provider.Call.Http.*` a single provider's codes. Matching is lexical over the
code alone; no pattern is scoped by the failure's origin. On a Step whose Call
targets a provider directly, the only call-provider codes that can arrive are
that provider's own, per its catalog. On a Step whose Call targets a Flow, a
`Provider.*` failure arising anywhere inside the called Flow propagates up
unchanged and matches the same way; there the code's identity segment is what
tells one origin's codes from another's.

`types` matches over the failure's `type`. Its entries name non-success Result
types — the spec-defined types or extension types (see
[Result types](../call-interface/#result-types)). `success` is not a permissible
entry: a matcher selects among failures, and no success reaches one.

`retryable` matches over the envelope's advisory signal (see
[The failure envelope](../call-interface/#the-failure-envelope)): `true` matches
only a failure that asserts `retryable: true`, and `false` only one that asserts
`retryable: false`. A failure whose `retryable` is unset — absent or `null`,
which the envelope makes equivalent — matches neither value: the signal's
silence satisfies no assertion about it.

> [!NOTE]
> The matcher is deliberately closed over the envelope's contract fields.
> `message` and `details` carry unstructured, provider-specific context and are
> not matchable; a workflow that must distinguish failures by something in
> `details` first names the distinction as a code — a middleware `onFailure`
> block that constructs a successor failure with a new `code` (see
> [`onFailure`](../middleware-mechanics/#onfailure)) — and then matches the
> name.

Implementations MUST accept any syntactically valid `codes` pattern. Tooling MAY
use provider and middleware catalogs to warn about patterns that reference
undeclared codes, particularly on a `Call` Step targeting a provider, where the
set of possible codes is statically computable. On a Step whose Call targets a
Flow, the codes that can propagate from inside the called Flow are not
statically bounded, which limits such analysis.

### `catch` clauses

Each `catch` clause is a conditional failure edge: when the Step's failure
matches, control transitions to the clause's `next`, the way `next` transitions
on success.

```json
"catch": [
  {
    "match": {
      "codes": [
        "Provider.Call.Payments.CardDeclined",
        "Provider.Call.Payments.InsufficientFunds"
      ]
    },
    "next": "retry-with-backup"
  },
  { "match": { "codes": ["*"] }, "next": "failed" }
]
```

| Field     | Type                     | Required | Default            | Expression      |
| --------- | ------------------------ | -------- | ------------------ | --------------- |
| `match`   | object (failure matcher) | required | —                  | no (structural) |
| `output`  | any                      | optional | `{{ step.input }}` | yes             |
| `assign`  | object                   | optional | `{}`               | yes (per value) |
| `next`    | string (Step name)       | required | —                  | no (structural) |
| `comment` | string                   | optional | —                  | no (literal)    |

Clauses are evaluated in order; the first clause whose `match` matches the
failure wins (see [Failure matching](#failure-matching)). Validators MAY warn
about a clause made unreachable by an earlier, broader one.

When a clause matches, control transitions to `next`, which resolves against the
same `steps` map as the Step it guards (see
[Step-name scoping](../flow-object/#step-name-scoping)). `output` shapes the
value the handler Step receives, and `assign` captures into `vars`. Because the
Step failed, no success `value` is available, so `output` defaults to
`{{ step.input }}`: the value the Step received passes through (see
[Absent fields and passthrough](../expressions/#absent-fields-and-passthrough)).

The clause's `output` and `assign` expressions, and the Step `catch` routes to,
can read the matched failure envelope. That failure remains the live failure
context across the handler path; if a handler Step itself fails, the new
failure's `previous` chains the one being handled, recording a failed recovery.
The binding that exposes the failure and the lifecycle by which it is set,
cleared, and chained are defined in [Execution context](../execution-context/).

### `catch` and frame-level failures

`catch` handles Step-level failures only. A failure that arises at the frame
level — from the Flow's own middleware, or from an external cancellation —
bypasses the step graph and therefore its `catch` clauses. Flow-level middleware
wraps the step graph as a whole, outside any individual Step's routing (see
[Middleware mechanics](../middleware-mechanics/)); cancellation and frame
completion are defined in [Execution model](../execution-model/).
