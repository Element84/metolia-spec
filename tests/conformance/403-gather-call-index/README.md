# 403 — Gather: call.index

The call template's fields evaluate afresh per dispatch, and `call.index` is the
dispatch's 0-based position: a `with` reading it configures each dispatch from
its own position.

## Checks

- `call.index` is available on `Gather`-dispatched calls, 0-based, in element
  order.
- The template's `with` evaluates per dispatch against that dispatch's bindings.

Reference: Step actions § The dispatch model; Execution context § call.
