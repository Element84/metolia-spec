# 021 — Sleep until a past instant

A `Sleep` whose `until` instant is already past completes the Step at once
rather than failing it, and the value passes through unchanged.

## Checks

- `until` accepts an RFC 3339 timestamp.
- A moment already reached completes the Step immediately; it is not an error.

Reference: Step actions § Sleep.
