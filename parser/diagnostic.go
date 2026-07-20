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
	// Code is a stable, kebab-case identifier for the kind of finding,
	// e.g. "duplicate-member". The set of warning codes the parser emits
	// is small, closed, and documented.
	Code string
	Pos  Position
	// Path locates the finding within the document as a dotted member
	// path with bracketed array indexes, e.g. "steps.greet.catch[0]".
	// Empty when no path applies (a syntax error, for instance).
	Path    string
	Message string
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
