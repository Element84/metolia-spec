# 201 — Catch clause order

Two `catch` clauses can match the same failure: a prefix pattern first, the
wildcard second. Clauses are evaluated in order, so the prefix clause wins and
the wildcard clause never runs.

## Checks

- Clauses are evaluated in array order; the first clause whose `match` matches
  the failure wins.
- A `Prefix.*` pattern matches any code under the prefix.

Reference: Steps and step mechanics § Failure matching, § catch clauses.
