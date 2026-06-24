---
title: "The Flow object"
weight: 60
---

A **Flow object** describes a workflow as a directed graph of named Steps:
execution enters the graph at the Flow's `entrypoint` and follows Step routing
until the Flow completes with a Result
([How a Flow completes](#how-a-flow-completes)).

The same object describes a whole workflow and a piece of one. A root document,
a named `flows` entry, and an inline `call.flow` value are all Flow objects with
the same fields and the same semantics
([Where Flows appear](#where-flows-appear)); only the root carries `$schema`
([`$schema`](#schema)). This makes the Flow the coarsest unit of composition in
MWL: subflows are named for reuse or embedded inline. A Flow executes inside its
own frame, an execution-time instantiation of the Flow with its own variables
and lifecycle (see [Execution model](../execution-model/)); the
[`vars` model](#the-vars-model) defined below is frame-scoped.

## A complete Flow

The smallest useful Flow names where to start, lists its Steps, and ends. The
following is a complete, valid root Flow:

```json
{
  "$schema": "https://mwl.dev/v0.1/flow/schema.json",
  "entrypoint": "greet",
  "steps": {
    "greet": {
      "action": "Call",
      "call": {
        "provider": "mwl:provider.call/example/http/v1",
        "with": { "method": "GET", "path": "/hello" }
      },
      "next": "done"
    },
    "done": { "action": "Return" }
  }
}
```

`$schema` declares the spec version (see [`$schema`](#schema)). `entrypoint`
names the Step where execution enters the graph; here, that Step is `greet`.
`steps` maps Step names to Step definitions: `greet` issues a `Call` and on
success routes to `done`, whose `Return` action completes the Flow. All other
Flow fields are optional: additional complexity is opt-in only as required by a
given use-case.

The following sections enumerate both the required and optional field keys and
define each in turn.

## Where Flows appear

A Flow object occurs in three contexts. The context determines where the
definition lives and what supplies the frame's input when it runs.

| Context             | Flow object location           | Frame input source         |
| ------------------- | ------------------------------ | -------------------------- |
| Root                | The top-level document         | The execution input        |
| Named `flows` entry | `flows.<Name>`                 | A `call`'s evaluated input |
| Inline `call.flow`  | A `call` object's `flow` field | A `call`'s evaluated input |

A named or inline Flow runs because a `call` object targets it, either as part
of a `Call` Step or a `Gather` dispatch whose `call` names a `flow` (see
[The Call interface and Result](../call-interface/) and
[Step actions](../step-actions/)). In both cases the frame's input is the value
the `call`'s `input` field produced when it was evaluated at the call site. The
platform starting an execution runs a root Flow, and that root frame's input is
the execution input JSON provided by the platform.

The frame's input is what the Flow receives, not necessarily what its entrypoint
Step sees: flow-level middleware wraps the step-graph, and its `onEntry` may
reshape the frame input on the way down to the entrypoint Step (see
[Middleware mechanics](../middleware-mechanics/)).

As all three contexts use the same object, the frame lifecycle, middleware
composition, parameter validation, and variable scoping described below apply
identically regardless of where the Flow appears.

## How a Flow completes

Every Flow execution ends by producing exactly one Result; this completion
contract, and the outcomes imposed from outside a Flow's definition such as
cancellation, are defined in [Execution model](../execution-model/). Within its
definition, a Flow completes in one of three ways:

- A terminal Step's `Return` action completes the Flow successfully: the data
  the Flow returns becomes its Result's `value`.
- A terminal Step's `Raise` action completes the Flow with a failure Result the
  author constructs.
- An unhandled failure completes the Flow: a failure Result that no `catch`
  clause matches propagates out of the frame and becomes the Flow's Result.

Terminal Steps and failure routing are defined in
[Steps and step mechanics](../step-mechanics/); the `Return` and `Raise` actions
in [Step actions](../step-actions/).

A success Result carries the returned `value`. A failure Result carries no
`value`; it instead carries a structured envelope of fields describing what went
wrong. Both shapes are defined in
[The Call interface and Result](../call-interface/), and they are the same
whether the Flow ran as the root of an execution or as a Call target: a caller
consumes a subflow's Result exactly as it consumes a provider's
([Flow-Call Result parity](../call-interface/#flow-call-result-parity)).

## Flow fields

A Flow object has the following keys. Only `$schema` is restricted to the root
document; every other key is available on every Flow, wherever it appears. Each
key has its own subsection below, in the order of the table.

| Field        | Type                 | Required        | Default | Expression      | Description                                                                            |
| ------------ | -------------------- | --------------- | ------- | --------------- | -------------------------------------------------------------------------------------- |
| `$schema`    | string (URI)         | required (root) | —       | no (structural) | Identifies the document as a Flow definition and its spec version. Root document only. |
| `comment`    | string               | optional        | —       | no (literal)    | Human-readable documentation. See [`comment`](#comment).                               |
| `entrypoint` | string (Step name)   | required        | —       | no (structural) | The Step where execution enters the graph. See [`entrypoint`](#entrypoint).            |
| `flows`      | object               | optional        | `{}`    | no (structural) | Map of name to Flow object: named subflows for reuse. See [`flows`](#flows).           |
| `middleware` | array                | optional        | `[]`    | no (structural) | Ordered middleware wrapping the Step graph. See [`middleware`](#middleware).           |
| `parameters` | object (JSON Schema) | optional        | —       | no (structural) | The Flow's parameter schema. See [`parameters`](#parameters).                          |
| `steps`      | object               | required        | —       | no (structural) | Map of Step names to Step definitions. See [`steps`](#steps).                          |

Note that all of these keys are structural (aside from `comment`, but this is a
literal for documentation only): their values are part of the definition, not
expressions evaluated at runtime. Expressions appear inside Step definitions,
middleware entries, and Calls, not in the keys that frame them.

### `$schema`

`$schema` is required on the root document and is the sole field that
distinguishes the root from a named or inline Flow. Its value names the spec
version the definition is authored against (see
[Schema documents](../definition-format/#schema-documents) for what the value
resolves to and the validation it enables).

### `comment`

A Flow object may carry an optional
[`comment`](../definition-format/#the-comment-field) to provide human-readable
documentation or other such context.

### `entrypoint`

`entrypoint` names the Step where execution enters the graph. Its value MUST be
a key of this Flow's `steps` object; resolution is scoped to that object (see
[Step-name scoping](#step-name-scoping)). When the frame is entered, execution
begins at the named Step and follows each Step's routing until the Flow
completes.

### `flows`

`flows` is a map from a name to a Flow object. Each entry is a reusable subflow
that a `call` can target by name. A Flow declared once under `flows` can be
called from many Steps, and from `Gather` dispatches, without repeating its
definition.

A named Flow is referenced by its bare name in a Call's `flow` field. The
following entry declares a `ProcessGranule` subflow with one parameter:

```json
"flows": {
  "ProcessGranule": {
    "comment": "Register a single granule with the catalog",
    "parameters": {
      "type": "object",
      "properties": { "collection": { "type": "string" } }
    },
    "entrypoint": "register",
    "steps": {
      "register": {
        "action": "Call",
        "call": {
          "provider": "mwl:provider.call/example/http/v1",
          "with": { "method": "POST", "path": "/granules", "collection": "{{ vars.collection }}" }
        },
        "next": "done"
      },
      "done": { "action": "Return" }
    }
  }
}
```

A Step then targets it by name, passing arguments through the Call's `with` and
a payload through `input`:

```json
"call": {
  "flow": "ProcessGranule",
  "with": { "collection": "modis-l1" }
}
```

A Flow that is used in exactly one place need not be named. A Call may embed a
Flow object directly in its `flow` field instead of referencing `flows`:

```json
"call": {
  "flow": {
    "entrypoint": "build-summary",
    "steps": {
      "build-summary": {
        "action": "Call",
        "call": {
          "provider": "mwl:provider.call/example/http/v1",
          "with": { "method": "POST", "path": "/summary" }
        },
        "next": "done"
      },
      "done": { "action": "Return" }
    }
  }
}
```

A named entry and an inline object are the same Flow object; naming only governs
reuse and reference. The call-site mechanics — how `provider` and `flow` relate,
how `input` and `with` are supplied — are defined in
[The Call interface and Result](../call-interface/).

A name declared in `flows` is visible to call sites in the declaring Flow and in
every Flow nested within it: a subflow declared once near the root can be called
from anywhere beneath its declaration. How a `flow` name resolves through the
nesting, and the constraints on references, are defined in
[Flow-name scoping](#flow-name-scoping).

### `middleware`

`middleware` is an ordered array of middleware entries that wrap the Flow's Step
graph: the entries form a stack around the graph as a whole, running on entry to
and exit from the frame. The entry shape, the phase model, and composition and
ordering are defined in [Middleware mechanics](../middleware-mechanics/); the
same array shape applies to a `Call` Step's `middleware`.

### `parameters`

`parameters` is a [JSON Schema](../definition-format/#schema-documents) document
describing the Flow's named parameters. The schema MUST have `"type": "object"`
at the top level; each property defines one named parameter. A parameter is
required when the schema's `required` array lists it. A property's `default`
declares the value the parameter takes when the caller does not supply it; a
parameter that is neither required nor defaulted is optional and, when not
supplied, is simply unbound.

Validation of arguments against the schema is closed by default. JSON Schema
alone leaves undeclared object members open; this format closes them: when the
schema does not set `additionalProperties`, it is evaluated as if it set
`"additionalProperties": false`, so an argument inside `with` whose name matches
no declared property fails validation. A schema that sets `additionalProperties`
itself, whether to `true`, `false`, or a subschema, is evaluated as written. A
Flow that declares no `parameters` takes no arguments: an empty or absent `with`
is valid, and any named argument fails validation.

A parameter schema is what the Call's `with` is validated against. Both a Flow
and a provider declare a `parameters` schema for the arguments they accept — the
relationship is symmetric, and is the first of the three axes a Call interacts
with its target along (see
[The three axes](../call-interface/#the-three-axes-parameters-with-and-input)).

Parameters are distinct from the Flow's
[data-plane](../concepts/#the-data-plane) input. They represent configuration —
behavioral knobs, operational settings, deployment-specific values — that should
not be mixed into the data payload. Decoupling the two lets a parent Flow tune a
subflow's behavior, or a Step tune its middleware, without threading
configuration through the data plane.

Caller-supplied parameter values are validated against the schema at frame
entry. `vars` is then seeded with the schema's declared `default` values,
overlaid by the validated arguments, before the first Step runs (see
[The `vars` model](#the-vars-model)): a supplied parameter binds its validated
value, an unsupplied parameter with a `default` binds that default, and a
parameter that is neither carries no binding — an expression that reads it
unguarded faults (see [Evaluation errors](../expressions/#evaluation-errors)).
When validation fails, the frame produces `System.ParameterValidationFailed`
([`System.ParameterValidationFailed`](#systemparametervalidationfailed)).

### `steps`

`steps` is a map from Step name to Step definition; it holds the Flow's Step
graph. Step names MUST be unique within the map, and all routing within the Flow
resolves against it (see [Step-name scoping](#step-name-scoping)). What a Step
is, its shared fields, and its lifecycle are defined in
[Steps and step mechanics](../step-mechanics/).

## The `vars` model

A Flow separates two kinds of incoming data: _configuration_, declared as named
`parameters` and supplied by the caller, and the _data payload_, supplied as
input. Parameters become the frame's variables; the data payload flows through
the Steps. This section defines the variable model the Flow establishes; the
schema that declares the parameters is described under
[`parameters`](#parameters).

`vars` is the frame's variable namespace. The Flow defines the frame, so `vars`
is scoped to that frame: it is established when the frame is entered and
persists for the life of the frame. Every expression evaluated within the Flow —
in Step definitions, in the Flow's own `middleware`, in Calls — reads `vars`.
The runtime shape of the `vars` binding within the broader data model is defined
in [Execution context](../execution-context/).

`vars` is populated from two sources:

- **At frame entry**, from the Flow's `parameters`: validated caller arguments
  and schema-declared defaults are injected. This is the initial contents of
  `vars`.
- **During execution**, by `assign`. A Step's `assign` writes to `vars` (see
  [Steps and step mechanics](../step-mechanics/)); a middleware phase's `assign`
  writes to `vars` (see [Middleware mechanics](../middleware-mechanics/)); a
  call arm's `assign` writes to `vars` at the call boundary (see
  [The arms](../call-interface/#the-arms-onsuccess-and-onfailure)), and it is
  there, where the target windows are in scope, that a completed inner Flow's
  variables are carried out, by capturing `flow.vars.<name>` (see
  [The target windows](../call-interface/#the-target-windows-flow-and-provider)).

`vars` is a single flat namespace, and a name holds one binding at a time. An
`assign` that writes an existing name, including a name seeded from
`parameters`, replaces that name's value: the last write wins. A `Gather`'s
concurrent dispatches are no exception: dispatches write no variables while the
fan-out runs — their arms' writes land at the action's completion, one dispatch
at a time, in dispatch order (see
[Frames and sequential execution](../execution-model/#frames-and-sequential-execution)).

Each frame has its own `vars`. A subflow does not see or share its caller's
variables: it is entered with a fresh `vars` seeded from its own `parameters`.
Configuration crosses the boundary as arguments on the Call's `with`, never by
sharing a variable namespace. This is what keeps a Flow's behavior a function of
its declared parameters rather than of ambient state.

## Validation

The specification's normative obligations are about **runtime behavior**. The
primary failure code produced by runtime validation is
`System.ParameterValidationFailed`, defined below. Many structural constraints
can additionally be checked without executing the Flow; those are described
under [Static checks](#static-checks).

### `System.ParameterValidationFailed`

A non-success Result of type `error` and code `System.ParameterValidationFailed`
is produced when a value fails validation against the constraint it is required
to satisfy: the JSON Schema of a Flow's or a provider's `parameters`, or the
type this specification fixes for a field, such as a duration field whose value
is not a valid duration. The constraint binds the value itself, however it was
produced: a literal written in the definition and an expression's result fail it
alike (see [The produced value](../expressions/#the-produced-value)). The
Result's `details` field carries a structured description of the validation
error: at minimum the schema path that failed and the observed value;
implementations MAY include the full JSON Schema validation report.

JSON Schema 2020-12 treats the `format` keyword as an annotation by default: a
declared format describes a value without constraining it. Parameter validation
does not follow that default. An implementation MUST evaluate `format` as an
assertion wherever a schema it validates against declares it, so a value that
does not match its declared format fails validation like any other constraint
violation. The parameter schemas this specification and its providers publish
rely on `format` for exactly such constraints—a duration parameter declares
`"format": "duration"`—and an annotation-only reading would pass a malformed
value through unchecked.

The failure surfaces at different points depending on what was being validated.
Each site below describes only its trigger; the code's meaning is defined here,
and the [Failure code reference](../failure-code-reference/) lists it.

- **Flow parameter values** (root or subflow): during frame entry, as part of
  variable initialization. The failure is the frame's Result; no middleware on
  that frame is established, and the Step graph does not run. For a subflow
  frame, the failure propagates to the enclosing context like any other frame
  failure — on a `Gather` dispatch, it is that dispatch's Result, counted by the
  `Gather`'s `completion` policy (see [Step actions](../step-actions/)).
- **Provider `with`**: at Call dispatch. On a `Call` Step, dispatch is the
  innermost operation in the middleware stack around it, so the failure bubbles
  outward through that stack on its failure path: catchable by the Step's
  `catch` clauses and retriable by `Retry` middleware in the stack (see
  [Steps and step mechanics](../step-mechanics/) and
  [Middleware mechanics](../middleware-mechanics/)). On a `Gather` dispatch, the
  validation failure is that dispatch's Result directly, counted by `completion`
  (see [Step actions](../step-actions/)).
- **Middleware `with`**: at the phase that carries it, when that phase runs — an
  `onEntry` `with` before the middleware's inner stack runs, an ascent phase's
  `with` as the Result rises past the entry. The failure is emitted from that
  middleware's position in its stack: visible to outer middleware, which can
  therefore catch or retry it, but not to the middleware that produced it or to
  inner middleware never established (see
  [Middleware mechanics](../middleware-mechanics/)).

### Static checks

Many of the specification's structural constraints depend only on the Flow
definition and can be checked without executing it. Platforms commonly perform
such checks at registration, submission, or authoring time — in tooling,
linters, and IDE integrations — to surface authoring errors early. The
specification does not require static checking: the runtime obligations above
are the normative surface, and a platform that performs no static checking
remains conformant as long as it validates at runtime.

Constraints amenable to static checking include the validity of `parameters`
schemas; the resolution of `provider` URIs against the platform's catalogs and
of `flow` names within their scope chains, and the acyclicity of the
flow-reference graph ([Flow-name scoping](#flow-name-scoping)); the match
between statically-resolvable `with` arguments and their declared schemas;
middleware applicability; Step reference scoping; the non-emptiness of a
`Gather`'s `calls` array and the positivity of its `concurrency` cap (see
[Step actions](../step-actions/)); and the absence of expressions in structural
fields. Each is specified in detail alongside the feature it constrains.

## Flow-name scoping

A `flow` name resolves lexically, against the `flows` maps of the Flow objects
that enclose the call site in the definition document. The name is matched by
exact comparison, first against the `flows` map of the Flow containing the call,
then against each enclosing Flow object's map outward to the root. The nearest
declaration wins: an inner entry shadows an outer entry of the same name for
every call site within the inner Flow. Shadowing is legal; tooling MAY warn when
an entry shadows an enclosing declaration. A `flow` name MUST resolve to an
entry in one of the maps on its chain.

The chain follows the nesting of Flow objects in the document, not the frames of
a running execution. A named Flow resolves its own `flow` references from its
declaration site, wherever it is called from; a caller's `flows` map never
influences what a callee's names mean. Scoping thereby keeps a Flow's behavior a
function of its definition, the same property the
[`vars` model](#the-vars-model) gives its state.

The contrast with [Step-name scoping](#step-name-scoping) is deliberate. A Step
reference is a routing edge inside a single frame's graph, and control cannot
jump between graphs, so a Step name has no meaning outside its own `steps` map.
A `flow` reference imports a definition, not state: the referenced Flow is
instantiated in a fresh frame, with its own `vars` seeded from its own
`parameters`, however far up the chain its declaration sits. The wider scope
shares definitions without sharing state.

References MUST NOT form a cycle. In the directed graph whose nodes are the
document's Flow objects and whose edges run from the Flow containing a `call` to
the Flow that call targets, whether named (as resolved by the chain) or inline,
no Flow reaches itself. Because `flow` is structural, this graph is fixed by the
definition, and the constraint is checkable without executing the Flow
([Static checks](#static-checks)). The chain places a named Flow's own name in
scope within its body, so it is the cycle rule that excludes self- and mutual
reference: a workflow that repeats does so within one graph, by routing
([Routing: `next` and terminal Steps](../step-mechanics/#routing-next-and-terminal-steps)),
not by a Flow re-entering itself.

## Step-name scoping

Step names MUST be unique within their containing `steps` object. All Step
references — `entrypoint`, a Step's `next`, and a `catch` clause's `next` —
resolve against the `steps` object that contains the referencing Step.
Cross-scope transitions are not permitted: a Step cannot target a Step in a
parent, child, or sibling `steps` map. A Flow's `steps` map is self-contained;
crossing between Flows is done by calling a Flow, not by referencing its Steps.

## Definition versioning

The only version a Flow records is the spec version in `$schema`. A definition
does not carry a version of its own, and the language defines no mechanism for
distinguishing one revision of a workflow from another. Keeping a version out of
Flows is intentional, as doing so avoids a class of problems that arise when
definitions carry their own version metadata: staleness, conflicts between the
declared version and the actual content, incompatible versioning semantics, and
ambiguity regarding what a version even means (particularly when definitions are
generated by tooling). The language is deliberately uninvolved: it describes
_what to do_, not _which revision of what-to-do this is_.

Therefore, all responsibilities regarding definition revisions, including
assigning version identifier(s), tracking/storing multiple versions, and
resolving which version to run, are all concerns outside the scope of this
specification. A definition should not, cannot, and does not know its version:
versioning is a higher-level concern, something the platform or an adjacent
service should manage, through content hashing, monotonic identifiers, or
whatever mechanism deemed most appropriate.
