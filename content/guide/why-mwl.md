---
title: "Why MWL"
weight: 15
---

MWL is one of many ways to orchestrate work. This page makes the case for it in
terms of what it makes easy: the things workflow authors regularly need that are
awkward to express, hard to operate, or impossible to inspect in other
approaches. It ends with an honest account of where MWL is not the right tool.

## What MWL makes easy

### Treating workflows as data

An MWL definition is a JSON document. Everything that follows from that is
boring, in the best way: definitions are validated with
[a published JSON Schema](/reference/definition-format/#schema-documents),
diffed in pull requests, code-reviewed line by line, version-controlled,
templated, and generated. SDKs in any host language, visual editors, and LLM
authoring tools all emit the same canonical artifact and get the same execution
semantics, so a team's choice of authoring surface is a preference, not a
commitment. Static analysis is real analysis: which providers a workflow
touches, what failure codes it handles, whether every route resolves — all
checkable without running anything.

### Composing operational behavior

Retry, timeout, looping, and cleanup are not fields on a step; they are
[middleware](/reference/middleware-mechanics/): ordered wrappers an author
stacks around a single dispatch or around a whole Flow. Ordering is meaningful
and explicit — a timeout outside a retry budgets all attempts together, inside
it each attempt separately — and the same mechanism carries platform-defined
behavior: caching, rate limiting, audit logging, publishing. Each entry's `when`
predicate gates it at runtime, so "retries off for this run" or "publish only in
production" is a parameter, not a second copy of the workflow. Operational
behavior that other systems bolt on around the engine lives in the definition,
reviewable next to the logic it wraps.

### Handling every failure the same way

Every non-success outcome — a provider error, a timeout, an engine validation
failure, a failure the workflow itself raises — is a Result carrying the same
[envelope](/reference/call-interface/#the-failure-envelope): a `type`, a dotted
`code`, a message, structured details, an advisory retry signal, and a chain of
what it superseded. One declarative matcher selects among all of them — by code,
exactly or by prefix (`Provider.Call.Payments.*`), by type, or by the retry
signal — and `catch` routing and `Retry` policies share it. When a failure is
translated, wrapped, or recovered along the way, the chain preserves what
actually happened. There is no second error system to learn and no failure
source that routes differently.

### Moving the boundary between service and workflow

A call targets a provider or a Flow, and
[either yields the same kind of Result](/reference/call-interface/#flow-call-result-parity).
That symmetry is worth more than it first appears: logic can start as a single
provider call, grow into a subflow that wraps the call with retries and
validation, and later become a different provider entirely — and the calling
Step never changes shape. The spec-defined
[`mock` provider](/reference/providers/call-providers/#the-mock-provider)
extends the symmetry to development: any call target can be stubbed with a
deterministic stand-in, so workflows are runnable, testable, and demonstrable on
any conformant implementation with no real integrations configured.

### Fanning out with a policy, not a prayer

[`Gather`](/reference/step-actions/#gather) runs many calls concurrently — one
per element of a collection, or a fixed set side by side — under an explicit
completion policy: how many successes the fan-out must achieve, and whether to
let in-flight work finish or cancel it once the outcome is determined. The
collected record holds one Result per dispatch, in order, whatever each outcome
was, so partial success is a first-class state to route on rather than an
exception to untangle: take the three fastest of ten, tolerate stragglers,
partition successes from failures and handle each.

### Keeping configuration out of the payload

A Flow declares typed, defaulted
[`parameters`](/reference/flow-object/#parameters); callers supply arguments
that are validated against them; the values land in variables every expression
can read. The data payload flows separately, Step to Step. Timeouts, retry
budgets, thresholds, and feature flags never have to ride inside the data being
processed, and a subflow's behavior is a function of its declared parameters,
never of ambient state.

## The landscape

Orchestration tools cluster into a few families, each with a legitimate sweet
spot.

**Code-as-workflow SDKs** express workflows in a host language with durable
execution underneath. They offer the full power of a programming language —
arbitrary control flow, native types, libraries — at the cost of the workflow
being a program: inspecting, diffing, and generating workflows means inspecting
code, and the authoring language is fixed by the runtime.

**Container-native DAG engines** model workflows as graphs of containerized
tasks exchanging artifacts, with the cluster as the unit of compute. They are a
natural fit when everything is already a container and artifact handling is the
core need; orchestration is coupled to the compute platform by design.

**Declarative step-graph languages** — MWL's family — express the workflow as a
data document a runtime interprets. The trade is the inverse of
code-as-workflow: less expressive freedom inside a step, full inspectability of
the whole. Within the family, MWL's distinctives are the ones above: one call
shape with provider/subflow parity, middleware as the composition surface for
operational behavior, one failure envelope, and a deliberately small action
vocabulary.

The families are complements more than competitors; many platforms run more than
one. The question is which trade fits the problem.

## Where MWL fits

- **You want workflow definitions to be data** — validated, diffed, reviewed,
  generated, and analyzed without execution.
- **You want operational concerns to compose** in author-controlled order,
  inside the definition, with one mechanism.
- **You want to decouple orchestration from compute.** The runtime sequences
  Steps and routes failures; the work happens in services behind providers,
  scaled and chosen independently.
- **You're building a platform.** MWL specifies the mechanism for providers and
  middleware; the catalog is yours. Two implementations with different catalogs
  still run any workflow whose providers they share.
- **Multiple teams, multiple languages, one artifact.** SDKs in different host
  languages interoperate by emitting the same canonical form.

## Where MWL doesn't fit

- **Fine-grained logic inside a step.** Anything that wants arbitrary code
  mid-step — complex bespoke decision logic, transactional state machines inside
  one operation — wants a host-language SDK or belongs inside a provider on the
  other side of a call.
- **Sub-millisecond orchestration latency.** The JSON definition and expression
  evaluation add overhead. MWL is built for workflows whose steps take seconds
  to minutes, not for hot-path request routing.
- **Workflows that are barely workflows.** A two-step validate-then-charge
  sequence is a function with two calls in it. MWL pays off when branching,
  fan-out, failure routing, and operational wrappers give the definition enough
  structure that data beats code for clarity.
- **Deep commitment to one platform's native integrations.** If a workload is
  already built against a specific platform's orchestration primitives and
  integration surface, the migration cost is real, and portability may not repay
  it.

## Reading further

- [Introduction](/guide/#introduction) — what MWL is at a glance.
- [The tour](../tour/) — the progressive walk through the language.
- [End-to-end example](../end-to-end-example/) — the features composed in one
  realistic workflow.
- [Design principles](/rationale/design-principles/) — the reasoning that
  produces MWL's shape.
- [Architecture and motivation](/rationale/architecture-and-motivation/) — why a
  workflow language at all, and why this one.
