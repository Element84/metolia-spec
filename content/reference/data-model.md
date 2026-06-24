---
title: "The data model"
weight: 30
---

This section defines the MWL data model: the types behind every value on
[the data plane](../concepts/#the-data-plane). Its rules are of two kinds, and
their scopes differ. The type rules define what can be a value at all; they
bound everything that flows through a workflow, including payloads a workflow
merely moves between external services. The format conventions, such as the
temporal profile below, are narrower: they apply only where this specification
itself defines the meaning of a value, and they place no requirement on the
format of user data moving through the data plane.

## The JSON data model

All data flowing through a workflow, including Step inputs and outputs, variable
bindings, the `value` of a Result, and expression results, is JSON as defined by
[RFC 8259](https://datatracker.ietf.org/doc/html/rfc8259). Every value is typed
as null, boolean, string, number, array, or object. This spec requires strict
RFC 8259 conformance with a few additional rules:

- String values are Unicode per RFC 8259.
- String comparison (in expressions, in failure matching, in object keys) is
  over the Unicode code point sequence; the spec does not apply Unicode
  normalization, so two strings that differ only by normalization form are
  distinct.
- Object key order is not significant; implementations MAY preserve key order
  for serialization round-trips but MUST NOT depend on it for semantics.
- Numbers follow RFC 8259 syntax; the spec does not distinguish integers from
  floating-point or mandate a specific numeric precision or width.
- A number MUST be a finite RFC 8259 value. The non-finite IEEE 754 values
  (`NaN`, positive infinity, negative infinity) have no RFC 8259 representation
  and are therefore not values in the MWL data model. Negative zero is a finite
  RFC 8259 number and is a value in the model; the spec carries it as written
  but does not guarantee it is distinguished from positive zero, and an
  implementation MAY treat the two as equal.
- For interoperability, definitions SHOULD limit numbers to the range and
  precision of an IEEE 754 double-precision value, which RFC 8259 identifies as
  the interoperable subset; implementations MAY reject or lose precision on
  numbers outside it.
- The model has no temporal type: a duration or timestamp is a string. The
  formats this specification's own temporal fields use are a format convention,
  given in the [temporal profile](#temporal-format-profile) below.
- Values that cannot be represented as JSON (binary data, non-finite numbers,
  circular references, language-specific objects) are not part of the MWL data
  model.

External service integrations via providers receive and return JSON. Expressions
evaluate to JSON values. The data model stands independently of the expression
language; the embedding syntax and the expression profile are defined in
[Expressions](../expressions/), but every expression regardless of profile or
expression language must produce a value in one of the JSON types.

The data model is the set of RFC 8259 JSON types and nothing more. Certain
fields read added meaning into a string value, as the temporal profile below
does for durations and timestamps, but these are a small, fixed set of
conventions layered onto the string type, not new types in the model. A value
that has no RFC 8259 type, such as a non-finite floating-point number, cannot be
represented and so cannot appear in a workflow. A system whose data includes
such a value is responsible for encoding it within the JSON types.

## A single number type

JSON, and so the data model, has one number type. It does not separate integers
from floating-point values: `5` and `5.0` are the same value, and the model
neither records nor preserves a distinction between them. An author should not
rely on a value staying "an integer" or "a float" as it flows through a
workflow; it is simply a number.

The interoperability guidance above — that definitions SHOULD keep numbers
within the range and precision of an IEEE 754 double — has a concrete
consequence worth stating plainly. A value that must survive beyond that range
or precision, the canonical case being a 64-bit integer identifier larger than
`2^53`, cannot be carried faithfully as a number: an implementation MAY lose
precision on it. Such a value SHOULD be carried as a **string**. This is the
conventional treatment for large identifiers and the one most service APIs
already follow; it keeps the value exact and end-to-end, since the model carries
a string unchanged.

## Temporal format profile

This profile governs the values of fields this specification defines as
temporal, wherever those values arise: authored in a definition, produced by an
expression, or written by the engine itself, as the timestamp fields on frame
and Step metadata are. It places no constraint on user data; a timestamp inside
a payload that a workflow moves between services may use whatever representation
its producer and consumer agree on.

Durations (for example, a Retry policy's `backoff.initial` or a `Sleep` Step's
`for`) use ISO 8601 duration format (e.g., `PT30S`, `PT1H`, `P1D`). Timestamps
(for example, a `Sleep` Step's `until`, or the timestamp fields on frame and
Step metadata) use RFC 3339 format (e.g., `2024-01-15T09:30:00Z`).
Implementations MUST accept these formats; implementations MAY accept additional
temporal formats as an extension.

A duration field names an interval to elapse from some reference instant: a
`Sleep` Step's `for` from the Step's entry, a Timeout's `duration` from the
bound's establishment, a Retry `backoff` delay from the moment the delay begins.
A well-formed duration MAY be zero or negative; a zero or negative duration
names an interval that has already elapsed, and the field's effect is satisfied
immediately. This is the same tolerance a timestamp field grants an instant in
the past, where the named moment has already arrived: an interval of zero or
less is not an error to reject but a deadline already met, just as a `Sleep`
`until` in the past completes the Step at once. Engines MUST NOT reject a
well-formed duration for being zero or negative; a value that is not a valid
duration is a separate matter, failing validation as
[`System.ParameterValidationFailed`](../flow-object/#systemparametervalidationfailed)
on the syntax alone.
