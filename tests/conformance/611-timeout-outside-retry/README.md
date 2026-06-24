# 611 — Timeout outside Retry

The composition rule made observable: a duration bound placed outside a retrying
middleware budgets all attempts together. Each attempt takes three seconds and
fails; the five-second bound spans the entry's whole participation, so it fires
partway through the second attempt and `Provider.Middleware.Timeout.Exceeded`
emerges — not `Retry.Exhausted`.

Case 612 is the same workload with the entries in the other order.

## Checks

- A `Timeout` bound captured at first entry spans every re-run of the inner
  scope by an entry inside it.
- When the bound fires mid-attempt, the scope is interrupted and the timeout
  failure emerges from the `Timeout` entry's position.

Reference: Middleware mechanics § The stack: ordering and composition;
Middleware providers § The Timeout middleware.
