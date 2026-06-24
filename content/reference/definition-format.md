---
title: "The definition format"
weight: 35
---

The canonical format for MWL workflow definitions is JSON (UTF-8). A **workflow
definition** is a single JSON document whose root is a
[Flow object](../flow-object/), and the values within the document are values of
[the data model](../data-model/). This section defines the rules that apply to
the definition as a document: well-formedness, the schema documents that
identify and validate it, and the `comment` field available on structures
throughout it.

## Well-formedness

RFC 8259 only recommends that the member names within a JSON object be unique.
This format requires it: within a single object, member names MUST be unique; a
workflow definition that repeats a member name within one object is ill-formed,
and implementations MUST reject it as invalid rather than choosing a winning
value.

## Schema documents

A root workflow definition carries a `$schema` key whose value is the URI of the
MWL meta-schema for the specification version by which the definition is
authored. The URI serves two purposes at once: it identifies the spec version,
and it dereferences to a
[JSON Schema (2020-12)](https://json-schema.org/specification) document against
which the whole definition MAY be validated. The MWL meta-schema describes the
structure of a definition, including its keys and their shapes. For a definition
authored against this version of the specification, the value is
[`https://mwl.dev/v0.1/flow/schema.json`](/v0.1/flow/schema.json).

The prose of this specification is normative, and the meta-schema is its
machine-checkable companion. The meta-schema validates structure, not the
constraints that span a document: that an `entrypoint` or a `next` resolves to a
Step, a `flow` name to a `flows` entry, or a `provider` URI to a catalog entry
are properties the prose defines and a schema cannot express (see
[Static checks](../flow-object/#static-checks)). Where the two disagree, the
prose governs and the schema document is in error.

Notably, such validation is a distinct concern from that of `parameters`
schemas, whether embedded in a definition by the root Flow or subflows, or
declared externally by any providers the definition references. Those
`parameters` schemas are likewise JSON Schema (2020-12) documents.
Implementations MUST evaluate every schema this format uses, the meta-schema and
`parameters` schemas alike, under the 2020-12 dialect; implementations MAY
accept additional dialects as an extension. How `parameters` schemas are
declared and validated is defined in [The Flow object](../flow-object/) and
[Providers](../providers/).

### Schema URIs

Every schema document this specification publishes lives at a URI of one layout:

```text
https://mwl.dev/<version>/<type>/schema.json
```

`<version>` is the specification version the schema belongs to: `v0.1`. Schemas
version with the specification: one version's schema documents form one set, and
a later version publishes its own under a new version segment. `<type>` names
the kind of document the schema describes, drawn from the type table below. The
table is the layout's extension seam: a later version or an extension
specification may add a document kind as a new row without changing the layout.

| `<type>`   | Document                                                                                                       | Schema document                                            |
| ---------- | -------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `flow`     | A workflow definition: the document this section defines                                                       | [`/v0.1/flow/schema.json`](/v0.1/flow/schema.json)         |
| `provider` | A provider definition (see [The provider definition document](../providers/#the-provider-definition-document)) | [`/v0.1/provider/schema.json`](/v0.1/provider/schema.json) |

A schema URI is the schema document's identity as well as its location: each
schema document declares the URI it is published at as its own `$id`, and a
document declares the schema it is authored against by carrying that URI in its
`$schema` key.

## The `comment` field

`comment` is an optional, human-readable string available on core structures
within this spec. `comment` values are intended to contain documentation or
other such contextual information for human readers and should carry no runtime
meaning. Every structure in the spec listing `comment` as a field defers the
field's meaning to this definition.

- Implementations **MUST** preserve `comment` values in any serialized
  round-trip of a definition. A definition read in, then written back out, must
  contain the same `comment` values in the same positions.
- Implementations **MAY** surface `comment` values in operational tooling:
  execution logs, debugging UIs, audit trails, run histories, error reports.
  Surfacing is encouraged where it helps operators understand intent, but is not
  required for conformance.
- Implementations **MUST NOT** interpret `comment` semantically. The engine MUST
  treat the value as opaque text: no parsing, no inference of intent, no
  behavioral effect.
