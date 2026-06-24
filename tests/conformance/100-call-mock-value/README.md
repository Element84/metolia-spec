# 100 — Call with a configured value

The basic dispatch: a `Call` Step targets the `mock` provider with a configured
`value`, and the Step's `output` default reads the success Result's value for
the successor.

## Checks

- A `call` dispatches to a provider named by URI and yields its Result.
- The `mock` produces a success Result carrying its `value` parameter.
- The `Call` Step's `output` default, `{{ step.result.value }}`, emits the
  Result's value on success.

Reference: The Call interface and Result; Step actions § Call; Call providers §
The mock provider.
