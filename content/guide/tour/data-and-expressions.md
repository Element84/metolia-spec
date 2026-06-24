---
title: "Data and expressions"
weight: 20
---

Every transition in a Flow hands one JSON value from a Step to its successor.
That stream of values — Step inputs and outputs, the value a `Return` emits, the
`value` a Result carries — is the **data plane**. This page shows how to observe
and shape it.

## Expressions

A string value in the definition is one of exactly two things: a literal, used
as written, or an **expression**, when one `{{ ... }}` pair spans the whole
value:

```json
"path": "/granules",
"collection": "{{ vars.collection }}"
```

The text inside the delimiters is a [CEL](https://cel.dev/) expression,
evaluated at runtime against the live execution state. The result has whatever
type the expression computes; the surrounding quotes are JSON syntax for an
expression-bearing value, not a claim that the result is a string. If
`vars.cacheTTL` is a number, `"ttl": "{{ vars.cacheTTL }}"` yields a number.

There is no in-string templating: an expression always occupies an entire value,
and building a string from parts is the expression's own work,
`"{{ '/granules/' + vars.collection }}"`. Object- and array-valued fields may
carry expressions at any string leaf, each leaf independently a literal or a
whole-value expression.

Expressions read a small set of **binding roots**, plain names with no sigil.
Three are in scope everywhere: `vars` (the Flow's variables), `frame` (the
current frame's input and metadata), and `execution` (the run's identity and
timing). Others appear where their construct does: `step` inside a Step, `call`
inside a call object, `match` in a `Match` clause, `middleware` in a middleware
phase, `failure` while a failure is being handled. Each page of this tour
introduces the roots its constructs bring.

## Shaping with `input` and `output`

Two Step fields shape the data crossing its boundary. `input` produces the value
the action consumes; `output` produces the value the Step emits on success. Both
default to passthrough: an absent `input` delivers the received value unchanged,
and an absent `output` emits the action's product unchanged (for a `Call`, the
value its Result carries).

```json
"fetch-items": {
  "action": "Call",
  "input": "{{ step.input.order }}",
  "call": {
    "provider": "mwl:provider.call/example/http/v1",
    "with": { "method": "GET", "path": "/items" }
  },
  "output": "{{ {'order': step.input, 'items': step.result.value.items} }}",
  "next": "process"
}
```

Here `input` narrows the incoming value to its `order` member before the call
machinery sees it, and `output` builds a new object pairing the received value
with part of the response. Note the two reads: `step.input` is always the value
the Step _received_, untouched by `input` shaping, and `step.result.value` is
the value the Call's Result carried. A Step never mutates anything; each field
produces a new value, and the bindings keep the originals readable.

When a Step exists only to reshape data, use `Pass`, the action that does
nothing but run its shaping fields:

```json
"wrap": {
  "action": "Pass",
  "output": "{{ {'features': step.input} }}",
  "next": "register"
}
```

## A few CEL notes

CEL is a small, deliberately bounded language: navigation
(`step.input.items[0].id`), operators (`+`, `==`, `&&`, `? :`), macros
(`filter`, `map`, `has()`), and a standard library, with no loops and no side
effects. Three things to know early:

- **Data-plane numbers are doubles.** Every number read through a binding
  arrives as a CEL `double`, and CEL does not mix numeric types in arithmetic:
  `1 + vars.ratio` is an error because `1` is an `int`. Write
  `1.0 +
  vars.ratio`, and prefer double literals (`1000.0`) when comparing
  against data. Equality is the exception: `vars.count == 1` works across
  numeric types.
- **Absent members fault.** Reading a key that isn't there is an evaluation
  error, not a null. Guard uncertain shapes with `has()` and the ternary:
  `{{ has(step.input.label) ? step.input.label : 'unlabeled' }}`.
- **Serialization is explicit.** Turning structured data into a string is
  `toJson(value)`; parsing is `fromJson(string)`. Nothing coerces values to text
  behind your back.

An expression that does fault produces a structured failure
(`System.ExpressionEvaluationError`) that routes like any other failure, which
is the subject of [Handling failures](../handling-failures/).

## Where the spec covers this

- [Expressions](/reference/expressions/) — the embedding, the evaluation
  contract, and the full CEL profile, including the number rules.
- [The data model](/reference/data-model/) — what values are: strict RFC 8259
  JSON, one number type, the temporal formats.
- [Data flow: `input` and `output`](/reference/step-mechanics/#data-flow-input-and-output)
  — the shaping fields and their defaults.
- [Data flow](/reference/data-flow/) — the end-to-end synthesis of how values
  move.
