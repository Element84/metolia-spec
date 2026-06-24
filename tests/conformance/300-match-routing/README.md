# 300 — Match routing

A routing `Match` with two ordered case clauses and a `default`. Which clause is
selected depends entirely on the input: clauses are tried in order, the first
whose predicate holds wins, and when none holds the required `default` routes.

## Scenarios

- `first-clause-wins` — both case predicates hold for the input, and the first
  wins: no later predicate evaluates, and the selected clause's `output` and
  `next` apply.
- `default-routes` — no case matches the input: selection falls through to
  `default`, which routes, shapes, and captures like any clause, without a
  `when`.

## Checks

- `cases` is ordered and the first clause whose `when` holds is selected.
- Predicates read the shaped value as `match.input`.
- The selected clause's `output` shapes what its `next` receives.
- When no case's `when` holds, `default` is selected.

Reference: Step actions § Match.
