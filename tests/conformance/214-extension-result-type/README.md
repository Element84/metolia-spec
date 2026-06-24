# 214 — An extension Result type

A dispatch fails with an extension type (`ProcessingError`, PascalCase by
convention). The type shares the envelope and the machinery: `catch` matches by
code regardless of type, and a bare `Raise` re-emits the envelope with its
extension type intact.

## Checks

- A non-success Result MAY carry an extension type; it is handled by the same
  matching and propagation machinery as the spec-defined types.
- `catch` matching is over the `code`; the failure path is selected by the
  Result being non-success, not by which non-success type it is.
- The extension type survives a bare re-raise unchanged.

Reference: The Call interface and Result § Result types; Steps and step
mechanics § Failure matching.
