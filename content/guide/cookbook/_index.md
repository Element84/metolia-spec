---
title: "Cookbook"
weight: 30
---

Recurring patterns, each worked through once so you don't have to derive it
under deadline. Every entry follows the same structure: the **problem**, the
**pattern** that solves it (with a concrete JSON example), **why this shape**
works, common **variations**, and **see also** pointers into the
[Reference](/reference/) for the formal semantics.

The entries assume the vocabulary of [the tour](/guide/tour/); read that first
if a construct here is unfamiliar.

## In this section

- **[Polling with timeout](polling/)** — `Loop` and `Timeout` composed around
  repeated status checks against a slow operation.
- **[Fan-out and partial success](fan-out/)** — `Gather` with an explicit
  completion policy, and shaping `step.results` when some dispatches fail.
- **[Retry composition](retry-composition/)** — multi-policy `Retry`, where
  `Timeout` sits relative to it, and three-layer stacks.
- **[Pagination accumulation](pagination/)** — `Loop`'s carried value driving a
  cursor, with `vars` accumulating pages.
- **[Saga-style compensation](saga/)** — `catch` routing to compensation Steps
  that undo partial work, then re-raise.
- **[Cleanup with `Finally`](finally-cleanup/)** — audit writes and resource
  release that run on every exit, including teardown.
- **[Conditional middleware](conditional-middleware/)** — `when` predicates
  turning operational behavior into parameters.
- **[Wrapping work in a subflow](subflow-wrapper/)** — the inline-Flow idiom for
  per-dispatch behavior, shared deadlines, and refactoring seams.
