---
title: "Guide"
weight: 10
cascade:
  type: docs
---

## Introduction

The Metolia Workflow Language (MWL) is a JSON-based language for describing
orchestrated workflows. A workflow is a **Flow**: a directed graph of named
**Steps**, each performing one action and then transitioning to the next. Steps
call external services, branch on conditions, fan work out concurrently, wait,
and handle failures. The definition is data, not code: it describes _what_
should happen and in _what order_, while the actual computation happens in
external services the workflow dispatches to. Readers who have used other
workflow orchestration systems will recognize the general shape — a step graph
with structured transitions and data flowing between steps.

MWL is designed around a small set of ideas:

- **Steps execute sequentially within a Flow.** Concurrency exists only where
  the workflow explicitly requests it, through the `Gather` action's fan-out.
- **One shape touches the outside world: the call.** A `call` object names a
  target, gives it arguments and a data payload, and yields a **Result** — a
  success carrying a value, or a structured failure. Only two actions dispatch
  calls (`Call`, one at a time, and `Gather`, many at once); everything else is
  internal control flow. The boundary where side effects can occur is always
  visible in the definition.
- **Providers and subflows are interchangeable targets.** A call can target a
  **provider** (a platform integration, addressed by URI) or a **Flow** (a named
  or inline subflow), and either yields the same kind of Result. Logic can move
  between "a service does it" and "a subflow does it" without the callers
  changing shape.
- **Failures wear one envelope.** Every failure — from a provider, a middleware,
  the engine, or the workflow's own `Raise` — carries the same structured fields
  and routes through the same `catch` machinery.
- **Operational behavior is composed, not baked in.** Retry, timeout, looping,
  cleanup, and platform-defined behaviors like caching are **middleware**:
  ordered wrappers around a dispatch or a whole Flow, composed in
  author-controlled order.
- **Configuration travels beside the data, not through it.** Parameters and
  variables form a control plane separate from the payload flowing between
  Steps, so behavioral knobs never contort the data being processed.
- **Computation embeds as expressions.** Dynamic values are written inside JSON
  string values (`{{ vars.collection }}`), keeping structure and computation as
  separate concerns. The workflow stays declarative; the expressions stay small.
- **How input arrives is not the language's business.** A Flow says what to do
  _given input_; HTTP endpoints, schedules, and event triggers are platform
  concerns.

## What's next?

Where to go next depends on what you came for. [Why MWL](why-mwl/) makes the
case for the language: what these ideas buy a workflow author, and where MWL is
and is not a fit. [The tour](tour/) teaches the language progressively, from a
two-Step Flow through subflows, fan-out, and middleware. The
[end-to-end example](end-to-end-example/) shows the features composed in one
realistic workflow, and the [cookbook](cookbook/) collects recurring patterns.
When you want precise semantics, the [Reference](/reference/) is the
specification; its [Concepts](/reference/concepts/) page is the spec-reader's
