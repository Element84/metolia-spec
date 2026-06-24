# 110 — Unhandled failure

A dispatch fails and the Step carries no `catch`: the failure propagates out of
the frame and becomes the Flow's Result, envelope intact.

## Checks

- The `mock` emits the failure envelope exactly as configured, with `type`
  defaulting to `"error"`.
- A failure no `catch` clause matches completes the Flow as its Result.
- The envelope's optional members (`message`, `details`, `retryable`) survive
  propagation unchanged.

Reference: The Flow object § How a Flow completes; The Call interface and Result
§ The failure envelope.
