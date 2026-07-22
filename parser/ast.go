package parser

// This file defines the AST shapes both document kinds share: reference
// names, positioned string literals, the expression node with its pair
// types, and the published Result-type vocabulary. Every node records the
// position of the JSON value it was bound from.

// Name is a string that refers to a named thing — a Step (entrypoint, next)
// or a named Flow (a call's flow target) — as written at the reference site.
type Name struct {
	Value string
	Pos   Position
}

// StringLit is a literal string with its position, used where a list of
// plain strings needs per-entry positions (failure-matcher patterns and
// type names).
type StringLit struct {
	Value string
	Pos   Position
}

// The spec-defined Result types. Published for consumer comparisons against
// type-bearing fields (RaiseResult.Type, FailureMatcher.Types); the
// vocabulary is open — extension Result types are legal beyond these five —
// so the constants are not a membership check, and the bind pass does not
// enforce one.
const (
	ResultTypeSuccess      = "success"
	ResultTypeError        = "error"
	ResultTypeCancellation = "cancellation"
	ResultTypeTimeout      = "timeout"
	ResultTypeSkipped      = "skipped"
)

// Expr is one embedded expression: a string whose entire content one
// delimiter pair set off. Body is the text between the delimiters with
// surrounding whitespace removed. The parser does not interpret the body;
// consumers that evaluate or analyze expressions parse it themselves.
type Expr struct {
	Body string
	Pos  Position
}

// OrExpr is a field that accepts either a literal of type T or an
// expression. Exactly one of Literal and Expr is set; the bind pass
// guarantees the invariant.
type OrExpr[T any] struct {
	Literal *T
	Expr    *Expr
}

// RetryableField is a retryable envelope field: a boolean, an explicit
// null (an assertion of "no assertion", distinct from omitting the field),
// or an expression. Exactly one of Literal, Null, and Expr is set.
type RetryableField struct {
	Literal *bool
	Null    bool
	Expr    *Expr
}

// PreviousField is a previous envelope field: a literal object, an explicit
// null (severing the failure chain, distinct from omitting the field), or
// an expression. Exactly one of Literal, Null, and Expr is set.
type PreviousField struct {
	Literal *Value
	Null    bool
	Expr    *Expr
}
