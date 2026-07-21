// The hand-designed JSON encoding of the parser's AST, per DESIGN.md ("The
// AST encoding makes the parser's decisions visible"). Go's default
// marshaling cannot express the AST -- Step is an interface, the pair types
// would leak Go field names, and the action discriminator lives in the type
// rather than a field -- so every shape is emitted explicitly. Key-order
// rules: an object's own "pos" is its last member; "namePos" immediately
// follows the "name" it locates; a one-key object's present key names the
// arm the author took, and bare JSON null never carries meaning.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Element84/metolia-spec/parser"
)

// writeEnvelope emits the {"filename", "ast", "diagnostics"} envelope,
// indented, with HTML escaping off so expression bodies read as written.
func writeEnvelope(w io.Writer, filename string, f *parser.Flow, ds parser.Diagnostics) error {
	env := newObj()
	if filename != "" {
		env.set("filename", filename)
	}
	if f != nil {
		env.set("ast", encFlow(f))
	} else {
		env.set("ast", nil)
	}
	dlist := make([]any, 0, len(ds))
	for _, d := range ds {
		dlist = append(dlist, encDiagnostic(d))
	}
	env.set("diagnostics", dlist)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// obj is a JSON object that marshals its members in insertion order.
// encoding/json's map type randomizes key order; deterministic output is a
// requirement here (golden files diff byte-for-byte).
type obj struct {
	keys []string
	vals []any
}

func newObj() *obj { return &obj{} }

func (o *obj) set(key string, val any) *obj {
	o.keys = append(o.keys, key)
	o.vals = append(o.vals, val)
	return o
}

func (o *obj) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := marshalNoEscape(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		vb, err := marshalNoEscape(o.vals[i])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// marshalNoEscape is json.Marshal without HTML escaping, so "&&" and "<" in
// expression bodies survive as written.
func marshalNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

func encPos(p parser.Position) *obj {
	// Filename is deliberately dropped: identical for every position in a
	// parse, it is hoisted to the envelope.
	return newObj().set("line", p.Line).set("col", p.Column).set("offset", p.Offset)
}

func encDiagnostic(d parser.Diagnostic) *obj {
	return newObj().
		set("severity", d.Severity.String()).
		set("code", d.Code).
		set("path", d.Path).
		set("message", d.Message).
		set("pos", encPos(d.Pos))
}

func encName(n parser.Name) *obj {
	return newObj().set("value", n.Value).set("pos", encPos(n.Pos))
}

func encStringLit(s parser.StringLit) *obj {
	return newObj().set("value", s.Value).set("pos", encPos(s.Pos))
}

func encExpr(e parser.Expr) *obj {
	return newObj().set("body", e.Body).set("pos", encPos(e.Pos))
}

// encOr encodes a literal-or-expression pair as a one-key object whose
// present key is the verdict.
func encOr[T any](o *parser.OrExpr[T], lit func(T) any) *obj {
	if o.Expr != nil {
		return newObj().set("expr", encExpr(*o.Expr))
	}
	return newObj().set("literal", lit(*o.Literal))
}

func litString(s string) any { return s }
func litBool(b bool) any     { return b }
func litInt(n int64) any     { return n }

func encRetryable(r *parser.RetryableField) *obj {
	switch {
	case r.Expr != nil:
		return newObj().set("expr", encExpr(*r.Expr))
	case r.Null:
		return newObj().set("null", true)
	default:
		return newObj().set("literal", *r.Literal)
	}
}

func encPrevious(p *parser.PreviousField) *obj {
	switch {
	case p.Expr != nil:
		return newObj().set("expr", encExpr(*p.Expr))
	case p.Null:
		return newObj().set("null", true)
	default:
		return newObj().set("literal", encValue(p.Literal, true))
	}
}

// encValue encodes a positioned value subtree. With classify set, string
// leaves are split into "string" and "expr" nodes by the same rule the
// parser applies everywhere (parser.ExprBody); without it (the parameters
// schema, which is not a template field) strings stay strings.
func encValue(v *parser.Value, classify bool) *obj {
	o := newObj()
	switch v.Kind {
	case parser.KindNull:
		o.set("kind", "null")
	case parser.KindBool:
		o.set("kind", "bool").set("value", v.Bool)
	case parser.KindNumber:
		// The literal text as written, emitted as a JSON string: no
		// float round-trip, no altered digits.
		o.set("kind", "number").set("value", v.Number)
	case parser.KindString:
		if body, ok := parser.ExprBody(v.Str); classify && ok {
			o.set("kind", "expr").set("body", body)
		} else {
			o.set("kind", "string").set("value", v.Str)
		}
	case parser.KindObject:
		o.set("kind", "object").set("members", encMembers(v.Object, classify))
	case parser.KindArray:
		elems := make([]any, 0, len(v.Array))
		for _, e := range v.Array {
			elems = append(elems, encValue(e, classify))
		}
		o.set("kind", "array").set("elements", elems)
	default:
		panic(fmt.Sprintf("unknown value kind %v", v.Kind))
	}
	return o.set("pos", encPos(v.Pos))
}

func encMembers(ms []parser.Member, classify bool) []any {
	out := make([]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, newObj().
			set("name", m.Name).
			set("namePos", encPos(m.NamePos)).
			set("value", encValue(m.Value, classify)))
	}
	return out
}

func encFlow(f *parser.Flow) *obj {
	o := newObj()
	if f.Schema != "" {
		o.set("$schema", f.Schema)
	}
	if f.Comment != "" {
		o.set("comment", f.Comment)
	}
	o.set("entrypoint", encName(f.Entrypoint))
	if f.Parameters != nil {
		o.set("parameters", encValue(f.Parameters, false))
	}
	if len(f.Middleware) > 0 {
		o.set("middleware", encMiddlewareList(f.Middleware))
	}
	if len(f.Flows) > 0 {
		flows := make([]any, 0, len(f.Flows))
		for _, nf := range f.Flows {
			flows = append(flows, newObj().
				set("name", nf.Name).
				set("namePos", encPos(nf.NamePos)).
				set("flow", encFlow(nf.Flow)))
		}
		o.set("flows", flows)
	}
	steps := make([]any, 0, len(f.Steps))
	for _, ns := range f.Steps {
		steps = append(steps, newObj().
			set("name", ns.Name).
			set("namePos", encPos(ns.NamePos)).
			set("step", encStep(ns.Step)))
	}
	o.set("steps", steps)
	return o.set("pos", encPos(f.Pos))
}

func encStep(s parser.Step) *obj {
	switch s := s.(type) {
	case *parser.CallStep:
		o := newObj().set("action", "Call")
		setComment(o, s.Comment)
		o.set("call", encCall(s.Call))
		setValue(o, "input", s.Input)
		setValue(o, "output", s.Output)
		setAssign(o, s.Assign)
		if len(s.Middleware) > 0 {
			o.set("middleware", encMiddlewareList(s.Middleware))
		}
		setCatch(o, s.Catch)
		o.set("next", encName(s.Next))
		return o.set("pos", encPos(s.Pos))
	case *parser.GatherStep:
		o := newObj().set("action", "Gather")
		setComment(o, s.Comment)
		if s.Over != nil {
			o.set("over", encOr(s.Over, func(vs []*parser.Value) any {
				elems := make([]any, 0, len(vs))
				for _, v := range vs {
					elems = append(elems, encValue(v, true))
				}
				return elems
			}))
		}
		if s.Call != nil {
			o.set("call", encCall(s.Call))
		}
		if len(s.Calls) > 0 {
			calls := make([]any, 0, len(s.Calls))
			for _, c := range s.Calls {
				calls = append(calls, encCall(c))
			}
			o.set("calls", calls)
		}
		if s.Completion != nil {
			o.set("completion", encCompletion(s.Completion))
		}
		if s.Concurrency != nil {
			o.set("concurrency", encConcurrency(s.Concurrency))
		}
		setValue(o, "output", s.Output)
		setAssign(o, s.Assign)
		setCatch(o, s.Catch)
		o.set("next", encName(s.Next))
		return o.set("pos", encPos(s.Pos))
	case *parser.MatchStep:
		o := newObj().set("action", "Match")
		setComment(o, s.Comment)
		setValue(o, "input", s.Input)
		cases := make([]any, 0, len(s.Cases))
		for _, c := range s.Cases {
			cases = append(cases, encMatchCase(c))
		}
		o.set("cases", cases)
		o.set("default", encMatchDefault(s.Default))
		return o.set("pos", encPos(s.Pos))
	case *parser.PassStep:
		o := newObj().set("action", "Pass")
		setComment(o, s.Comment)
		setValue(o, "output", s.Output)
		setAssign(o, s.Assign)
		o.set("next", encName(s.Next))
		return o.set("pos", encPos(s.Pos))
	case *parser.SleepStep:
		o := newObj().set("action", "Sleep")
		setComment(o, s.Comment)
		if s.For != nil {
			o.set("for", encOr(s.For, litString))
		}
		if s.Until != nil {
			o.set("until", encOr(s.Until, litString))
		}
		o.set("next", encName(s.Next))
		return o.set("pos", encPos(s.Pos))
	case *parser.ReturnStep:
		o := newObj().set("action", "Return")
		setComment(o, s.Comment)
		setValue(o, "value", s.Value)
		return o.set("pos", encPos(s.Pos))
	case *parser.RaiseStep:
		o := newObj().set("action", "Raise")
		setComment(o, s.Comment)
		if s.Result != nil {
			o.set("result", encRaiseResult(s.Result))
		}
		return o.set("pos", encPos(s.Pos))
	default:
		panic(fmt.Sprintf("unknown step type %T", s))
	}
}

func setComment(o *obj, comment string) {
	if comment != "" {
		o.set("comment", comment)
	}
}

func setValue(o *obj, key string, v *parser.Value) {
	if v != nil {
		o.set(key, encValue(v, true))
	}
}

func setAssign(o *obj, ms []parser.Member) {
	if len(ms) > 0 {
		o.set("assign", encMembers(ms, true))
	}
}

func setCatch(o *obj, cs []parser.CatchClause) {
	if len(cs) == 0 {
		return
	}
	out := make([]any, 0, len(cs))
	for _, c := range cs {
		out = append(out, encCatchClause(c))
	}
	o.set("catch", out)
}

func encCall(c *parser.Call) *obj {
	o := newObj()
	setComment(o, c.Comment)
	if c.Provider != nil {
		o.set("provider", encProviderRef(*c.Provider))
	}
	if c.Flow != nil {
		o.set("flow", encFlowTarget(c.Flow))
	}
	setValue(o, "input", c.Input)
	if c.With != nil {
		o.set("with", encOr(c.With, func(ms []parser.Member) any {
			return encMembers(ms, true)
		}))
	}
	if c.OnSuccess != nil {
		s := newObj()
		setValue(s, "value", c.OnSuccess.Value)
		setAssign(s, c.OnSuccess.Assign)
		o.set("onSuccess", s.set("pos", encPos(c.OnSuccess.Pos)))
	}
	if c.OnFailure != nil {
		f := newObj()
		setAssign(f, c.OnFailure.Assign)
		o.set("onFailure", f.set("pos", encPos(c.OnFailure.Pos)))
	}
	return o.set("pos", encPos(c.Pos))
}

func encProviderRef(p parser.ProviderRef) *obj {
	return newObj().set("uri", p.URI).set("pos", encPos(p.Pos))
}

// encFlowTarget encodes a call's flow target as a one-key object: "name"
// for a named subflow reference, "inline" for a Flow written in place.
func encFlowTarget(t *parser.FlowTarget) *obj {
	if t.Name != nil {
		return newObj().set("name", encName(*t.Name))
	}
	return newObj().set("inline", encFlow(t.Inline))
}

func encCatchClause(c parser.CatchClause) *obj {
	o := newObj()
	setComment(o, c.Comment)
	o.set("match", encMatcher(c.Match))
	setValue(o, "output", c.Output)
	setAssign(o, c.Assign)
	o.set("next", encName(c.Next))
	return o.set("pos", encPos(c.Pos))
}

func encMatcher(m parser.FailureMatcher) *obj {
	o := newObj()
	if len(m.Codes) > 0 {
		o.set("codes", encStringLits(m.Codes))
	}
	if len(m.Types) > 0 {
		o.set("types", encStringLits(m.Types))
	}
	if m.Retryable != nil {
		o.set("retryable", *m.Retryable)
	}
	return o.set("pos", encPos(m.Pos))
}

func encStringLits(ss []parser.StringLit) []any {
	out := make([]any, 0, len(ss))
	for _, s := range ss {
		out = append(out, encStringLit(s))
	}
	return out
}

func encMatchCase(c parser.MatchCase) *obj {
	o := newObj()
	setComment(o, c.Comment)
	o.set("when", encExpr(c.When))
	setValue(o, "output", c.Output)
	setAssign(o, c.Assign)
	o.set("next", encName(c.Next))
	return o.set("pos", encPos(c.Pos))
}

func encMatchDefault(d parser.MatchDefault) *obj {
	o := newObj()
	setComment(o, d.Comment)
	setValue(o, "output", d.Output)
	setAssign(o, d.Assign)
	o.set("next", encName(d.Next))
	return o.set("pos", encPos(d.Pos))
}

func encCompletion(c *parser.Completion) *obj {
	o := newObj().set("successes", encOr(&c.Successes, litInt))
	if c.Wait != nil {
		o.set("wait", *c.Wait)
	}
	return o.set("pos", encPos(c.Pos))
}

// encConcurrency keeps the written form: a cap value, or the explicit null
// the author wrote (equivalent to absent, preserved as written).
func encConcurrency(c *parser.ConcurrencyCap) *obj {
	o := newObj()
	if c.Value != nil {
		o.set("value", *c.Value)
	} else {
		o.set("null", true)
	}
	return o.set("pos", encPos(c.Pos))
}

func encRaiseResult(r *parser.RaiseResult) *obj {
	o := newObj()
	if r.Type != nil {
		o.set("type", encOr(r.Type, litString))
	}
	o.set("code", encOr(&r.Code, litString))
	if r.Message != nil {
		o.set("message", encOr(r.Message, litString))
	}
	setValue(o, "details", r.Details)
	if r.Retryable != nil {
		o.set("retryable", encRetryable(r.Retryable))
	}
	if r.Previous != nil {
		o.set("previous", encPrevious(r.Previous))
	}
	return o.set("pos", encPos(r.Pos))
}

func encMiddlewareList(ms []parser.MiddlewareEntry) []any {
	out := make([]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, encMiddlewareEntry(m))
	}
	return out
}

func encMiddlewareEntry(m parser.MiddlewareEntry) *obj {
	o := newObj()
	setComment(o, m.Comment)
	o.set("provider", encProviderRef(m.Provider))
	if m.OnEntry != nil {
		p := newObj()
		setPhaseCommon(p, m.OnEntry.When, m.OnEntry.With, m.OnEntry.Assign)
		setValue(p, "output", m.OnEntry.Output)
		o.set("onEntry", p.set("pos", encPos(m.OnEntry.Pos)))
	}
	if m.OnSuccess != nil {
		p := newObj()
		setPhaseCommon(p, m.OnSuccess.When, m.OnSuccess.With, m.OnSuccess.Assign)
		setValue(p, "value", m.OnSuccess.Value)
		o.set("onSuccess", p.set("pos", encPos(m.OnSuccess.Pos)))
	}
	if m.OnFailure != nil {
		p := newObj()
		setPhaseCommon(p, m.OnFailure.When, m.OnFailure.With, m.OnFailure.Assign)
		if m.OnFailure.Type != nil {
			p.set("type", encOr(m.OnFailure.Type, litString))
		}
		if m.OnFailure.Code != nil {
			p.set("code", encOr(m.OnFailure.Code, litString))
		}
		if m.OnFailure.Message != nil {
			p.set("message", encOr(m.OnFailure.Message, litString))
		}
		setValue(p, "details", m.OnFailure.Details)
		if m.OnFailure.Retryable != nil {
			p.set("retryable", encRetryable(m.OnFailure.Retryable))
		}
		if m.OnFailure.Previous != nil {
			p.set("previous", encPrevious(m.OnFailure.Previous))
		}
		o.set("onFailure", p.set("pos", encPos(m.OnFailure.Pos)))
	}
	if m.OnAlways != nil {
		p := newObj()
		setPhaseCommon(p, m.OnAlways.When, m.OnAlways.With, m.OnAlways.Assign)
		o.set("onAlways", p.set("pos", encPos(m.OnAlways.Pos)))
	}
	return o.set("pos", encPos(m.Pos))
}

func setPhaseCommon(o *obj, when *parser.OrExpr[bool], with *parser.OrExpr[[]parser.Member], assign []parser.Member) {
	if when != nil {
		o.set("when", encOr(when, litBool))
	}
	if with != nil {
		o.set("with", encOr(with, func(ms []parser.Member) any {
			return encMembers(ms, true)
		}))
	}
	setAssign(o, assign)
}
