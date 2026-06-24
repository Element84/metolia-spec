---
title: "Subflows"
weight: 60
---

A call's target doesn't have to be a provider. It can be another Flow, and
because a completed Flow yields the same kind of Result a provider does, the
caller can't tell the difference. Subflows are how a workflow grows structure
without its callers changing shape.

## Named Flows: the `flows` map

A Flow declares reusable subflows in its `flows` map, and a call targets one by
name:

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "flows": {
    "RegisterGranule": {
      "comment": "Validate one granule and register it with the catalog",
      "parameters": {
        "type": "object",
        "properties": { "collection": { "type": "string" } },
        "required": ["collection"]
      },
      "entrypoint": "validate",
      "steps": {
        "validate": {
          "action": "Call",
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "with": { "method": "POST", "path": "/granules/validate" }
          },
          "next": "register"
        },
        "register": {
          "action": "Call",
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "with": {
              "method": "POST",
              "path": "{{ '/collections/' + vars.collection + '/granules' }}"
            }
          },
          "next": "done"
        },
        "done": { "action": "Return" }
      }
    }
  },
  "entrypoint": "process",
  "steps": {
    "process": {
      "action": "Call",
      "call": {
        "flow": "RegisterGranule",
        "with": { "collection": "modis-l1" }
      },
      "next": "done"
    },
    "done": { "action": "Return" }
  }
}
```

The call site reads exactly like a provider call: a target, arguments in `with`,
a payload on the default `input` passthrough. The subflow declares `parameters`
the same way any Flow does, the arguments are validated the same way any
target's are, and the values seed the subflow's `vars`. One Call interface, two
kinds of target.

A Flow used in exactly one place doesn't need a name; the `flow` field accepts
an inline Flow object directly. Naming is for reuse and reference, nothing more
— a named entry and an inline object are the same construct.

## Isolation: a subflow is a function of its inputs

A called Flow runs in its own **frame**: fresh `vars` seeded from its own
`parameters`, its own Step state, its own lifecycle. It does not see its
caller's variables, and nothing crosses the boundary implicitly. Data goes in
through the call's `input` and `with`; what comes back is the Result.

When the caller needs more than the Result's value — an inner variable, the
subflow's timing — the call's arms are the seam. A flow target opens the `flow`
window there: the completed frame, with its `result`, its final `vars`, its
`input`, and its `metadata`. Capture is explicit:

```json
"call": {
  "flow": "RegisterGranule",
  "with": { "collection": "modis-l1" },
  "onSuccess": {
    "assign": { "registeredCount": "{{ flow.vars.count }}" }
  }
}
```

The discipline is the point: a subflow's behavior is a function of its declared
parameters and its input, never of ambient state, and what it surfaces is what
its caller deliberately captures.

## Scoping: names carry inward

A name declared in `flows` is visible to call sites in the declaring Flow and in
every Flow nested within it: declare `RegisterGranule` once near the root and
call it from anywhere beneath. Resolution is lexical — the document's nesting,
not the runtime call path — checking the containing Flow's map first, then each
enclosing Flow's outward, with the nearest declaration winning. A callee
resolves its own references from where it was _declared_, so what a subflow
means never depends on who called it.

References cannot form a cycle: no Flow may reach itself through the
flow-reference graph, so recursion is not expressible. A workflow that repeats
does so within one graph by routing `next` backward, or with the
[`Loop` middleware](../middleware/#loop-repeat-while).

## Why this matters

Result parity makes the provider/subflow boundary a refactoring seam rather than
an architectural decision. Start with a provider call; when the operation grows
validation, retries, or a second call, wrap it in a subflow and point the call's
target at the Flow; if it later becomes a real service, point the target back at
a provider. Callers never change. The same parity is what makes `Gather` (next
page) indifferent to what it fans out over, and it gives platforms a clean path
to features like cross-workflow invocation as ordinary providers.

## Where the spec covers this

- [`flows`](/reference/flow-object/#flows) and
  [Flow-name scoping](/reference/flow-object/#flow-name-scoping) — declaration
  and resolution.
- [Flow-Call Result parity](/reference/call-interface/#flow-call-result-parity)
  — the contract that makes targets interchangeable.
- [The target windows](/reference/call-interface/#the-target-windows-flow-and-provider)
  — what the arms see.
- [The `vars` model](/reference/flow-object/#the-vars-model) — frame isolation.
