---
title: "Handling failures"
weight: 50
---

External calls fail, data disappoints, and budgets run out. In MWL every one of
those outcomes is a failure Result wearing the
[same envelope](../calls-and-results/#the-result), and one mechanism routes on
them all: `catch`.

## `catch`: the failure analogue of `next`

Where `next` routes a Step's success, `catch` routes its failure. It is an
ordered list of clauses, each matching failures and naming a handler Step:

```json
"charge-payment": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "POST", "path": "/billing/charge" }
  },
  "next": "done",
  "catch": [
    {
      "match": {
        "codes": [
          "Provider.Call.Payments.CardDeclined",
          "Provider.Call.Payments.InsufficientFunds"
        ]
      },
      "output": "{{ {'order': step.input, 'reason': failure.code} }}",
      "next": "notify-customer"
    },
    { "match": { "codes": ["*"] }, "next": "failed" }
  ]
}
```

Each clause selects failures with a `match` object — a **failure matcher** —
whose members each constrain one field of the failure: `codes` patterns, `types`
(match a `timeout` differently from an `error`), and `retryable` (the advisory
signal a failure can carry). Every member present must match. The `codes`
grammar is small and closed: `"Prefix.Code"` matches exactly, `"Prefix.*"`
matches a prefix, `"*"` matches anything. Codes are dotted strings whose first
segment names their origin by convention — `System.` for the engine,
`Provider.Call.<name>.` and `Provider.Middleware.<name>.` for providers,
anything else for workflow authors — so one pattern can take a single code, one
provider's codes, all provider failures, or everything. Clauses are tried in
order and the first match wins.

The matcher deliberately stops at those three fields: `message` and `details`
carry unstructured, provider-specific context and are not matchable. When you
need to route on something buried in `details` — an HTTP status, a vendor error
class — first name the distinction as a code, with a middleware
[`onFailure` block](/reference/middleware-mechanics/#onfailure) that constructs
a successor failure carrying the new code, and then match the name.

A matching clause is a conditional edge: control transitions to its `next`, its
`output` shapes what the handler receives (defaulting to the value the failed
Step received, since there is no success value to pass), and its `assign`
captures. A failure no clause matches propagates out of the Flow and becomes the
Flow's Result; `catch` is for the failures you have a plan for.

Only the call-dispatching actions, `Call` and `Gather`, carry `catch` — they are
the Steps whose ordinary work can fail recoverably. Any Step can fail (an
expression fault is a failure like any other), but a failure on a Step without
`catch` simply propagates.

## The `failure` context

When a Step resolves to a failure, the envelope becomes readable as the
`failure` binding, and it stays readable down the whole handler path: the
clause's fields, the handler Step, and every Step after it until a success
clears the context.

```json
"catch": [
  {
    "match": { "codes": ["*"] },
    "assign": {
      "failedStep": "{{ step.name }}",
      "failureCode": "{{ failure.code }}"
    },
    "next": "report-failure"
  }
]
```

The clause's expressions run at the failing Step, so `step` is still that Step:
capturing `step.name` beside `failure.code` records where it happened along with
what happened.

## `Raise`: producing a failure

`Raise` is the terminal failure: it ends the Flow with an envelope the author
constructs.

```json
"reject-order": {
  "action": "Raise",
  "result": {
    "code": "Orders.InvalidAmount",
    "message": "Order amount must be positive",
    "details": { "amount": "{{ step.input.amount }}" }
  }
}
```

`code` is the only required member; `type` defaults to `error`. The code is
yours to choose — author space is everything outside the `System.` and
`Provider.` conventions — and upstream callers `catch` it like any other
failure, because it is one.

A `Raise` with no `result` at all re-raises: it re-emits the active `failure`
unchanged. That is the natural terminal for a handler path that deals with what
it can and propagates the rest:

```json
"failed": { "action": "Raise" }
```

## Failures keep their history

When one failure supersedes another — a handler Step fails while handling, a
middleware translates an envelope, cleanup fails during teardown — the engine
chains the superseded failure as the new one's `previous`. Reading down the
chain reads the history: what was raised, on top of what it displaced, down to
the original cause. Translation never destroys evidence, and a `previous` set
explicitly to `null` is the deliberate way to drop history when carrying it
onward is unwanted.

Two failure codes you will meet before any provider's: a faulting expression
produces `System.ExpressionEvaluationError`, and a `with` or parameter value
that fails schema validation produces `System.ParameterValidationFailed`.
Catching the first is possible but usually wrong — it nearly always marks an
authoring bug, better fixed than handled; prefer the
[defensive constructs](../data-and-expressions/#a-few-cel-notes) that keep the
expression from faulting at all.

## Where the spec covers this

- [Failures and `catch`](/reference/step-mechanics/#failures-and-catch) — the
  matching grammar and clause mechanics.
- [The failure envelope](/reference/call-interface/#the-failure-envelope) — the
  shape and the code namespaces.
- [`Raise`](/reference/step-actions/#raise) — construction, re-raise, and code
  conventions.
- [`failure`](/reference/execution-context/#failure) — the context's lifecycle
  and chaining rules.
- [Failure code reference](/reference/failure-code-reference/) — every
  spec-defined code.
