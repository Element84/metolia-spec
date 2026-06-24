---
title: "Metolia Workflow Language"
toc: false
outputs: ["html", "SpecMarkdown"]
---

## MWL

A JSON-based language for describing orchestrated workflows.

A workflow is a directed graph of named **steps** — each performing an action,
then transitioning to the next. Steps can call external services, branch on
conditions, run work concurrently, wait, and handle failures. The workflow
definition is data, not code: it describes _what_ should happen and in _what
order_, while the actual computation happens in external services the workflow
invokes.

<!-- dprint-ignore-start -->
{{< cards cols="1" >}}
  {{< card link="/guide/" title="Guide" subtitle="Learn MWL. Introduction, by-example tour, cookbook patterns, a worked end-to-end workflow, and the glossary." >}}
  {{< card link="/reference/" title="Reference" subtitle="The normative specification. Data model, expressions, execution model, step actions, middleware, providers, conformance." >}}
  {{< card link="/rationale/" title="Rationale" subtitle="The reasoning behind the design. Principles, non-goals, architecture, and the change log." >}}
{{< /cards >}}
<!-- dprint-ignore-end -->
