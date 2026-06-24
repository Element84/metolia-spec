# 502 — Subflow vars isolation

The caller binds a variable, then calls a subflow whose expression reads the
same name. Each frame has its own `vars`: the subflow is entered with a fresh
namespace seeded only from its own `parameters`, so the unguarded read faults
and the subflow frame fails with `System.ExpressionEvaluationError`, which
propagates to complete the Flow.

`message` is omitted from the expected envelope: its wording is the
implementation's.

## Checks

- A subflow does not see or share its caller's variables.
- An expression reading an unbound name faults; the failure is the subflow
  frame's Result and propagates like any frame failure.

Reference: The Flow object § The vars model; Expressions § Evaluation errors.
