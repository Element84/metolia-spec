---
title: "Design principles"
weight: 10
---

MWL is small. The language vocabulary is deliberately narrow — a handful of Step
actions, one dispatch shape, one extension seam through providers — and that
narrowness is the point. Most of what authors might want to do is composed from
those primitives rather than expressed in dedicated language constructs. Every
choice in MWL's design serves this constraint: separate orchestration from
computation, keep the data plane free of control concerns, give failures one
routing path, give common operations one composition mechanism.

This section documents the design principles that produce that shape. It is not
normative — the formal specification is the [Reference](/reference/) — but it
explains _why_ the language is the way it is, which is useful context for anyone
implementing MWL, extending it through providers, or evaluating it for adoption.

## JSON as a compilation target

MWL's canonical form is JSON. This is a deliberate choice about _layer_, not
about authoring ergonomics: the JSON form is an intermediate representation, not
a final authoring surface. Anything that compiles to MWL is a valid way to write
workflows — language-native SDKs, YAML preprocessors, visual editors, LLM-driven
authoring tools, schema-driven generators. The JSON form is what the runtime
executes and what tooling inspects; the authoring form can be whatever fits the
audience.

This makes the language _less_ opinionated about authoring, not more. A
code-based developer experience is a layer above MWL, not a replacement for it.
A workflow author writing Python through an SDK and a workflow author writing
JSON directly produce the same artifact and get the same execution semantics.
Several properties follow:

- **Language-agnostic SDKs.** Any host language can produce MWL, and SDKs in
  different languages can interoperate by producing the same canonical form.
- **Static inspectability.** A workflow definition is data; it can be validated,
  diffed, code-reviewed, version-controlled, and analyzed without execution.
- **Trivial schema validation.** Submission-time validation is a JSON Schema
  check against the published
  [meta-schema](/reference/definition-format/#schema-documents), not a runtime
  concern.
- **Amenability to non-human authoring.** Visual editors and LLM generators emit
  JSON natively.

The JSON IR has a paradigm cost worth being explicit about: it is what the
platform shows back to authors. Execution histories, validation errors, the
event log, debugging surfaces — all speak in MWL regardless of how authors wrote
the workflow. An author writing through an SDK is still going to see MWL in
operational tooling, and the mental model of "what my code becomes" is an extra
layer they have to carry. The trade is real; it is accepted because the
alternative — picking a single host language as canonical — gives up
multi-surface authoring and adds opacity to system-level tooling.

## Messages, not files

The previous section was about the _form_ of the language; this one is about the
_contract_ between Steps. The two are related — the JSON IR gives the message
contract somewhere to live — but the message contract is a stance about data
modeling, not about authoring surfaces.

MWL passes structured JSON messages between Steps. The language has no opinion
on the shape of the message — it does not require any particular schema — but
the fact that the message is a structured object rather than an opaque file path
matters.

In a file-based orchestrator, authors think about files and references to them.
In MWL, authors think about messages, with files (when present) appearing as
fields within messages rather than as the message itself. This is a stronger
contract than it looks: even file-based orchestrators are passing references to
files rather than the files themselves — they just call those references "paths"
instead of "messages." Treating metadata as the message makes the contract
explicit, and gives workflows direct access to the metadata describing whatever
artifacts the workflow produces. Steps can act on metadata without fetching the
underlying artifacts when the operation does not require them.

The language is neutral about message content: a workflow can pass STAC items,
custom JSON objects, file references, control values, or any other
JSON-representable shape. The point is that the _shape_ is structured, not that
the _schema_ is fixed.

## Orchestration is separate from computation

The MWL runtime orchestrates; it does not compute. Processing happens in
external services accessed through the provider mechanism. This separation is a
design virtue independent of any specific runtime:

- The runtime's responsibilities are narrow — sequencing, data flow, failure
  routing, middleware composition — and stay narrow as the platform grows.
- Compute infrastructure can be chosen and scaled independently from
  orchestration infrastructure. Heavy or specialized compute (GPUs, large
  memory, custom dependency stacks) is the provider's concern, not the
  runtime's.
- The runtime's implementation surface is small enough to reason about, test,
  and operate without committing the team to a general-purpose compute platform.

The boundary has a single shape: the
[`call` object](/reference/call-interface/). A call is the only construct that
reaches outside the workflow, and only two actions dispatch one — `Call`, which
dispatches exactly one, and `Gather`, which dispatches many concurrently. Every
other Step action (`Match`, `Pass`, `Sleep`, `Return`, `Raise`) is internal
control flow: routing, data transformation, waiting, signaling completion or
failure. One construct marking the line where side effects can occur simplifies
reasoning, validation, and operational tooling alike.

## One Call interface

A call names a target, supplies it arguments and a data payload, and yields a
Result. MWL gives that interaction exactly one shape and lets two kinds of
target plug into it: a **provider** (an external integration, addressed by URI)
and a **Flow** (a named or inline subflow). Both kinds declare a `parameters`
schema for the arguments they accept, both receive `with` and `input` along the
same
[three axes](/reference/call-interface/#the-three-axes-parameters-with-and-input),
and both yield the same kind of Result
([Flow-Call Result parity](/reference/call-interface/#flow-call-result-parity)).
The consumer never branches on target kind.

The unification pays for itself several times over:

- **Subflows came nearly for free.** Because a completed Flow yields an ordinary
  Result, letting a call target a Flow required no new invocation machinery, no
  special sub-workflow construct, and no second failure path. The
  [`flows` map](/reference/flow-object/#flows) exists because the Call interface
  made it cheap, not because reuse was a goal (see
  [Non-goals](/rationale/non-goals/#reuse-as-a-language-goal)).
- **One fan-out primitive serves every target.** A `Gather` dispatches `call`
  objects, so the same fan-out runs providers, named Flows, inline Flows, or a
  mix of them, and a call template moves between a `Call` Step and a `Gather`
  unchanged.
- **Substituting a target is a local edit.** Swapping a provider for a subflow
  that wraps it (adding retry inside, say, or composing several providers)
  changes the call's target field and nothing else. Refactoring a workflow's
  structure does not ripple through its consumers.
- **Platforms inherit a stable seam.** Anything that can produce a Result can
  stand behind a call — a mock during development, a provider in production, a
  subflow when logic grows — and tooling that understands the call shape
  understands every dispatch in every workflow.

## Control plane and data plane are separated

A recurring problem in workflow languages designed without an explicit variable
mechanism is that control state has to multiplex through the data plane. Authors
thread control values through the data payload because there is no other path;
payloads get contorted to keep control concerns alive across Step boundaries;
integrations require parameter values to be delivered through the data plane
alongside the data being processed.

MWL is designed from the outset with the planes separated:

- **Variables (`vars`)** carry named values across Steps without contaminating
  the data flowing between them. Variable scope follows the frame structure
  ([The `vars` model](/reference/flow-object/#the-vars-model)), and `assign` is
  available at every Result-consuming seam — Steps, clauses, middleware phases,
  a call's arms.
- **Dedicated argument channels on calls.** `with` configures the target;
  `input` carries the data being processed. The two are distinct fields with
  distinct roles, and `with` is validated against the target's declared
  `parameters` schema
  ([The three axes](/reference/call-interface/#the-three-axes-parameters-with-and-input)).
- **`parameters` on Flow objects** inject configuration into `vars` at frame
  entry, keeping behavioral configuration (timeouts, retry budgets,
  deployment-specific settings) out of the data payload
  ([`parameters`](/reference/flow-object/#parameters)).

The result is that an author can express what a Step _does_ without conflating
it with what state the Step _carries_.

## One failure envelope, one routing path

Every non-success outcome in MWL — provider failures, middleware failures,
engine failures, author-raised failures — produces a structured Result with the
same shape: `type`, `code`, `message`, `details`, `retryable`, and optional
`previous`
([The failure envelope](/reference/call-interface/#the-failure-envelope)). All
non-success Result types flow through the same matching machinery, regardless of
origin.

A workflow author writes one `catch` clause that handles
`Provider.Call.Payments.CardDeclined`, another that handles
`Provider.Middleware.Timeout.Exceeded`, another that handles
`System.GatherCompletionUnmet`, and another that handles a domain-specific code
raised by their own `Raise` Step — and all four are expressed the same way.
There is no separate language construct for "handle a provider error" versus
"handle a timeout"; the failure shape, the failure matcher (one declarative
grammar over the envelope's contract fields, shared by `catch` clauses and
`Retry` policies), and the routing mechanism are the same across all failure
sources.

Because the failure envelope sits outside the data plane, `catch` can route on
failure metadata without contaminating the data flowing between Steps. And
because the envelope chains — a superseding failure carries what it displaced in
`previous` — translating, wrapping, or recovering from a failure never destroys
the history of what actually happened. The failure matcher and clause mechanics
are specified in
[Failures and `catch`](/reference/step-mechanics/#failures-and-catch); the
envelope in
[The Call interface and Result](/reference/call-interface/#the-failure-envelope).

## A small vocabulary

The Step-action vocabulary is deliberately constrained: seven actions cover the
full space of workflow control flow. Three design choices keep the vocabulary
small without sacrificing expressiveness:

- **One concurrency primitive.** Concurrent execution comes in two shapes:
  running a sub-workflow per element of a collection, and running a fixed set of
  dispatches side by side. MWL unifies them under a single `Gather` action — the
  iterate form (`over` with a call template) and the scatter form (a literal
  `calls` array) — with one completion policy (`completion`), one collected
  record (`step.results`), and one dispatch shape, the `call` object. See
  [`Gather`](/reference/step-actions/#gather).
- **Execution wrappers are middleware, not Step-level fields.** Retry and
  timeout are not properties of a `Call` Step; they are middleware that wrap an
  operation — a single dispatch, or a whole Flow's Step graph — and combine with
  each other and with platform-defined middleware in author-controlled order.
  See [Middleware mechanics](/reference/middleware-mechanics/).
- **Expressions handle all data shaping, uniformly.** There are no separate path
  syntaxes for input selection, output construction, parameter resolution, or
  result transformation — the same expression embedding, the same binding roots,
  and the same passthrough defaults apply at every shaping seam. See
  [Expressions](/reference/expressions/).

The goal is a small set of primitives, each individually simple, each broadly
applicable — and then stop. New primitives earn their place by enabling
something genuinely inexpressible with the current set, not by being more
convenient for a specific pattern ([Non-goals](/rationale/non-goals/)).

## Expression languages are an extension surface

MWL needs computed values — predicates, transformations, dynamic configuration —
but it does not need to own the language they are computed in. The specification
defines a language-agnostic _embedding_ (a string value whose entire content one
delimiter pair sets off is an expression) and an _evaluation contract_ (the
bindings in scope, what the result means for the field, what happens on
failure). The expression language plugs into that contract, and the delimiter
pair identifies the language: a future language arrives as a new delimiter row
and a new profile, with no change to any field that carries expressions. The
specification mandates no language at all — `Match` and `Loop` cannot function
without one, but which one is the implementation's conformance claim to state
([Expressions](/reference/expressions/)).

What _is_ uniform is the embedding. Wherever an expression appears — an `input`,
a `with` field, a `when` predicate, an `assign` value — it drops into the same
field surface the same way, whatever language the delimiters name. Authors learn
where expressions go once; the language inside the delimiters is a separate,
swappable decision.

This version defines one language:
[CEL](/reference/expressions/#the-cel-profile), enclosed in `{{ }}`. CEL was
chosen first because it is strictly explicit and bounded. Explicitness cuts both
ways — a CEL expression shows exactly what it does, at the cost of verbosity and
some sharp edges (no mixed-type arithmetic being the one authors meet first) —
but bounded evaluation is what a workflow engine wants from an embedded
language: no Turing-complete programs hiding in string fields. Among the
languages that clearly met MWL's needs under that constraint, CEL is the most
widely supported, with multiple mature implementations and an upstream
specification. The choice of _first_ language is not a final verdict; other
languages (JSONata and Expr among them) remain under consideration and can be
added at any time through the same seam.

## Middleware as a composable vocabulary

Workflow languages typically give authors a small fixed set of execution
wrappers — retry and timeout, usually — and treat anything else as an
out-of-band concern handled outside the language. The result is that operations
like rate limiting, circuit breaking, caching, idempotency checks, audit
logging, and distributed tracing end up wrapping the workflow runtime externally
rather than composing into the workflow definition. The state these operations
need lives outside the engine; their semantics are conventions, not language
guarantees.

MWL models the full space of execution wrappers with one mechanism: an ordered
stack of middleware entries around an operation, each entry participating
through four phases — `onEntry` on the way down, `onSuccess` or `onFailure` and
then `onAlways` on the way back out
([The phase model](/reference/middleware-mechanics/#the-phase-model)). The same
vocabulary — stack order, phase blocks, `when` gating, `with` configuration —
applies whether the middleware is doing retry, timeout, caching, audit logging,
or anything else. Middleware are providers, declared through the same catalog
contract as the spec-defined four
([Middleware providers](/reference/providers/middleware-providers/)).

Three properties of this design are worth calling out:

- **Composability beats specialization.** A naive retry is not always what you
  want — if many parallel workflows are hitting the same shared resource, blind
  retry makes the problem worse, and a coordinated rate limiter or circuit
  breaker is the right primitive. Middleware lets authors compose retry with
  rate limiting, timeout with retry, caching with both, in the order that
  produces the semantics they need. The same entries in a different order are a
  different composition — a duration bound outside a retrying entry budgets all
  attempts together; inside it, each attempt separately
  ([ordering and composition](/reference/middleware-mechanics/#the-stack-ordering-and-composition)).
- **The phase model separates the author's shaping from the middleware's
  action.** A phase block carries the author's own expressions (shape the value,
  rewrite the envelope, capture variables) beside the middleware's
  implementation-defined action, configured by `with` and gated by `when`. The
  author's data-flow code runs regardless of the action, so observing,
  translating, and capturing at a boundary never depends on what the middleware
  does there.
- **Middleware composes at the Flow level, not only at the dispatch level.**
  Operations like caching, timeout, retry, and audit logging apply just as
  naturally to a whole Flow execution as to a single dispatch. The `middleware`
  array on a Flow object wraps the Step-graph execution; the array on a `Call`
  Step wraps its dispatch. The composition semantics and phase model are
  identical at both levels — the distinction lies only in what constitutes the
  inner operation.

Operations that other systems implement as out-of-band wrappers around the
workflow runtime — idempotency caches, lifecycle notification hooks, publishing
pipelines — become first-class middleware compositions in MWL. They live in the
workflow definition, not alongside it. This is the property that lets MWL's
middleware mechanism be useful well beyond the patterns the spec defines
directly.

## Putting it together

The principles compose. The narrow Step-action vocabulary works because
middleware absorbs the operations a larger vocabulary would otherwise need to
name. The unified failure envelope works because `Raise` produces the same shape
any other failure source produces. The control-plane/data-plane separation works
because variables exist as a first-class language construct rather than a
payload convention. The Call interface works because Flows and providers honor
the same Result contract. The JSON IR works because authors can produce it from
any host language.

A short example shows several of these principles intersecting. Consider a
`Call` Step that charges a customer: its call targets a payments provider; its
middleware stack is `Retry` outside `Timeout`, with a `Finally` entry outermost
auditing the outcome; and a `catch` clause routes the card-declined failure to a
notification path. Five principles are doing distinct work in that one Step:

- _Orchestration is separate from computation_: the actual charge happens in the
  provider; the workflow only orchestrates the dispatch and the surrounding
  behavior.
- _One Call interface_: nothing about the Step changes if the target later
  becomes a subflow that wraps the provider — the call names a different target
  and every consumer is untouched.
- _Middleware as a composable vocabulary_: `Retry` outside `Timeout` gives each
  attempt its own duration budget; swapping the order would bound all attempts
  together. The composition is authored, not baked into the Step.
- _One failure envelope_: the `catch` clause matches
  `Provider.Call.Payments.CardDeclined` the same way it would match
  `Provider.Middleware.Retry.Exhausted` or `System.Cancelled`. The author writes
  one shape, not three.
- _Control plane / data plane separation_: the `Finally` entry's audit call
  reads the Result in flight at its position (`middleware.result`) — control
  metadata — rather than threading the outcome through the payload between
  Steps.

Reading the same fragment through any single principle gives an incomplete
picture. Reading it through all five at once shows why the language's surface is
as small as it is: each principle is doing some of the work that a more verbose
language would force into the Step definition itself.

The formal specification ([Reference](/reference/)) defines each of these
mechanisms precisely. This section's purpose is to make the design visible _as
design_ — so that the spec's small, opinionated shape reads as intentional
rather than arbitrary.
