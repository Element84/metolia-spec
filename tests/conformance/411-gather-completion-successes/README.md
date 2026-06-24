# 411 — Gather: a partial-success policy

The same fan-out as case 410, under `completion: { "successes": 1 }`: a single
success satisfies the policy, so the `Gather` succeeds despite the failed
dispatch. With `wait` defaulted to `true`, every dispatch still runs to
completion, and the `output` projection collects the successes in order.

## Checks

- `completion.successes` defines what the fan-out must achieve; failures beyond
  it are observed data, not caught failures.
- Under `wait: true`, pending dispatches run to completion after the outcome is
  determined.
- The `output` default projects only the succeeded dispatches' values, in
  dispatch order.

Reference: Step actions § completion: the completion policy, § Failures and
catch.
