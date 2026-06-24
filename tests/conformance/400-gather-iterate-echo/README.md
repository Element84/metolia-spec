# 400 — Gather: iterate form

A fan-out in the iterate form: one dispatch per element of the input array, each
a bare `mock` that echoes its element. The `output` default, the success
projection over `step.results`, returns the elements in dispatch order.

## Checks

- `over` is evaluated once and each element makes one dispatch of the `call`
  template, arriving at the call boundary as `call.input`.
- `step.results` is position-faithful: one Result per dispatch, in element
  order.
- The `Gather` `output` default projects the succeeded dispatches' values, in
  order.

Reference: Step actions § Gather.
