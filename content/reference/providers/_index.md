---
title: "Providers"
weight: 120
singleFileSkipTOC: true
---

A **provider** is a named integration the language dispatches to: the extension
seam through which platform capability enters a workflow. Providers come in two
kinds, parallel in shape. A **call provider** is a Call target: an external
service or platform capability a `call` object names in its `provider` field
(see [The Call interface and Result](../call-interface/)). A **middleware
provider** plugs into the phase model, wrapping a Call or a Step graph (see
[Middleware mechanics](../middleware-mechanics/)).

The specification defines the contract a provider plugs into: how a provider is
identified, what its catalog entry declares, and how a use of it is validated.
The concrete provider list, including or in addition to the set this
specification defines itself, is a platform's to supply. This page defines what
is common to both kinds; [Call providers](call-providers/) and
[Middleware providers](middleware-providers/) document each catalog.

## Provider URIs

A provider is identified by a URI: the value of the `provider` field on a `call`
object and on a middleware entry. The URI's **scheme** identifies the layout of
the identifier, and the scheme fixes what the rest of the URI means.

The scheme is a pluggability seam, parallel to the way an expression's
delimiters identify its language (see [Expressions](../expressions/)): a later
version or an extension specification could admit another scheme as a new row
with its own layout and resolution rules without subverting anything defined
here. For example, an `https` URI scheme could be supported as an extension, to
both identify providers and point to their published specifications.

MWL currently defines one scheme:

| Scheme | Layout                           |
| ------ | -------------------------------- |
| `mwl`  | `mwl:<type>/<namespace>/<name…>` |

Implementations MUST support the `mwl` scheme as this scheme is used to define
the URIs for the set of language-specified providers.

### `mwl` URI scheme

Under the `mwl` scheme, an identifier has three parts, separated by slashes; a
dot subdivides within a part:

- `<type>`: what kind of extension the URI identifies, drawn from the type table
  below. A dot subdivides a type within a family: `provider.call` and
  `provider.middleware` are the two provider types. The type is implied by the
  field that carries the URI, but carrying it in the identifier keeps the
  identifier self-describing, and it makes cross-type misuse, i.e., a middleware
  provider named as a Call target, detectable from the definition alone.
- `<namespace>`: the defining authority, i.e., who publishes the provider's
  specification.
- `<name…>`: one or more segments naming the provider, structured as the
  authority chooses. Everything after the namespace is the authority's.

This version defines two types:

| Type                  | Extension                                                             |
| --------------------- | --------------------------------------------------------------------- |
| `provider.call`       | A call provider ([Call providers](call-providers/))                   |
| `provider.middleware` | A middleware provider ([Middleware providers](middleware-providers/)) |

A URI whose type the specification does not define is not a valid `mwl` URI. The
type table is the scheme's own extension seam: a later version or an extension
specification may add a type as a new row — dotted where a family warrants
subdivision, plain where it does not — and the layout never changes: type,
namespace, and name keep their positions however the type set grows.

A provider's identity is the URI as a whole, matched exactly. The language
attaches no semantics to the name segments; in particular, it never compares or
resolves versions.

#### Syntax

The scheme's syntax, in ABNF
([RFC 5234](https://datatracker.ietf.org/doc/html/rfc5234)):

```abnf
mwl-uri   = "mwl:" type "/" namespace "/" name
type      = segment
namespace = segment
name      = segment *( "/" segment )
segment   = 1*( ALPHA / DIGIT / "-" / "_" / "." )
```

The valid `type` values are those the type table above enumerates.

A segment MUST NOT be `.` or `..`, which carry path semantics in generic URI
processing; excluding them keeps an `mwl` URI inert under any URI library. An
`mwl` URI has no query or fragment component, and no authority component: its
path is rootless, the scheme followed directly by the first segment — no `//`,
`?`, `#`, or leading `/`. The `//` of generic URI syntax introduces a _network_
authority, a host; the `mwl` namespace is a _naming_ authority, not a host, and
it travels in the path, the convention of identifier schemes that carry naming
authorities of their own, such as `tag:` and `urn:`.

Matching is case-sensitive, and the scheme name is written in lowercase, `mwl`.
Because identity is exact match, `Http` and `http` name different providers; the
RECOMMENDED style is lowercase with hyphens, which every identifier in this
specification follows.

Percent-encoding is not part of the scheme: `%` is not a permitted character.
Every permitted character is among RFC 3986's unreserved characters, so an `mwl`
URI is a valid URI exactly as written and has exactly one spelling — a
deliberate property, since exact-match identity tolerates no alternative
encodings of the same name. The `.` is the within-a-part subdivider wherever a
part needs one: a type's family (`provider.call`), a DNS-shaped namespace
(`example.com`), a dotted version segment (`v1.2`).

#### Versions

The RECOMMENDED convention is to end a provider's name with a version segment
like `mwl:provider.call/example/http/v1`. All providers defined in this
specification follow this convention. The version is part of the name, not a
structural element of the identifier: `…/http/v1` and `…/http/v2` are simply two
providers. What may change within one version and what requires a new one is
versioning policy, which is the defining authority's to state, in the provider's
specification or elsewhere.

#### Reserved namespaces

Two namespaces are reserved in the `mwl` scheme:

- `mwl` is the specification's own. The providers this specification defines
  live under it and a platform or third party MUST NOT define a provider in it:
  a URI in the `mwl` namespace always means a spec-defined provider.
- `example` is reserved for documentation. It never identifies a real provider,
  and a platform's catalog MUST NOT contain an entry under it. Documentation and
  teaching material use it freely, certain never to collide with a deployed
  provider; this specification's own examples do.

Beyond the two reservations, namespace uniqueness is platform governance, as
`codePrefix` uniqueness is below: the platform whose catalog resolves the URIs
decides who publishes under which namespace.

### Resolution

A provider URI is a catalog key: the platform resolves it against its catalog to
an implementation. Whether every URI a definition references resolves, and
whether each names the type its position requires, are statically checkable
properties of the definition (see [Static checks](../flow-object/#static-checks)
and [Validation](../middleware-mechanics/#validation)).

## The provider catalog

Every provider a platform supports has an entry in the platform's catalog. The
entry is the provider's contract: what the engine consults at dispatch and what
tooling validates against. Every entry, for either kind, declares:

- the provider's **URI** ([Provider URIs](#provider-uris));
- its **`codePrefix`** — its identity segment in failure codes (see
  [The failure catalog](#the-failure-catalog));
- its **parameter schema or schemas** — what a `with` is validated against (see
  [Parameter validation](#parameter-validation));
- its **failure catalog** — the codes it can emit (see
  [The failure catalog](#the-failure-catalog));
- the **metadata it exposes** — a call provider's `provider.metadata` window
  members (see [`provider`](../execution-context/#provider)), a middleware's
  contributed members (see
  [Middleware-contributed metadata](../execution-context/#middleware-contributed-metadata)).

What else an entry declares is kind-specific and documented with each catalog.
For example, consider a middleware's per-phase actions, gateable phases, and
acceptance semantics (see
[What a middleware declares](../middleware-mechanics/#what-a-middleware-declares)).

The specification does not prescribe how a platform organizes its catalog. A
fixed set of providers, an extensible registry, or anything between satisfies it
so long as the platform resolves provider URIs and validates `with` values
against the declared schemas at runtime.

Provider definitions are independent specifications: each declares a URI,
parameter schemas, a failure catalog, and a metadata surface, and a definition
can originate from this specification, a platform vendor, or a community effort.
An implementation advertises which provider specifications it supports, and a
workflow is portable to any implementation that supports the providers it
references. A workflow's portability is therefore a function of provider
adoption. The language itself is fully portable; providers are where
platform-specific behavior enters.

### The provider definition document

A provider definition's interchange form is a JSON document carrying every
declaration of the catalog entry in a fixed shape, together with a
human-readable `description`. The document's `$schema` key names the MWL
provider-definition schema for the specification version the definition is
authored against, a schema URI as defined in
[Schema documents](../definition-format/#schema-documents); for this version,
[`https://mwl.dev/v0.1/provider/schema.json`](/v0.1/provider/schema.json). One
schema serves both provider kinds, discriminated by the type segment of the
document's `uri`, and a definition MAY be validated against it. As with the
meta-schema, the prose contract is normative and the schema is its
machine-checkable companion.

The providers this specification defines are published in this form, each linked
from the page that specifies it. The
[`mock` provider's](call-providers/#the-mock-provider) document,
[`mock.v1.json`](mock.v1.json), doubles as the worked example for provider
authors.

## Parameter validation

A call's `with` is validated against the target provider's declared parameter
schema when the Call dispatches; a middleware phase's `with` is validated
against the schema the middleware's contract declares for that phase, when the
phase runs (see [Validation](../middleware-mechanics/#validation)). A value that
fails produces `System.ParameterValidationFailed` (see
[the definition](../flow-object/#systemparametervalidationfailed)). The
schema-evaluation rules — a parameter is required exactly when the schema's
`required` lists it, and validation is closed by default — are defined with
[`parameters`](../flow-object/#parameters) and apply to provider schemas
identically: the symmetry is the `with`-to-`parameters` axis of
[The three axes](../call-interface/#the-three-axes-parameters-with-and-input).

The data channel is provider-opaque: the specification requires no schema for
the `input` a provider accepts or for the value it produces. Validating those,
where wanted, is the business of tooling above the language — SDKs, registries,
IDE integrations — that knows what a specific dispatch resolves to.

## The failure catalog

Every catalog entry declares a **`codePrefix`**: the identity segment of the
codes the provider emits, in the taxonomy `Provider.<Kind>.<codePrefix>.<Code>`
— `Provider.Call.Http.ConnectionFailed`, `Provider.Middleware.Retry.Exhausted`.
The namespace scheme those codes inhabit is defined with the failure envelope
(see [Code namespaces](../call-interface/#code-namespaces)); a catalog declares
each provider's slice of it. `codePrefix` uniqueness within a platform's catalog
is platform governance.

A failure catalog declares **closed codes**, the specific codes the provider can
emit, and **open sub-prefixes**, regions where codes not knowable in advance may
appear — typically because they originate in user code or external systems the
provider surfaces rather than defines. The declaration's interchange form is
JSON:

```json
{
  "closed": [
    "Provider.Call.Http.ConnectionFailed",
    "Provider.Call.Http.Throttled",
    "Provider.Call.Http.NonSuccessStatus"
  ],
  "open": []
}
```

A function-runtime provider that surfaces failures thrown by the user code it
runs declares an open sub-prefix for them:

```json
{
  "closed": ["Provider.Call.Function.Timeout"],
  "open": ["Provider.Call.Function.Errors.*"]
}
```

A sub-prefix of `*` alone declares the whole code space open: the honest catalog
of a provider whose codes are entirely caller-determined, such as the
[`mock` provider](call-providers/#the-mock-provider).

Closed is a constraint on the provider, not on the string space: a conformant
provider emits, under its prefix, only the codes its catalog declares plus codes
within its open sub-prefixes. Nothing partitions code values at runtime (see
[Code namespaces](../call-interface/#code-namespaces)).

A catalog SHOULD carry a human-readable description for each closed code and
each open sub-prefix; the specification does not fix where or in what format
those descriptions live.

Tooling MAY use failure catalogs to warn about matcher `codes` patterns that
reference no declared code (see
[Failure matching](../step-mechanics/#failure-matching)). An open sub-prefix
bounds such analysis: any code within it is declared.

## In this section

- **[Call providers](call-providers/)** — the call-target catalog: what a call
  provider's entry declares, including its `provider.metadata` window members.
- **[Middleware providers](middleware-providers/)** — the middleware catalog:
  the spec-defined middlewares with their per-phase schemas, failure codes, and
  composition notes, plus the platform-extensible mechanism.
