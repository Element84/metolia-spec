# Parser design decisions

A running record of the design decisions behind the MWL parser. This is a
working document, not a formal decision log: entries are revised as the design
changes, and a decision that hardens may later graduate to its own record. Each
entry states what was decided and why, so a later reader need not reconstruct
the reasoning from the code.

## Decisions at a glance

1. [Monorepo with a Go workspace](#monorepo-with-a-go-workspace) — the parser
   lives with the spec, so a change to both is one pull request; it is a
   separate Go module so it stays independently importable and versionable.
2. [Open- and closed-source posture](#open--and-closed-source-posture) — the
   parser and a local runtime stay open in this repository as the adoption
   on-ramp; a separate production runtime is the closed product, kept elsewhere.
3. [The core parser is catalog-free](#the-core-parser-is-catalog-free) — parsing
   and structural validation need no provider catalog; a provider URI is checked
   as a well-formed name, not resolved.
4. [Provider-aware validation is a separate layer](#provider-aware-validation-is-a-separate-layer)
   — resolving URIs to definitions and validating `with` against them is an
   optional pass, possibly its own module later. _Future._
5. [Embedded schemas](#embedded-schemas) — the parser embeds the JSON Schemas
   with `go:embed`, guarded by a drift check against the published originals.

## Monorepo with a Go workspace

The parser lives in this repository alongside the specification, its JSON
Schemas, and its conformance tests. A change that touches the language and the
parser together (a new field, a changed constraint) is then one pull request
against one repository, which keeps the prose, the schemas, the parser, and the
tests from drifting apart. The parser's test suite runs against the existing
fixtures under `tests/`, so the conformance corpus that gates the spec also
gates the parser.

The parser is its own Go module (`parser/go.mod`), separate from the root
`go.mod` that exists only to load the Hugo theme. A `go.work` file at the
repository root includes `./parser` and nothing else. The workspace is
development-time glue that lets a single checkout build and test everything
together; it does not affect what a consumer importing the parser pulls in.

A separate module for the parser means it is independently importable and
independently versionable: a third party building MWL tooling depends on the
parser without inheriting the spec site's or a runtime's dependencies. It also
draws the boundary between what lives in this repository and what a closed
production runtime, kept elsewhere, would import: that runtime depends on the
parser module without any of it needing to move.

## Open- and closed-source posture

Three artifacts, three postures. The parser and a local runtime stay open and
live in this repository; a production runtime is the product, closed, and lives
elsewhere.

The parser and the local runtime are open to support adoption. The parser is the
shared, checkable contract for the language: it is what lets others build
linters, editor integrations, and validators against MWL without a runtime. The
local runtime lets people actually run MWL without the product. Together they
are the on-ramp to the spec; closing either would concede the spec's openness by
leaving the language readable but not usable without the product.

The production runtime is the commercial engine and may be closed. Because it is
a separate module that imports the parser, keeping it closed and elsewhere costs
the open artifacts nothing: the monorepo still holds the spec, the parser, and
the local runtime together, so a change across those three remains one pull
request.

## The core parser is catalog-free

A Flow names providers by URI but does not embed their contracts. Parsing a Flow
and validating its structure need none of those contracts: the specification's
static checks are that a Flow is well-formed, that its step graph is sound, and
that each provider URI is a syntactically valid name of the type its position
requires. A provider URI is checked as a well-formed name and left unresolved;
whether it names a provider that exists is not asked here.

The specification supports this split. It frames catalog resolution and `with`
validation as runtime activities — a call's `with` is validated "when the Call
dispatches" — and casts tooling-time validation as an optional courtesy. So the
core parser takes no provider catalog: parsing is total and offline, and never
depends on which providers a platform supplies.

## Provider-aware validation is a separate layer

Beyond structure, a document can be checked against the providers it references:
that each URI resolves to a definition, and that a call's `with` validates
against the resolved provider's parameter schema. This is the optional pass the
specification assigns to the runtime at dispatch and offers to tooling. A free,
open validator of this kind is worth shipping to support adoption.

It is a distinct layer from the core parser because it needs things the parser
does not: a catalog of provider definitions to resolve against, and a JSON
Schema evaluation engine to check `with` values. Keeping it separate means a
consumer who only parses and navigates — a formatter, a graph tool — does not
inherit a schema engine it never calls.

Whether this layer is a package inside the parser module or its own module is
deferred. The criterion is dependency weight: promote it to its own module when
the schema-evaluation dependency is real and a parser-only consumer would
otherwise carry it. A package split is cheap and reversible; a module split is a
public commitment, so it waits for that pressure to appear.

_Future; nothing built yet._

## Embedded schemas

The parser embeds the JSON Schemas (`static/v0.1/`) into itself with `go:embed`,
so a consumer who imports the parser has what structural validation checks
against without a copy of this repository's tree.

Embedding duplicates bytes that also live in the source tree, which can drift. A
check (in `scripts/check-schemas.sh` or an equivalent test) compares the
embedded copies against the published originals so that a spec-side change which
forgets the parser fails CI. This is the parser-side counterpart to the schema-
and-test review the repository already requires for spec changes.

_Decided in principle; not yet implemented._
