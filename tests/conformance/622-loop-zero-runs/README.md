# 622 — Loop: zero runs

The `Loop` entry's `onEntry.when` is false, so the action admits no run at all —
not even the first. The entry emits its `onEntry` `output` product as a success
Result: the carried value of a loop of zero iterations, here the passthrough
default, the value the entry received.

## Checks

- `Loop`'s action owns every run including the first; gated off at `onEntry`,
  the wrapped operation never runs.
- The entry emits its `onEntry` output product as a success Result.

Reference: Middleware providers § The Loop middleware.
