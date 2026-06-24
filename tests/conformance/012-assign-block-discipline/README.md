# 012 — Assign block discipline

Two `Pass` Steps write variables. The second Step's `assign` block both
reassigns `x` and reads it into `y`: the read sees the value from before the
block, while the reassignment takes effect for subsequent Steps.

## Checks

- `assign` binds names into `vars` on successful Step exit.
- Within one `assign` block, every expression evaluates against the variable
  state from before the block: `y` reads the prior `x`.
- A write to an existing name replaces it: the last write wins.

Reference: The Flow object § The vars model; Steps and step mechanics §
Variables: assign.
