---
title: "Change Log"
weight: 40
---

This page tracks changes to the MWL specification across releases.

The specification is currently pre-release: no version has been cut, and there
is no commitment to backwards compatibility until v0.1. The structure below is
scaffolding for change tracking starting from that first release.

## Format

Each release is its own section. Within a section:

- **Summary** — a one-paragraph description of what changed and why.
- **Breaking changes** — incompatibilities that workflow authors or implementers
  need to address.
- **Additions** — new features or sections.
- **Changes** — modifications to existing features that don't break
  compatibility.
- **Editorial** — non-normative changes (wording, organization, examples).

Per-page editorial changes don't need to be logged here — `git log` is
authoritative for that. Language-level changes (semantics, syntax, conformance
requirements) do.

## v0.1 (unreleased)

Initial release. Establishes the language baseline: step actions, middleware,
the failure model, the execution model, providers, and the expression language.
See the [Reference](/reference/) for the full specification and the
[Guide](/guide/) for learning material.

Until v0.1 is cut, the spec is in flux and no compatibility guarantees apply.
