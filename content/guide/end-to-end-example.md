---
title: "End-to-end example"
weight: 40
---

The [tour](../tour/) introduces features one at a time. This page shows a single
workflow that puts them together — small enough to read in one sitting, complete
enough to demonstrate how the pieces compose.

The workflow processes an incoming order: it validates the customer, checks
inventory for every line item concurrently (each check retried independently),
routes high-value orders through manual review, charges payment with retries and
a per-attempt timeout, and audits the outcome however the run ends. Every major
language feature appears at least once.

## The workflow

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "comment": "Process an order: validate, check inventory concurrently, route on value, charge payment, audit the outcome.",
  "parameters": {
    "type": "object",
    "properties": {
      "deadline": { "type": "string", "format": "duration", "default": "PT5M" },
      "paymentAttempts": { "type": "integer", "default": 4 },
      "highValueThreshold": { "type": "number", "default": 1000 }
    }
  },
  "middleware": [
    {
      "provider": "mwl:provider.middleware/mwl/finally/v1",
      "onAlways": {
        "with": {
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "input": "{{ {'execution': execution.id, 'result': middleware.result} }}",
            "with": { "method": "POST", "path": "/audit/orders" }
          }
        }
      }
    },
    {
      "provider": "mwl:provider.middleware/mwl/timeout/v1",
      "onEntry": { "with": { "duration": "{{ vars.deadline }}" } }
    }
  ],
  "flows": {
    "CheckItem": {
      "comment": "Check inventory for one line item; fail if unavailable",
      "entrypoint": "check",
      "steps": {
        "check": {
          "action": "Call",
          "call": {
            "provider": "mwl:provider.call/example/http/v1",
            "with": { "method": "POST", "path": "/inventory/check" }
          },
          "middleware": [
            {
              "provider": "mwl:provider.middleware/mwl/retry/v1",
              "onEntry": {
                "with": {
                  "policies": [
                    {
                      "match": { "codes": ["Provider.Call.Http.*"] },
                      "attempts": 3,
                      "backoff": { "initial": "PT1S", "rate": 2 }
                    }
                  ]
                }
              }
            }
          ],
          "next": "verify"
        },
        "verify": {
          "action": "Match",
          "cases": [
            { "when": "{{ match.input.available }}", "next": "done" }
          ],
          "default": { "next": "insufficient" }
        },
        "insufficient": {
          "action": "Raise",
          "result": {
            "code": "Inventory.InsufficientStock",
            "message": "line item is not available",
            "details": { "sku": "{{ frame.input.sku }}" }
          }
        },
        "done": { "action": "Return" }
      }
    }
  },
  "entrypoint": "validate-customer",
  "steps": {
    "validate-customer": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "POST", "path": "/customers/validate" }
      },
      "assign": { "customerId": "{{ step.result.value.id }}" },
      "output": "{{ step.input }}",
      "next": "check-inventory"
    },

    "check-inventory": {
      "action": "Gather",
      "over": "{{ step.input.lineItems }}",
      "call": { "flow": "CheckItem" },
      "concurrency": 10,
      "output": "{{ {'order': step.input, 'inventory': step.results.map(r, r.value)} }}",
      "next": "route-by-value",
      "catch": [
        {
          "match": { "codes": ["System.GatherCompletionUnmet"] },
          "next": "reject-order"
        }
      ]
    },

    "route-by-value": {
      "action": "Match",
      "cases": [
        {
          "when": "{{ match.input.order.amount > vars.highValueThreshold }}",
          "next": "manual-review"
        }
      ],
      "default": { "next": "charge-payment" }
    },

    "manual-review": {
      "action": "Call",
      "input": "{{ {'customerId': vars.customerId, 'order': step.input.order} }}",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "POST", "path": "/orders/review" }
      },
      "output": "{{ step.input }}",
      "next": "charge-payment"
    },

    "charge-payment": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "POST", "path": "/billing/charge" },
        "onSuccess": {
          "assign": { "chargeId": "{{ call.result.value.chargeId }}" }
        },
        "onFailure": {
          "assign": { "failedRequestId": "{{ provider.metadata.requestId }}" }
        }
      },
      "middleware": [
        {
          "provider": "mwl:provider.middleware/mwl/retry/v1",
          "onEntry": {
            "with": {
              "policies": [
                {
                  "match": { "codes": ["Provider.Call.Http.Throttled"] },
                  "attempts": "{{ vars.paymentAttempts }}",
                  "backoff": { "initial": "PT2S", "rate": 2, "jitter": "full" }
                },
                {
                  "match": { "codes": ["Provider.Call.Http.ConnectionFailed"] },
                  "attempts": 3,
                  "backoff": { "initial": "PT1S", "rate": 2 }
                }
              ]
            }
          }
        },
        {
          "provider": "mwl:provider.middleware/mwl/timeout/v1",
          "onEntry": { "with": { "duration": "PT30S" } }
        }
      ],
      "next": "done",
      "catch": [
        {
          "match": {
            "codes": [
              "Provider.Call.Payments.CardDeclined",
              "Provider.Call.Payments.InsufficientFunds"
            ]
          },
          "output": "{{ {'customerId': vars.customerId, 'reason': failure.code, 'requestId': vars.failedRequestId} }}",
          "next": "notify-customer"
        }
      ]
    },

    "notify-customer": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "POST", "path": "/notifications/payment-failed" }
      },
      "next": "done"
    },

    "reject-order": {
      "action": "Raise",
      "result": {
        "code": "Orders.Rejected",
        "message": "one or more line items unavailable"
      }
    },

    "done": { "action": "Return" }
  }
}
```

## How the pieces compose

### `parameters` as the control plane

`deadline`, `paymentAttempts`, and `highValueThreshold` are operational knobs.
Each has a default, so the workflow runs with no arguments; each lands in `vars`
at frame entry, readable by every expression downstream — including the
Flow-level `middleware` and the retry policy that reads
`{{ vars.paymentAttempts }}`. Configuration steers the run without ever touching
the order data flowing between Steps.

### Flow-level middleware: the run's outer skin

Two entries wrap the whole Step graph. `Timeout` bounds total execution: if the
run exceeds `deadline`, the graph is preempted and the Flow fails with
`Provider.Middleware.Timeout.Exceeded` — a frame-level failure that, per the
[scoping rule](/reference/execution-model/#the-scoping-rule), no `catch` inside
the graph can intercept. `Finally` sits outside even that: its audit call runs
at `onAlways` on every exit — success, rejection, payment failure, timeout,
cancellation — and reads the final Result as `middleware.result`. Outermost
position is what makes it the last word.

### A named subflow as the unit of work

`CheckItem` is declared once in `flows` and dispatched per line item. It owns
its own retry (each item's transient check failures retry independently), its
own `Match`, and its own failure: an unavailable item completes the subflow with
`Inventory.InsufficientStock` via `Raise`. The line item rides passthrough
defaults the whole way: the `Gather` element becomes `call.input`, the frame's
input, the `check` Step's input, and the provider's payload, with `frame.input`
still naming it when the `Raise` needs the SKU.

### Fan-out under the default policy

`check-inventory` dispatches `CheckItem` per element of `step.input.lineItems`,
at most ten in flight. The absent `completion` means every dispatch must
succeed; one insufficient item makes the policy unachievable and the `Gather`
fails with `System.GatherCompletionUnmet`, which the `catch` routes to
`reject-order`. The `Raise` there constructs `Orders.Rejected` while that
failure is active, so the engine chains it: the workflow's failure Result reads
`Orders.Rejected`, with the policy failure and each item's
`Inventory.InsufficientStock` evidence beneath it.

### Shaping to keep two things in flight

The `Gather`'s `output` pairs the original order with the inventory results,
because `route-by-value` and `charge-payment` need the order while `summarize`
-style consumers need the checks. Where a Step's action would otherwise replace
the value in flight (`validate-customer`, `manual-review`), an explicit `output`
of `{{ step.input }}` keeps the order moving while `assign` captures what the
response contributed. Data plane for the order; control plane for `customerId`.

### The payment Step: composition in miniature

`charge-payment` is the tour in one Step. The call's arms capture at the
boundary: `chargeId` on success; on failure, the provider window's `requestId` —
context that exists nowhere else once the dispatch is gone, and that the `catch`
clause's `output` then hands to `notify-customer`. `Retry`-outside-`Timeout`
gives each attempt its own 30 seconds, with throttle and connection failures on
separate budgets. A declined card matches no retry policy, surfaces immediately,
and routes to notification as an expected business outcome; anything uncaught
(retry exhaustion, the timeout) propagates and fails the workflow through the
audit's watchful exit.

### One success path

Every successful route converges on `done`, a bare `Return` whose value is
whatever reached it. There is no special exit construct: a terminal Step
completes the frame, and the frame's Result is what the platform — or a calling
Flow — consumes.

## What it doesn't show

- **`Loop`.** No polling or pagination here; see
  [Polling with timeout](../cookbook/polling/) and
  [Pagination accumulation](../cookbook/pagination/).
- **`Sleep`.** Pausing for a duration or until an instant; it appears in the
  polling pattern.
- **The scatter form.** `Gather` can also run a fixed set of different calls
  side by side; see
  [the tour](../tour/gather/#the-scatter-form-a-fixed-set-of-dispatches).
- **Partial success.** This workflow wants all checks to pass; tolerating
  failures with `completion` and shaping `step.results` is
  [Fan-out and partial success](../cookbook/fan-out/).
- **Conditional middleware.** Gating behavior with `when`; see
  [Conditional middleware](../cookbook/conditional-middleware/).

The [Reference](/reference/) formalizes each feature individually. This page is
the integrated view: how the features interlock when an author composes them.
