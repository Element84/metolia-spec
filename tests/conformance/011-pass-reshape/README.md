# 011 — Pass reshape

A `Pass` Step reshapes the value between two Steps: its `output` expression
wraps the received value, and the successor returns the reshaped result.

## Checks

- `Pass` performs no action work; its `output` shapes the emitted value.
- A transition hands one Step's output to its successor as input.
- An expression produces a structured value (a CEL map literal).

Reference: Step actions § Pass; Steps and step mechanics § Data flow.
