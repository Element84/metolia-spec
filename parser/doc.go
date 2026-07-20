// Package parser parses Metolia Workflow Language (MWL) definition
// documents: workflow definitions (Flow documents) and provider definitions.
//
// The parser is catalog-free and does no file I/O: callers hand it source
// bytes, and it produces a typed representation plus a list of diagnostics.
// It validates structure (each object the right shape) and the document-local
// static checks the specification defines; it does not resolve provider URIs
// against a catalog, evaluate JSON Schemas, or parse expression bodies.
//
// The parser emits a small, closed set of warnings that arise as by-products
// of parsing itself. It is not a complete linter: findings that need
// knowledge beyond the definition, such as provider catalogs or expression
// analysis, belong to separate tools built on top of this package.
package parser
