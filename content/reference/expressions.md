---
title: "Expressions"
weight: 40
---

Many fields in a workflow definition carry values computed at runtime rather
than written as literals — data transformations, predicates, variable bindings,
and dynamic configuration. These fields hold an _expression_: a small program,
written in an expression language, that the platform evaluates against the
current execution context to produce a value.

An expression is an MWL concept; the expression _language_ sits on top of it.
MWL defines a language-agnostic _embedding_ — how an expression is written into
the JSON document and what its result means — together with an _evaluation
contract_: the bindings an expression is evaluated against, how its result type
relates to the field it occupies, and what happens when evaluation fails. An
expression language plugs into that contract, supplying the concrete syntax,
types, and operators. A language is paired with its own _delimiter pair_, and
the delimiter is what identifies the language for a given value — so more than
one language could be supported at once, each value declaring its language by
the delimiters that enclose it. This is a pluggability seam, parallel to the way
[providers](../providers/) are the seam for call targets: an additional language
could be added by specifying its delimiters and writing its profile subsection,
with no change to the contract or to any section that uses expressions.

This specification version defines exactly one language:
[CEL](#the-cel-profile), the Common Expression Language, enclosed in `{{ }}`.
Supporting an expression language is not required of a conforming
implementation, and no specific language is mandated; an implementation MAY
support CEL, another language, or none. CEL is the language this version
standardizes, however, and an implementation SHOULD support it, so that
workflows written against it are portable. A definition that embeds an
expression in a language an implementation does not support is outside what that
implementation can run: the implementation MUST reject it rather than read the
expression string as a literal. Which languages an implementation supports is
stated by its conformance claim (see [Conformance](../conformance/)). Everything
in this section _outside_ the CEL profile subsection is the language-agnostic
contract; the examples throughout use concrete CEL because it is the only
language this version defines, but the rules they illustrate are properties of
the contract, not of CEL.

The same reading applies to defaults. Where this specification gives the default
of an expression-bearing field as an expression, such as `{{ step.input }}` or
`{{ call.result.value }}`, the default is defined by the behavior that
expression denotes; the expression form states the behavior precisely and is
what an author would write to restate it explicitly. In all but one case that
behavior is a passthrough — the value of the binding path the expression names,
unchanged. The one exception is the `Gather` `output` default, a defined
projection over the Step's collected Results (see
[Step actions](../step-actions/)). Either way, the expression is notation for
what the platform does, not a dependency on it doing evaluation: the engine
realizes every default natively, and no default requires an expression
evaluator.

## The embedding

A string value in the document is one of exactly two things: a _literal_, used
as written, or a single _expression_. It is an expression when a delimiter pair
sets off its entire content — the value begins with an opening delimiter and
ends with the matching closing delimiter, with that one pair spanning the whole
value. Any other string is a literal. An expression therefore always occupies an
entire value; an expression is never embedded within surrounding text.

The text between the delimiters is the expression body, evaluated using the
language that the delimiter pair identifies. This version defines one delimiter
pair:

| Opening | Closing | Language                | Conformance profile                                |
| ------- | ------- | ----------------------- | -------------------------------------------------- |
| `{{`    | `}}`    | [CEL](#the-cel-profile) | `https://mwl.dev/v0.1/conformance/expressions/cel` |

```json
"path": "/granules",
"collection": "{{ vars.collection }}"
```

Here `"/granules"` is a literal and `"{{ vars.collection }}"` is an expression.
The delimiter pair is the sole mechanism by which an implementation identifies
the expression language for a given value. An additional language introduced by
a later version or an extension specification would carry its own distinct
delimiter pair — a new row in this table — so that every expression's language
is unambiguous from its delimiters alone, without disturbing the rules described
here.

Whitespace between the delimiters and the expression body is not significant.
The delimiters do not nest: an expression body is handed to its language as-is,
and all further composition is the language's own syntax.

### The produced value

The value an expression produces has whatever JSON type the expression computes:
null, a number, a string, a boolean, an array, or an object. That result, with
its type, becomes the field's value. The result type is the _expression's_, not
the string's, even though the expression is written inside a JSON string in the
source document. A field written as a string in the document may therefore carry
a value of any type at runtime:

```json
"ttl": "{{ vars.cacheTTL }}"
```

If `vars.cacheTTL` is null, the field's runtime value is null, not the string
`"null"`; if it is a number, the field's value is that number. The surrounding
quotes are the JSON document's syntax for an expression-bearing value, not an
assertion that the result is a string.

A field's type may be known in two ways:

- The spec fixes it, for a field whose type this specification defines.
- A schema constrains it, the usual source being a Flow's or a Provider's
  `parameters`.

Where the type is known, the result must be of that type. The result is not
coerced to fit the field, so an expression that produces the wrong type is a
validation failure rather than a silent conversion, as
[Evaluation errors](#evaluation-errors) describes. Where the type is unknown,
neither fixed by the spec nor constrained by a schema, the field accepts any
value the expression produces.

Because an expression spans the whole value, composing a string from parts is
the work of the expression itself, using the language's own string operations
rather than any document-level templating:

```json
"path": "{{ '/granules/' + vars.collection }}"
```

This expression concatenates a literal and a binding with CEL's `+` operator;
its result is a string, so the field's value is a string. Expression languages
provide functions and operators for this kind of computation — building and
measuring strings, arithmetic, constructing objects and arrays — beyond plain
navigation; what they offer, and how it is written, is a property of the
language in use (for CEL, see [the CEL profile](#the-cel-profile)).

> [!NOTE]
> Turning structured data into a string is a distinct, explicit operation
>
> Some targets accept a string where the data at hand is an object or array — an
> HTTP request body with no media type to interpret it, or a notification
> `message` built from structured context. Producing a string from structured
> data is _serialization_, not the implicit value-to-text coercion this
> specification avoids: it is an explicit operation the author writes into the
> expression, not a conversion the embedding performs. The sanctioned mechanism
> is a JSON-serialization capability the expression profile provides, kept on
> the expression side of the
> [expression-provider boundary](#the-expression-provider-boundary) because it
> is pure, deterministic shaping. For CEL it is the
> [`toJson`/`fromJson`](#mwl-functions) functions.

### Expressions in object and array fields

A field whose value is a JSON object or array may hold expressions at any string
leaf. Each leaf is independently a literal or a whole-value expression, under
the same rule, and an expression leaf contributes its typed result:

```json
"with": {
  "method": "POST",
  "collection": "{{ vars.collection }}",
  "resources": { "cpu": 16, "memoryMB": 31000 }
}
```

## Where expressions may appear

Many value-carrying fields accept an expression in place of a literal. Some of
these field names recur across the specification, carrying an expression at
every site they appear but evaluated against whatever bindings are in scope
there:

- `output` — the value a construct emits (a Step's normal exit, a `catch`
  clause, a `Match` clause, a middleware `onEntry` phase).
- `assign` — values captured into the frame's variables, available beside every
  shaping field above except on the terminal actions (a Step, a clause, a
  middleware phase, a Call's arms).
- `input` — the data a construct works on (a Step, a `Call`).
- `with` — the configuration passed to a call target or a middleware phase. Its
  fields each accept an expression, or one whole-value expression produces the
  entire object.
- `value` — the value placed in a success Result (a Call's `onSuccess` arm, a
  `Return`, a middleware `onSuccess` phase).
- `when` — the predicate gating a `Match` clause or a middleware phase.

Others are specific to a single site:

- `over` — the collection a `Gather` iterates, in its iterate form.
- `for` — the duration a `Sleep` waits.
- `until` — the timestamp a `Sleep` waits until.

Each section's field documentation is the authoritative account of which of its
fields accept an expression and what bindings that expression is evaluated
against; consult it for any specific field.

Discriminators and static identifiers do not accept expressions — a field such
as `action`, `type`, `provider`, `next`, or a Step name is resolved from the
definition alone, before any execution context exists to evaluate against. An
implementation MUST reject an embedded expression in such a field.

## Absent fields and passthrough

A field that accepts an expression has a defined value when it is absent, not
merely when it is present. For the data-flow fields — those that shape what a
construct receives or emits — an absent field means _passthrough_: the
construct's data flows through unchanged, exactly as though the field held an
expression returning that data as-is. Omitting a Step's `output` is therefore
not "emit nothing"; it is "emit the result unchanged." Each such field's
passthrough value — what flows when it is absent, and the value an expression
shapes when it is present — is a property of the field, given with that field's
documentation and consolidated in [Execution context](../execution-context/).
This section establishes only that the absent case is defined and defaults to
passthrough; it does not enumerate per-field defaults.

## Evaluation context: the binding roots

An expression is evaluated against a set of named _bindings_ — the live data the
platform exposes to it. Each binding is reached by a bare root name; this
specification uses no sigil or prefix on binding names.

The binding _roots_ — the top-level names and what each holds — are listed
below. The members under each root and their runtime semantics are detailed in
[Execution context](../execution-context/), the reference for the runtime data
model.

| Root         | What it holds                                                                       | Defined in                                                      |
| ------------ | ----------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `vars`       | The frame's variables: declared parameters and assigned values.                     | [The Flow object](../flow-object/), [Steps](../step-mechanics/) |
| `execution`  | The execution: identity, timing, and the platform surface.                          | [Execution context](../execution-context/)                      |
| `frame`      | The current frame: its input and execution metadata.                                | [Execution context](../execution-context/)                      |
| `step`       | The currently executing Step: its input, result or collected Results, and metadata. | [Steps and step mechanics](../step-mechanics/)                  |
| `call`       | The current Call: its data payload, result, and metadata.                           | [The Call interface and Result](../call-interface/)             |
| `flow`       | A flow target's completed frame, in the call's arms.                                | [The Call interface and Result](../call-interface/)             |
| `provider`   | A provider target's completed execution, in the call's arms.                        | [The Call interface and Result](../call-interface/)             |
| `match`      | The data handed to the current `Match`.                                             | [Step actions](../step-actions/)                                |
| `middleware` | The input and result at the current middleware position.                            | [Middleware mechanics](../middleware-mechanics/)                |
| `failure`    | The live failure context: the failure envelope being handled.                       | [Execution context](../execution-context/)                      |

Which roots are in scope depends on where the expression appears: a Call's
`with` sees `call`, a call's arms see its target's window (`flow` or
`provider`), a middleware phase sees `middleware`, a `Match` clause sees
`match`, and a `catch` clause's expressions see `failure`. Each field's own
documentation states the bindings available to it.

## Evaluation errors

When an expression cannot be evaluated — a type error, a reference to an absent
value where a value is required, or any other runtime fault — the evaluation
produces a non-success Result of type `error` with the code
`System.ExpressionEvaluationError`. The Result type and its envelope are defined
in [The Call interface and Result](../call-interface/); this code is listed in
the [Failure code reference](../failure-code-reference/). The failure follows
the same propagation machinery as any other non-success Result.

An evaluation error is not intended to be a recovery mechanism. Expression
profiles provide defensive constructs — presence checks, guarded access,
default-on-absent forms, short-circuit boolean operators — that prevent
evaluation errors before they arise; these are the recommended way to handle
expected absence or variation in data shape. A `catch` clause MAY match
`System.ExpressionEvaluationError`, because it flows through the same matching
machinery as any other failure, but doing so is discouraged: an evaluation error
generally signals an authoring bug or a data-shape assumption that did not hold,
better addressed by hardening the expression or validating data upstream than by
catching the error.

Two further cases are distinct from a failure to evaluate, because in each the
expression evaluates _successfully_ and the problem is with the value it yields.
Neither is a `System.ExpressionEvaluationError`:

- A result that is a valid value but fails a schema constraint, including a
  value of the wrong type for a field whose type is known, produces
  `System.ParameterValidationFailed`. A `3` where a string is required is such a
  case. The validation surfaces that raise it are defined in
  [The Flow object](../flow-object/).
- A result with no faithful representation in the [data model](../data-model/)
  produces `System.UnrepresentableValue`. This covers a value that is not any of
  the JSON types, such as a non-finite number, and a value that is nominally a
  number but carries magnitude or precision the data model cannot preserve, such
  as an integer beyond the range a double represents exactly. In every case the
  result could not faithfully be a field value of any type, so it is not a
  schema failure; it arises from the expression language's own type system
  reaching beyond what the data model carries. The profile for a language states
  which of its types can produce this, and how an author converts such a value
  explicitly to avoid it.

Both codes are listed in the
[Failure code reference](../failure-code-reference/).

## Predicates and `when`

An expression used as a predicate is evaluated for a boolean result. Expression
languages evaluate truthiness in various ways, and a value that one language
treats as true another may reject or treat as false. Authors are therefore
encouraged to write predicates that return an explicit boolean, to avert
ambiguity arising from differences in how languages interpret a non-boolean
result. The [CEL profile](#truthiness) states how CEL predicates in particular
are interpreted as booleans.

The recurring predicate field is `when`. The same field name carries a predicate
at two sites:

- A `Match` clause's `when` selects whether that clause matches, defined in
  [Step actions](../step-actions/).
- A middleware phase's `when` conditions whether that phase's middleware action
  runs, defined in [Middleware mechanics](../middleware-mechanics/).

Both uses share one contract: `when` holds an expression evaluated for a
boolean. What each `when` gates, the bindings it sees, and its default value are
properties of the site, given in the sections linked above. A `when` that fails
to evaluate is an evaluation error like any other (see
[Evaluation errors](#evaluation-errors)); failure is not absorbed into a
non-match.

## The CEL profile

The Common Expression Language ([CEL](https://cel.dev/)) is the expression
profile this specification version defines, and the one it recommends an
implementation support. CEL is associated with the `{{ }}` embedding described
above. The material in this subsection is CEL-specific; everything outside it is
the language-agnostic contract that any profile satisfies.

### Conformance profile

CEL support is claimed as the conformance profile
`https://mwl.dev/v0.1/conformance/expressions/cel` (see
[Conformance](../conformance/)). An implementation that advertises CEL support
MUST support CEL as defined by the CEL language specification: its standard
syntax, type system, macros, operators, and built-in functions. MWL defines no
reduced subset of the language. On top of core CEL, MWL adds a small set of
functions defined in [MWL functions](#mwl-functions) below, which such an
implementation MUST also provide. Core CEL together with these functions is the
floor for CEL support: a workflow that uses only them is portable across every
implementation that advertises CEL. The
[recommended extensions](#recommended-extensions) that follow sit above this
floor and are encouraged but not required.

### Recommended extensions

Some operations recur in workflow authoring that core CEL cannot express on its
own — splitting and formatting strings, range and set operations over lists,
encoding a small value. These are all pure, deterministic data shaping, and so
belong on the expression side of the
[expression-provider boundary](#the-expression-provider-boundary) rather than
behind a provider Call. CEL covers them through _extension libraries_ layered
onto the core language. To keep authors from each reaching for a different
mechanism for the same need, this specification RECOMMENDS that a conforming
platform provide the following capabilities. An implementation SHOULD provide
them; a workflow that relies on one is portable across implementations that
follow the recommendation.

The recommendation is stated against the
[cel-go `ext` package](https://pkg.go.dev/github.com/google/cel-go/ext), the
reference realization named in the third column; each capability is
independently claimable by the conformance profile in the fourth (see
[Conformance](../conformance/)):

| Capability          | Operations                                                      | cel-go extension | Conformance profile                                         |
| ------------------- | --------------------------------------------------------------- | ---------------- | ----------------------------------------------------------- |
| String manipulation | `split`, `join`, `format`, `substring`, case folding, trimming. | `strings`        | `https://mwl.dev/v0.1/conformance/expressions/cel/strings`  |
| List manipulation   | range generation, slicing, flattening, sorting, distinct.       | `lists`          | `https://mwl.dev/v0.1/conformance/expressions/cel/lists`    |
| Set operations      | membership, equivalence, and intersection over lists.           | `sets`           | `https://mwl.dev/v0.1/conformance/expressions/cel/sets`     |
| Encoding            | base64 encode/decode.                                           | `encoders`       | `https://mwl.dev/v0.1/conformance/expressions/cel/encoders` |
| Numeric aggregates  | greatest/least and related reductions over collections.         | `math`           | `https://mwl.dev/v0.1/conformance/expressions/cel/math`     |

> [!NOTE]
> The CEL extension story is still being formalized upstream
>
> Extension libraries are layered onto core CEL and are not yet uniformly part
> of the CEL specification: at present only string manipulation is defined as a
> specification extension
> ([cel-spec `doc/extensions/strings.md`](https://github.com/google/cel-spec/blob/master/doc/extensions/strings.md)).
> The others exist as implementation libraries. This specification names the
> [cel-go `ext`](https://pkg.go.dev/github.com/google/cel-go/ext) extensions as
> the concrete recommendation because formalization upstream will inevitably
> follow the same boundaries; the operation names and behavior follow cel-go
> until the CEL specification formalizes them. Another conforming CEL
> implementation may package or name the same capabilities differently; a claim
> of one of these extension profiles binds the operations that extension
> defines, with the semantics cel-go documents, however the implementation
> packages them.

The recommendation is a floor for _portability_, not a license for unbounded
computation. Any operation that is nondeterministic or side-effecting remains a
provider concern regardless of what an extension makes syntactically possible —
see [the expression-provider boundary](#the-expression-provider-boundary). An
implementation MAY provide further extensions beyond those recommended here; a
workflow that relies on one is portable only across implementations that also
provide it.

### Binding access

A CEL expression reaches the
[binding roots](#evaluation-context-the-binding-roots) as ordinary identifiers
and reads their members with the dot and index operators:

```json
"items": "{{ middleware.result.value.features }}",
"over": "{{ step.input.features }}",
"replace": "{{ vars.replace == false }}"
```

The roots in scope at a given site, and the members beneath each root, are as
enumerated above and detailed in [Execution context](../execution-context/).

### Result values

A CEL expression's result becomes a [data model](../data-model/) value, so its
CEL type is read as a JSON type. CEL's type system is wider than the JSON types,
and the mapping passes through only those CEL types that have a single, lossless
JSON form. The rest have none; rather than impose an encoding, MWL leaves the
choice to the author, who converts such a value explicitly with the function
shown before it becomes a result:

| CEL type                | Data model value | Conversion to a data model value                   |
| ----------------------- | ---------------- | -------------------------------------------------- |
| `bool`                  | boolean          | direct                                             |
| `int`, `uint` (≤ 2^53)  | number           | direct                                             |
| `int`, `uint` (\> 2^53) | —                | `double(n)` (lossy) or `string(n)` (exact)         |
| `double` (finite)       | number           | direct                                             |
| `double` (non-finite)   | —                | not representable                                  |
| `string`                | string           | direct                                             |
| `bytes`                 | —                | `base64.encode(b)` → string                        |
| `list`                  | array            | direct                                             |
| `map` with string keys  | object           | direct                                             |
| `map`, other key types  | —                | build the `map` with string keys                   |
| `null`                  | null             | direct                                             |
| `timestamp`             | —                | `string(t)` → RFC 3339 string                      |
| `duration`              | —                | `string(d)` → seconds string (`300s`), or          |
|                         |                  | `durationToIso8601(d)` → ISO 8601 string (`PT30S`) |

A row marked — has no data model value; such a result raises
`System.UnrepresentableValue` unless the expression converts it first using the
conversion shown. A non-finite `double`, the result of dividing by zero for
instance, has no conversion: it is not a value in the data model. The `string`
function and the `base64` functions are core CEL and the `encoders` extension;
`durationToIso8601` is one of the [MWL functions](#mwl-functions) below, which
produces the duration string MWL's own duration-typed fields expect.

### Numbers

CEL's type system separates numbers into `int` (signed 64-bit), `uint` (unsigned
64-bit), and `double` (IEEE 754), where the
[data model](../data-model/#a-single-number-type) has a single number type. MWL
bridges the two so that the common case — arithmetic over the numbers in a
workflow's data — needs no ceremony, while making the one place friction remains
explicit.

A JSON number from [the data plane](../concepts/#the-data-plane) enters CEL as a
`double`. Every number reached through a binding (`vars.count`,
`step.input.limit`, an element of a list in `call.input`) is a `double`,
regardless of whether it was written with a fractional part. This matches the
data model's single number type: an author does not have to track whether a
given value arrived as `5` or `5.0`, because both are the same `double`, and
arithmetic among data-plane numbers composes without casts.
`vars.total / vars.count` and `vars.price * vars.quantity` are all `double`
operations and just work.

The bridge is asymmetric by design: every data-plane number comes _in_ as a
`double`, but any of CEL's numeric types — `int`, `uint`, or `double` — may go
_out_ and become a result, since the standard library and arithmetic can produce
each. This leads to the one residual edge an author should know.

> [!NOTE]
> CEL's standard library returns `int`, and CEL does not mix numeric types
>
> A few CEL operations yield `int` rather than `double`: `size()` returns an
> `int`, list indices and string positions are `int`, and an integer literal
> such as `1` is an `int`. CEL performs no implicit conversion among `int`,
> `uint`, and `double` — `1 + 2.0` does not dispatch — so combining one of these
> `int` values with a data-plane `double` requires an explicit conversion. An
> integer literal added to a data-plane number is the case authors meet first:
> `1 + vars.ratio` does not dispatch, because `1` is an `int` and `vars.ratio`
> is a `double`. Write the literal as a `double`, `1.0 + vars.ratio` or
> `double(1) + vars.ratio`; or, where the number is known to be integral or
> truncation toward zero is intended, convert the binding instead,
> `1 + int(vars.ratio)`. Two forms cover nearly every case:
>
> - Use a `double` in arithmetic with a data-plane number:
>   `double(size(items)) + vars.ratio`.
> - Use an `int` where CEL expects one, such as a list index built from data:
>   `items[int(vars.offset)]`.
>
> Equality is the exception and needs no cast: `vars.count == 1` compares across
> numeric types and is `true` for a `vars.count` of `1`. And because data-plane
> numbers are `double`, integer division does not arise from them; it appears
> only between values that are themselves `int` (two literals, or stdlib
> results), where CEL's `/` truncates — `5 / 2` is `2`, not `2.5`.
>
> Division by zero splits along the same `int`/`double` line. Integer division
> by zero, `5 / 0`, is a CEL evaluation error: it raises
> `System.ExpressionEvaluationError` and the expression never produces a value.
> Double division by zero, `5.0 / 0.0`, follows IEEE 754 and _succeeds_,
> yielding `+Inf` — a value with no data model representation, which then raises
> `System.UnrepresentableValue` when the result is marshalled, per
> [Evaluation errors](#evaluation-errors). The two reach different codes because
> one is a failure to evaluate and the other is a successful evaluation of an
> unrepresentable value.

A CEL `int` or `uint` is a 64-bit integer, so it can hold a whole value beyond
`2^53`, the point past which an IEEE 754 double can no longer represent every
integer exactly. Such a result has magnitude the data model cannot preserve, so
it raises `System.UnrepresentableValue` per
[Evaluation errors](#evaluation-errors) rather than being emitted as a lossy
number. The author converts explicitly, choosing what to trade — `double(n)` to
accept the precision loss deliberately, or `string(n)` to carry the value
exactly as text, the conventional treatment for a 64-bit identifier.

The boundary is asymmetric about precision, and deliberately so. Inbound,
conversion cannot fail: a data-plane number beyond `2^53` becomes the nearest
`double` silently. Such a number exists only where an implementation chose to
preserve it, because the [data model](../data-model/#a-single-number-type) caps
its interoperability guarantee at the double range and permits loss beyond it;
reading the value subjects it to no loss it was not already subject to.
Outbound, an `int` or `uint` beyond `2^53` raises rather than rounds, because
the exact value is still in hand and the author can still choose what to trade;
silent rounding here would manufacture a loss at the only point where it could
have been prevented.

A finite `double` is the model's number form and is emitted directly; a value
beyond the finite double range is not a finite `double` but `+Inf` or `-Inf`,
which raises as the non-finite case below. The numeric results an author can
produce, and what becomes of each:

| Result                                           | Becomes                                    |
| ------------------------------------------------ | ------------------------------------------ |
| `int`/`uint` with magnitude ≤ 2^53               | a JSON number, exact                       |
| `int`/`uint` with magnitude \> 2^53              | `System.UnrepresentableValue` — cast first |
| finite `double`, any magnitude                   | a JSON number                              |
| non-finite `double` (`NaN`, `±Inf`, `5.0 / 0.0`) | `System.UnrepresentableValue`              |

### MWL functions

Beyond core CEL and the recommended extensions, MWL defines a small set of
functions that a CEL implementation MUST provide to conform. Most are pure and
deterministic, fitting the
[expression-provider boundary](#the-expression-provider-boundary); the
conversion functions come as pairs that convert in both directions between a CEL
value and a data model value. The two clock functions are the exception to
purity and carry their own rules, given below.

The first pair serializes a value to and from a JSON string. A target that
expects a string built from structured data, an HTTP body with no media type or
a notification message assembled from context, needs the data turned into text;
this is the serialization [the embedding](#the-embedding) leaves to the author.

- `toJson(value)` returns the canonical JSON string for any data model value:
  its [RFC 8785](https://datatracker.ietf.org/doc/html/rfc8785) (JSON
  Canonicalization Scheme) serialization. The form is compact, object members
  are sorted by key, and the number and string renderings are fixed by the
  scheme, so equal values yield byte-identical strings, which matters where the
  string is used as a cache key, a signature, or a fingerprint.
- `fromJson(string)` parses a JSON string and returns the data model value it
  encodes.

```json
"body": "{{ toJson(step.result.value) }}"
```

The second pair bridges CEL's `duration` type and the ISO 8601 duration string
the [temporal profile](../data-model/#temporal-format-profile) uses. CEL's own
`duration()` and `string()` read and write the protobuf seconds form (`300s`),
not ISO 8601, so these functions cover the form MWL's duration-typed fields
expect.

- `durationToIso8601(duration)` returns the canonical ISO 8601 string for a
  `duration`: hours, minutes, then seconds (`PT1H30M`), where hours are the
  largest unit used (`PT26H`), zero-valued components are omitted, a fractional
  part appears on the seconds component (`PT0.5S`), and the zero duration is
  `PT0S`. A negative duration takes a leading minus (`-PT30S`), the ISO 8601-2
  extension.
- `durationFromIso8601(string)` parses an ISO 8601 duration string into a
  `duration`, accepting any valid duration form, not only the canonical one,
  including the leading-minus negative form.

A duration carried through the workflow as an ISO 8601 string, a workflow
parameter passed to a `Sleep`, needs no conversion: it is already a string and
flows through unchanged. These functions are for computing on a duration, where
`durationFromIso8601` parses the string into a `duration` to operate on and
`durationToIso8601` renders the result back. CEL's `timestamp` type needs no
such pair, as CEL's `timestamp()` and `string()` already use RFC 3339, the form
the temporal profile uses for timestamps.

Two functions read the current time. CEL has no clock access of its own, so MWL
supplies both, each returning a CEL `timestamp` and differing only in their
stability:

- `now()` returns a `timestamp` of the time at which the current construct
  execution was entered (the constructs and their executions are defined in
  [Expression evaluation timing](../execution-model/#expression-evaluation-timing)).
  Every `now()` evaluated within one construct execution returns that same
  instant, so an expression can compute on it and reason about it: two `now()`
  calls in the same construct agree, and `now()` in a Step's `assign` matches
  `now()` in the same Step's `output`.
- `wallTime()` returns a `timestamp` of the actual current time, read fresh at
  each evaluation. Two `wallTime()` calls need not agree even within one
  expression.

> [!IMPORTANT]
> `now()` is stable within a construct execution, not across re-execution
>
> A construct that re-runs its inner scope, like a `Retry` attempt or a `Loop`
> iteration, is a new execution, so `now()` takes a fresh pin for each attempt:
> stable across the expressions of one attempt, but advancing from one attempt
> to the next. A value that must stay fixed across attempts is captured into
> `vars` on first evaluation and read from the variable thereafter, following
> the general rule for nondeterministic values defined in
> [the execution model](../execution-model/). `wallTime()` is never stable; it
> is for the rare case that genuinely needs the wall-clock reading at the moment
> of evaluation, and a value derived from it that must persist is likewise
> captured into `vars`.

### Defensive constructs

The defensive constructs that [Evaluation errors](#evaluation-errors) recommends
are realized in CEL as the `has()` macro for presence testing, the ternary
conditional `cond ? a : b` for default-on-absent, and the short-circuit boolean
operators `&&` and `||`. All are part of core CEL.

### Truthiness

A CEL predicate evaluates to a boolean directly; CEL does not coerce arbitrary
values to boolean the way some languages do. An expression used where a boolean
is expected SHOULD therefore evaluate to a `bool` — a comparison, a
boolean-typed binding, or an explicit boolean built from them. A non-boolean
result at a predicate site is an evaluation error, handled per
[Evaluation errors](#evaluation-errors).

### The expression-provider boundary

CEL computes values; it does not perform side effects. This draws a clean line
between work that belongs in an expression and work that belongs in a
[provider](../providers/) Call:

- **Pure, deterministic data shaping** — selecting, comparing, arithmetic,
  constructing objects and arrays, building and splitting strings, encoding a
  value — is an expression, written inline at the call site with no provider
  involved. Core CEL covers most of it; the
  [recommended extensions](#recommended-extensions) cover the rest.
- **Nondeterministic** work, such as generating a UUID or drawing a random
  number, is a provider concern, not an expression. An expression is evaluated
  afresh each time its containing construct runs, so a nondeterministic
  expression would yield a different value on each evaluation; a value that must
  persist is produced by a Call and captured into `vars`. Reading the clock is
  the one nondeterministic operation MWL exposes as an expression, through
  [`now()` and `wallTime()`](#mwl-functions), because a timestamp is needed too
  pervasively to route through a provider; a clock value that must be stable
  across re-evaluation is still captured into `vars` like any other.
- **Side-effecting** work — anything that touches the outside world, such as
  fetching or storing data — is a provider Call, even where an extension makes
  some part of it syntactically expressible.

The line is drawn by the _nature_ of the operation, not the size of its data: a
pure transform stays an expression whether it runs over a small value or a large
one. Expressions are evaluated by the engine itself, not inside a target, so an
author should still weigh the cost of shaping a very large payload inline
against doing it inside the target it feeds; that is a performance judgment, not
a change in where the operation belongs.

This boundary is why MWL defines no catalog of computational "intrinsic"
functions. A selection-only query language needs such a catalog to compute at
all; CEL computes natively, so pure shaping is just an expression, and
everything beyond pure shaping is deliberately pushed across the seam to a
provider, where nondeterminism and side effects are governed.
