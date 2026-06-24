# 640 — Middleware: value threading

Author shaping on both crossings of one entry: `onEntry.output` reshapes the
value on the way down into the dispatch, and `onSuccess.value` reshapes the
rising value on the way up. The carrier is a `Timeout` entry whose bound never
fires — author shaping belongs to the phase block, not to the middleware's
action, so any entry carries it.

## Checks

- `onEntry.output` becomes the input of the wrapped operation; the echoing mock
  receives the shaped value.
- `onSuccess.value` produces the value the entry emits upward; the Step's
  `output` default reads it.
- `middleware.input` and `middleware.result` are in scope in their phases.

Reference: Middleware mechanics § How values thread the stack, § Author shaping
and the middleware action.
