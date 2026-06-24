# 105 — Provider-side with validation

A `with` argument the `mock`'s parameter schema does not declare. Validation
against a provider's `parameters` is closed by default exactly as a Flow's is,
so the dispatch fails with `System.ParameterValidationFailed`. The dual of case
511, which validates the flow-target half of the symmetry.

`details` is omitted from the expected envelope: its content is the
implementation's.

## Checks

- A call's `with` is validated against the target provider's declared parameter
  schema at dispatch.
- An undeclared argument fails validation; the failure is the dispatch's Result
  and, uncaught, completes the Flow.

Reference: Providers § Parameter validation; The Flow object §
System.ParameterValidationFailed.
