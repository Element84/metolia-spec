# 610 — Timeout: bound exceeded

A `Timeout` entry bounds a dispatch whose `mock` is configured to delay well
past the bound. The bound elapses before any Result is accepted, the action
interrupts the scope, and `Provider.Middleware.Timeout.Exceeded` emerges from
the entry's position as an ordinary failure.

`message` and `details` are omitted from the expected envelope: the
specification fixes only the failure's type and code.

## Checks

- The mock's `delay` waits before resolving, exercising the bound
  deterministically.
- When the bound elapses before acceptance, the inner scope is interrupted and
  the entry emits a failure of type `timeout` and code
  `Provider.Middleware.Timeout.Exceeded`.

Reference: Middleware providers § The Timeout middleware; Call providers § The
mock provider (delay).
