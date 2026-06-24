# 641 — Middleware: stacked onFailure translations

Two entries carry `onFailure` author shaping around a failing dispatch. The
failure ascends: the inner entry's block constructs a successor first, then the
outer entry's constructs another from that — the outermost entry has the final
word. Each construction chains what it superseded, and each unwritten envelope
field is taken from the superseded failure, so the original `message` survives
two translations and the inner translation's `details` reaches the top.

The carriers are `Timeout` entries whose bounds never fire: author shaping
belongs to the phase block, whatever the middleware's action.

## Checks

- Ascent runs phases in reverse array order; the outer translation supersedes
  the inner one.
- Writing any envelope field constructs a new failure; unwritten fields inherit
  from the superseded failure.
- Each supersession chains via `previous`: the Result carries all three
  failures, newest first.

Reference: Middleware mechanics § The stack: ordering and composition, §
onFailure.
