# 650 — Flow-level middleware: shaping the frame input

A `middleware` array on the Flow object wraps the Step graph: the entry's
`onEntry.output` reshapes the frame input on the way down, so the entrypoint
Step receives the shaped value, not the raw execution input. The frame's input
is what the Flow receives; what its entrypoint sees is the stack's to shape.

## Checks

- Flow-level middleware wraps the graph from `entrypoint` to terminal
  completion.
- The entry's `onEntry.output` becomes the value the entrypoint Step receives.
- The graph's success Result rises through the stack unchanged under the shaping
  defaults.

Reference: Middleware mechanics § Where middleware attaches; The Flow object §
Where Flows appear.
