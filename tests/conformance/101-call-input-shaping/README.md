# 101 — The input channel

The data channel: the call's `input` expression shapes the payload the target
receives, and a bare `mock` echoes it back as its success value. The shaped
payload, not the raw execution input, is what returns.

## Checks

- `input` is a data channel separate from `with`; its expression shapes what the
  target receives from the inbound payload (`call.input`).
- A bare `mock` (no `value`, no `failure`) echoes the dispatch's input.

Reference: The Call interface and Result § The three axes; Call providers § The
mock provider.
