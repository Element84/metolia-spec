# 631 — Finally: a failed cleanup chains a failed Result

The dual of case 630: the wrapped dispatch fails, then the cleanup fails too.
The cleanup's failure supersedes the failure in flight, and a superseded failure
— unlike the displaced success of 630 — is chained via `previous`: both failures
surface, in supersession order.

## Checks

- A failed cleanup supersedes the Result in flight under the `onAlways` rules.
- The superseded failure is chained as the superseding failure's `previous`.

Reference: Middleware providers § The Finally middleware; Middleware mechanics §
onAlways.
