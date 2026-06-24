# 230 — The failure binding

A handler path reads the failure being handled: the `catch` routes to a Step
whose expressions read the matched envelope through the `failure` binding, and
the Flow returns what it read.

## Checks

- `failure` is set when a Step resolves to a failure Result and remains the live
  context across the handler path.
- The binding exposes the envelope's members (`failure.code`,
  `failure.message`).

Reference: Execution context § failure; Steps and step mechanics § catch
clauses.
