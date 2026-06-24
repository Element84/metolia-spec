# 200 — Catch on an exact code

A dispatch fails with an authored code, and a `catch` clause matching that exact
code routes to a handler. The clause's `output` shapes what the handler
receives, replacing the failure path's passthrough default.

## Checks

- A `catch` clause matches the Step's failure Result by exact code.
- The matched clause's `output` shapes the value the handler Step receives.
- The handler completes the Flow successfully: a caught failure does not fail
  the frame.

Reference: Steps and step mechanics § Failures and catch, § catch clauses.
