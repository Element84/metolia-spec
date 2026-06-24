# 213 — Severing the chain

The same handler shape as case 212, but the constructed failure writes
`previous: null` itself. Writing `previous` overrides the engine's link, so the
active failure's history is deliberately dropped.

## Checks

- A failure-constructing site MAY set `previous` explicitly, overriding the
  engine's chaining.
- `previous: null` severs the chain: the emitted failure carries no history.

Reference: Step actions § Raise; The Call interface and Result § Code namespaces
(previous); Execution context § Chaining.
