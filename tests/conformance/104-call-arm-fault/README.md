# 104 — A faulting arm fails the Call

The target succeeds, but the success arm's `value` expression faults. A fault in
any of the call object's fields fails the Call execution itself — the arms are
no exception — and the resulting failure takes the same path a failed dispatch
takes.

## Checks

- A faulting `onSuccess.value` fails the Call even where the target succeeded.
- The failure is `System.ExpressionEvaluationError` and, uncaught, completes the
  Flow.

Reference: The Call interface and Result § Faults in the call object's fields;
Expressions § Evaluation errors.
