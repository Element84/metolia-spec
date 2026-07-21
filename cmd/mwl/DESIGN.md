# CLI design decisions

The companion to `parser/DESIGN.md` for the `mwl` command-line tool. Same
charter: a working document, not a formal decision log. Entries record what was
decided and why and are revised as the design changes; a decision that hardens
may later graduate to its own record.

## Decisions at a glance

1. [One binary, its own module](#one-binary-its-own-module)—a single `mwl`
   binary services the repository; the module lives at `cmd/mwl` so the binary
   inherits that name, joins the workspace, and depends only on the standard
   library and the parser.
2. [`parse` names the document kind](#parse-names-the-document-kind)—
   `mwl parse flow <file>` and `mwl parse provider <file>`; `-` reads stdin; the
   CLI does the file I/O the parser deliberately does not.
3. [Each stream has one contract](#each-stream-has-one-contract)—stderr carries
   text diagnostics for humans, always; stdout carries the requested JSON
   artifact or nothing; no flag changes what a stream is for.
4. [Exit codes separate the verdict from the tool](#exit-codes-separate-the-verdict-from-the-tool)
   —0 parsed, 1 rejected, 2 the tool itself failed; `--strict` promotes warnings
   to a failing exit.
5. [The `--ast` envelope is the one machine artifact](#the---ast-envelope-is-the-one-machine-artifact)
   —a single JSON object carrying the AST and the diagnostics together, one
   schema for success and failure alike.
6. [The AST encoding makes the parser's decisions visible](#the-ast-encoding-makes-the-parsers-decisions-visible)
   —discriminated Steps, tagged literal-or-expression pairs, tagged value trees
   with classified string leaves, positions on every node.
7. [Golden files document the shape](#golden-files-document-the-shape)—
   committed fixture-to-output pairs are the executable specification of the
   `--ast` output; a formal JSON Schema is deferred behind a named trigger.
8. [Deferred features](#deferred-features)—a control-flow outline view, MWL
   re-emission, and a diagnostics-only JSON flag are all designed or sketched
   and deliberately not built.

## One binary, its own module

A single CLI services anything in this repository that needs one: parsing today,
validation and a local runtime later as those layers appear. Separate binaries
per concern would multiply installation and discovery costs for no isolation
benefit; subcommands are the natural growth axis.

The module lives at `cmd/mwl` (module
`github.com/Element84/metolia-spec/cmd/mwl`, `use ./cmd/mwl` in `go.work`). The
nested directory is not organizational: `go install` names the binary after the
main package's directory, so `cmd/mwl` is what makes the installed tool `mwl`
rather than `cmd`.

A separate module keeps the parser's dependency posture intact—nothing is added
to `parser/go.mod` for the CLI's sake. The CLI itself also carries no
third-party dependencies for now: the surface is two subcommands and two flags,
comfortably within the standard library's `flag` package and a small
hand-written dispatch. A CLI framework (cobra) earns adoption when `validate` or
`run` arrive with real flag surfaces of their own; adopting one later changes
nothing user-visible if command names stay stable.

## `parse` names the document kind

```
mwl parse flow <file>       parse a workflow definition
mwl parse provider <file>   parse a provider definition
```

`parse` is deliberately not `validate`: the parser checks structure and
document-local rules—the specification's rejection set plus by-product
warnings—and a future provider-aware validator is a different, catalog-bearing
layer. Naming today's command `parse` keeps `validate` free for that layer.

The document kind is a subcommand rather than a flag or sniffed content: the two
kinds have different schemas, different entry points, and a caller always knows
which one it holds. `parse provider` ships when the parser grows
`ParseProvider`; the surface is designed for both from the start.

The positional argument is a file path, or `-` for stdin, so the tool composes
in pipelines (`git show HEAD:flow.json | mwl parse flow -`). The parser takes
bytes and never touches files (`parser/DESIGN.md`, "The API takes bytes, not
files"); the CLI is the layer that reads the file or the pipe and passes the
name along via `parser.WithFilename` for diagnostic labels. Stdin input gets no
filename label.

## Each stream has one contract

| Stream | Carries                                                     |
| ------ | ----------------------------------------------------------- |
| stderr | text diagnostics for humans; silent when the parse is clean |
| stdout | the requested machine artifact (`--ast`), otherwise nothing |

Neither contract varies by flag. Text renders as
`file:line:col: severity: message [code]`—the format terminals and editors
already recognize—one finding per line, in document order.

With `--ast`, diagnostics appear twice: text on stderr and data inside the
stdout envelope. This is one set of findings rendered once per audience, not
redundancy. A pipeline has both audiences at once: in
`mwl parse flow x.json --ast | jq .ast`, jq consumes stdout while the human
watching the terminal still sees warnings on stderr, because a pipe connects
only stdout. Either copy is trivially dismissed (`2>/dev/null` for pure data,
`>/dev/null` for a pure check).

## Exit codes separate the verdict from the tool

| Code | Meaning                                                           |
| ---- | ----------------------------------------------------------------- |
| 0    | the document parsed; warnings may have printed                    |
| 1    | the document was rejected (error diagnostics, no AST)             |
| 2    | the tool failed to run: unreadable input, unknown flag, bad usage |

The default treats warnings as exit 0 because the specification treats MAY-warn
findings as legal documents; a checker that fails legal input by default
contradicts the language it checks. `--strict` opts a caller into the higher
bar: any diagnostic at all exits 1.

Exit 2 exists so automation can distinguish "invalid MWL" from "the check never
ran"—a typo'd path in CI must not read as a rejected document.

## The `--ast` envelope is the one machine artifact

```json
{
  "filename": "flow.json",
  "ast": { "...": "..." },
  "diagnostics": [
    {
      "severity": "warning",
      "code": "shadowed-flow",
      "path": "flows.Greet",
      "message": "...",
      "pos": { "line": 12, "col": 5, "offset": 214 }
    }
  ]
}
```

One schema always: on a failed parse `ast` is `null` and `diagnostics` carries
the errors; on a clean parse `diagnostics` is empty. `filename` is hoisted to
the envelope—it is identical for every position in a parse, so no per-node
repetition—and absent for stdin input. `severity` is the string form (`"error"`,
`"warning"`); `path` is the dotted member path and may be empty where none
applies.

The envelope's stability posture: **unstable before v1**. The shape may change;
consumers script against it accordingly. See
[Golden files document the shape](#golden-files-document-the-shape) for how
changes stay visible.

The envelope is a **machine artifact**. Its audience is tooling, so its shapes
mirror the parser's AST types faithfully and make no attempt to resemble the
source document or to read well for a human. This is a deliberate boundary, not
an omission: readability accommodations (position stripping, extra uniformity
tags beyond what the AST types carry) were tried and removed, because each one
made the output a half-measure between the internal view and the source view, at
home in neither. Where the grammar forced a construct (a Match case's `when`
admits only an expression), the encoding carries no discriminator—there was no
decision to record. The human-facing rendering of "what the parser understood"
is MWL re-emission (see [Deferred features](#deferred-features)): a document in
the language the author wrote, comparable to the source directly.

## The AST encoding makes the parser's decisions visible

Go's default marshaling cannot express the AST—`Step` is an interface, the pair
types would leak Go field names, and the discriminator lives in the type rather
than a field—so the encoding is hand-designed. The organizing principle: the
JSON shows every decision the parser made while binding, in the source's own
vocabulary.

The implementation derives the bulk of the encoding from the AST structs by
reflection: field names are camelCased, nil pointers and empty slices are
omitted (which collapses the pair types to their one-key form on its own), and
unexported fields are invisible. Hand-written code is reserved for the decisions
the struct shapes cannot express: the action discriminator, position rendering,
value trees with classified leaves, the explicit-null pair types, and the key
rules below. Two consequences follow. A field added to an AST struct appears in
the output with no encoder change, surfacing as a golden-file diff for review,
and a new Step variant fails tests until its discriminator is mapped. And
because keys derive from Go field names, renaming an AST field changes the
output contract—the goldens catch it, but a rename is an encoding change, not an
internal refactor.

- **Positions** are `{ "line": 6, "col": 5, "offset": 112 }` on every node, keys
  always in that order. Line and col are 1-based, col counts bytes; offset is
  0-based from the start of the source. Placement is uniform: `pos` is always
  the **last** member of the object it locates, so every object reads
  content-first; `namePos` is the exception in name-bearing entries, sitting
  immediately after the `name` it locates rather than trailing a large sibling
  subtree.
- **Steps** are `{ "name", "namePos", "step" }` entries in document order (the
  AST's ordered-collection decision surfacing), and each `step` object carries
  `"action": "Call"` and so on—the discriminator re-attached as the field it was
  in the source. Field keys are the source's camelCase keys.
- **Name references** (`entrypoint`, `next`, a flow target's name) are
  `{ "value": "done", "pos": ... }`.
- **Literal-or-expression pairs** (`OrExpr`) encode as a one-key object whose
  present key is the verdict: `{ "literal": 5 }` or
  `{ "expr": { "body": "vars.n + 1", "pos": ... } }`. The `retryable` and
  `previous` fields, where the spec distinguishes an explicit `null` from an
  absent field, add a third arm: `{ "null": true }`. A wrapper object rather
  than a bare JSON `null` because many JSON consumers cannot distinguish a key
  holding `null` from a missing key; the encoding never uses bare `null` as
  data, so written-null and absent can never blur.
- **Absent optional fields are absent**—no defaults filled in, so the output
  states what the document says, not what a runtime would assume.
- **Value trees** (template fields: `input`, `output`, `value`, `with` objects,
  `assign` entries) are tagged nodes, `{ "kind": "...", "pos": ..., ... }`,
  because mirroring the source JSON directly would make a wrapped string leaf
  indistinguishable from a source object that happens to have the same member
  names. Per kind: `bool` and `string` carry `"value"`; `number` carries
  `"value"` as a JSON **string** holding the literal text exactly as written in
  the source, never a re-decoded float; `object` carries `"members"` as
  `{ "name", "namePos", "value" }` entries in document order; `array` carries
  `"elements"`; `null` carries nothing further. String leaves inside template
  fields are classified at emission with the same rule the parser applies
  everywhere (`parser.ExprBody`): an expression-shaped leaf becomes
  `{ "kind": "expr", "body": ..., "pos": ... }` rather than a `string` node. The
  body is the delimiter-stripped, whitespace-trimmed text; the raw source string
  is not preserved, matching the specification's declaration that the
  surrounding whitespace is insignificant.

A worked fragment—the first case of `300-match-routing`'s Match step:

```json
{
  "name": "route",
  "namePos": { "line": 6, "col": 5, "offset": 148 },
  "step": {
    "action": "Match",
    "cases": [
      {
        "when": {
          "expr": {
            "body": "match.input.status == 'approved' && match.input.amount > 1000.0",
            "pos": { "line": 10, "col": 19, "offset": 240 }
          }
        },
        "output": {
          "kind": "string",
          "value": "manual-review",
          "pos": { "line": 11, "col": 21, "offset": 330 }
        },
        "next": {
          "value": "done",
          "pos": { "line": 12, "col": 19, "offset": 360 }
        },
        "pos": { "line": 9, "col": 9, "offset": 210 }
      }
    ],
    "pos": { "line": 6, "col": 14, "offset": 157 }
  }
}
```

(Offsets illustrative.) The `when` string became an `expr` with the delimiters
stripped; the `output` string stayed a `string`. In the source these are the
same JSON kind—the AST shows which one the parser decided each was, which is the
class of question the dump exists to answer.

## Golden files document the shape

The CLI's test suite commits fixture-to-output pairs: a handful of
representative definitions (a linear chain, a Match, both Gather forms,
expression-bearing fields, a failing parse) each beside the exact envelope
`--ast` produces for it. Tests run the CLI and diff against the golden file.

This is the middle rung between a prose description that drifts silently and a
formal JSON Schema whose authoring weight (the recursive Step variants alone
rival the flow schema) is premature for a shape marked unstable:

- A committed expected-output file beside a readable fixture is executable
  documentation—a consumer learns the shape from worked examples rather than a
  grammar.
- The shape cannot drift silently: changing the encoding fails a test, and
  updating the golden file is a reviewable diff that records exactly what
  changed—the same discipline this repository applies to the spec, at
  test-weight.
- Formalization has a trigger instead of a deadline: when the shape stabilizes
  or a real external consumer appears, publish `static/v0.1/ast/schema.json`
  beside the flow and provider schemas, wire corpus validation into
  `check-schemas.sh`, and the golden files become its fixtures.

## Deferred features

- **Control-flow outline** (`--graph`): a colored, tree-connector rendering of
  the step graph was designed in full and deferred—the JSON envelope already
  carries everything it would show, and the rendering code is a real maintenance
  surface. It returns if actual usage keeps asking for it.
- **MWL re-emission**: serializing the AST back to a definition document—the
  true round trip, and the seed of a formatter. Kept out of `parse`; byte
  fidelity is a non-goal the parser's design already declares, so this arrives
  as its own command with its own contract. First in line among the deferrals:
  it is the human-facing rendering of a parse result, the counterpart to the
  machine-only envelope.
- **Diagnostics-only JSON**: a flag emitting just the diagnostics array.
  `--ast | jq .diagnostics` covers the need today; the flag is cheap to add if a
  consumer objects to paying for the AST.
- **`parse provider`**: designed into the surface, wired when the parser grows
  `ParseProvider`.
