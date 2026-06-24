# 401 — Gather: scatter form

A fan-out in the scatter form: one dispatch per entry of a literal `calls`
array, mixing provider targets with an inline-flow target. The collected order
is the `calls` order regardless of completion order, and the flow target's
Result is consumed exactly as the providers' are.

## Checks

- `calls` makes one independently configured dispatch per entry.
- `step.results` holds dispatch Results in `calls` order.
- A dispatch MAY target an inline Flow; its success Result is consumed through
  the same projection (Flow-Call Result parity).

Reference: Step actions § Gather; The Call interface and Result § Unified
targets.
