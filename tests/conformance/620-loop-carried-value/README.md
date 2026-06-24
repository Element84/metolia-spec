# 620 — Loop: the carried value

A bounded loop: the `Loop` entry's `onSuccess.when` continuation reads the
entry's `iteration` count, and the default carried value feeds each run's output
to the next run as its input. The mock appends one character per run, so three
runs turn the empty-string input into `"xxx"`.

## Checks

- `Loop`'s action owns every run; `onSuccess.when` is the continuation, and the
  entry emits when it is false.
- The carried value's default passes each iteration's produced value through as
  the next run's input.
- `middleware.metadata.iteration` counts runs, `1` on the first.

Reference: Middleware providers § The Loop middleware.
