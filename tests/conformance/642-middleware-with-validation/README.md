# 642 — Middleware: phase with validation

A bare `Retry` entry: no phase blocks at all. An absent block is equivalent to
an empty one, so `onEntry` still runs with an empty `with` — which fails the
schema the middleware declares for that phase, since `policies` is required. The
`System.ParameterValidationFailed` failure is emitted from the entry's position
on the descent: the wrapped dispatch never runs.

`details` is omitted from the expected envelope: its content is the
implementation's.

## Checks

- A phase's `with` is validated against the middleware's declared schema when
  the phase runs; absent `with` validates as `{}`.
- An `onEntry` failure means the wrapped operation never runs.
- The failure, uncaught, completes the Flow.

Reference: Middleware mechanics § Validation, § When a phase fails; Middleware
providers § The Retry middleware.
