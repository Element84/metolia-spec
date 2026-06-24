# 202 — Catch matching on type, members as a conjunction

A mock dispatch fails with type `timeout`. The first `catch` clause's matcher
names a matching `codes` pattern but constrains `types` to `error`: every member
present must match, so the clause does not. The second clause matches the
failure's type alone and routes to the handler.

## Checks

- A matcher's `types` member matches when the failure's `type` is any of the
  named non-success types.
- A matcher matches only when every member present matches: a `codes` match
  alone does not satisfy a matcher whose `types` member does not match.

Reference: Steps and step mechanics § Failure matching, § catch clauses.
