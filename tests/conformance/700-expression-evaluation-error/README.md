# 700 — Expression evaluation error

A `Pass` Step's `output` expression reads a member that does not exist. The
evaluation faults, the frame fails with `System.ExpressionEvaluationError`, and
— `Pass` carrying no `catch` — the failure completes the Flow.

`message` is omitted from the expected envelope: its wording is the
implementation's.

## Checks

- An expression that cannot be evaluated produces a failure of type `error` and
  code `System.ExpressionEvaluationError`.
- The failure propagates out of the frame as its Result.

Reference: Expressions § Evaluation errors; Steps and step mechanics § Failures
and catch.
