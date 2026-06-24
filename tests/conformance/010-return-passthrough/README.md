# 010 — Return passthrough

The minimal Flow: a single terminal `Return` with no `value`. The execution
input reaches the entrypoint Step and passes through to the success Result
unchanged. The definition contains no expressions, so it runs on an
implementation with no expression evaluator.

## Scenarios

- `basic` — a simple object passes through unchanged.
- `data-fidelity` — a structurally rich value passes through untouched: unicode
  strings, escapes, safe-range integers, decimals, booleans, `null` members, and
  empty and deeply nested containers. Because the definition contains no
  expressions, the scenario isolates the data plane's fidelity from everything
  else.

## Checks

- Execution enters at `entrypoint` and a terminal `Return` completes the Flow.
- `Return`'s `value` default passes the Step's received value through.
- The success Result carries the returned data as `value`.
- The data plane carries RFC 8259 values without alteration: no key
  reordering-sensitive comparison, no string normalization, no numeric value
  drift within the safe integer range.

Reference: The Flow object § How a Flow completes; Step actions § Return; The
data model; Concepts § The data plane.
