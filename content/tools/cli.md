---
title: "The MWL CLI"
weight: 30
---

The command-line tool for the Metolia Workflow Language. Its first subcommand,
`mwl parse`, checks that a workflow or provider definition parses and reports
diagnostics, optionally emitting the parsed AST as JSON: a thin surface over
[the parser](../parser/), one document in, that document's diagnostics out.
Subcommands are the tool's growth axis; the linter and validator to follow will
arrive as new ones beside `parse`.

## Install

From the root of the
[specification repository](https://github.com/Element84/metolia-spec):

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
(`{"filename", "ast", "diagnostics"}`; `ast` is `null` when the parse failed;
`filename` is omitted when reading stdin), so it pipes cleanly. The envelope's
shape is unstable before v1.

| Exit code | Meaning                                                             |
| --------- | ------------------------------------------------------------------- |
| `0`       | the document parsed (warnings may have printed)                     |
| `1`       | the document was rejected, or produced any diagnostic in `--strict` |
| `2`       | the tool itself failed to run                                       |
