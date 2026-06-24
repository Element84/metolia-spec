# 501 — Parameter defaults and overlay

The subflow declares two parameters with defaults; the caller supplies one.
`vars` seeds with the schema's defaults overlaid by the validated arguments: the
supplied parameter binds the caller's value, the unsupplied one binds its
default.

## Checks

- A supplied parameter binds its validated value.
- An unsupplied parameter with a `default` binds that default.
- The two sources compose in one `vars` namespace at frame entry.

Reference: The Flow object § parameters, § The vars model.
