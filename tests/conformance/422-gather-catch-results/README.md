# 422 — Gather: catch reads the completed record

The fan-out fails its policy, and the matching `catch` clause's `output` reads
`step.results`. By the resolution guarantee, every slot is a Result by the time
the clause's expressions run, so the clause projects the successes out of a
failed `Gather` and the Flow returns them.

## Checks

- A `Gather`'s `catch` matches the `Gather`'s own failure
  (`System.GatherCompletionUnmet`), never a dispatch's.
- When the `Gather` fails, every dispatch has already resolved: the clause's
  expressions read `step.results` complete.
- The success projection over a partially failed record yields the succeeded
  values in dispatch order.

Reference: Step actions § The collected Results, § Failures and catch.
