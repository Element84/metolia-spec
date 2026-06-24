---
title: "Saga-style compensation"
weight: 50
---

## Problem

A multi-Step operation commits real side effects as it goes — reserve inventory,
charge the card, schedule fulfillment — and a failure partway through must undo
the work already done, then still fail the workflow with an honest account of
what happened.

## Pattern

Capture what undo will need as you commit each effect; route failures to
compensation Steps with `catch`, deepest effect first; and end the compensation
path with a `Raise` that carries the original failure.

```json
"steps": {
  "reserve-inventory": {
    "action": "Call",
    "call": {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "POST", "path": "/inventory/reserve" }
    },
    "assign": { "reservationId": "{{ step.result.value.reservationId }}" },
    "next": "charge-payment"
  },
  "charge-payment": {
    "action": "Call",
    "call": {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "POST", "path": "/billing/charge" }
    },
    "assign": { "chargeId": "{{ step.result.value.chargeId }}" },
    "next": "schedule-fulfillment",
    "catch": [
      {
        "match": { "codes": ["*"] },
        "assign": { "originalFailure": "{{ failure }}" },
        "next": "release-inventory"
      }
    ]
  },
  "schedule-fulfillment": {
    "action": "Call",
    "call": {
      "provider": "mwl:provider.call/example/http/v1",
      "with": { "method": "POST", "path": "/fulfillment/schedule" }
    },
    "next": "done",
    "catch": [
      {
        "match": { "codes": ["*"] },
        "assign": { "originalFailure": "{{ failure }}" },
        "next": "refund-payment"
      }
    ]
  },
  "refund-payment": {
    "action": "Call",
    "call": {
      "provider": "mwl:provider.call/example/http/v1",
      "input": "{{ {'chargeId': vars.chargeId} }}",
      "with": { "method": "POST", "path": "/billing/refund" }
    },
    "next": "release-inventory"
  },
  "release-inventory": {
    "action": "Call",
    "call": {
      "provider": "mwl:provider.call/example/http/v1",
      "input": "{{ {'reservationId': vars.reservationId} }}",
      "with": { "method": "POST", "path": "/inventory/release" }
    },
    "next": "compensated"
  },
  "compensated": {
    "action": "Raise",
    "result": {
      "code": "Orders.CompensatedFailure",
      "message": "order failed; partial work was rolled back",
      "previous": "{{ vars.originalFailure }}"
    }
  },
  "done": { "action": "Return" }
}
```

## Why this shape

Undo needs identities, so commit Steps capture them. `reservationId` and
`chargeId` go into `vars` the moment each effect exists; the compensation Steps
read them back however far downstream the failure strikes. This is the control
plane doing exactly its job: undo bookkeeping never rides the payload.

Compensation is reverse routing. Each commit Step's `catch` routes to the
compensation for the _previous_ effects: a charge failure releases the
reservation; a fulfillment failure refunds, then releases. The graph encodes the
unwind order explicitly, and each compensation is an ordinary `Call` that can
have its own retries, timeout, and even its own `catch`.

Capture the failure before compensating. The `failure` context is cleared by the
first _successful_ Step completion after it is set — and compensation Steps
succeeding is the plan. A bare `Raise` at the end of the path would therefore
find nothing to re-raise. Capturing the envelope into `vars.originalFailure` at
the clause keeps it, and the terminal `Raise` chains it as `previous`: the
workflow fails with a clear "compensated" verdict on top and the true cause one
link down.

Failed compensation tells on itself. If `release-inventory` itself fails, it has
no `catch`, so the new failure propagates out of the Flow — and because it arose
while a failure was being handled, the engine chains the one being handled as
its `previous`. Nothing about a botched rollback is silently lost.

## Variations

- **Compensate only what's compensable.** Use narrower `match` codes than `"*"`:
  route business failures (declined card) to compensation, but let
  infrastructure failures propagate uncompensated for an operator, or retry them
  first with `Retry` middleware before `catch` ever sees them.
- **Track progress instead of branching.** With many effects, assign a
  `vars.completedSteps` list as you go and route every `catch` to one
  compensation entry point that consults it via `Match`.
- **Compensation as a subflow.** Package the undo Steps as a named Flow taking
  the captured IDs as `parameters`, and call it from each `catch` route; the
  guard rails of [Subflows](/guide/tour/subflows/) apply.
- **Compensation versus cleanup.** Reach for this pattern when undo is
  conditional on _which_ failure occurred and involves real decisions. When
  something must simply happen on every exit (release a lock, write an audit
  record), that is [`Finally`](../finally-cleanup/), not a saga.

## See also

- [Failures and `catch`](/reference/step-mechanics/#failures-and-catch) — the
  routing and clause mechanics.
- [`failure`](/reference/execution-context/#failure) — the context's lifecycle
  (including when it clears) and the chaining rules.
- [`Raise`](/reference/step-actions/#raise) — constructing a failure with an
  explicit `previous`.
