# 511 — Closed-by-default parameter validation

The called subflow declares no `parameters`, so it takes no arguments: any named
argument fails validation. The dispatch produces
`System.ParameterValidationFailed`, which propagates uncaught as the Flow's
Result.

`details` is omitted from the expected envelope: the specification requires it
to describe the validation error but leaves the exact content to the
implementation.

## Checks

- A Flow that declares no `parameters` rejects any named argument: validation is
  closed by default.
- The failure is the subflow frame's Result, produced at frame entry without
  running its Step graph, and propagates like any frame failure.

Reference: The Flow object § parameters, § System.ParameterValidationFailed.
