---
title: "The tour"
weight: 20
---

A progressive walk through MWL. Each page introduces one layer of the language,
builds on the pages before it, and ends with pointers into the
[Reference](/reference/) for the precise semantics. Read in order if you are
new; each page also stands alone well enough to refresh one topic.

## In this section

- **[Your first Flow](your-first-flow/)** — The smallest complete workflow:
  Steps, a Call, and a Return.
- **[Data and expressions](data-and-expressions/)** — How values move between
  Steps, and how `{{ ... }}` expressions shape them.
- **[Branching and variables](branching-and-variables/)** — Routing with
  `Match`, carrying values with `vars`, and configuring a Flow with
  `parameters`.
- **[Calls and Results](calls-and-results/)** — The call object in full:
  targets, `with` versus `input`, the arms, and the Result every call yields.
- **[Handling failures](handling-failures/)** — The failure envelope, `catch`
  routing, the `failure` context, and `Raise`.
- **[Subflows](subflows/)** — Named and inline Flows, parameter passing,
  isolation, and what calling a Flow shares with calling a provider.
- **[Concurrency with Gather](gather/)** — Fan-out in two forms, completion
  policies, and the collected Results.
- **[Middleware](middleware/)** — Wrapping dispatches and Flows with retry,
  timeout, looping, cleanup, and platform behaviors.
