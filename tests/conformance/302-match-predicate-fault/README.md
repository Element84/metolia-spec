# 302 — Match: a faulting predicate is an evaluation error

The input is an empty object, so the case predicate's member access faults. A
`when` that fails to evaluate is an evaluation error like any other: the frame
fails with `System.ExpressionEvaluationError`. Failure is not absorbed into a
non-match, and selection does not fall through to `default`.

## Checks

- A faulting `when` fails the frame with `System.ExpressionEvaluationError`.
- The failure is not treated as a non-match, and `default` is not selected.

Reference: Step actions § Predicates and failure; Expressions § Evaluation
errors.
