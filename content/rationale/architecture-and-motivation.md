---
title: "Architecture and motivation"
weight: 30
---

[Design principles](../design-principles/) explains the shape decisions _given
that_ the language exists. This page covers the decisions before that: what
problem a workflow language solves, why that problem calls for an orchestrator
and a language at all, and why the answer is a new language rather than an
adopted one. Like everything in the Rationale, it is non-normative.

## The problem

The workloads MWL targets are pipelines: produce a particular output from a
particular input through a particular sequence of operations, where the
operations run in external services, take seconds to hours, fail routinely, and
must be observable after the fact. Three demands recur across every such system:

- **Repeatability.** The sequence is predetermined. Given the same input and the
  same definition, the system should do the same thing, and an operator should
  be able to read the definition and know what that is.
- **Failure as a first-class outcome.** At scale, failures are routine events,
  not exceptions: services throttle, networks blip, data disappoints. The system
  has to classify, route, retry, and report failures as part of its ordinary
  operation.
- **Accountability.** Users depend on being told what happened in each run, and
  whether that matches what was expected. "What happened" has to be a queryable
  fact, not an inference from logs.

A workflow system is the piece of infrastructure that takes these demands off
each pipeline's hands. The question is what shape it should have.

## Why orchestration

One architectural alternative deserves engagement before any language question:
choreography. In a choreographed system there is no coordinator; independent
components subscribe to events, do their work, and emit events, and the system's
behavior emerges from the producer-consumer relationships.

Choreography is genuinely the right pattern at the _system_ level, where
autonomous components owned by independent teams integrate without central
ownership: no producer needs to know its consumers, and no single team owns the
end-to-end shape. A platform built around MWL can and should use event-driven
integration between its major components.

At the _workflow_ level the pattern inverts. The metaphor that names the pattern
is instructive: a choreographed performance works because each dancer's own
sequence is rehearsed and predetermined — each dancer is _orchestrated_ — and
the choreographer composes a dance out of orchestrated parts. Remove the
predetermined sequences and ask each dancer to react to whatever is happening
nearby, and the result is not a dance but improv. A workflow is the dancer, not
the dance: a predetermined sequence with a defined input and output, whose every
run someone must be able to account for. That is orchestration, by definition.

Nor does "distributed orchestration" dissolve the need. An orchestrator built
out of choreographed workers — picking work off queues, advancing state in a
shared store — still requires a definition of what is being orchestrated (a
workflow language, by whatever name), a state store that knows what each
execution is and what comes next (persistence, the hard part), and code that
applies definitions to state transitions (a runtime, however many processes it
spans). Distributing the runtime relocates it; it does not eliminate it.

So: an orchestrator, driven by definitions. The next question is what kind of
definition.

## Why a data-shaped language

"DSL versus code" is a false dichotomy. Every workflow orchestration system is a
DSL, including those marketed as code-based: a code-first workflow framework is
a DSL embedded in a host language and accessed through an SDK, with conventions
(no nondeterminism, no I/O outside designated activities) that the host language
cannot enforce. The real choice is between a DSL shaped as _code_ and one shaped
as _data_. Two arguments push MWL to the data shape.

First, a data-shaped language stands on its own. A code-shaped DSL cannot be
authored without an SDK; the SDK effectively _is_ the language, and every
authoring language someone wants is a vendor-shipped SDK away. A data-shaped
language's canonical form is the artifact itself: a runtime can ship and be
useful before any SDK exists, and SDKs, templating systems, visual editors, and
LLM generators are optional surfaces that compile down to it. The same property
makes definitions structurally analyzable — schema validation, structural
diffing, lineage analysis — in ways arbitrary host-language code is not.

Second, a data-shaped language _structurally_ separates orchestration from
computation. A code-shaped definition has the host language's full
expressiveness inside it, so the runtime must execute or simulate author code
and police the conventions that keep it replayable. JSON cannot express
computation at all: there are no loops or function calls to police. Computation
has to happen somewhere else — in providers, in bounded expressions, in
middleware — and the separation is enforced by the data model rather than by
author discipline or runtime detection. The
[expression-provider boundary](/reference/expressions/#the-expression-provider-boundary)
is this argument carried into the language's details: pure shaping may be an
expression; everything nondeterministic or side-effecting lives behind a
provider.

## Why a new language

Given a data-shaped workflow language, why design one rather than adopt one?

The positive case: the language is the interface, and not only the authoring
interface. SDKs may be the _write_ surface, but the language defines the _read_
surface — execution histories, validation errors, event structures, everything
the platform shows back to authors and operators is shaped by the language
regardless of how the workflow was written. Adopting a language means accepting
someone else's design for the surface your users live in.

The negative case is grounded in experience. MWL's direct antecedent is
[Cirrus](https://github.com/cirrus-geo/cirrus-geo), an open-source geospatial
pipeline framework built on AWS Step Functions. Cirrus needed operations the
underlying language lacked — idempotency checks, lifecycle notifications, result
publishing — and so implemented them as infrastructure wrapped _around_ workflow
executions. The wrap then became an interface in its own right: a partial, leaky
one, defined by what the adopted language did not provide. Its assumptions (in
Cirrus's case, geospatial ones) baked themselves into every workflow, whether
wanted or not, and its operations were invisible to the workflow definitions
they modified. The same trap waits at the bottom of every adopt-and-wrap path;
extending an adopted language instead creates a dialect no upstream runtime
supports, which is a new language with extra steps.

The candidates each fail in their own way, with respect:

- **Amazon States Language.** ASL is genuinely good at this job, and MWL borrows
  ideas from it gladly. But its license permits no modified publication, so it
  cannot be forked or trimmed; implementing it faithfully would still not be
  compatible with the service integrations real ASL workflows depend on, so "we
  run ASL" would not mean what it appears to mean; and adopting it cedes the
  language's direction to its vendor. A clean-room successor that keeps what ASL
  got right is permissible, and MWL is in large part that.
- **Container-native and serverless workflow DSLs.** The data-shaped
  alternatives get much right (small step vocabularies, operationally usable
  representations), and some of their ideas are worth borrowing outright. Where
  they fall short for MWL's goals is extensibility at the language level: fixed
  step types with no provider-style extension seam, compute models oriented to a
  specific dispatch substrate, and no first-class middleware — so the
  cross-cutting operations MWL cares most about would again live as wrappers
  outside the definition.
- **Code-shaped frameworks.** Ruled out by the data-shape arguments above,
  independent of any individual system's merits.

## Lessons carried from Cirrus

Being Cirrus's successor in spirit, MWL preserves what it validated and fixes
what it taught.

Preserved: failures are inevitable and must be first-class; the message between
steps is structured metadata, not files, with file references as fields _within_
messages (a STAC item is just a JSON object, which is why geospatial fits MWL
natively without MWL knowing anything about geospatial); and orchestration stays
decoupled from compute, which is what lets heavy and specialized processing
scale independently.

Fixed: the operations Cirrus performed externally, because its language had
nowhere to put them, are exactly the shape of MWL's middleware. An idempotency
check is a caching middleware; lifecycle notification is an observing middleware
or a runtime event; result publishing is a middleware taking an expression that
selects what to publish. Three properties of that translation drove the design:

- **Everything is opt-in.** Behaviors a wrapper imposed on every workflow become
  behaviors authors compose deliberately; a workflow that doesn't need
  idempotency doesn't carry a cache.
- **The mechanism is general.** Rate limiting, circuit breaking, tracing,
  validation — anything that wraps an operation is expressible the same way. The
  mechanism was not designed around the antecedent's three operations; they
  merely fit inside it.
- **The payload belongs to the user.** Middleware are configured with
  expressions that point into the author's payload, whatever its shape; the
  system never forces a payload structure so its machinery can find things.

## An open language

MWL is deliberately separable from any runtime that executes it. The
specification, its JSON Schema artifacts, and its conformance test corpus are
meant to be published openly, for two reasons that reinforce each other.
Openness is the portability guarantee made credible: a workflow's definition
outlives any one platform when the language it is written in belongs to no one
platform. And openness is design review: a language hardens better in public,
where implementers and authors who owe it nothing can object before
compatibility freezes the mistakes in. The conformance corpus — executable
workflows built on the spec-defined
[`mock` provider](/reference/providers/call-providers/#the-mock-provider) — is
what makes "an implementation of MWL" a testable claim rather than a marketing
one.
