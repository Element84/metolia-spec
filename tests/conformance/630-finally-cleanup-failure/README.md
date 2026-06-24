# 630 — Finally: a failed cleanup supersedes

The wrapped dispatch succeeds, then the `Finally` entry's cleanup call fails. A
failed cleanup is real and must surface: the cleanup's failure supersedes the
Result in flight under the `onAlways` rules. The displaced Result was a success,
so nothing is chained: the chain records failures, not the success they
displaced.

## Checks

- `Finally` dispatches its structural `call` parameter on the way out.
- A failed cleanup supersedes the in-flight Result, carrying its originator's
  code (`Finally`'s own catalog is empty).
- A superseded success is displaced without a `previous` link.

Reference: Middleware providers § The Finally middleware; Middleware mechanics §
onAlways.
