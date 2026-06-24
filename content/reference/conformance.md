---
title: "Conformance"
weight: 130
---

This appendix defines what it means to conform to this specification. It names
the conformance profiles an implementation claims, and it consolidates the
reference's normative statements into one index for implementers. Apart from the
statements this appendix itself defines—the profile and claim rules, the
[tooling guidance](#tooling-guidance), and
[the platform boundary](#the-platform-boundary)—everything here is a summary of
a requirement stated in an owning section. The owning section is authoritative;
where a summary and its source diverge, the source governs.

## Normative language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this
specification are to be interpreted as described in BCP 14
([RFC 2119](https://datatracker.ietf.org/doc/html/rfc2119),
[RFC 8174](https://datatracker.ietf.org/doc/html/rfc8174)) when, and only when,
they appear in all capitals, as shown here.

A keyword marks a requirement's strength; it does not bound what is binding.
Behavior the reference states declaratively—"the engine links the superseded
failure as `previous`"—binds a conforming implementation as surely as a MUST.
The keywords appear where strength needs saying: where a requirement is
absolute, where it may be set aside for good reason, or where behavior is
genuinely discretionary.

## Normative and informative content

The reference is normative, with two exceptions. [Concepts](../concepts/) is an
informative orientation: it introduces the model's vocabulary, and the terms it
defines, [the data plane](../concepts/#the-data-plane) and
[the control plane](../concepts/#the-control-plane), are used normatively
elsewhere, but it states no requirements of its own. [Data flow](../data-flow/)
is an informative synthesis: every rule it composes is defined in an owning
section. Where either's phrasing differs from an owning section's, the owning
section governs. The [Guide](/guide/) and [Rationale](/rationale/) are
informative in their entirety.

Examples throughout the reference are illustrative. The provider specification
documents published with the reference are normative for the providers they
specify (see [Call providers](../providers/call-providers/) and
[Middleware providers](../providers/middleware-providers/)).

## What a requirement binds

Every requirement binds one of four subjects, named in the
[requirements index](#the-requirements-index)'s Binds column:

- An **implementation**: the engine and the platform around it, the software
  that accepts, validates, and executes workflow definitions. Where the
  reference says "the engine" or "a platform", the requirement binds the
  implementation.
- A **definition**: a workflow definition. A definition that violates such a
  requirement is invalid; an implementation rejects the violation where it is
  statically detectable, and otherwise fails it at the validation surface that
  detects it (see [Validation](../flow-object/#validation)).
- A **provider**: a call or middleware provider, through its contract, its
  catalog entry, and the surfaces it declares. The spec-defined providers ship
  with the implementation; the requirements bind a third-party provider equally.
- **Tooling**: validators, linters, editors, and other software that reads
  definitions without executing them. The recurring shape of a tooling
  requirement is advisory: warn, never reject.

## Conformance profiles

A _conformance profile_ is a named subset of this specification's requirements,
identified by a URI. Profiles are the units of conformance: an implementation
conforms to this specification by satisfying the core profile, and its
conformance claim states which further profiles it satisfies.

### Profile URIs

A profile URI is an opaque identifier. Implementations and tooling compare
profile URIs character for character, exactly as written here, and never parse
or compare them structurally—the same identity rule provider URIs follow (see
[Provider URIs](../providers/#provider-uris)). Like the
[schema documents](../definition-format/#schema-documents), profiles are
versioned artifacts of a specification release: the path carries the release
version, and a release that changes a profile publishes the changed profile at
its version. Each URI carries the same commitment the workflow `$schema` URI
makes: it dereferences to the published definition of its profile (see
[Schema URIs](../definition-format/#schema-uris)).

URIs under `https://mwl.dev/v0.1/conformance/` are minted only by this
specification. A platform MAY define profiles of its own, for extension behavior
it offers beyond this specification, under URIs it controls; what such a profile
requires is its publisher's to define, and a claim that includes one is portable
only where the URI is honored.

### The profiles

This version defines seven profiles:

| Profile      | URI                                                         |
| ------------ | ----------------------------------------------------------- |
| Core         | `https://mwl.dev/v0.1/conformance/core`                     |
| CEL          | `https://mwl.dev/v0.1/conformance/expressions/cel`          |
| CEL strings  | `https://mwl.dev/v0.1/conformance/expressions/cel/strings`  |
| CEL lists    | `https://mwl.dev/v0.1/conformance/expressions/cel/lists`    |
| CEL sets     | `https://mwl.dev/v0.1/conformance/expressions/cel/sets`     |
| CEL encoders | `https://mwl.dev/v0.1/conformance/expressions/cel/encoders` |
| CEL math     | `https://mwl.dev/v0.1/conformance/expressions/cel/math`     |

#### Core

Core is every requirement of the normative reference except those scoped to an
expression-language profile: the data model and definition format, the Flow,
Step, and call model, all seven actions, middleware mechanics, the execution
model and execution context, parameter validation, the provider model and the
`mwl` URI scheme, and the spec-defined providers, `mock` and the four
middlewares. Core requires no expression evaluation: every shaping default is
defined by the behavior it denotes and realized natively (see
[Expressions](../expressions/)).

#### The expression profiles

The CEL profile binds the
[CEL conformance profile](../expressions/#conformance-profile) as Expressions
defines it: full core CEL plus the MWL functions. An implementation SHOULD
satisfy it (see [Expressions](../expressions/)).

The five extension profiles, one per
[recommended extension](../expressions/#recommended-extensions) capability, each
bind the operations their extension defines, with the semantics cel-go
documents, however an implementation packages them. They are RECOMMENDED for an
implementation claiming the CEL profile. An extension profile extends its base:
a claim includes an extension profile only alongside the profile it extends.

A future expression language enters the way
[the embedding](../expressions/#the-embedding) admits it, a new delimiter row
and its own profile subsection, and mints its profile as a new URI under
`…/conformance/expressions/`.

### Conformance claims

A _conformance claim_ is the set of profile URIs an implementation satisfies.

- A claim MUST include the core profile.
- A claim that includes an extension profile MUST include the profile it
  extends.
- Each URI in a claim binds every requirement of that profile, at the release
  the URI names.

The claim is the entire statement: this specification defines no levels, tiers,
or partial conformance beyond the profiles named. How a platform publishes or
advertises its claim is a platform concern (see
[The platform boundary](#the-platform-boundary)).

## Expression-language support

Supporting an expression language is not required for conformance.
[Expressions](../expressions/) makes the support posture explicit—an
implementation MAY support CEL, another language, or none, and SHOULD support
CEL—and defines every shaping default by the behavior it denotes, so a core-only
implementation realizes the defaults without an evaluator.

The expression-free subset is a coherent static workflow language. A definition
with no expressions dispatches provider and flow calls, routes on failures, fans
out over declared work, sleeps, fails, and returns; every value is a literal and
every default a passthrough. Expressions light up the dynamic half: shaped
inputs and outputs, computed configuration, captured variables, and predicates.

Three constructs depend on expressions:

- `Match` is wholly dependent: every `cases` clause requires a `when`
  expression, making it the one action with no expression-free form (see
  [`Match`](../step-actions/#match)).
- `Loop` is wholly dependent in use: a `Loop` entry's only terminator is
  `onSuccess.when`, which every valid entry writes (see
  [The `Loop` middleware](../providers/middleware-providers/#the-loop-middleware)).
  A core-only implementation still provides the middleware; the dependence binds
  the definitions that attach it.
- `Gather`'s iterate form is practically dependent: `over` accepts a literal
  array, but the form exists to fan out over runtime data (see
  [`Gather`](../step-actions/#gather)).

A definition is within an implementation's claim only where every expression it
embeds is in a language the claim covers. An implementation MUST reject a
definition that embeds an expression in a language it does not support; an
expression is never read as a literal (see [Expressions](../expressions/)).

## Tooling guidance

This section owns the two statements below; every other tooling entry in the
[requirements index](#the-requirements-index) summarizes a statement owned
elsewhere.

Middleware composition is free by design: ordering is the author's to choose,
and the language does not invalidate a composition for being unusual (see
[The stack: ordering and composition](../middleware-mechanics/#the-stack-ordering-and-composition)).
Freedom is not advice, and advice is tooling's to give:

- Tooling MAY warn about an unusual composition: an ordering whose behavior is
  well defined but unlikely to be what the author meant.
- Tooling SHOULD warn about work amplification: a re-running middleware nested
  inside another re-running middleware multiplies attempts, and the product
  grows quickly (see
  [Re-execution and re-entry](../middleware-mechanics/#re-execution-and-re-entry)).

## The platform boundary

This specification defines a language and its observable execution semantics.
Some adjacent concerns are deliberately the platform's; conformance neither
requires nor measures them.

Delivery guarantees under platform failure are platform concerns. The execution
semantics imply at-least-once dispatch—the idempotency idiom built from
`execution.id` and `step.id` exists for exactly that (see
[`step`](../execution-context/#step))—but what a platform guarantees when its
infrastructure fails, and how, is the platform's contract with its users, not
the language's.

The same boundary holds elsewhere. What precedes the root frame—submission,
queueing, scheduling—is outside the model (see
[`execution`](../execution-context/#execution)). How terminal states are
surfaced (see [Result types](../call-interface/#result-types)), how the provider
catalog is published (see
[The provider catalog](../providers/#the-provider-catalog)), and how a
conformance claim is advertised are operational surfaces of the platform.

## Implementation-defined behavior

These choices are deliberately the implementation's. Each is bounded by its
owning section; differing choices here are all conformant.

- Internal architecture: anything that preserves the execution model's rules as
  observable behavior (see
  [The completion contract](../execution-model/#the-completion-contract)).
- Whether, and when, static checks run before execution (see
  [Static checks](../flow-object/#static-checks)).
- The provider catalog's contents, and the governance of namespaces beyond the
  reserved two (see [The provider catalog](../providers/#the-provider-catalog)
  and [Reserved namespaces](../providers/#reserved-namespaces)).
- Which expression languages are supported; the conformance claim states the
  choice (see [Expressions](../expressions/)).
- Additional non-success Result types, and the mapping of Result types to
  terminal states (see [Result types](../call-interface/#result-types)).
- The depth at which a `previous` chain truncates (see
  [Chaining](../execution-context/#chaining)).
- The size cap, if any, on `System.GatherCompletionUnmet`'s `failures` list (see
  [`System.GatherCompletionUnmet`](../step-actions/#systemgathercompletionunmet)).
- Which dispatches start first under a `Gather` `concurrency` cap, and a bound,
  if any, on the number of dispatches a single fan-out may enumerate (see
  [`Gather`](../step-actions/#gather)).
- Key-order preservation in serialization, and precision handling beyond the
  interoperable number range (see [The data model](../data-model/)).
- Additional temporal formats (see
  [Temporal format profile](../data-model/#temporal-format-profile)) and
  additional schema dialects (see
  [Schema documents](../definition-format/#schema-documents)).
- The members of the declared extension surfaces: `execution.platform` (see
  [`execution`](../execution-context/#execution)), a call provider's window
  metadata (see
  [The metadata schema](../providers/call-providers/#the-metadata-schema)), and
  a middleware's contributed metadata (see
  [Contributed metadata](../providers/middleware-providers/#contributed-metadata)).

## The requirements index

The tables below collect the reference's keyword statements: each row summarizes
one requirement, names what it binds, and links the owning section that states
it in full. The owning section is authoritative. Requirements scoped to an
expression profile are marked "claiming CEL"; every other row is core.

### The data model

| Level    | Binds          | Requirement                                                                                     | Defined in                                                        |
| -------- | -------------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| MUST     | definition     | A number is a finite RFC 8259 value; the non-finite IEEE 754 values are not data-model values.  | [The JSON data model](../data-model/#the-json-data-model)         |
| MUST NOT | implementation | Depend on object key order for semantics; preserving order for round-trips is permitted.        | [The JSON data model](../data-model/#the-json-data-model)         |
| MAY      | implementation | Treat negative zero and positive zero as equal.                                                 | [The JSON data model](../data-model/#the-json-data-model)         |
| SHOULD   | definition     | Keep numbers within IEEE 754 double-precision range and precision.                              | [The JSON data model](../data-model/#the-json-data-model)         |
| MAY      | implementation | Reject, or lose precision on, a number outside the interoperable range.                         | [The JSON data model](../data-model/#the-json-data-model)         |
| SHOULD   | definition     | Carry an integer identifier beyond 2^53 as a string.                                            | [A single number type](../data-model/#a-single-number-type)       |
| MUST     | implementation | Accept the temporal profile's timestamp and duration formats; further formats are an extension. | [Temporal format profile](../data-model/#temporal-format-profile) |
| MUST NOT | implementation | Reject a well-formed zero or negative duration; the consuming construct defines its meaning.    | [Temporal format profile](../data-model/#temporal-format-profile) |

### The definition format

| Level    | Binds          | Requirement                                                                                                                                          | Defined in                                                     |
| -------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| MUST     | definition     | Member names are unique within a single JSON object.                                                                                                 | [Well-formedness](../definition-format/#well-formedness)       |
| MUST     | implementation | Reject a definition with duplicate member names as ill-formed.                                                                                       | [Well-formedness](../definition-format/#well-formedness)       |
| MUST     | implementation | Evaluate every schema the format uses, meta-schema and `parameters` alike, under the JSON Schema 2020-12 dialect; further dialects are an extension. | [Schema documents](../definition-format/#schema-documents)     |
| MAY      | tooling        | Validate a definition against the published meta-schema.                                                                                             | [Schema documents](../definition-format/#schema-documents)     |
| MUST     | implementation | Preserve `comment` values across serialization round-trips.                                                                                          | [The `comment` field](../definition-format/#the-comment-field) |
| MUST NOT | implementation | Interpret `comment` semantically.                                                                                                                    | [The `comment` field](../definition-format/#the-comment-field) |
| MAY      | tooling        | Surface `comment` values in operational tooling.                                                                                                     | [The `comment` field](../definition-format/#the-comment-field) |

### Expressions

| Level  | Binds          | Requirement                                                                                                                      | Defined in                                                                   |
| ------ | -------------- | -------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| MAY    | implementation | Support CEL, another expression language, or none.                                                                               | [Expressions](../expressions/)                                               |
| SHOULD | implementation | Support CEL, the language this version standardizes.                                                                             | [Expressions](../expressions/)                                               |
| MUST   | implementation | Reject a definition embedding an expression in a language it does not support; an expression is never read as a literal.         | [Expressions](../expressions/)                                               |
| MUST   | implementation | Reject an embedded expression in a discriminator or static identifier field (`action`, `type`, `provider`, `next`, a Step name). | [Where expressions may appear](../expressions/#where-expressions-may-appear) |
| MAY    | definition     | Match `System.ExpressionEvaluationError` in a `catch`; doing so is discouraged.                                                  | [Evaluation errors](../expressions/#evaluation-errors)                       |
| MUST   | implementation | Claiming CEL: support full core CEL as the CEL specification defines it, with no reduced subset.                                 | [Conformance profile](../expressions/#conformance-profile)                   |
| MUST   | implementation | Claiming CEL: provide the MWL functions (`toJson`/`fromJson`, `durationFromIso8601`/`durationToIso8601`, `now()`/`wallTime()`).  | [MWL functions](../expressions/#mwl-functions)                               |
| SHOULD | implementation | Claiming CEL: provide the recommended extension capabilities.                                                                    | [Recommended extensions](../expressions/#recommended-extensions)             |
| MAY    | implementation | Provide further extensions beyond those recommended.                                                                             | [Recommended extensions](../expressions/#recommended-extensions)             |
| SHOULD | definition     | A CEL predicate evaluates to an explicit `bool`.                                                                                 | [Truthiness](../expressions/#truthiness)                                     |

### The Call interface and Results

| Level      | Binds                      | Requirement                                                                                                    | Defined in                                                      |
| ---------- | -------------------------- | -------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| MUST       | definition                 | A call names exactly one target, `provider` or `flow`: never both, never neither.                              | [The call object](../call-interface/#the-call-object)           |
| MAY        | implementation             | Define additional non-success Result types.                                                                    | [Result types](../call-interface/#result-types)                 |
| SHOULD     | implementation, definition | An extension Result type is PascalCase; the lowercase type space is reserved to this specification.            | [Result types](../call-interface/#result-types)                 |
| MAY        | implementation             | Surface terminal states by mapping Result types, for operational purposes.                                     | [Result types](../call-interface/#result-types)                 |
| MUST       | implementation             | Treat an absent `retryable` and an explicit `null` alike.                                                      | [The failure envelope](../call-interface/#the-failure-envelope) |
| MUST       | implementation             | Emit under `System.` only the codes this specification enumerates.                                             | [Code namespaces](../call-interface/#code-namespaces)           |
| SHOULD NOT | definition                 | Mint a `System.` code from workflow logic.                                                                     | [Code namespaces](../call-interface/#code-namespaces)           |
| MAY        | definition                 | A failure-constructing site sets `previous` explicitly, overriding the engine's link; `null` severs the chain. | [Code namespaces](../call-interface/#code-namespaces)           |

### The Flow object and validation

| Level | Binds          | Requirement                                                                                                | Defined in                                                                            |
| ----- | -------------- | ---------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| MUST  | definition     | `entrypoint` names a key of the Flow's own `steps` object.                                                 | [`entrypoint`](../flow-object/#entrypoint)                                            |
| MUST  | definition     | A `parameters` schema has `"type": "object"` at its top level.                                             | [`parameters`](../flow-object/#parameters)                                            |
| MUST  | definition     | Step names are unique within their containing `steps` object; all routing resolves within that object.     | [Step-name scoping](../flow-object/#step-name-scoping)                                |
| MUST  | definition     | A `flow` name resolves within its lexical chain of enclosing `flows` maps; the nearest declaration wins.   | [Flow-name scoping](../flow-object/#flow-name-scoping)                                |
| MUST  | definition     | `flow` references form no cycle: no Flow reaches itself through the document's call targets.               | [Flow-name scoping](../flow-object/#flow-name-scoping)                                |
| MAY   | tooling        | Warn when a `flows` entry shadows a declaration in an enclosing Flow.                                      | [Flow-name scoping](../flow-object/#flow-name-scoping)                                |
| MUST  | implementation | Evaluate `format` as an assertion in parameter validation.                                                 | [`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed) |
| MAY   | implementation | Include the full JSON Schema validation report in a `System.ParameterValidationFailed` Result's `details`. | [`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed) |

### Steps

| Level | Binds          | Requirement                                                                                  | Defined in                                                                                                        |
| ----- | -------------- | -------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| MUST  | definition     | `next` names a key of the same `steps` map as the Step that carries it.                      | [Routing: `next` and terminal Steps](../step-mechanics/#routing-next-and-terminal-steps)                          |
| MUST  | definition     | Every Step has a defined exit path.                                                          | [Routing: `next` and terminal Steps](../step-mechanics/#routing-next-and-terminal-steps)                          |
| MUST  | definition     | A failure matcher has at least one member.                                                   | [Failure matching](../step-mechanics/#failure-matching)                                                           |
| MUST  | implementation | Accept any syntactically valid `codes` pattern, declared in a catalog or not.                | [Failure matching](../step-mechanics/#failure-matching)                                                           |
| MAY   | tooling        | Warn, using provider and middleware catalogs, about a pattern that matches no declared code. | [Failure matching](../step-mechanics/#failure-matching), [The failure catalog](../providers/#the-failure-catalog) |
| MAY   | tooling        | Warn about a `catch` clause made unreachable by an earlier, broader one.                     | [`catch` clauses](../step-mechanics/#catch-clauses)                                                               |

### Middleware

| Level  | Binds      | Requirement                                                                          | Defined in                                                                        |
| ------ | ---------- | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------- |
| MUST   | provider   | A control action that races the operation it wraps defines its acceptance semantics. | [The phase model](../middleware-mechanics/#the-phase-model)                       |
| MUST   | definition | A failure constructed in `onFailure` has a non-success `type`.                       | [`onFailure`](../middleware-mechanics/#onfailure)                                 |
| MUST   | provider   | A middleware's contract documents which phases expose a gateable action.             | [What a middleware declares](../middleware-mechanics/#what-a-middleware-declares) |
| MAY    | tooling    | Warn about an unusual middleware composition.                                        | [Tooling guidance](#tooling-guidance)                                             |
| SHOULD | tooling    | Warn about work amplification from nested re-running middleware.                     | [Tooling guidance](#tooling-guidance)                                             |

### Execution model

| Level    | Binds          | Requirement                                                                                                        | Defined in                                                                       |
| -------- | -------------- | ------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------- |
| MUST     | implementation | Preserve the execution model's rules as observable behavior, whatever the internal architecture.                   | [The completion contract](../execution-model/#the-completion-contract)           |
| MUST     | implementation | Evaluate each expression-valued field exactly once per execution of its containing construct.                      | [Expression evaluation timing](../execution-model/#expression-evaluation-timing) |
| MUST     | definition     | Capture a value that must stay stable across re-executions into `vars` on first evaluation.                        | [Nondeterministic sources](../execution-model/#nondeterministic-sources)         |
| MUST     | implementation | `now()` returns the current construct execution's entry instant, at every evaluation within it.                    | [The clock pin](../execution-model/#the-clock-pin)                               |
| MUST     | implementation | An interrupting construct constructs its explanatory failure and imposes a cancellation chaining it as `previous`. | [The unwind](../execution-model/#the-unwind)                                     |
| MUST     | implementation | Convert a bare own cancellation at the owner's seam: the cancellation pops, its `previous` continues forward.      | [The conversion seam](../execution-model/#the-conversion-seam)                   |
| MUST NOT | implementation | Convert a chain whose head a cleanup failure superseded; it ascends as-is.                                         | [The conversion seam](../execution-model/#the-conversion-seam)                   |
| MAY      | implementation | Surface an external cancellation's operational reason through the Result's `message` or `details`.                 | [External cancellation](../execution-model/#external-cancellation)               |

#### Observable behavior

The completion contract's observable-behavior requirement covers every rule the
execution model states, keyworded or not. Among them, these are easy to miss and
load-bearing for concurrency and failure handling; each link is the defining
passage:

- Frame immutability: a frame's variables change only on the frame's own serial
  thread of evaluation. While a `Gather` fan-out is in flight, no write-capable
  evaluation runs in the frame, and a dispatch's call fields read the variable
  state at the action's start (see
  [Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).
- Deferred arms: a `Gather` dispatch's arms evaluate at fan-out completion, in
  dispatch order, exactly once per settled dispatch; the arms read the target
  windows and `call.result`, never `step.results` (see
  [The arms at fan-out completion](../step-actions/#the-arms-at-fan-out-completion)
  and [When the arms run](../call-interface/#when-the-arms-run)).
- Two-stage determination: `completion` reads settled Result types, while the
  `Gather`'s own Result is determined after the arms run, from the final
  `step.results`; an arm fault only tightens the outcome, and settled
  dispatches' arms run even on the `Gather`'s own-failure path (see
  [`completion`: the completion policy](../step-actions/#completion-the-completion-policy)).
- The `Retry` restore: on re-entry the frame's variables restore to their
  post-`onEntry` state; the re-entering phase's `assign` evaluates against the
  attempt's aftermath and applies onto the restored state, and no restore
  happens on final emission (see
  [The `Retry` middleware](../providers/middleware-providers/#the-retry-middleware)).
- The unwind's acceptance boundary: interruption takes only work not yet
  committed. A target execution whose Result the platform has already accepted
  resolves as that Result; an interrupted target execution's Result is the
  cancellation in flight (see [The unwind](../execution-model/#the-unwind)).
- Failure-arm faults: a fault in a call's failure arm supersedes the failure in
  hand and chains it as `previous` (see
  [Faults in the call object's fields](../call-interface/#faults-in-the-call-objects-fields)).

### Execution context

| Level    | Binds          | Requirement                                                                                                 | Defined in                                     |
| -------- | -------------- | ----------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| MUST NOT | implementation | Expose execution-context members beyond those this specification defines and its declared extension points. | [`execution`](../execution-context/#execution) |
| MAY      | implementation | Expose additional runtime data under `execution.platform`, including pre-run instants.                      | [`execution`](../execution-context/#execution) |
| MAY      | implementation | Truncate a `previous` chain at a platform-defined depth.                                                    | [Chaining](../execution-context/#chaining)     |
| MUST     | implementation | Replace the deepest retained `previous`, on truncation, with a `System.FailureChainTruncated` error.        | [Chaining](../execution-context/#chaining)     |

### Step actions

| Level    | Binds          | Requirement                                                                                                        | Defined in                                                                             |
| -------- | -------------- | ------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- |
| MUST     | definition     | A `Gather` carries exactly one of its two forms: `over` with `call`, or `calls`.                                   | [`Gather`](../step-actions/#gather)                                                    |
| MUST     | definition     | `calls` is a non-empty array.                                                                                      | [`Gather`](../step-actions/#gather)                                                    |
| MUST     | definition     | A `concurrency` cap is a positive integer.                                                                         | [`Gather`](../step-actions/#gather)                                                    |
| MUST     | implementation | Fail a `Gather` whose `over` result is not an array with `System.ParameterValidationFailed`.                       | [The iterate form: `over` and `call`](../step-actions/#the-iterate-form-over-and-call) |
| MAY      | implementation | Cap the `System.GatherCompletionUnmet` `failures` list at a platform-defined size; `failureCount` is never capped. | [`System.GatherCompletionUnmet`](../step-actions/#systemgathercompletionunmet)         |
| MAY      | implementation | Bound the number of dispatches a single fan-out may enumerate, at a platform-defined, documented limit.            | [`Gather`](../step-actions/#gather)                                                    |
| MUST     | implementation | Fail a `Gather` that exceeds a fan-out bound rather than truncate the fan-out.                                     | [`Gather`](../step-actions/#gather)                                                    |
| SHOULD   | implementation | Fail a fan-out that exceeds the bound at enumeration, before any dispatch starts.                                  | [`Gather`](../step-actions/#gather)                                                    |
| MAY      | tooling        | Warn about an apparent tautology, contradiction, or unreachable `Match` clause.                                    | [Predicates and failure](../step-actions/#predicates-and-failure)                      |
| MUST NOT | tooling        | Reject a definition for an apparent tautology, contradiction, or unreachable clause.                               | [Predicates and failure](../step-actions/#predicates-and-failure)                      |
| MUST     | definition     | A `Sleep` carries exactly one of `for` and `until`.                                                                | [`Sleep`](../step-actions/#sleep)                                                      |
| MUST     | definition     | A `Raise` result `type`, when written, is a non-success type.                                                      | [`Raise`](../step-actions/#raise)                                                      |
| MAY      | tooling        | Warn about a convention-trespassing `code` in a `Raise`.                                                           | [`Raise`](../step-actions/#raise)                                                      |
| MUST NOT | tooling        | Reject a definition for its choice of `code`.                                                                      | [`Raise`](../step-actions/#raise)                                                      |

### Providers

| Level       | Binds          | Requirement                                                                                                                          | Defined in                                                                                                |
| ----------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| MUST        | implementation | Support the `mwl` URI scheme.                                                                                                        | [Provider URIs](../providers/#provider-uris)                                                              |
| MUST        | provider       | A `mwl` URI conforms to the scheme's syntax: three slash-separated parts, the segment charset, and no authority, query, or fragment. | [Syntax](../providers/#syntax)                                                                            |
| MUST NOT    | provider       | Use `.` or `..` as a URI segment.                                                                                                    | [Syntax](../providers/#syntax)                                                                            |
| RECOMMENDED | provider       | Identifiers are lowercase with hyphens, and a name ends with a version segment.                                                      | [Syntax](../providers/#syntax), [Versions](../providers/#versions)                                        |
| MUST NOT    | provider       | Define a provider in the `mwl` namespace.                                                                                            | [Reserved namespaces](../providers/#reserved-namespaces)                                                  |
| MUST NOT    | implementation | Catalog an entry under the `example` namespace.                                                                                      | [Reserved namespaces](../providers/#reserved-namespaces)                                                  |
| SHOULD      | provider       | A failure catalog carries a description for each closed code and open sub-prefix.                                                    | [The failure catalog](../providers/#the-failure-catalog)                                                  |
| MAY         | tooling        | Validate a provider specification document against its published schema.                                                             | [The provider definition document](../providers/#the-provider-definition-document)                        |
| MUST NOT    | provider       | Expose window metadata members beyond the declared metadata schema.                                                                  | [The metadata schema](../providers/call-providers/#the-metadata-schema)                                   |
| MUST        | implementation | Provide the `mock` provider.                                                                                                         | [The `mock` provider](../providers/call-providers/#the-mock-provider)                                     |
| MUST        | implementation | Provide the four spec-defined middlewares: `Retry`, `Timeout`, `Loop`, and `Finally`.                                                | [Spec-defined middleware providers](../providers/middleware-providers/#spec-defined-middleware-providers) |
| MAY         | provider       | Declare a `with` parameter structural: a definition, not an evaluated value.                                                         | [Structural parameters](../providers/middleware-providers/#structural-parameters)                         |
| MUST NOT    | provider       | Expose contributed metadata members beyond the declared schema.                                                                      | [Contributed metadata](../providers/middleware-providers/#contributed-metadata)                           |
| MUST NOT    | provider       | Name a contributed member `enteredAt` or `exitedAt`.                                                                                 | [Contributed metadata](../providers/middleware-providers/#contributed-metadata)                           |
| MUST        | definition     | A `Loop` entry writes `onSuccess.when`.                                                                                              | [The `Loop` middleware](../providers/middleware-providers/#the-loop-middleware)                           |

## Implementation notes

This section is informative.

### Window retention under `Gather`

The deferred arms read their dispatches' target windows when they run, at
fan-out completion (see
[The arms at fan-out completion](../step-actions/#the-arms-at-fan-out-completion)).
Every completed target's window—for a flow target, the completed frame—is
therefore held from the dispatch's settlement until the fan-out completes. At a
large fan-out this is a real memory consideration: the windows an implementation
must be able to serve at arm time scale with the fan-out, not with what the arms
ultimately capture.

### The conformance corpus

A suite of specification-owned, runnable workflows, exercising the language
feature by feature against the `mock` provider (see
[The `mock` provider](../providers/call-providers/#the-mock-provider)), is
planned as a companion artifact for validating implementations. It is not part
of this specification version.
