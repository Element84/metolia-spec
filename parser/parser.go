package parser

import "sort"

// Option configures a parse.
type Option func(*options)

type options struct {
	filename string
}

// WithFilename sets the label diagnostics use in positions, e.g. the
// "flow.json" in "flow.json:42:10". The parser never reads the file; the
// name is only a label, and without it positions render as "42:10".
func WithFilename(name string) Option {
	return func(o *options) { o.filename = name }
}

// ParseFlow parses a workflow definition: a JSON document whose root is a
// Flow object. It runs the three passes — decode, bind, check — collecting
// diagnostics across all of them, returned in document order.
//
// The returned Flow is nil when any diagnostic has error severity; a
// non-nil Flow satisfies every structural and document-local rule the
// specification states, and may be accompanied by warnings. The error set
// is exactly the specification's rejection set. Provider URIs are checked
// as well-formed names, not resolved; validating against provider
// definitions is a separate concern.
func ParseFlow(src []byte, opts ...Option) (*Flow, Diagnostics) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	tree, diags := decode(src, o.filename)
	if tree == nil {
		return nil, diags
	}

	flow, bindDiags := bind(tree)
	diags = append(diags, bindDiags...)

	// The check pass assumes an AST that bound without errors; with
	// errors present the parse fails regardless, so it is skipped.
	if !diags.HasErrors() {
		diags = append(diags, check(flow)...)
	}

	sortByPosition(diags)
	if diags.HasErrors() {
		return nil, diags
	}
	return flow, diags
}

// sortByPosition orders diagnostics by source offset, so findings arrive in
// document order regardless of which pass produced them. The sort is
// stable: diagnostics at the same offset keep their emission order.
func sortByPosition(diags Diagnostics) {
	sort.SliceStable(diags, func(i, j int) bool {
		return diags[i].Pos.Offset < diags[j].Pos.Offset
	})
}
