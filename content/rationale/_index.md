---
title: "Rationale"
weight: 30
cascade:
  type: docs
---

Non-normative. Read this to understand _why_ MWL is the way it is — the design
decisions, the alternatives that were considered and rejected, and the broader
motivation behind the language's shape.

The Rationale is not part of the formal specification. The
[Reference](/reference/) section is authoritative for what MWL _is_; this
section explains the reasoning that produced it. Implementers don't need to read
Rationale to build a conformant runtime, but authors evaluating MWL, reviewers
proposing changes, and anyone trying to understand why a particular decision was
made will find the context here.

## In this section

- **[Design principles](design-principles/)** — The "why" behind the language's
  shape: JSON as a compilation target, messages-not-files, orchestration
  separated from computation, the unified Call interface, control/data plane
  separation, one failure envelope, a small vocabulary, expression languages as
  an extension surface, and middleware as a composable vocabulary.
- **[Non-goals](non-goals/)** — Design directions that were considered and
  rejected. Read this before proposing a new language feature.
- **[Architecture and motivation](architecture-and-motivation/)** — The prior
  decisions: what problem a workflow language solves, why orchestration, why a
  data-shaped language, and why a new one rather than an adopted one.
- **[Change log](changelog/)** — Version-to-version differences and the
  reasoning behind each significant change.
