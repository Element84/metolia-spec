# 701 — The literal boundary

A string value containing a delimiter pair that does not span its entire
content. A value is an expression only when one delimiter pair sets off the
whole string; anything else is a literal, taken as written — including this one,
which begins with `{{` but carries a suffix.

## Checks

- A string is an expression only when the delimiter pair spans the entire value;
  a partial pair makes the value a literal.
- The literal passes through evaluation untouched, delimiters and all.

Reference: Expressions § (the embedding: values and expressions).
