package parser

import (
	"fmt"
	"strings"
)

// Severity ranks a diagnostic. Errors are exactly the specification's
// rejection set: a document with an error-severity diagnostic is not valid
// MWL. Warnings never affect whether a parse succeeds.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return fmt.Sprintf("Severity(%d)", int(s))
	}
}

// The error codes a Diagnostic.Code can carry, enumerated here in full. The
// error set is exactly the specification's rejection set.
//
// Codes emitted by the decode pass:
const (
	// CodeJSONSyntax reports that the document is not valid JSON.
	CodeJSONSyntax = "json-syntax"
	// CodeEmptyDocument reports a document with no content.
	CodeEmptyDocument = "empty-document"
	// CodeDuplicateMember reports a member name repeated within one object.
	CodeDuplicateMember = "duplicate-member"
	// CodeTrailingContent reports content after the root value.
	CodeTrailingContent = "trailing-content"
)

// Codes emitted by the bind pass:
const (
	// CodeWrongType reports a value of the wrong JSON kind for its field.
	CodeWrongType = "wrong-type"
	// CodeUnknownMember reports a member its containing shape does not
	// accept.
	CodeUnknownMember = "unknown-member"
	// CodeMissingMember reports an absent required member.
	CodeMissingMember = "missing-member"
	// CodeInvalidValue reports a value of the right kind that is not a
	// permitted value (an unknown action, an empty calls array, a
	// concurrency below 1).
	CodeInvalidValue = "invalid-value"
	// CodeExclusiveMembers reports a violation of an exactly-one rule
	// (a call's provider or flow, a Gather's two forms, a Sleep's for or
	// until).
	CodeExclusiveMembers = "exclusive-members"
	// CodeExpressionNotPermitted reports an expression embedded in a
	// structural field, which the specification requires rejecting.
	CodeExpressionNotPermitted = "expression-not-permitted"
	// CodeInvalidProviderURI reports a provider URI that is not a
	// well-formed name of the kind its position requires.
	CodeInvalidProviderURI = "invalid-provider-uri"
)

// Codes emitted by the check pass:
const (
	// CodeUnresolvedStep reports an entrypoint or next that names no Step
	// in its own Flow's steps map.
	CodeUnresolvedStep = "unresolved-step"
	// CodeUnresolvedFlow reports a flow name that resolves to no flows
	// entry on its lexical scope chain.
	CodeUnresolvedFlow = "unresolved-flow"
	// CodeFlowCycle reports a cycle in the flow-reference graph.
	CodeFlowCycle = "flow-cycle"
)

// Warning codes. This is the complete set of warnings the parser emits; it
// is deliberately limited to findings that arise as by-products of parsing
// itself. The parser is not a complete linter (see the package
// documentation).
const (
	// CodeShadowedFlow reports a flows entry that shadows a declaration
	// of the same name in an enclosing Flow. Shadowing is legal; the
	// nearest declaration wins at every call site within the inner Flow.
	CodeShadowedFlow = "shadowed-flow"
	// CodeCatalogOutsidePrefix reports a failure-catalog entry — a closed
	// code, or an open sub-prefix other than the lone "*" — whose kind or
	// codePrefix segment disagrees with the provider's own. Legal, but
	// rarely what the author meant: the catalog constrains the provider's
	// emissions, and those sit under its own slice of the taxonomy.
	CodeCatalogOutsidePrefix = "catalog-outside-prefix"
	// CodeStructuralUnknownParameter reports a structural name matching
	// no key of its phase's parameters top-level properties. Legal,
	// because a parameter schema may admit members in other ways, but the
	// declaration marks nothing.
	CodeStructuralUnknownParameter = "structural-unknown-parameter"
)

// severityOf returns the severity a code always carries: severity is a
// property of the code, not of the emission site. The warning cases are the
// complete warning list; every other code is an error.
func severityOf(code string) Severity {
	switch code {
	case CodeShadowedFlow, CodeCatalogOutsidePrefix, CodeStructuralUnknownParameter:
		return SeverityWarning
	default:
		return SeverityError
	}
}

// Position is a location in the source. Line and Column are 1-based; Column
// counts bytes within the line. Filename is the label supplied by
// WithFilename and may be empty.
type Position struct {
	Filename string
	Offset   int
	Line     int
	Column   int
}

func (p Position) String() string {
	if p.Filename == "" {
		return fmt.Sprintf("%d:%d", p.Line, p.Column)
	}
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}

// Diagnostic is one finding from a parse.
type Diagnostic struct {
	Severity Severity
	Code     string // the kind of finding, from the Code constants above
	Pos      Position
	Path     string // dotted path to the finding, e.g. "steps.greet.catch[0]"; empty when none applies
	Message  string
}

func (d Diagnostic) String() string {
	var b strings.Builder
	b.WriteString(d.Pos.String())
	b.WriteString(": ")
	b.WriteString(d.Severity.String())
	b.WriteString(": ")
	b.WriteString(d.Message)
	if d.Code != "" {
		fmt.Fprintf(&b, " [%s]", d.Code)
	}
	return b.String()
}

// Diagnostics is the ordered list of findings from a parse.
type Diagnostics []Diagnostic

// HasErrors reports whether any diagnostic has error severity.
func (ds Diagnostics) HasErrors() bool {
	for _, d := range ds {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// pathSeg is one segment of a document path: an object member name or an
// array index.
type pathSeg struct {
	key   string
	idx   int
	isIdx bool
}

// renderPath renders segments as a dotted path with bracketed indexes,
// e.g. "steps.greet.catch[0]". A non-empty finalKey is appended as a last
// member segment.
func renderPath(segs []pathSeg, finalKey string) string {
	var b strings.Builder
	writeKey := func(key string) {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(key)
	}
	for _, s := range segs {
		if s.isIdx {
			fmt.Fprintf(&b, "[%d]", s.idx)
		} else {
			writeKey(s.key)
		}
	}
	if finalKey != "" {
		writeKey(finalKey)
	}
	return b.String()
}

// reporter accumulates a pass's diagnostics and the document path of its
// walk. Every pass carries one: the decoder and binder embed it directly,
// the checkers embed it beside their own state.
type reporter struct {
	diags Diagnostics
	path  []pathSeg
}

func (r *reporter) pushKey(name string) { r.path = append(r.path, pathSeg{key: name}) }
func (r *reporter) pushIdx(i int)       { r.path = append(r.path, pathSeg{idx: i, isIdx: true}) }
func (r *reporter) pop()                { r.path = r.path[:len(r.path)-1] }

// report records one finding at the walk's current path; the severity is
// the code's own.
func (r *reporter) report(code string, pos Position, format string, args ...any) {
	r.reportPath(code, pos, renderPath(r.path, ""), format, args...)
}

// reportPath records one finding at an explicitly rendered path, for sites
// where the path is not the walk's current one: a final member segment
// appended, or an edge recorded on an earlier walk.
func (r *reporter) reportPath(code string, pos Position, path, format string, args ...any) {
	r.diags = append(r.diags, Diagnostic{
		Severity: severityOf(code),
		Code:     code,
		Pos:      pos,
		Path:     path,
		Message:  fmt.Sprintf(format, args...),
	})
}
