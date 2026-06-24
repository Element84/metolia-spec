# 612 — Timeout inside Retry

The other order of case 611: the bound sits inside the retrying entry, so it is
re-established per run and budgets each attempt separately. Each three-second
attempt finishes within its own five-second bound, both attempts fail on their
own terms, and the `Retry` budget exhausts — the bound never fires, even though
the attempts total six seconds.

## Checks

- A `Timeout` inside a re-running entry is re-established per run, bounding each
  attempt separately.
- The same entries in a different order are a different composition: this
  workload yields `Retry.Exhausted` where case 611 yields `Timeout.Exceeded`.

Reference: Middleware mechanics § The stack: ordering and composition, §
Re-execution and re-entry.
