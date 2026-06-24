---
title: "Call providers"
weight: 10
---

A **call provider** is a Call target: the integration a `call` object names in
its `provider` field. This page defines the call-provider contract—what a call
provider's catalog entry declares, and the semantics every dispatch to one
shares—and the one call provider this specification defines, the
[`mock` provider](#the-mock-provider). The `call` object itself, and how its
Result is consumed, are defined in
[The Call interface and Result](../../call-interface/).

## The dispatch contract

A dispatch hands the provider two things and gets one back. The provider
receives its **arguments**—the call's `with`, validated against the parameter
schema the provider declares (see
[Parameter validation](../#parameter-validation) and
[The three axes](../../call-interface/#the-three-axes-parameters-with-and-input))—and
the **data payload**, the call's evaluated `input`, which the specification
leaves opaque (see [The provider catalog](../#the-provider-catalog)). It
produces exactly one Result: a success carrying a `value`, or a failure under
the codes its catalog declares (see
[The Result](../../call-interface/#the-result)).

A provider target's Result is consumed exactly as a flow target's, through the
same `type` discrimination and the same `value` (see
[Flow-Call Result parity](../../call-interface/#flow-call-result-parity)). A
`with` that fails validation makes `System.ParameterValidationFailed` the
dispatch's Result (see
[`System.ParameterValidationFailed`](../../flow-object/#systemparametervalidationfailed)),
rising through the Step's middleware stack like any failure from the dispatch
position (see [Where `catch` sits](../../step-mechanics/#where-catch-sits)).

The Call's metadata record brackets the dispatch; its `dispatchedAt` and
`acceptedAt` instants are defined with the action (see
[`Call` metadata](../../step-actions/#call-metadata)). The call's arms consume
the completed dispatch through the `provider` window—the provider's `input`,
`result`, and `metadata`—whatever the outcome: the success arm beside the
produced value, the failure arm beside the envelope (see
[The target windows](../../call-interface/#the-target-windows-flow-and-provider)).

## What a call provider declares

A call provider's catalog entry carries the declarations common to every
provider: its URI, of type `provider.call`; its `codePrefix` and failure
catalog; its parameter schema (see
[The provider catalog](../#the-provider-catalog)). Call provider catalog entries
also contain one declaration specific to its kind: the metadata schema for its
window. For a complete call-provider specification in its machine-consumable
form, see the `mock` provider's: [`mock.v1.json`](../mock.v1.json).

### The metadata schema

A call provider declares the members of its `provider.metadata` window as a JSON
Schema describing the object the window exposes, the same declaration device as
`parameters`. The entry for an HTTP provider might declare:

```json
{
  "type": "object",
  "properties": {
    "requestId": { "type": "string" },
    "status": { "type": "number" }
  }
}
```

A provider that declares no metadata schema exposes nothing: its
`provider.metadata` is empty. A provider MUST NOT expose members beyond its
declared schema: the declaration is what makes the window a declared extension
point rather than an ad-hoc surface (see
[`execution`](../../execution-context/#execution)). No member names are
reserved; the namespace is wholly the provider's, and the engine's timing of the
dispatch lives on the Call's own record instead (see
[`provider`](../../execution-context/#provider)).

## Spec-defined call providers

This specification defines one call provider: the `mock` provider, below.
Concrete integrations are a platform's to define and an implementation's to
advertise (see [The provider catalog](../#the-provider-catalog)); the `http` and
`container` providers appearing in this specification's examples are
illustrations under the reserved `example` namespace, not specifications (see
[Reserved namespaces](../#reserved-namespaces)).

### The `mock` provider

```
mwl:provider.call/mwl/mock/v1
```

The `mock` provider is a configurable stand-in: it produces a Result from its
arguments alone, deterministically, touching nothing outside the execution. It
exists so that workflows can run without real integrations and it emulates the
full range of dispatch outcomes: success with a chosen value, any failure,
configurable latency, and window metadata. While `mock` is not generally useful
in real workflows, it exists to support test workflows, conformance validation
of an implementation, and stubbing a dependency during development. An
implementation MUST provide it; the test workflows that validate an
implementation depend on it (see [Conformance](../../conformance/)). It is
usable wherever a call is: a `Call` Step's dispatch, or any dispatch of a
`Gather`.

The `mock`'s specification document, the contract of this section in its
machine-consumable form, is published beside this page:
[`mock.v1.json`](../mock.v1.json). It is normative for the `mock`, and it
doubles as the worked example of a provider specification document for provider
authors.

With an interface like that of the `Return` and `Raise` Steps combined, the
`mock` either returns a value or produces a failure. Bare—no `with` at all—it
echoes: a success Result whose `value` is the dispatch's `input`.

| Parameter  | Type              | Required | Default                | Description                                           |
| ---------- | ----------------- | -------- | ---------------------- | ----------------------------------------------------- |
| `value`    | any               | optional | the dispatch's `input` | The `value` of the success Result produced.           |
| `failure`  | object \| null    | optional | —                      | A failure Result to produce instead (below).          |
| `delay`    | string (duration) | optional | none                   | An ISO 8601 duration to wait before resolving.        |
| `metadata` | object            | optional | `{}`                   | Exposed verbatim as the window's `provider.metadata`. |

A non-null `failure` takes precedence over `value`: with both supplied, the mock
produces the failure and `value` goes unread. A `failure` of `null` is
equivalent to omitting it—the escape that lets a single expression choose
between the success and failure paths, since an expression can compute a field's
value but cannot remove the field. Precedence is what makes the escape
composable: a `value` beside a conditional `failure` supplies the success
branch. An empty `failure` object is neither path: `code` is required, so `{}`
fails validation.

`failure`'s members are the authorable fields of the failure envelope (see
[The failure envelope](../../call-interface/#the-failure-envelope)), as a
`Raise`'s `result` accepts them (see [`Raise`](../../step-actions/#raise)):
`type` (optional, defaults to `"error"`, MUST be a non-success type), `code`
(required), `message`, `details`, `retryable`, and `previous`. The produced
Result is the envelope exactly as configured. Note that, while configurable,
`previous` defaults to absent: the engine links failure chains at the language's
failure-constructing sites, not at a provider boundary (see
[Chaining](../../execution-context/#chaining)).

The point of a configured failure is where it originates: at the dispatch
position, beneath the Step's middleware stack, so it rises through `Retry`
matching, `onFailure` translation, and the Step's `catch` exactly as a real
provider's failure would. A `Raise`, which is terminal and sits in the Step
graph, cannot emulate this.

```json
{
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/mwl/mock/v1",
    "with": {
      "failure": {
        "code": "Provider.Call.Payments.CardDeclined",
        "message": "emulated decline"
      }
    }
  },
  "next": "done",
  "catch": [
    { "match": { "codes": ["Provider.Call.Payments.*"] }, "next": "fallback" }
  ]
}
```

The `code` is the author's to choose, including codes under other providers'
prefixes, as the example shows: emulation is this provider's documented purpose,
and it carries the same standing as a `Raise`'s author-chosen codes (see
[Code namespaces](../../call-interface/#code-namespaces)). Its catalog says so
honestly: `codePrefix` `Mock`, no closed codes, and the whole code space open.

`delay` waits the given duration before resolving, whatever the outcome
configured. It exists to exercise duration-bound behavior, such as `Timeout`
middleware's preemption and acceptance semantics or cancellation during
in-flight work (see [Cancellation](../../execution-model/#cancellation)), with
timing that is otherwise deterministic.

`metadata` is exposed verbatim as `provider.metadata` in the call's arms,
whatever the outcome configured: a failure arm capturing
`provider.metadata.requestId` from an emulated decline reads exactly what was
supplied. The provider's declared metadata schema is deliberately permissive as
an unconstrained object, since its window's content is, like its codes, the
author's to configure.

The `mock` is stateless: its Result is a function of its arguments, and every
dispatch with the same arguments behaves the same. A sequence that varies across
attempts is not expressible with it alone. To emulate something like "fail
twice, then succeed", an author tracks the attempt count in `vars` and computes
`failure` from it, yielding `null` — the success path — once the count is
reached.
