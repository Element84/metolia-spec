# 020 — Sleep for a duration

A `Sleep` Step pauses the frame for one second, then transitions. `Sleep` is a
pure pause: it carries no shaping fields, and the value it received passes
through unchanged.

## Checks

- `for` accepts an ISO 8601 duration and the Step completes after it elapses.
- The value the Step received passes through to its successor unchanged.

Reference: Step actions § Sleep.
