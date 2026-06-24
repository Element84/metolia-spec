---
title: "Your first Flow"
weight: 10
---

A workflow definition is a single JSON document whose root object is a **Flow**.
The following is complete and valid: a Flow of two Steps, one that calls an
external HTTP service and one that ends the workflow.

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "comment": "Fetch a greeting and return it",
  "entrypoint": "greet",
  "steps": {
    "greet": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "GET", "path": "/hello" }
      },
      "next": "done"
    },
    "done": { "action": "Return" }
  }
}
```

Walk it top to bottom:

- **`$schema`** identifies the document as an MWL Flow definition and names the
  spec version it is written against. It also points at a JSON Schema the
  document can be validated with, before anything runs.
- **`entrypoint`** names the **Step** where execution enters. **`steps`** maps
  Step names to Step definitions; names are yours to choose and only need to be
  unique within the map.
- Each Step has an **`action`**: the one thing it does. `Call` dispatches a
  **call** to an external integration; `Return` ends the Flow successfully.
- **`call`** is the dispatch. Its `provider` field names the target by URI, and
  **`with`** supplies the target's arguments, validated against the parameter
  schema the provider declares. The language doesn't know what an HTTP provider
  does; it knows how to dispatch to it and that a **Result** comes back.
- **`next`** names the Step to transition to on success. A Step either
  transitions via `next` or is terminal: `Return` ends the Flow with a success
  Result, and `Raise` (you'll meet it in
  [Handling failures](../handling-failures/)) ends it with a failure.

Run it mentally: execution enters at `greet`, the call dispatches, the provider
responds, and control follows `next` to `done`, whose `Return` completes the
Flow. That is the whole execution loop — run the current Step; if it
transitions, follow `next` and repeat; if it is terminal, the Flow completes.

Data moved through that Flow even though nothing mentioned data. Whatever input
the Flow was started with arrived at `greet`; the provider's response became
`greet`'s output; that output arrived at `done`, and `Return` returned it as the
Flow's result. Every value-shaping field in MWL has a passthrough default, so a
Flow that says nothing about data still has completely defined data flow. The
[next page](../data-and-expressions/) makes that flow visible and shows how to
shape it.

> [!NOTE]
> The `example` namespace is for documentation
>
> Provider URIs under `example` (`mwl:provider.call/example/http/v1`) are
> illustrations; no real catalog defines them. The one provider every conformant
> implementation does ship is the spec-defined
> [`mock`](/reference/providers/call-providers/#the-mock-provider)
> (`mwl:provider.call/mwl/mock/v1`), a deterministic stand-in that makes any
> example in this guide runnable: swap a target for the `mock` and configure the
> Result you want it to produce.

## Where the spec covers this

- [Concepts](/reference/concepts/) — the model in one page.
- [The Flow object](/reference/flow-object/) — every Flow field.
- [The definition format](/reference/definition-format/) — `$schema`,
  well-formedness, and `comment`.
- [Steps and step mechanics](/reference/step-mechanics/) — routing and the
  shared Step fields.
