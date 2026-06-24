# 500 — Calling a named subflow

The composition pattern: a named `flows` entry with a required parameter, called
with `with` arguments and the data payload on `input`. The subflow reads its
argument from its own `vars`, and the caller captures one of the completed
frame's variables through the `flow` window at the call boundary.

## Checks

- A `call` targets a named Flow; `with` arguments are validated against the
  subflow's `parameters` and seed its `vars`.
- The subflow's frame input is the call's evaluated `input` (here the
  passthrough default).
- The completed frame's variables are reachable only as `flow.vars.<name>` in
  the call's arms; the arm's `assign` carries the value out.

Reference: The Flow object § flows, § parameters; The Call interface and Result
§ The target windows.
