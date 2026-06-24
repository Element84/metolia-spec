# 211 — Bare re-raise

A handler path that propagates what it caught: a `catch` clause routes the
failure to a bare `Raise`, which re-emits the frame's active failure unchanged.
No new failure is constructed and no `previous` link is added.

## Checks

- The matched failure remains the live failure context across the handler path.
- A bare `Raise` re-emits the active failure exactly: every envelope member
  survives, and the chain is untouched.

Reference: Step actions § Bare re-raise; Steps and step mechanics § catch
clauses.
