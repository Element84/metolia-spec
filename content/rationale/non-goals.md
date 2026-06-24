---
title: "Non-goals"
weight: 20
---

MWL is deliberately scoped. The following are design directions that have been
considered and rejected. Future revisions should preserve these decisions rather
than relitigate them.

## Event-driven triggers and invocation

MWL does not define how workflow executions are initiated. The language
describes _what to do given input_; it has no constructs for HTTP endpoints,
scheduled triggers, queue subscriptions, file-arrival hooks, or any other
mechanism by which inputs reach a workflow. These are platform concerns.

The reasoning mirrors the
[Orchestration is separate from computation](/rationale/design-principles/#orchestration-is-separate-from-computation)
principle: the runtime orchestrates a Flow given an input, and what produces
that input is outside the Flow's definition. A platform may surface workflow
invocation through any number of front-end mechanisms; the workflow doesn't know
and shouldn't need to know.

This keeps the language tightly scoped and avoids the problem of every
platform's trigger model leaking into the language definition. Authors who want
to model "this workflow runs every five minutes" or "this workflow runs when an
object lands in a bucket" handle that at the platform layer, not in the workflow
definition.

## Reuse as a language goal

MWL does not pursue reuse as a design goal. The language does contain a
reuse-shaped feature: a Flow declared once in a
[`flows` map](/reference/flow-object/#flows) can be called from many Steps and
from `Gather` dispatches. But that feature was not the product of a reuse
requirement. It follows from the unified Call interface: once a Flow and a
provider yield the same kind of Result through the same call shape
([Flow-Call Result parity](/reference/call-interface/#flow-call-result-parity)),
letting a call target a named Flow costs the language almost nothing, and the
`flows` map fell out of the unification. The win is taken; the goal is
unchanged.

The boundary is the definition document. Flow references resolve lexically
within the document that contains them: a call may target an entry in its own
Flow's `flows` map or in the map of any enclosing Flow
([Flow-name scoping](/reference/flow-object/#flow-name-scoping)). There are no
imports, no registry references, no fragment inclusion, and no mechanism by
which one definition names a Flow declared in another. A definition is
self-contained, and its behavior is a function of the document alone — the same
property the scoping rules and the `vars` model protect at runtime.

Authors who want reuse beyond that boundary — shared Step libraries,
parameterized templates, organization-wide subflows — should produce MWL
definitions through higher-layer tooling: programmatic SDKs, templating systems,
or composition libraries that compile into self-contained MWL documents. Reuse
expressed through duplication at compile time is acceptable; workflow
definitions are not typically large enough for duplication to be a real cost,
and the runtime gains from self-contained definitions (simpler validation,
simpler execution, simpler reasoning) are tangible.

Reuse should not be confused with composability, which has always been a goal
and is sought throughout the specification. Composable interfaces are often
simpler and yet more expressive, and thus more capable: one call shape consumed
by both `Call` and `Gather`, middleware entries that stack in author-chosen
order at two attachment levels, Flows that nest because every Flow presents the
same Result contract. The non-goal here is narrow — the language does not chase
mechanisms for _sharing definitions_ — and it implies nothing against the
composition of the constructs the language already has.

### Cross-definition invocation

Within one definition, calling a Flow is first-class. Across definitions, the
language has no concept at all: no action, reference form, or primitive for
starting another workflow execution. If a platform offers a provider that starts
other workflow executions, that is a provider concern; workflows targeting it
are dispatching to an external integration, not invoking anything the language
models. Flow-Call Result parity intentionally smooths that path: because every
completed Flow yields an ordinary Result, a platform's "run a workflow" provider
can present to its caller exactly like any other call target.

## Data typing across the Step boundary

MWL does not enforce schemas or type contracts on the data flowing between
Steps. The spec defines the _shape_ of the execution context (`step.input`, a
Result's `value`, the failure envelope) but treats the _content_ of those fields
as opaque. Inter-Step data is whatever the previous Step produced; the language
does not validate it against a declared type before delivering it to the next
Step.

This is not a statement that typing and schema enforcement lack value; to the
contrary, they are extremely valuable. The position is that such enforcement
belongs at a higher layer: one that has insight into what specific providers
accept and return, what shape a given `Call` Step's output will take, and what
downstream Steps expect. Tooling above the language such as SDKs, registries,
IDE integrations, linters, etc. can provide typed contracts with richer
knowledge than the language definition alone possesses. The language defines
data flow mechanics; data flow validation is a tooling concern.

The one place the language does validate values is the control plane: a call's
`with` is validated against its target's declared `parameters` schema, because
configuration is a contract the target itself declares
([The three axes](/reference/call-interface/#the-three-axes-parameters-with-and-input)).
The data channel stays open by design.

## Versioning within the language

A workflow definition carries no version field of its own beyond the spec URI in
`$schema`, which is not the workflow definition's version but the version of MWL
in which it is authored. A definition cannot and does not know its version:
versioning is a higher-level concern, something the platform should manage
through content hashing, monotonic identifiers, or whatever mechanism the
platform chooses. The language is deliberately uninvolved: it describes _what to
do_, not _which revision of what-to-do this is_.

This avoids a class of problems that arise when definitions carry their own
version metadata — staleness, conflicts between the declared version and the
actual content, and the question of what "version" even means when definitions
are generated by tooling rather than hand-edited. The normative statement of
this position lives in the reference
([Definition versioning](/reference/flow-object/#definition-versioning)); this
entry records that it is a deliberate boundary, not an omission.

## A required provider catalog

The specification requires a small, fixed implementation floor: the four
spec-defined middlewares (`Retry`, `Timeout`, `Loop`, `Finally`) and one call
provider, the [`mock`](/reference/providers/call-providers/#the-mock-provider).
Beyond that floor, requiring providers is a non-goal. The spec defines the
_mechanism_ by which platforms declare and validate providers, but it does not
ship a standard catalog of common integrations (HTTP, container execution, queue
dispatch, etc.), and it does not intend to grow one as a requirement.

The reasoning: implementations should not be forced to support an integration
that makes no sense in their environment, even at some cost to out-of-the-box
interoperability. A workflow's portability is a function of provider adoption,
and that adoption is better driven by published provider specifications that
platforms opt into than by spec mandate.

The `mock` provider is the sole concession, and it is admitted for two reasons.
First, it allows example workflows that are executable on any conformant
implementation, with no real integrations configured. Second, because those
example workflows can exercise every facet of the language itself, they double
as conformance tests: a corpus of `mock`-only workflows can validate an
implementation's correctness end to end. A capability that exists to test the
language, rather than to integrate with anything, is the one provider that
belongs to the language.

As implementations mature and consensus consolidates around specific
integrations, the spec may onboard provider specifications as _recommended_
extensions. The base spec is unlikely to ever require more than it does today.

## Vocabulary expansion and syntactic sugar

MWL aims to adopt the simplest abstractions that are still powerful enough to
cover the full space of workflow control flow — and then stop. The language
resists both directions of complexity: accumulating specialized constructs for
patterns already expressible with existing primitives, and introducing fewer but
more complex abstractions that would be harder to learn and reason about. The
goal is the sweet spot: a small set of primitives, each individually simple,
each broadly applicable.

The current vocabulary reflects this. `Match` handles conditional routing. The
[`Loop` middleware](/reference/providers/middleware-providers/#the-loop-middleware)
handles iterative re-execution with structured termination; for simple cases,
`next` targeting a previously-executed Step also expresses looping. `Gather`
handles fan-out and collection in its two forms. These are general-purpose
primitives whose composition covers patterns that other languages address with
dedicated syntax — `if/else`, `while`, `for-each`, `try/catch`. MWL does not
introduce those constructs because the existing primitives already express them,
and each additional construct would widen the vocabulary without widening the
capability. It would also introduce ambiguity about which construct to use for a
given pattern — a cost that falls on every workflow author and every tool that
processes MWL definitions.

New primitives should earn their place by enabling something genuinely
inexpressible with the current set, not by being more convenient for a specific
pattern. Convenience is the domain of higher-layer tooling — SDKs, builders,
visual editors — that can present ergonomic interfaces while emitting the
constrained MWL form.
