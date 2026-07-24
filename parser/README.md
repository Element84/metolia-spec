# parser

A zero-dependency Go parser for Metolia Workflow Language (MWL) definition
documents: workflow definitions (`ParseFlow`) and provider definitions
(`ParseProvider`). Callers hand it source bytes; it returns a typed tree plus a
list of diagnostics. It is catalog-free and does no file I/O.

This page describes the architecture: what the passes are, what each produces,
and which files belong to which. The decisions behind the architecture — what
was considered and why the alternatives lost — live in [DESIGN.md](DESIGN.md).

## The pipeline

Every parse runs the same three passes, each consuming the previous pass's
output. There is one list of diagnostics for the whole parse: each pass appends
the problems it detects, and the finished list is sorted into document order
before it is returned, so diagnostics read in source order regardless of which
pass produced them.

```mermaid
flowchart LR
    src["source bytes"] --> decode
    decode -- "positioned *Value tree" --> bind
    bind -- "typed AST" --> check
    check --> out["*Flow / *Provider<br>+ Diagnostics"]
```

### decode

`decode.go` turns the source bytes into a tree of positioned `Value` nodes
(`value.go`): every JSON value, tagged with its byte offset, line, and column.
This is the pass that rejects malformed JSON, empty documents, duplicate object
members, and trailing content. Everything downstream works on this tree; no
later pass touches the bytes.

### bind

The bind pass (`bind.go`, plus `bind_flow.go` and `bind_provider.go` per
document kind) walks the value tree and builds the typed AST (`ast.go`,
`ast_flow.go`, `ast_provider.go`). This is the JSON-to-Go boundary, and shape is
enforced as it is crossed: each value must be the JSON kind its field requires
(an object for `steps`, a string for `next`); each object may carry only the
members its shape accepts, and must carry the required ones; the exactly-one
pairs (a call's `provider` or `flow`, a Gather's two forms, a Sleep's `for` or
`until`) must have exactly one side present; and expressions may appear only in
fields that permit them. A violation becomes a diagnostic and binding moves on,
so one parse reports every shape problem in the document, not just the first.

### check

The check pass (`check_flow.go` and `check_provider.go`) runs the rules that
span the bound document: every `entrypoint` and `next` resolves to a step, every
flow name resolves on its lexical scope chain, and the flow-reference graph is
acyclic; for providers, the failure catalog and structural declarations are
examined. The parser's warnings — flow shadowing, catalog entries outside the
provider's prefix, structural names matching no parameter — all come from this
pass. It assumes an AST that bound without errors and is skipped otherwise.

## The contract

`ParseFlow` and `ParseProvider` return a nil tree when any diagnostic has error
severity, and the error set is exactly the specification's rejection set: a
document with an error is not valid MWL, and every valid document parses. A
non-nil tree satisfies every structural and document-local rule the
specification states, may be accompanied by warnings, and defaults nothing — a
nil field means the member was absent from the document.

## File map

| Files                                         | Role                                                                                 |
| --------------------------------------------- | ------------------------------------------------------------------------------------ |
| `parser.go`                                   | `ParseFlow` / `ParseProvider`: run the passes, sort the diagnostics                  |
| `decode.go`, `value.go`                       | the decode pass and the positioned JSON tree it produces                             |
| `bind.go`, `bind_flow.go`, `bind_provider.go` | the bind pass: shared field shapes, then one file per document kind                  |
| `check_flow.go`, `check_provider.go`          | the check pass, one file per document kind                                           |
| `ast.go`, `ast_flow.go`, `ast_provider.go`    | the typed AST: shared shapes, the Flow tree, the Provider tree                       |
| `diagnostic.go`                               | severities, the full code enumeration, positions, the reporter every pass walks with |
| `expr.go`                                     | `ExprBody`: the specification's expression-embedding rule                            |
| `doc.go`                                      | package documentation                                                                |

## What it deliberately does not do

The parser stops at the document boundary: provider URIs are checked as
well-formed names but never resolved against a catalog, embedded JSON Schemas
(`parameters`, `metadata`) are retained as value trees but never evaluated, and
expression bodies are recognized but never parsed. Checks that need knowledge
beyond the definition belong to separate tools built on top of this package;
[DESIGN.md](DESIGN.md) records why each boundary sits where it does.
