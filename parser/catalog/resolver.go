// Package catalog resolves a provider URI to its provider definition document.
//
// A Flow names providers by URI but does not embed their contracts. A Resolver
// is how a platform supplies a referenced provider's definition to the parser,
// wherever it keeps them (a directory, a registry, a URL).
package catalog

// Resolver returns the raw bytes of the provider definition named by uri.
// Decoding and validating them are the parser's job, not the resolver's.
type Resolver interface {
	Resolve(uri string) ([]byte, error)
}
