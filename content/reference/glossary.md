---
title: "Glossary"
weight: 150
---

This glossary is informative. Each entry gives a working definition and points
to the section that owns the term; the owning section is authoritative where the
two differ. Terms are listed alphabetically.

<!-- dprint-ignore-start -->

Arm
: One of the two Result-consuming blocks on a `call` object, `onSuccess` and
  `onFailure`. Exactly one runs when the target's Result settles, selected by
  the Result's `type`: `onSuccess` carries `value` (shaping) and `assign`;
  `onFailure` carries `assign` alone.
  [The arms](../call-interface/#the-arms-onsuccess-and-onfailure).

`assign`
: A map from name to expression whose values are written into the frame's
  variables. It is available at every Result-consuming seam — a Step, a `Match`
  or `catch` clause, a middleware phase, a call's arms — except the terminal
  actions. Every expression in one `assign` block evaluates against the
  variable state from before the block ran.
  [Variables: `assign`](../step-mechanics/#variables-assign).

Binding root
: A bare top-level name through which an expression reads the running
  workflow's state: `vars`, `execution`, `frame`, `step`, `call`, `flow`,
  `provider`, `match`, `middleware`, and `failure`. Which roots are in scope
  depends on where the expression appears.
  [Evaluation context](../expressions/#evaluation-context-the-binding-roots),
  [Execution context](../execution-context/).

Call
: The dispatch unit: a request to run a target and obtain its Result. A `call`
  object names exactly one target (`provider` or `flow`), supplies it
  arguments (`with`) and a data payload (`input`), and consumes the settled
  Result through its arms. A `Call` Step dispatches one call; a `Gather`
  dispatches many concurrently.
  [The call object](../call-interface/#the-call-object).

Call provider
: A provider that serves as a Call target: an integration a `call` names in
  its `provider` field, which receives the call's `with` and `input` and
  produces exactly one Result. [Call providers](../providers/call-providers/).

Carried value
: The value a `Loop` entry threads from one iteration to the next: the
  product of its `onSuccess` `value`, fed to the re-entered scope as its input
  or, on the final iteration, emitted upward as the entry's success value.
  [The `Loop` middleware](../providers/middleware-providers/#the-loop-middleware).

`catch`
: A Step's failure-path routing: an ordered list of clauses, each selecting
  failures with a failure matcher and routing to a `next` Step, carried by
  the call-dispatching actions (`Call` and `Gather`). The first matching
  clause wins; an unmatched failure propagates out of the frame.
  [Failures and `catch`](../step-mechanics/#failures-and-catch).

Clock pin
: The single entry instant recorded for each construct execution (a Step pass,
  a middleware phase run, a Call execution). The `now()` function returns it,
  so `now()` is stable within one construct execution and advances across
  re-executions. [The clock pin](../execution-model/#the-clock-pin).

`completion`
: A `Gather`'s completion policy: the number of dispatch `successes` the
  fan-out must achieve, and whether the `Gather` `wait`s for in-flight
  dispatches once the outcome is determined.
  [`completion`: the completion policy](../step-actions/#completion-the-completion-policy).

Control plane
: The state that steers a workflow rather than flowing through it:
  parameters and the `vars` they seed, `assign` writes, metadata records, and
  the failure context. Configuration crosses boundaries explicitly, as `with`
  arguments. [The control plane](../concepts/#the-control-plane).

Data plane
: The values that flow through a workflow end to end: Step inputs and
  outputs, the `value` a success Result carries, and what expressions
  evaluate to. The data plane carries JSON values per the data model.
  [The data plane](../concepts/#the-data-plane),
  [The data model](../data-model/).

Dispatch
: One execution of a `call` object made by a `Gather`: per element of the
  `over` collection in the iterate form, per entry of `calls` in the scatter
  form. Every dispatch resolves to exactly one Result holding its position in
  `step.results`, whether it settled, was cancelled, or was skipped.
  [The dispatch model](../step-actions/#the-dispatch-model).

Establishment
: A middleware entry is established once its `onEntry` phase has completed.
  Established entries are guaranteed their `onAlways` phase on the way out,
  including when the scope is torn down by an interruption.
  [The stack](../middleware-mechanics/#the-stack-ordering-and-composition),
  [The unwind](../execution-model/#the-unwind).

Expression
: A JSON string value whose entire content is set off by a recognized
  delimiter pair (for CEL, `{{ }}`), evaluated at runtime against the
  execution context to produce a typed data-model value. Any other string is
  a literal. The delimiter pair identifies the expression language.
  [Expressions](../expressions/).

Failure context
: The frame's live failure: the `failure` binding, holding the failure
  envelope being handled, or null when none is active. It is set when a Step
  resolves to a failure Result and cleared by the first successful Step
  completion after that. [`failure`](../execution-context/#failure).

Failure envelope
: The structured shape every non-success Result carries: `type`, `code`,
  `message`, `details`, `retryable`, and `previous`. All non-success Result
  types share it regardless of origin, and `catch`, `Retry` policies, and
  `onFailure` phases all work over it.
  [The failure envelope](../call-interface/#the-failure-envelope).

Failure matcher
: The language's one failure-selection grammar: a structural object (`match`)
  whose members — `codes` patterns, `types`, `retryable` — each constrain one
  contract field of the failure envelope, with every member present required
  to match. `catch` clauses and `Retry` policies select failures with it.
  [Failure matching](../step-mechanics/#failure-matching).

Flow
: The unit of definition: an object pairing an `entrypoint` with a `steps`
  map, optionally with named subflows (`flows`), `parameters`, and
  `middleware`. The same object describes a whole workflow and a piece of
  one; a Flow runs inside a frame and completes with exactly one Result.
  [The Flow object](../flow-object/).

`flows`
: A Flow field mapping names to Flow objects: named subflows a `call` can
  target by name. References resolve lexically — against the containing
  Flow's map, then each enclosing Flow's outward — with the nearest
  declaration winning, and the reference graph must be acyclic.
  [`flows`](../flow-object/#flows),
  [Flow-name scoping](../flow-object/#flow-name-scoping).

Frame
: The execution-time instantiation of a Flow: its own variables, input,
  metadata, and lifecycle. A frame evaluates serially — one Step at a time —
  and all concurrency is between the target executions its Steps have
  outstanding. Frames are isolated; data crosses the boundary only through a
  call's `input` and `with` on the way in and its Result on the way out.
  [Frames and sequential execution](../execution-model/#frames-and-sequential-execution).

`input`
: A field shaping the data a construct works on: a Step's `input` produces
  the value its action consumes, and a `call`'s `input` produces the payload
  delivered to the target. Absent, the field is a passthrough.
  [Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output),
  [The three axes](../call-interface/#the-three-axes-parameters-with-and-input).

Iterate form
: The `Gather` form pairing `over` (an expression producing an array) with
  `call` (a call template): one dispatch per element, the element arriving as
  `call.input` and its position as `call.index`.
  [The iterate form](../step-actions/#the-iterate-form-over-and-call).

Middleware entry
: One middleware in a stack: a middleware-provider URI plus up to four phase
  blocks (`onEntry`, `onSuccess`, `onFailure`, `onAlways`) configuring the
  author's shaping and the middleware's action at each phase.
  [The middleware entry](../middleware-mechanics/#the-middleware-entry).

Middleware provider
: A provider that plugs into the phase model: the integration a middleware
  entry names, whose catalog contract declares its per-phase actions,
  parameter schemas, attachment levels, contributed metadata, and failure
  codes. [Middleware providers](../providers/middleware-providers/).

`output`
: A field shaping the value a construct emits on success: a Step's emitted
  value, a clause's, or a middleware `onEntry`'s product passed inward.
  Absent, each site's defined default applies — passthrough everywhere except
  the `Gather` success projection.
  [Data flow: `input` and `output`](../step-mechanics/#data-flow-input-and-output),
  [Field defaults](../execution-context/#field-defaults-and-passthrough).

`parameters`
: A JSON Schema (2020-12) declaring the arguments a target accepts. A Flow
  declares its own; a provider declares its through the catalog. A call's
  `with` is validated against it, and a Flow's validated arguments and
  schema defaults seed the frame's `vars` at entry.
  [`parameters`](../flow-object/#parameters).

Phase
: A crossing of a middleware entry's boundary: `onEntry` on the way down;
  `onSuccess` or `onFailure`, then `onAlways`, on the way up. A phase block
  is the author's configuration of one phase: `when` gating the action,
  `with` configuring it, a shaping key, and `assign`.
  [The phase model](../middleware-mechanics/#the-phase-model).

Provider
: A named integration the language dispatches to, identified by URI and
  resolved against the platform's catalog: the extension seam through which
  platform capability enters a workflow. Two kinds exist, call providers and
  middleware providers. [Providers](../providers/).

Result
: The discriminated value recording one outcome: a success carrying a
  `value`, or a non-success carrying the failure envelope. Every Call
  produces exactly one Result, and so does every frame — which is what lets
  a Flow be called like a provider.
  [The Result](../call-interface/#the-result).

Result type
: The `type` field of a Result. Five spec-defined lowercase values:
  `success`, plus the non-success `error`, `cancellation`, `timeout`, and
  `skipped`. Extension types are PascalCase and share the envelope and the
  handling machinery. [Result types](../call-interface/#result-types).

Scatter form
: The `Gather` form carrying `calls`, a literal non-empty array of call
  objects: one dispatch per entry, each independently targeted and
  configured, with the Step's received value arriving as every dispatch's
  `call.input`. [The scatter form](../step-actions/#the-scatter-form-calls).

Step
: A named entry in a Flow's `steps` map: the unit of execution inside a
  frame. A Step names an `action` and, around it, routes (`next`, `catch`),
  shapes data (`input`, `output`), and captures variables (`assign`).
  [Steps and step mechanics](../step-mechanics/).

Step action
: The verb a Step performs, named by its `action` discriminator. Seven are
  defined: `Call`, `Gather`, `Match`, `Pass`, `Sleep`, `Return`, and
  `Raise`. [Step actions](../step-actions/).

Step graph
: The connected set of Steps within a single `steps` map, executed from
  `entrypoint` toward terminal completion. The Step graph is the operation a
  Flow-level middleware stack wraps.
  [Where middleware attaches](../middleware-mechanics/#where-middleware-attaches).

Structural field
: A field whose value is part of the definition and never an expression:
  `action`, `next`, `provider`, a `flow` name, a `catch` clause's `match`,
  and the other discriminators and static identifiers. An implementation
  rejects an embedded expression in such a field.
  [Where expressions may appear](../expressions/#where-expressions-may-appear).

Target
: What a call runs: a provider or a Flow. A **target execution** is one run
  of a target — a frame, or a provider execution — and every target
  execution completes exactly once, with exactly one Result.
  [The completion contract](../execution-model/#the-completion-contract).

Target window
: The binding exposing a settled target to its call's arms, named for the
  target field the call wrote: `flow` is the completed frame (its `result`,
  `vars`, `input`, and `metadata`); `provider` is the provider's completed
  execution (its `input`, `result`, and declared `metadata`). `call.result`
  denotes the window's `result` for either target kind.
  [The target windows](../call-interface/#the-target-windows-flow-and-provider).

Unwind
: The teardown of an interrupted scope: execution stops, and the engine
  ascends out of the scope running established middleware entries' `onAlways`
  phases — and nothing else — innermost outward.
  [The unwind](../execution-model/#the-unwind).

`vars`
: The frame's variable namespace: a flat object seeded from the Flow's
  `parameters` at frame entry and written by `assign` during execution. Each
  frame has its own; a subflow never shares its caller's.
  [The `vars` model](../flow-object/#the-vars-model).

`with`
: The arguments a call or a middleware phase supplies, validated against the
  target's or the middleware's declared `parameters` schema. Distinct from
  `input`: `with` configures the target, `input` carries the data being
  processed.
  [The three axes](../call-interface/#the-three-axes-parameters-with-and-input).

Workflow definition
: The top-level JSON document describing a workflow: a root Flow object
  carrying `$schema`. [The definition format](../definition-format/).

<!-- dprint-ignore-end -->
