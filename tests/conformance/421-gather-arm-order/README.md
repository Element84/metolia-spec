# 421 — Gather: arms run in dispatch order

Each dispatch's success arm appends its element to a shared variable. Under
`Gather`, arm evaluation is deferred to fan-out completion: the arms run one
dispatch at a time, in dispatch order, each reading the variable state every
lower-indexed dispatch's arm left. Cross-dispatch accumulation is therefore
deterministic — the result is exactly `"abc"`, whatever order the dispatches
completed in.

## Checks

- Arms do not run as each Result settles; they run at fan-out completion, in
  dispatch order, each arm one `assign` block.
- The dispatch's context (`call.input`, `call.result`) is alive when its arm
  runs.
- The Step's `output` evaluates after the action, observing all captures.

Reference: Step actions § The arms at fan-out completion.
