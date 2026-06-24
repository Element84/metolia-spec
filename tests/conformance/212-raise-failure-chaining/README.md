# 212 — Failure chaining

A handler constructs a new failure while one is active: the engine links the
failure being handled as the new failure's `previous`, recording the
supersession.

## Checks

- A `Raise` with a `result` constructs a new failure; unwritten fields are
  absent (`type` defaults to `"error"`).
- Constructing a failure while one is active chains the active failure as
  `previous`, unwritten.

Reference: Step actions § Raise; Execution context § Chaining.
