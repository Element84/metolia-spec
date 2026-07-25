---
title: "The parser"
weight: 10
---

A zero-dependency Go package, part of the
[specification repository](https://github.com/Element84/metolia-spec), that
parses MWL definition documents: `parser.ParseFlow` for a workflow definition,
`parser.ParseProvider` for a provider definition. Callers hand it source bytes;
it returns a typed tree plus a list of diagnostics in document order. It is
catalog-free and does no file I/O.

## The contract

`ParseFlow` and `ParseProvider` return a nil tree when any diagnostic has error
severity, and the error set is exactly the specification's rejection set: a
document with an error is not valid MWL, and every valid document parses. A
non-nil tree satisfies every structural and document-local rule the
specification states, may be accompanied by warnings, and defaults nothing: a
nil field means the member was absent from the document.

The trees themselves, the Flow AST and the Provider AST, are documented field by
field in the [AST explorer](../ast-explorer/).

## What it deliberately does not do

The parser stops at the document boundary: provider URIs are checked as
well-formed names but never resolved against a catalog, embedded JSON Schemas
(`parameters`, `metadata`) are retained as value trees but never evaluated, and
expression bodies are recognized but never parsed. Checks that need knowledge
beyond the definition belong to separate tools built on top of this package.
