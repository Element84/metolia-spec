# 102 — The arms: shaping and capture

The success arm reshapes the value the Call's Result carries, and its `assign`
captures a member of the `provider` window into `vars` at the only seam where
that window is in scope. A later Step reads both back.

## Checks

- `onSuccess.value` produces the value the Call's success Result carries; the
  Step's `output` default then reads the reshaped value.
- The `mock`'s `metadata` parameter is exposed verbatim as `provider.metadata`,
  readable in the arms.
- An arm's `assign` carries call-boundary data forward in `vars`; nothing
  crosses outward on its own.

Reference: The Call interface and Result § The arms, § The target windows; Call
providers § The mock provider.
