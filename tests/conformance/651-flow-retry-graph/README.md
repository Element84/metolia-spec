# 651 — Flow-level Retry: re-running the graph

A `Retry` entry on the Flow wraps the whole Step graph. The graph routes on a
counter and raises until the counter reaches two; each raised failure reaches
the flow-level entry as the graph's Result, the entry restores the variables and
applies the `assign` carry, and the graph re-runs from `entrypoint`. The third
run routes to success.

## Checks

- Flow-level `Retry` re-enters the Step graph as a whole; a failure a `catch`
  never handles reaches it as the graph's Result.
- Re-entry restores the frame's variables to their post-`onEntry` state, with
  the gap's `assign` as the carry — at the Flow level exactly as at the Step
  level.
- The recovered success is the frame's Result.

Reference: Middleware mechanics § Where middleware attaches; Middleware
providers § The Retry middleware.
