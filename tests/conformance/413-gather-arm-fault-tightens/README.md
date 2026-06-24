# 413 — Gather: an arm fault tightens the settled outcome

A `Gather`'s outcome is determined in two stages. `completion` reads the
dispatches' _settled_ Results — what each target produced — but the `Gather`'s
own verdict is computed only after the arms run, from the final `step.results`.
The two determinations can differ only by _tightening_: an `onSuccess` arm that
faults turns a settled success into that dispatch's failure, never the reverse.
So a fan-out that met its policy at settlement can still fail once the arms run.

The definition is a general-purpose probe. Each input element carries a `with`
object passed straight through to the `mock` provider as the dispatch's whole
arguments object (a whole-value expression at `with`), so an element configures
its own settled outcome — a success with a chosen `value`, a configured
`failure`, a `delay`. A sibling `faultArm` flag independently makes that
dispatch's `onSuccess.value` dereference an absent path, faulting the arm. The
`completion.successes` policy arrives as a workflow parameter, so one definition
probes several outcomes.

## Scenarios

- `all-succeed` — three settled successes, no tightening; the policy is met and
  the `output` default projects the values in dispatch order.
- `settled-failure-observed` — one dispatch is configured to fail at the mock. A
  _settled_ failure is observed data counted by `completion`; under
  `successes: 2` the two successes still meet the policy.
- `arm-fault-tightens-to-unmet` — all three dispatches settle successfully, so
  `completion: { successes: 3 }` is met at settlement; but one dispatch's arm
  faults, dropping the count to two. The `Gather` fails with
  `System.GatherCompletionUnmet`. An implementation that decided the verdict at
  settlement would report success and fail this scenario.
- `arm-fault-survives` — the same arm fault under `successes: 2`: tightening
  drops the count to two, which still clears, so the `Gather` succeeds.
- `mixed-causes-unmet` — a settled failure and an arm fault both count against
  the policy. The evidence in `details.failures` tells the two causes apart by
  `code`, each at its own dispatch `index`.
- `order-under-delay` — three successes with descending per-dispatch `delay`, so
  completion order is the reverse of dispatch order. The output projection is in
  dispatch order regardless.

## Checks

- `completion` reads settled Result types; the `Gather`'s final outcome is
  computed after the arms, from the arm-finalized `step.results`.
- An arm fault tightens only: it can turn a settled success into a failure,
  never a settled failure into a success.
- The settled dispatches' arms run at fan-out completion even when the `Gather`
  is going to fail from its own machinery, before the failure Result is built.
- `System.GatherCompletionUnmet`'s `details` pair each non-success dispatch's
  `index` with its `result`; arm-fault and settled-failure causes are
  distinguishable by `code`.
- `step.results` and the `output` projection are in dispatch order, not
  completion order.

Reference: Step actions § completion: the completion policy, § The arms at
fan-out completion; The Call interface and Result § Faults in the call object's
fields.
