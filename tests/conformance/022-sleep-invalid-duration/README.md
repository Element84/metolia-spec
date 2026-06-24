# 022 — Sleep with an invalid duration

A `Sleep` whose `for` value is not a valid ISO 8601 duration. The constraint
binds the value itself, so the literal fails exactly as a computed value would:
`System.ParameterValidationFailed`, the code for a value that fails the type
this specification fixes for a field.

`details` is omitted from the expected envelope: its content is the
implementation's.

## Checks

- A value that is not a valid duration for a duration-typed field produces
  `System.ParameterValidationFailed`.
- `Sleep` has no `catch`; the failure propagates as the Flow's Result.

Reference: Step actions § Sleep; The Flow object §
System.ParameterValidationFailed.
