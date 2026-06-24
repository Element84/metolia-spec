# 402 — Gather: the `over` domain

The same iterate `Gather` probed at the edges of `over`'s domain. The `over`
expression's result MUST be an array, but which array, and whether it is one at
all, is data-dependent: the definition is fixed and the input decides the
outcome.

## Scenarios

- `empty-array` — an `over` that produces an empty array is legal: the `Gather`
  makes zero dispatches and completes with `step.results` empty, so the `output`
  projection is an empty array.
- `non-array` — the `over` expression produces a string. Its result MUST be an
  array, so the `Gather` MUST fail with `System.ParameterValidationFailed`: a
  value of the wrong type for a field whose type is known. No dispatch is made.
  `details` is omitted from the expected envelope: its content is the
  implementation's.

## Checks

- An empty `over` result makes zero dispatches and the action completes.
- The `output` default projects an empty `step.results` to `[]`.
- A non-array `over` result fails the `Gather` with
  `System.ParameterValidationFailed`.
- The failure is the `Gather`'s own and, uncaught, completes the Flow.

Reference: Step actions § The iterate form; The Flow object §
System.ParameterValidationFailed.
