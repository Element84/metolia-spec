# 412 — Gather: wait false

Two dispatches: one resolves immediately, one is delayed thirty seconds. Under
`completion: { "successes": 1, "wait": false }`, the first success determines
the outcome, and the `Gather` does not wait: the unfinished dispatch is
cancelled in flight or skipped if never started. Either way its slot in
`step.results` holds a non-success Result, so the success projection contains
only the fast dispatch's value — and the case completes in about a second, not
thirty.

Which of `System.GatherDispatchCancelled` and `System.GatherDispatchSkipped` the
unfinished dispatch resolves as depends on scheduling, so the expected Result
deliberately asserts only the projection, which excludes it either way.

## Checks

- The `Gather` succeeds the moment `successes` is reached.
- Under `wait: false`, the pending dispatch is resolved (cancelled or skipped)
  rather than awaited; the resolution guarantee still holds.
- The `output` projection excludes the non-success slot.

Reference: Step actions § completion: the completion policy, §
System.GatherDispatchSkipped.
