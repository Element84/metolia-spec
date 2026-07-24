---
title: "Tooling"
weight: 25
singleFileExclude: true
cascade:
  type: docs
---

Documentation for the Go tooling that ships alongside the specification, with a
linter and validator to follow. Where the [Reference](/reference/) defines the
language itself, this section documents the tools that read it.

<!-- dprint-ignore-start -->
{{< cards >}}
  {{< card link="parser" title="The parser" subtitle="The Go package that turns a definition document into a typed tree plus diagnostics." >}}
  {{< card link="ast-explorer" title="AST explorer" subtitle="The parser's companion reference: the Flow and Provider trees it returns, field by field." >}}
  {{< card link="cli" title="The MWL CLI" subtitle="The parser at the command line: check a document, read its diagnostics, emit its AST." >}}
{{< /cards >}}
<!-- dprint-ignore-end -->
