# 030 — Root parameters

A root Flow declares two defaulted parameters. The platform starting the
execution is the root frame's caller: the arguments it supplies are validated
against the Flow's `parameters` schema at frame entry exactly as a subflow
caller's `with` would be, and `vars` seeds with the schema's defaults overlaid
by the validated arguments.

## Scenarios

- `defaults` — no arguments are supplied: both parameters bind their schema
  `default` values.
- `overlay` — one argument is supplied, the other is not: the supplied parameter
  binds the platform's value, the unsupplied one binds its default.
- `undeclared-argument` — the supplied arguments include a name matching no
  declared property: closed-by-default validation rejects it, the frame produces
  `System.ParameterValidationFailed` without running the Step graph, and as the
  root frame's Result it ends the execution. `details` is omitted from the
  expected envelope: its content is the implementation's.

## Checks

- The root frame's parameter handling is the subflow frame's: same validation,
  same defaults overlay, same `vars` seeding.
- A supplied argument binds its validated value; an unsupplied parameter with a
  `default` binds that default.
- An undeclared argument fails validation at frame entry, and the failure is the
  execution's Result directly.

Reference: The Flow object § parameters, § The vars model, §
System.ParameterValidationFailed; Execution model § The frame lifecycle.
