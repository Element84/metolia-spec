---
title: "Calls and Results"
weight: 40
---

The call is MWL's one shape for getting work done elsewhere. Everything outside
the workflow — services, platform capabilities, and, as the
[Subflows](../subflows/) page shows, other Flows — sits behind a `call` object,
and every call yields a **Result**. This page takes the shape apart.

## The call object

```json
"charge-payment": {
  "action": "Call",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "POST", "path": "/billing/charge" },
    "input": "{{ {'orderId': vars.orderId, 'amount': call.input.amount} }}",
    "onSuccess": {
      "value": "{{ call.result.value.body }}",
      "assign": { "chargeId": "{{ call.result.value.body.id }}" }
    }
  },
  "next": "done"
}
```

A call names exactly one **target**: `provider` (a platform integration,
addressed by URI) or `flow` (a subflow). Around the target sit two data channels
and two Result-consuming **arms**.

### `with` and `input`: configuration versus data

The two channels are never interchangeable.

- **`with`** is the arguments. Every target declares a `parameters` schema for
  the arguments it accepts — a provider through its catalog, a Flow in its own
  definition — and the call's `with` is validated against it at dispatch, the
  way arguments meet a signature.
- **`input`** is the data payload, a separate channel analogous to standard
  input. The target reads its configuration from `with` and its working data
  from `input`. The default is passthrough: `{{ call.input }}`, the payload
  arriving at the call's position, flows to the target untouched unless the
  field reshapes it.

The split is the control-plane/data-plane separation at the call boundary: "how
to do it" rides `with`; "what to do it to" rides `input`.

### The arms: consuming the Result

When the target's Result settles, exactly one arm runs, selected by the Result's
type. **`onSuccess`** shapes and captures: its `value` produces the value the
Call's success Result carries (default: the target's value, unchanged), and its
`assign` writes variables. **`onFailure`** captures only: the failure travels
onward exactly as the target produced it, but the arm's `assign` can save
context from the failed dispatch before it is gone.

The arms matter because of what is in scope there and nowhere else. Inside the
call object, `call.input` is the inbound payload and `call.result` is the
settled Result; in the arms, the **target window** opens — a binding named for
the target field. For a provider target, `provider` carries the provider's
`input`, `result`, and whatever `metadata` its catalog declares (a request ID, a
status code); for a flow target, `flow` is the completed frame. Whatever an arm
does not capture into `vars` is gone with the call:

```json
"onFailure": {
  "assign": { "failedRequestId": "{{ provider.metadata.requestId }}" }
}
```

## The Result

Every call yields exactly one Result, a discriminated union on `type`:

```json
{ "type": "success", "value": { "id": "ch_1907", "status": "captured" } }
```

```json
{
  "type": "error",
  "code": "Provider.Call.Payments.CardDeclined",
  "message": "card declined by issuer",
  "details": { "reason": "insufficient_funds" },
  "retryable": false
}
```

`success` carries a `value` and nothing else. The four non-success types
(`error`, `cancellation`, `timeout`, `skipped`) all carry the same **failure
envelope**: a dotted `code` identifying what happened, an optional `message` and
structured `details`, an advisory `retryable`, and `previous`, a chained prior
failure. One envelope for every failure source is the foundation the
[next page](../handling-failures/) builds on.

The same contract scales up: a whole Flow also completes with exactly one
Result, which is why a subflow can stand wherever a provider can.

## Stubbing a call: the `mock` provider

One call provider is defined by the spec itself and present on every conformant
implementation:

```json
"call": {
  "provider": "mwl:provider.call/mwl/mock/v1",
  "with": { "value": { "id": "ch_test", "status": "captured" } }
}
```

The `mock` produces a Result from its arguments alone: a success with a chosen
`value`, or any failure you configure (with optional latency and window
metadata). It makes every example in this guide runnable without integrations,
stubs dependencies during development, and — because the failure it emits
originates at the real dispatch position — exercises retry, translation, and
`catch` machinery exactly as a real provider failure would.

## Where the spec covers this

- [The Call interface and Result](/reference/call-interface/) — the call object,
  the three axes, the arms, the windows, and the Result.
- [`Call`](/reference/step-actions/#call) — the Step around the dispatch.
- [Call providers](/reference/providers/call-providers/) — the dispatch contract
  and the `mock`.
- [Providers](/reference/providers/) — URIs, catalogs, and parameter validation.
