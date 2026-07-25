# `mwl`

The command-line tool for the Metolia Workflow Language. It parses a workflow or
provider definition and reports diagnostics, optionally emitting the parsed AST
as JSON. The decisions behind the surface are recorded in
[DESIGN.md](DESIGN.md).

## Install

From the repository root:

```sh
go install ./cmd/mwl        # installs "mwl" to GOBIN
go build ./cmd/mwl          # or: build a local ./mwl binary
```

## Usage

```sh
mwl parse flow flow.json                     # check that a document parses
git show HEAD:flow.json | mwl parse flow -   # "-" reads stdin
mwl parse flow flow.json --ast | jq .ast     # emit the parsed AST as JSON
mwl parse flow flow.json --strict            # exit 1 on warnings too
mwl parse provider mock.v1.json              # same surface for a provider
                                             # definition
```

Diagnostics print to stderr as `file:line:col: severity: message [code]`; a
clean parse prints nothing. stdout carries only the `--ast` JSON envelope
(`{"filename", "ast", "diagnostics"}`; `ast` is `null` when the parse failed),
so it pipes cleanly.

Exit codes: `0` the document parsed (warnings may have printed), `1` the
document was rejected (or produced any diagnostic under `--strict`), `2` the
tool itself failed to run.

The envelope's shape is unstable before v1 and is documented by DESIGN.md and
the golden files under `testdata/golden/` — committed, exact outputs for
representative fixtures, which double as worked examples of the encoding.

## Development

The CLI is its own Go module in the repository workspace, depending only on the
standard library and the parser module (`../../parser`).

```sh
go test ./...                # run the suite, including golden comparisons
go test ./... -update        # regenerate golden files after a deliberate
                             # encoding change; review the diff
```

`testdata/` is excluded from dprint on purpose: the golden files bake in the
fixtures' byte offsets and line/column positions, so reformatting either side
would break the comparison.
