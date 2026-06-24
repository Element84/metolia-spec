---
title: "Reference"
weight: 20
cascade:
  type: docs
outputs: ["html", "SpecMarkdown"]
singleFileSkipTOC: true
---

**Status:** Pre-release. The spec is in flux and no compatibility guarantees
apply until v0.1 is cut.

For learning material (introduction, tutorials, cookbook), see the
[Guide](/guide/). For design reasoning and non-goals, see
[Rationale](/rationale/).

## In this document

- **[Concepts](concepts/)** — _orientation._ Informative: a minimal Flow, the
  execution loop, and the model's vocabulary, each concept pointing to its
  owning section. Defines the term "the data plane".
- **[The data model](data-model/)** — _foundation._ The RFC 8259 value model,
  the single number type, and the temporal format profile.
- **[The definition format](definition-format/)** — _foundation._ The definition
  as a JSON document: well-formedness, schema documents, and `comment`.
- **[Expressions](expressions/)** — _foundation._ The `{{ ... }}` embedding, the
  CEL conformance profile, and the binding namespaces.
- **[The Call interface and Result](call-interface/)** — _spine._ The
  first-class `call` object: target (`provider` | `flow`), `input`, `with`, and
  the `onSuccess`/`onFailure` arms; and the **Result** it yields (success
  `value` and the failure envelope). Defined here, before its consumers.
  Consumed by the `Call` and `Gather` actions.
- **[The Flow object](flow-object/)** — _spine._ The one Flow object and all its
  keys; where it appears (root, named `flows`, inline); the `vars` model;
  Step-name scoping; parameter validation.
- **[Steps and step mechanics](step-mechanics/)** — _spine._ What a Step is;
  shared fields (`input`/`output`, `assign`); the Step lifecycle; `catch` and
  failure matching (against the envelope defined in The Call interface and
  Result).
- **[Middleware mechanics](middleware-mechanics/)** — _spine._ The phase model
  (`onEntry`/`onSuccess`/`onFailure`/`onAlways`), composition and ordering, the
  contract, and how middleware wraps each Call a Step dispatches (Step-level) or
  a Step-graph (Flow-level).
- **[Execution model](execution-model/)** — The completion contract, the frame
  lifecycle, and cancellation.
- **[Execution context](execution-context/)** — The runtime data exposed to
  expressions: frames, frame/Step metadata, and the failure context.
- **[Step actions](step-actions/)** — The Step actions, each with its complete
  field set. `Call` and `Gather` consume the Call interface.
- **[Providers](providers/)** — The extension catalogs: the provider model, Call
  providers, and middleware providers (the spec-defined middleware catalog plus
  the platform-extensible mechanism).
- **[Data flow](data-flow/)** — _synthesis._ Informative: how data moves end to
  end, across Steps, the middleware stack, subflow and Gather boundaries, and
  the failure path, composing the owning sections' rules.
- **[Conformance](conformance/)** — _appendix._ The conformance profiles and
  claims, and the consolidated requirements index.
- **[Failure code reference](failure-code-reference/)** — _appendix._ Every
  spec-defined failure code in one table.
- **[Glossary](glossary/)** — _appendix._ Informative: working definitions of
  the specification's terms, each pointing to its owning section.
