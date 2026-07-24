package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// This file holds the machinery both bind passes share: the binder, a
// reporter carrying the generic field-shape helpers the flow and provider
// grammars both draw on.

type binder struct {
	reporter
}

func (b *binder) unknownMember(m Member, in string) {
	b.report(CodeUnknownMember, m.NamePos, "unknown member %q in %s", m.Name, in)
}

func (b *binder) missingMember(pos Position, what string) {
	b.report(CodeMissingMember, pos, "%s", what)
}

// requireString reports a wrong-type error unless v is a string.
func (b *binder) requireString(v *Value, what string) (string, bool) {
	if v.Kind != KindString {
		b.report(CodeWrongType, v.Pos, "%s must be a string, not %s", what, v.Kind)
		return "", false
	}
	return v.Str, true
}

// bindName binds a structural reference (entrypoint, next, a flow name): a
// string that does not accept an expression.
func (b *binder) bindName(v *Value, what string) Name {
	s, ok := b.requireString(v, what)
	if !ok {
		return Name{Pos: v.Pos}
	}
	if _, isExpr := ExprBody(s); isExpr {
		b.report(CodeExpressionNotPermitted, v.Pos,
			"%s is structural and does not accept an expression", what)
	}
	return Name{Value: s, Pos: v.Pos}
}

func (b *binder) bindComment(v *Value) string {
	s, _ := b.requireString(v, "comment")
	return s
}

// Shared field shapes ------------------------------------------------------

// bindObjectSchema binds an embedded JSON Schema field (parameters,
// metadata) that must describe an object at its top level. The subtree is
// retained as-is; the one check is the declared "type": "object".
func (b *binder) bindObjectSchema(v *Value, what string) *Value {
	if v.Kind != KindObject {
		b.report(CodeWrongType, v.Pos, "%s must be a JSON Schema object, not %s", what, v.Kind)
		return nil
	}
	t := v.Field("type")
	switch {
	case t == nil:
		b.missingMember(v.Pos, fmt.Sprintf(`a %s schema must declare "type": "object"`, what))
	case t.Kind != KindString || t.Str != "object":
		b.report(CodeInvalidValue, t.Pos, `a %s schema must declare "type": "object"`, what)
	}
	return v
}

// providerURIKind reports which provider kind uri's type segment names,
// from its prefix alone; whether the rest of the URI is well-formed is
// validProviderURI's question.
func providerURIKind(uri string) (string, bool) {
	for _, kind := range []string{"call", "middleware"} {
		if strings.HasPrefix(uri, "mwl:provider."+kind+"/") {
			return kind, true
		}
	}
	return "", false
}

// invalidProviderURI reports a URI that is not a well-formed name of the
// given kind; the message is shared by every site that checks one.
func (b *binder) invalidProviderURI(pos Position, uri, kind string) {
	b.report(CodeInvalidProviderURI, pos,
		"%q is not a well-formed %s-provider URI; expected mwl:provider.%s/<namespace>/<name>",
		uri, kind, kind)
}

// validProviderURI reports whether uri is a well-formed provider URI of the
// given kind ("call" or "middleware"): mwl:provider.<kind>/ followed by two
// or more segments of [A-Za-z0-9._-], none of which is "." or "..".
func validProviderURI(uri, kind string) bool {
	rest, ok := strings.CutPrefix(uri, "mwl:provider."+kind+"/")
	if !ok {
		return false
	}
	segs := strings.Split(rest, "/")
	if len(segs) < 2 {
		return false
	}
	for _, seg := range segs {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for i := 0; i < len(seg); i++ {
			c := seg[i]
			switch {
			case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
				c == '.', c == '_', c == '-':
			default:
				return false
			}
		}
	}
	return true
}

// Expression-bearing field shapes -----------------------------------------

func (b *binder) bindAssign(v *Value) []Member {
	if v.Kind != KindObject {
		b.report(CodeWrongType, v.Pos, "assign must be an object, not %s", v.Kind)
		return nil
	}
	return v.Object
}

func (b *binder) bindWith(v *Value) *OrExpr[[]Member] {
	switch v.Kind {
	case KindObject:
		lit := v.Object
		return &OrExpr[[]Member]{Literal: &lit}
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return &OrExpr[[]Member]{Expr: &Expr{Body: body, Pos: v.Pos}}
		}
		b.report(CodeWrongType, v.Pos, "with must be an object or an expression")
	default:
		b.report(CodeWrongType, v.Pos, "with must be an object or an expression, not %s", v.Kind)
	}
	return nil
}

// bindExpr binds a field that requires an expression, such as a cases
// clause's when.
func (b *binder) bindExpr(v *Value, what string) *Expr {
	s, ok := b.requireString(v, what)
	if !ok {
		return nil
	}
	body, isExpr := ExprBody(s)
	if !isExpr {
		b.report(CodeInvalidValue, v.Pos, "%s must be an expression", what)
		return nil
	}
	return &Expr{Body: body, Pos: v.Pos}
}

func (b *binder) bindBoolOrExpr(v *Value, what string) *OrExpr[bool] {
	switch v.Kind {
	case KindBool:
		lit := v.Bool
		return &OrExpr[bool]{Literal: &lit}
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return &OrExpr[bool]{Expr: &Expr{Body: body, Pos: v.Pos}}
		}
		b.report(CodeWrongType, v.Pos, "%s must be a boolean or an expression", what)
	default:
		b.report(CodeWrongType, v.Pos, "%s must be a boolean or an expression, not %s", what, v.Kind)
	}
	return nil
}

func (b *binder) bindIntOrExpr(v *Value, what string) (OrExpr[int64], bool) {
	switch v.Kind {
	case KindNumber:
		n, ok := intFromLexeme(v.Number)
		if !ok {
			b.report(CodeInvalidValue, v.Pos, "%s must be an integer", what)
			return OrExpr[int64]{}, false
		}
		return OrExpr[int64]{Literal: &n}, true
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return OrExpr[int64]{Expr: &Expr{Body: body, Pos: v.Pos}}, true
		}
		b.report(CodeWrongType, v.Pos, "%s must be an integer or an expression", what)
	default:
		b.report(CodeWrongType, v.Pos, "%s must be an integer or an expression, not %s", what, v.Kind)
	}
	return OrExpr[int64]{}, false
}

func (b *binder) bindStringOrExpr(v *Value, what string) (OrExpr[string], bool) {
	s, ok := b.requireString(v, what)
	if !ok {
		return OrExpr[string]{}, false
	}
	if body, isExpr := ExprBody(s); isExpr {
		return OrExpr[string]{Expr: &Expr{Body: body, Pos: v.Pos}}, true
	}
	return OrExpr[string]{Literal: &s}, true
}

// bindResultType binds a type envelope field: a string-or-expression that,
// as a literal, must name a non-success Result type. Both constructed
// failures — a Raise result and an onFailure phase — carry it.
func (b *binder) bindResultType(v *Value) *OrExpr[string] {
	x, ok := b.bindStringOrExpr(v, "type")
	if !ok {
		return nil
	}
	if x.Literal != nil && *x.Literal == ResultTypeSuccess {
		b.report(CodeInvalidValue, v.Pos, "type must be a non-success Result type")
	}
	return &x
}

func (b *binder) bindRetryable(v *Value) *RetryableField {
	switch v.Kind {
	case KindBool:
		lit := v.Bool
		return &RetryableField{Literal: &lit}
	case KindNull:
		return &RetryableField{Null: true}
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return &RetryableField{Expr: &Expr{Body: body, Pos: v.Pos}}
		}
		b.report(CodeWrongType, v.Pos, "retryable must be a boolean, null, or an expression")
	default:
		b.report(CodeWrongType, v.Pos, "retryable must be a boolean, null, or an expression, not %s", v.Kind)
	}
	return nil
}

func (b *binder) bindPrevious(v *Value) *PreviousField {
	switch v.Kind {
	case KindObject:
		return &PreviousField{Literal: v}
	case KindNull:
		return &PreviousField{Null: true}
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return &PreviousField{Expr: &Expr{Body: body, Pos: v.Pos}}
		}
		b.report(CodeWrongType, v.Pos, "previous must be an object, null, or an expression")
	default:
		b.report(CodeWrongType, v.Pos, "previous must be an object, null, or an expression, not %s", v.Kind)
	}
	return nil
}

// intFromLexeme parses a JSON number lexeme as an integer. Following JSON
// Schema's "integer", a number with a zero fractional part (1.0, 1e3)
// qualifies.
func intFromLexeme(s string) (int64, bool) {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f != math.Trunc(f) || math.Abs(f) > 1<<53 {
		return 0, false
	}
	return int64(f), true
}
