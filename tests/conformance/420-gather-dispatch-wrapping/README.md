# 420 — Gather: wrapping a dispatch with a flow target

A `Gather` carries no `middleware`: a dispatch that needs wrapped behavior
targets a Flow and carries the wrapper inside it. Here the call template is an
inline Flow whose `Call` Step carries a `Retry` stack, so every element's
processing retries independently within its own frame. One element always fails,
exhausts its dispatch's own budget, and the frame's
`Provider.Middleware.Retry.Exhausted` Result is what `completion` counts and
what the `System.GatherCompletionUnmet` evidence carries.

## Checks

- A need for middleware around a dispatch is a need for a frame around it: the
  inline flow target carries the stack, and the element flows through on the
  default passthroughs with no extra plumbing.
- Each dispatch's retry is its own; a dispatch's Result is its frame's one
  emerging Result.
- The `GatherCompletionUnmet` evidence carries the exhausted envelope,
  `previous` chain intact.

Reference: Step actions § Wrapping a dispatch: flows, not middleware; Middleware
providers § The Retry middleware.
