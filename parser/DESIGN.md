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
5. [The parser is three passes](#the-parser-is-three-passes) — decode builds the
   positioned value tree, bind builds the typed AST and validates structure,
   check enforces the cross-document rules; each pass collects diagnostics.
6. [Decoding goes through a positioned value tree](#decoding-goes-through-a-positioned-value-tree)
   — a token-level decode builds a generic JSON tree carrying self-contained
   positions, document order, and duplicate-member rejection; template fields
   keep their subtrees after binding.
7. [The Step is a closed interface sum](#the-step-is-a-closed-interface-sum) —
   each action is its own struct carrying exactly the fields the spec gives it;
   `Step` is an interface closed by an unexported marker method.
8. [Named collections keep document order](#named-collections-keep-document-order)
   — `steps` and `flows` are ordered lists with name indexes, so diagnostics are
   deterministic; value-and-order round-trip stays possible, byte fidelity is a
   non-goal.
9. [Expressions are recognized, not parsed](#expressions-are-recognized-not-parsed)
   — `Expr` stores body text and position, no CEL dependency; a generic
   literal-or-expression pair type; template fields hold pre-classified value
   trees.
10. [The typed model goes as deep as the spec's opinions](#the-typed-model-goes-as-deep-as-the-specs-opinions)
    — every construct the spec constrains binds to a typed node; where the
    spec's opinions stop (a flow's template fields, a provider's embedded
    schemas), the parser retains a positioned value tree instead.
11. [Structural validation is hand-written, with the schemas as test oracle](#structural-validation-is-hand-written-with-the-schemas-as-test-oracle)
    — the bind pass validates structure itself, with no JSON Schema engine
    anywhere in the module; the fixture corpus is the shared oracle, with the
    schema-side and parser-side suites each held to its verdicts.
12. [Diagnostics are collected, with severities](#diagnostics-are-collected-with-severities)
    — a parse reports everything it finds in one run; any error-severity
    diagnostic means a nil AST; the error set is exactly the spec's rejection
    set.
13. [The parser warns only as a by-product](#the-parser-warns-only-as-a-by-product)
    — warnings are limited to findings the parse already computes, drawn from a
    short, closed, documented list of codes; a complete linter is a separate
    layer.
14. [The API takes bytes, not files](#the-api-takes-bytes-not-files) —
    `ParseFlow` and `ParseProvider` in package `parser` take source bytes and do
    no file I/O; a filename is an optional diagnostic label supplied by
    functional option.
15. [Closed string sets are published as constants](#closed-string-sets-are-published-as-constants)
    — a value set that survives into the AST as string data gets exported
    plain-string constants, shared with the bind pass; a set the AST encodes as
    types gets none.
16. [The typed AST normalizes empty spellings](#the-typed-ast-normalizes-empty-spellings)
    — a spelling that contributes no value (an explicit null documented as
    equivalent to absence, an empty optional collection) binds identically to an
    absent field; re-emission from the typed AST is canonical, and syntactic
    fidelity is the decode tree's to provide.

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

## The parser is three passes

The parser runs three passes in order. Each answers one question, and each
collects diagnostics as it goes:

| Pass   | Question it answers                                                                                                     | From → to                            |
| ------ | ----------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| decode | Is this legal input at the document level: valid JSON, no duplicate members, nothing after the root value?              | source bytes → positioned value tree |
| bind   | Is each object the right shape: required fields present, types correct, no unrecognized fields, per-object constraints? | value tree → typed AST               |
| check  | Do the pieces agree with each other: references resolve, flow names resolve, no cycles, every Step has an exit path?    | typed AST → diagnostics only         |

A pass runs only when the one before it left something to run on: a syntax error
leaves no tree, so bind never runs; bind errors leave no AST worth checking.
Within a pass, one finding does not stop the pass itself — bind keeps binding
the remaining Steps past a bad one (see
[Diagnostics are collected, with severities](#diagnostics-are-collected-with-severities)).

The division of knowledge follows the division of work: decode is generic JSON
plus positions, MWL's grammar lives in bind, and the rules that span the
document live in check.

## Decoding goes through a positioned value tree

The definition format requires an implementation to reject a document that
repeats a member name within one object. `encoding/json` silently keeps the last
duplicate, so plain unmarshalling cannot implement the spec, and the decode pass
reads token by token regardless of any other choice. That fixed cost pays for
the things that are painful to retrofit: source positions and document order.

The tree the pass produces is small generic JSON: every node carries its
position, every object records its members in document order, and duplicate
names are rejected as they are read. A position is stored fully resolved
(offset, line, and column, computed at node creation from a line index built
once per source), so nodes are self-contained: a consumer holding one can render
a location without carrying a translation table beside the tree. The bind pass
carries each node's position onto the AST node it becomes, so a diagnostic can
report `flow.json:42:10` beside the path (`steps.greet.next`). The tree is not
discarded wholesale after binding: template fields (see
[Expressions are recognized, not parsed](#expressions-are-recognized-not-parsed))
hold their subtrees directly.

## The Step is a closed interface sum

A Step's `action` selects which fields the rest of the Step accepts, and the
seven actions share almost nothing: only `comment` is universal, `next` appears
on five of seven, `catch` on two, `middleware` on one, and `Match` carries its
routing on clauses rather than on the Step. A single struct holding every field
would advertise fields the spec says the action does not accept, and every
consumer would re-derive the applicability table at runtime.

Each action is therefore its own struct (`CallStep`, `GatherStep`, and so on)
carrying exactly the fields the spec defines for it, and `Step` is an interface
those structs satisfy through an unexported marker method, which closes the set:
no package outside the parser can add a variant. Consumers dispatch by type
switch; Go does not check such switches for exhaustiveness, and a linter such as
go-check-sumtype covers that gap. The variant structs mirror the schema's
per-variant definitions (`stepCall`, `stepGather`, and so on) one to one, the
same schema-to-parser correspondence the differential test guards (see
[Structural validation is hand-written, with the schemas as test oracle](#structural-validation-is-hand-written-with-the-schemas-as-test-oracle)).

The interface treatment is reserved for the seven-way sum, and the reservation
follows a rule rather than a headcount. Go has no native sum type, so either
encoding of a discriminated union is a workaround with its own failure mode.
Pointer arms leave the exactly-one invariant invisible to the compiler, a
comment enforced by bind-pass discipline; at two arms the risk is negligible,
since a reader sees both pointers side by side and an `if`/`else` cannot skip a
case. The interface keeps each variant's field set honest and its dispatch
checkable, since a linter can hold a type switch to exhaustiveness, but no field
can live on an interface, so whatever the variants share is duplicated into each
or hidden behind methods. A union is therefore encoded as a struct with one
pointer per arm and a bind-time invariant that exactly one arm is set, and it
graduates to a closed sum when the variant count makes the one-arm invariant
illegible and exhaustive handling the thing needing a machine check.

Among MWL's unions, only the Step crosses that line: seven variants sharing only
`comment`. The two-way shapes stay on the arms side: a call's provider-or-flow
target, a `Gather`'s iterate-or-scatter form, a `Sleep`'s `for`-or-`until`, and
a provider definition's two kinds, where one `Provider` struct carries the
fields the kinds share (the URI, `codePrefix`, the description, the failure
catalog, the metadata schema) and a pointer arm per kind carries the rest.

## Named collections keep document order

`steps` and `flows` are JSON objects, and the data model declares object key
order insignificant, but the AST stores both as lists in document order with a
name-to-entry index built at bind, not as Go maps. The driver is determinism: Go
randomizes map iteration, and diagnostics should arrive in document order,
identically on every run.

Order fidelity also keeps re-serialization useful without promising byte
fidelity. A tool that reads a definition, edits it mechanically, and writes it
back can emit the same values in the same order, so the diff is the edit, up to
canonicalization of empty spellings (see
[The typed AST normalizes empty spellings](#the-typed-ast-normalizes-empty-spellings));
indentation and other lexical choices are a formatter's concern and a non-goal
here. The spec's round-trip obligation, that `comment` values survive a read and
rewrite, is met by modeling `comment` on every structure that accepts it.

## Expressions are recognized, not parsed

An expression is a string whose entire content one delimiter pair sets off;
every other string is a literal. The rule is purely syntactic, so the parser
classifies every string once, while reading, and records the answer in the AST;
no two consumers can disagree about what is an expression. The `Expr` node holds
the body text and its position, nothing more: parsing the body is the expression
language's domain and would pull in cel-go, a dependency no static check needs.
The core parser stays out of the expression language the way it stays out of
provider catalogs; the runtime and a linter parse `Expr.Body` themselves.

Classification serves three field shapes:

- A structural field (`action`, `next`, `provider`, `entrypoint`) rejects an
  expression-shaped string, a rejection the spec requires.
- A literal-or-expression field (`when`, `over`, `completion.successes`, `with`,
  `for`, `until`) binds to a generic pair type, `OrExpr[T]`, with exactly one of
  its literal and expression arms set. `retryable` and `previous`, where an
  explicit `null` differs from an absent field, get one-off pair types with a
  null flag.
- A template field (`input`, `output`, `value`, an `assign` entry's value, a
  `with` object) holds its positioned value subtree, whose string leaves are
  pre-classified into literals and `Expr`s.

## The typed model goes as deep as the spec's opinions

Bind produces a typed node for every construct the specification constrains, and
stops typing exactly where the specification stops having opinions: what remains
is retained as a positioned value tree, held but not interpreted. In a Flow the
boundary falls at the template fields (`input`, `output`, `value`, a `with`
object, an `assign` entry's value), whose interiors are the author's arbitrary
JSON; the specification keeps one opinion inside them, that a string leaf is a
literal or an expression, and that one opinion is applied to the retained tree
as leaf classification. In a provider definition the boundary falls at the
embedded parameter and metadata schemas, whose interiors are JSON Schema; no
opinion applies inside them, so their string leaves stay plain strings,
providers having no expression fields.

The rule guards against a typed model of a foreign grammar that goes partway.
Modeling the inside of an embedded schema (its `properties`, its `required`)
would look authoritative while silently mangling whatever it does not model
(`$ref`, composition, conditionals); modeling all of it means implementing JSON
Schema, the dependency the parse path deliberately excludes. The same holds for
expression bodies and CEL. Where another language begins, the parser records
position and content and leaves interpretation to that language's own tooling.

## Structural validation is hand-written, with the schemas as test oracle

The bind pass validates structure itself, in hand-written Go, rather than
delegating to a JSON Schema engine evaluating the published schema. Three
reasons, in order of weight. First, error quality is the parser's main product:
the bind pass knows the `action` discriminator, so it can say "a `Call` Step
requires `next`" with a file position, where a schema engine reports a failed
seven-way `oneOf` against a decoded value, positionless. Second, the authority
order: the prose governs and the schema may be in error, so a parser wired to
the schema inherits schema bugs, while a hand-written parser encodes the prose
and surfaces any disagreement as a failing fixture. Third, an engine would not
reduce code: the bind pass must examine every field anyway to build the typed
AST, so schema evaluation would be a second opinion carrying the module's first
dependency, not a replacement for the bind logic.

What keeps the hand-written pass honest is a differential check over the fixture
corpus (the schema fixtures under `tests/schemas/` plus the published provider
definitions), held from both sides by two suites that run together in the
pre-commit hook. `scripts/check-schemas.sh` asserts the schemas' verdict on
every fixture: the valid set validates, the invalid set is rejected. The
parser's corpus tests (`TestParseFlowFixtureCorpus`,
`TestParseProviderFixtureCorpus`) assert the parser's verdict on the same files:
clean parses on the valid set, error diagnostics and a nil AST on the invalid
set. A parser-schema divergence on any fixture therefore fails one of the two
suites. Agreement is checked transitively through the corpus rather than by
evaluating the schemas in-process, so no JSON Schema engine appears anywhere in
the module and `parser/go.mod` stays dependency-free even for tests.

The corpus check is also the parser-side drift check. The repository already
requires every normative spec change to land with schema and fixture updates, so
a change that reaches the schema and fixtures but not the parser fails the
parser's suite on the next run. The alternative, embedding the schema bytes with
`go:embed`, is rejected: with no engine in the parse path, embedded bytes have
nothing to check against at runtime, and behavioral agreement over the corpus is
a stronger guarantee than byte identity of copies. One limit is accepted
knowingly: agreement is only as wide as the corpus, so a divergence on a
document shape no fixture exercises survives until a fixture names it — the
price of keeping the engine out.

## Diagnostics are collected, with severities

A parse reports everything it can find in one run rather than stopping at the
first failure. The authoring loop is the reason: renaming a Step can dangle
three `next` references at once, and a fail-fast parser surfaces them one
fix-and-rerun at a time. The passes support collection naturally — static checks
are independent of one another, and the bind pass can finish binding one Step
after rejecting a field in another. Only a JSON syntax error in the decode pass
genuinely halts progress, since there is no tree left to read.

Each diagnostic carries a severity, a stable code, a source position, a path
(`steps.greet.next`), and a message. Two contracts bound the model. The error
set is exactly the specification's rejection set: the parser never rejects a
legal definition, and anything the spec marks MAY-warn is at most a warning. And
errors mean no AST: `ParseFlow` returns a nil Flow when any error-severity
diagnostic exists. A best-effort partial AST for editors, in the style of
`go/parser`, is deferred until an integration asks for it; loosening that
contract later is compatible, tightening it is not.

## The parser warns only as a by-product

The specification marks several findings as MAY-warn, MUST-NOT-reject: flow-name
shadowing, unreachable `catch` clauses, code-namespace trespass. These are lint
findings — legal definitions that are probably not what the author meant — and
the linter is a separate consumer of the parser, not the parser itself.

The parser still emits the subset of those warnings that fall out of work
parsing already does. Flow-name shadowing is the canonical case: resolution
walks the scope chain anyway, so the shadowed declaration is already in hand,
and suppressing the observation would only force a linter to re-implement scope
resolution. Anything requiring knowledge beyond the definition (provider
catalogs) or machinery the parser does not carry (CEL analysis) belongs to the
linter layer.

The boundary is made legible rather than asserted: the parser's warning codes
form a short, closed, documented list, so "the parser is not a complete linter"
is checkable from the docs, and consumers can key suppression and tests off the
codes.

## The API takes bytes, not files

The entry points are `ParseFlow(src []byte, opts ...Option)` and
`ParseProvider(src []byte, opts ...Option)`, both in package `parser`
(`parser.ParseFlow(...)`, on the precedent of `go/parser`). Provider definition
documents are part of the language, with their own published schema and
fixtures, and parsing one needs no catalog, so both document kinds share the
module and the pipeline.

The parser does no file I/O: callers hand it bytes, which come from a file in a
CLI, an unsaved buffer in an editor, or a request body on a platform. A filename
is therefore never an input, only a label for diagnostics, and it is supplied as
a functional option (`parser.WithFilename("flow.json")`) rather than a
positional parameter, so callers without a file do not pass an empty
placeholder. The options form also lets future knobs join without breaking the
signature.

## Closed string sets are published as constants

Several fields carry values drawn from a fixed vocabulary: a phase action's
`kind`, a middleware provider's attachment levels, the spec-defined Result
types. Consumers compare against these values — a linter dispatching on an
action kind, a validator inspecting matcher types — and a comparison against a
mistyped literal fails silently. The package therefore exports each such
vocabulary as constants (`ActionKindControl = "control"`), turning the typo into
a compile error at the consumer's site.

The rule draws the boundary: a closed set is published when its values survive
into the AST as string data, and not when the AST encodes the set as types. The
Step `action` values get no constants because no consumer compares against one
as a string — the discriminator becomes the variant struct, and a type switch
already gives the compiler something to check. A constant with no comparison
site protects nothing. Consumers that render or re-emit a Step do need the
source spelling back, so the `Step` interface carries it as an `Action() string`
method each variant implements: the discriminator travels with the variant, and
a new variant without one fails to compile rather than surfacing as a missing
entry in some consumer's table.

Two properties keep the constants honest. The bind pass validates against the
same identifiers it publishes, so the enforced set and the published set cannot
drift. And they are plain untyped constants, not a defined string type: Go's
defined types do not close a value set (any string constant converts), so a
`type ActionKind string` would promise a check the compiler does not perform,
while untyped constants compare against `StringLit.Value` without conversion.
Where a vocabulary is open — extension Result types are legal beyond the
spec-defined five — the constants are published vocabulary for comparison, not a
membership check, and the bind pass does not enforce membership.

## The typed AST normalizes empty spellings

JSON offers more than one spelling of "nothing here": an optional member can be
absent, an empty collection (`"catch": []`), or an explicit null the data model
documents as equivalent to absence (`"concurrency": null`). The typed AST
records meaning, so these spellings bind identically: a member whose spelling
contributes no value is indistinguishable from an absent one. The line is
content, not syntax. An authored value always survives, even when it restates a
default — `"wait": true` is a value, and `Completion.Wait` keeps it — and an
explicit null that means something different from absence (`retryable`,
`previous`) is likewise a value, carried by its pair type. Only the spellings of
nothing collapse: `concurrency` carries a cap value or nothing at all.

Two consequences follow. Re-emission from the typed AST is canonical: a tool
that reads a definition and writes it back emits one spelling of nothing —
absence — the way gofmt emits one spelling of a program, and decision 8's
round-trip promise reads accordingly: "the same values in the same order" means
semantic values, and an edit tool's diff may additionally canonicalize an empty
spelling the author wrote. And the preservation machinery the rule forecloses
stays unbuilt: distinguishing written-empty from absent would put presence
machinery (a pointer-to-slice, a wrapper struct) on every optional collection,
taxing every consumer of the AST for a distinction none of them read. Go's own
nil-versus-empty slice distinction is not that machinery: it is invisible to
consumers, silently destroyed by ordinary operations, and no test would hold it.

The consumer that genuinely wants spelling fidelity is a formatter, or any
rewrite tool whose diff must be exactly its edit — and what such a tool needs is
member-level syntax the typed AST rightly discards: presence, document order,
the number lexeme as written, positions into the source. That layer already
exists as the decode pass's positioned value tree, where `"catch": []` is simply
a member holding an empty array. The tree is unexported today; when a
syntax-level tool becomes real, exporting it is the API addition, and the typed
AST stays a purely semantic model rather than becoming a partial syntactic one.
