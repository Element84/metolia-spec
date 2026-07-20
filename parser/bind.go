package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// flowSchemaURI is the $schema value a root workflow definition must carry.
const flowSchemaURI = "https://mwl.dev/v0.1/flow/schema.json"

// bind is the structural pass: it walks the positioned value tree and
// produces the typed AST, validating along the way that each object is the
// right shape — required members present, types correct, no unrecognized
// members, and the per-object constraints the schema and prose state. It
// binds what it can past individual findings so one bad Step does not hide
// another's diagnostics.
func bind(root *Value) (*Flow, Diagnostics) {
	b := &binder{}
	if root.Kind != KindObject {
		b.errorAt(CodeWrongType, root.Pos,
			"the root of a workflow definition must be an object, not %s", root.Kind)
		return nil, b.diags
	}
	f := b.bindFlow(root, true)
	return f, b.diags
}

type binder struct {
	diags Diagnostics
	path  []pathSeg
}

func (b *binder) pushKey(name string) { b.path = append(b.path, pathSeg{key: name}) }
func (b *binder) pushIdx(i int)       { b.path = append(b.path, pathSeg{idx: i, isIdx: true}) }
func (b *binder) pop()                { b.path = b.path[:len(b.path)-1] }

func (b *binder) errorAt(code string, pos Position, format string, args ...any) {
	b.diags = append(b.diags, Diagnostic{
		Severity: SeverityError,
		Code:     code,
		Pos:      pos,
		Path:     renderPath(b.path, ""),
		Message:  fmt.Sprintf(format, args...),
	})
}

func (b *binder) unknownMember(m Member, in string) {
	b.errorAt(CodeUnknownMember, m.NamePos, "unknown member %q in %s", m.Name, in)
}

func (b *binder) missingMember(pos Position, what string) {
	b.errorAt(CodeMissingMember, pos, "%s", what)
}

// requireString reports a wrong-type error unless v is a string.
func (b *binder) requireString(v *Value, what string) (string, bool) {
	if v.Kind != KindString {
		b.errorAt(CodeWrongType, v.Pos, "%s must be a string, not %s", what, v.Kind)
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
		b.errorAt(CodeExpressionNotPermitted, v.Pos,
			"%s is structural and does not accept an expression", what)
	}
	return Name{Value: s, Pos: v.Pos}
}

func (b *binder) bindComment(v *Value) string {
	s, _ := b.requireString(v, "comment")
	return s
}

// Flow --------------------------------------------------------------------

func (b *binder) bindFlow(v *Value, root bool) *Flow {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a Flow must be an object, not %s", v.Kind)
		return nil
	}
	f := &Flow{Pos: v.Pos}
	var hasSchema, hasEntrypoint, hasSteps bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "$schema":
			if !root {
				b.errorAt(CodeUnknownMember, m.NamePos,
					"unknown member %q: $schema is permitted on the root document only", m.Name)
				break
			}
			hasSchema = true
			if s, ok := b.requireString(m.Value, "$schema"); ok {
				f.Schema = s
				if s != flowSchemaURI {
					b.errorAt(CodeInvalidValue, m.Value.Pos, "$schema must be %q", flowSchemaURI)
				}
			}
		case "comment":
			f.Comment = b.bindComment(m.Value)
		case "entrypoint":
			hasEntrypoint = true
			f.Entrypoint = b.bindName(m.Value, "entrypoint")
		case "flows":
			f.Flows, f.flowIndex = b.bindFlows(m.Value)
		case "middleware":
			f.Middleware = b.bindMiddlewareStack(m.Value)
		case "parameters":
			f.Parameters = b.bindParameters(m.Value)
		case "steps":
			hasSteps = true
			f.Steps, f.stepIndex = b.bindSteps(m.Value)
		default:
			b.unknownMember(m, "a Flow")
		}
		b.pop()
	}
	if root && !hasSchema {
		b.missingMember(v.Pos, "the root document requires $schema")
	}
	if !hasEntrypoint {
		b.missingMember(v.Pos, "a Flow requires entrypoint")
	}
	if !hasSteps {
		b.missingMember(v.Pos, "a Flow requires steps")
	}
	return f
}

func (b *binder) bindFlows(v *Value) ([]NamedFlow, map[string]int) {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "flows must be an object of named Flows, not %s", v.Kind)
		return nil, nil
	}
	var flows []NamedFlow
	index := make(map[string]int)
	for _, m := range v.Object {
		if _, isExpr := ExprBody(m.Name); isExpr {
			b.errorAt(CodeExpressionNotPermitted, m.NamePos,
				"a flow name is structural and does not accept an expression")
		}
		b.pushKey(m.Name)
		fl := b.bindFlow(m.Value, false)
		b.pop()
		if fl != nil {
			index[m.Name] = len(flows)
			flows = append(flows, NamedFlow{Name: m.Name, NamePos: m.NamePos, Flow: fl})
		}
	}
	return flows, index
}

func (b *binder) bindSteps(v *Value) ([]NamedStep, map[string]int) {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "steps must be an object of named Steps, not %s", v.Kind)
		return nil, nil
	}
	if len(v.Object) == 0 {
		b.errorAt(CodeInvalidValue, v.Pos, "steps must define at least one Step")
	}
	var steps []NamedStep
	index := make(map[string]int)
	for _, m := range v.Object {
		if _, isExpr := ExprBody(m.Name); isExpr {
			b.errorAt(CodeExpressionNotPermitted, m.NamePos,
				"a Step name is structural and does not accept an expression")
		}
		b.pushKey(m.Name)
		s := b.bindStep(m.Value)
		b.pop()
		if s != nil {
			index[m.Name] = len(steps)
			steps = append(steps, NamedStep{Name: m.Name, NamePos: m.NamePos, Step: s})
		}
	}
	return steps, index
}

func (b *binder) bindParameters(v *Value) *Value {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "parameters must be a JSON Schema object, not %s", v.Kind)
		return nil
	}
	t := v.Field("type")
	switch {
	case t == nil:
		b.missingMember(v.Pos, `a parameters schema must declare "type": "object"`)
	case t.Kind != KindString || t.Str != "object":
		b.errorAt(CodeInvalidValue, t.Pos, `a parameters schema must declare "type": "object"`)
	}
	return v
}

// Steps -------------------------------------------------------------------

func (b *binder) bindStep(v *Value) Step {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a Step must be an object, not %s", v.Kind)
		return nil
	}
	action := v.Field("action")
	if action == nil {
		b.missingMember(v.Pos, "a Step requires action")
		return nil
	}
	name, ok := b.requireString(action, "action")
	if !ok {
		return nil
	}
	switch name {
	case "Call":
		return b.bindCallStep(v)
	case "Gather":
		return b.bindGatherStep(v)
	case "Match":
		return b.bindMatchStep(v)
	case "Pass":
		return b.bindPassStep(v)
	case "Sleep":
		return b.bindSleepStep(v)
	case "Return":
		return b.bindReturnStep(v)
	case "Raise":
		return b.bindRaiseStep(v)
	default:
		b.errorAt(CodeInvalidValue, action.Pos,
			"unknown action %q; expected Call, Gather, Match, Pass, Sleep, Return, or Raise", name)
		return nil
	}
}

func (b *binder) bindCallStep(v *Value) *CallStep {
	s := &CallStep{Pos: v.Pos}
	var hasCall, hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "call":
			hasCall = true
			s.Call = b.bindCall(m.Value)
		case "input":
			s.Input = m.Value
		case "output":
			s.Output = m.Value
		case "assign":
			s.Assign = b.bindAssign(m.Value)
		case "middleware":
			s.Middleware = b.bindMiddlewareStack(m.Value)
		case "catch":
			s.Catch = b.bindCatch(m.Value)
		case "next":
			hasNext = true
			s.Next = b.bindName(m.Value, "next")
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Call Step")
		}
		b.pop()
	}
	if !hasCall {
		b.missingMember(v.Pos, "a Call Step requires call")
	}
	if !hasNext {
		b.missingMember(v.Pos, "a Call Step requires next")
	}
	return s
}

func (b *binder) bindGatherStep(v *Value) *GatherStep {
	s := &GatherStep{Pos: v.Pos}
	var hasOver, hasCall, hasCalls, hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "over":
			hasOver = true
			s.Over = b.bindOver(m.Value)
		case "call":
			hasCall = true
			s.Call = b.bindCall(m.Value)
		case "calls":
			hasCalls = true
			s.Calls = b.bindCalls(m.Value)
		case "completion":
			s.Completion = b.bindCompletion(m.Value)
		case "concurrency":
			s.Concurrency = b.bindConcurrency(m.Value)
		case "output":
			s.Output = m.Value
		case "assign":
			s.Assign = b.bindAssign(m.Value)
		case "catch":
			s.Catch = b.bindCatch(m.Value)
		case "next":
			hasNext = true
			s.Next = b.bindName(m.Value, "next")
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Gather Step")
		}
		b.pop()
	}
	switch {
	case hasCalls && (hasOver || hasCall):
		b.errorAt(CodeExclusiveMembers, v.Pos,
			"a Gather carries exactly one of its two forms: over with call, or calls")
	case !hasCalls && !hasOver && !hasCall:
		b.errorAt(CodeExclusiveMembers, v.Pos,
			"a Gather carries exactly one of its two forms: over with call, or calls")
	case hasOver && !hasCall:
		b.missingMember(v.Pos, "the iterate form requires call beside over")
	case hasCall && !hasOver:
		b.missingMember(v.Pos, "the iterate form requires over beside call")
	}
	if !hasNext {
		b.missingMember(v.Pos, "a Gather Step requires next")
	}
	return s
}

func (b *binder) bindOver(v *Value) *OrExpr[[]*Value] {
	switch v.Kind {
	case KindArray:
		lit := v.Array
		return &OrExpr[[]*Value]{Literal: &lit}
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return &OrExpr[[]*Value]{Expr: &Expr{Body: body, Pos: v.Pos}}
		}
		b.errorAt(CodeWrongType, v.Pos, "over must be an array or an expression")
	default:
		b.errorAt(CodeWrongType, v.Pos, "over must be an array or an expression, not %s", v.Kind)
	}
	return nil
}

func (b *binder) bindCalls(v *Value) []*Call {
	if v.Kind != KindArray {
		b.errorAt(CodeWrongType, v.Pos, "calls must be an array of call objects, not %s", v.Kind)
		return nil
	}
	if len(v.Array) == 0 {
		b.errorAt(CodeInvalidValue, v.Pos, "calls must be a non-empty array")
		return nil
	}
	var calls []*Call
	for i, e := range v.Array {
		b.pushIdx(i)
		c := b.bindCall(e)
		b.pop()
		if c != nil {
			calls = append(calls, c)
		}
	}
	return calls
}

func (b *binder) bindCompletion(v *Value) *Completion {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "completion must be an object, not %s", v.Kind)
		return nil
	}
	c := &Completion{Pos: v.Pos}
	var hasSuccesses bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "successes":
			hasSuccesses = true
			if x, ok := b.bindIntOrExpr(m.Value, "successes"); ok {
				c.Successes = x
			}
		case "wait":
			if m.Value.Kind != KindBool {
				b.errorAt(CodeWrongType, m.Value.Pos,
					"wait must be a literal boolean; it does not accept an expression")
			} else {
				lit := m.Value.Bool
				c.Wait = &lit
			}
		default:
			b.unknownMember(m, "a completion policy")
		}
		b.pop()
	}
	if !hasSuccesses {
		b.missingMember(v.Pos, "a completion policy requires successes")
	}
	return c
}

func (b *binder) bindConcurrency(v *Value) *ConcurrencyCap {
	switch v.Kind {
	case KindNull:
		return &ConcurrencyCap{Pos: v.Pos}
	case KindNumber:
		n, ok := intFromLexeme(v.Number)
		if !ok {
			b.errorAt(CodeInvalidValue, v.Pos, "concurrency must be an integer")
			return nil
		}
		if n < 1 {
			b.errorAt(CodeInvalidValue, v.Pos, "concurrency must be at least 1")
			return nil
		}
		return &ConcurrencyCap{Pos: v.Pos, Value: &n}
	default:
		b.errorAt(CodeWrongType, v.Pos,
			"concurrency must be an integer or null; it does not accept an expression")
		return nil
	}
}

func (b *binder) bindMatchStep(v *Value) *MatchStep {
	s := &MatchStep{Pos: v.Pos}
	var hasCases, hasDefault bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "input":
			s.Input = m.Value
		case "cases":
			hasCases = true
			s.Cases = b.bindMatchCases(m.Value)
		case "default":
			hasDefault = true
			s.Default = b.bindMatchDefault(m.Value)
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Match Step")
		}
		b.pop()
	}
	if !hasCases {
		b.missingMember(v.Pos, "a Match Step requires cases")
	}
	if !hasDefault {
		b.missingMember(v.Pos, "a Match Step requires default")
	}
	return s
}

func (b *binder) bindMatchCases(v *Value) []MatchCase {
	if v.Kind != KindArray {
		b.errorAt(CodeWrongType, v.Pos, "cases must be an array of clauses, not %s", v.Kind)
		return nil
	}
	var cases []MatchCase
	for i, e := range v.Array {
		b.pushIdx(i)
		if c, ok := b.bindMatchCase(e); ok {
			cases = append(cases, c)
		}
		b.pop()
	}
	return cases
}

func (b *binder) bindMatchCase(v *Value) (MatchCase, bool) {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a cases clause must be an object, not %s", v.Kind)
		return MatchCase{}, false
	}
	c := MatchCase{Pos: v.Pos}
	var hasWhen, hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "when":
			hasWhen = true
			if e := b.bindExpr(m.Value, "when"); e != nil {
				c.When = *e
			}
		case "output":
			c.Output = m.Value
		case "assign":
			c.Assign = b.bindAssign(m.Value)
		case "next":
			hasNext = true
			c.Next = b.bindName(m.Value, "next")
		case "comment":
			c.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a cases clause")
		}
		b.pop()
	}
	if !hasWhen {
		b.missingMember(v.Pos, "a cases clause requires when")
	}
	if !hasNext {
		b.missingMember(v.Pos, "a cases clause requires next")
	}
	return c, true
}

func (b *binder) bindMatchDefault(v *Value) MatchDefault {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "default must be an object, not %s", v.Kind)
		return MatchDefault{}
	}
	d := MatchDefault{Pos: v.Pos}
	var hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "when":
			b.errorAt(CodeUnknownMember, m.NamePos,
				"unknown member \"when\": the default clause carries no predicate")
		case "output":
			d.Output = m.Value
		case "assign":
			d.Assign = b.bindAssign(m.Value)
		case "next":
			hasNext = true
			d.Next = b.bindName(m.Value, "next")
		case "comment":
			d.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "the default clause")
		}
		b.pop()
	}
	if !hasNext {
		b.missingMember(v.Pos, "the default clause requires next")
	}
	return d
}

func (b *binder) bindPassStep(v *Value) *PassStep {
	s := &PassStep{Pos: v.Pos}
	var hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "output":
			s.Output = m.Value
		case "assign":
			s.Assign = b.bindAssign(m.Value)
		case "next":
			hasNext = true
			s.Next = b.bindName(m.Value, "next")
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Pass Step")
		}
		b.pop()
	}
	if !hasNext {
		b.missingMember(v.Pos, "a Pass Step requires next")
	}
	return s
}

func (b *binder) bindSleepStep(v *Value) *SleepStep {
	s := &SleepStep{Pos: v.Pos}
	var hasFor, hasUntil, hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "for":
			hasFor = true
			if x, ok := b.bindStringOrExpr(m.Value, "for"); ok {
				s.For = &x
			}
		case "until":
			hasUntil = true
			if x, ok := b.bindStringOrExpr(m.Value, "until"); ok {
				s.Until = &x
			}
		case "next":
			hasNext = true
			s.Next = b.bindName(m.Value, "next")
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Sleep Step")
		}
		b.pop()
	}
	if hasFor == hasUntil {
		b.errorAt(CodeExclusiveMembers, v.Pos, "a Sleep carries exactly one of for and until")
	}
	if !hasNext {
		b.missingMember(v.Pos, "a Sleep Step requires next")
	}
	return s
}

func (b *binder) bindReturnStep(v *Value) *ReturnStep {
	s := &ReturnStep{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "value":
			s.Value = m.Value
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Return Step")
		}
		b.pop()
	}
	return s
}

func (b *binder) bindRaiseStep(v *Value) *RaiseStep {
	s := &RaiseStep{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
		case "result":
			s.Result = b.bindRaiseResult(m.Value)
		case "comment":
			s.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a Raise Step")
		}
		b.pop()
	}
	return s
}

func (b *binder) bindRaiseResult(v *Value) *RaiseResult {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "result must be an object, not %s", v.Kind)
		return nil
	}
	r := &RaiseResult{Pos: v.Pos}
	var hasCode bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "type":
			if x, ok := b.bindStringOrExpr(m.Value, "type"); ok {
				if x.Literal != nil && *x.Literal == "success" {
					b.errorAt(CodeInvalidValue, m.Value.Pos, "type must be a non-success Result type")
				}
				r.Type = &x
			}
		case "code":
			hasCode = true
			if x, ok := b.bindStringOrExpr(m.Value, "code"); ok {
				r.Code = x
			}
		case "message":
			if x, ok := b.bindStringOrExpr(m.Value, "message"); ok {
				r.Message = &x
			}
		case "details":
			r.Details = m.Value
		case "retryable":
			r.Retryable = b.bindRetryable(m.Value)
		case "previous":
			r.Previous = b.bindPrevious(m.Value)
		default:
			b.unknownMember(m, "a Raise result")
		}
		b.pop()
	}
	if !hasCode {
		b.missingMember(v.Pos, "a Raise result requires code")
	}
	return r
}

// Calls -------------------------------------------------------------------

func (b *binder) bindCall(v *Value) *Call {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a call must be an object, not %s", v.Kind)
		return nil
	}
	c := &Call{Pos: v.Pos}
	var hasProvider, hasFlow bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "provider":
			hasProvider = true
			c.Provider = b.bindProviderRef(m.Value, "call")
		case "flow":
			hasFlow = true
			c.Flow = b.bindFlowTarget(m.Value)
		case "input":
			c.Input = m.Value
		case "with":
			c.With = b.bindWith(m.Value)
		case "onSuccess":
			c.OnSuccess = b.bindSuccessArm(m.Value)
		case "onFailure":
			c.OnFailure = b.bindFailureArm(m.Value)
		case "comment":
			c.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a call object")
		}
		b.pop()
	}
	if hasProvider == hasFlow {
		b.errorAt(CodeExclusiveMembers, v.Pos,
			"a call names exactly one target: provider or flow")
	}
	return c
}

func (b *binder) bindFlowTarget(v *Value) *FlowTarget {
	switch v.Kind {
	case KindString:
		if _, isExpr := ExprBody(v.Str); isExpr {
			b.errorAt(CodeExpressionNotPermitted, v.Pos,
				"flow is structural and does not accept an expression")
			return nil
		}
		return &FlowTarget{Name: &Name{Value: v.Str, Pos: v.Pos}}
	case KindObject:
		if fl := b.bindFlow(v, false); fl != nil {
			return &FlowTarget{Inline: fl}
		}
		return nil
	default:
		b.errorAt(CodeWrongType, v.Pos, "flow must be a name or a Flow object, not %s", v.Kind)
		return nil
	}
}

func (b *binder) bindProviderRef(v *Value, kind string) *ProviderRef {
	s, ok := b.requireString(v, "provider")
	if !ok {
		return nil
	}
	if _, isExpr := ExprBody(s); isExpr {
		b.errorAt(CodeExpressionNotPermitted, v.Pos,
			"provider is structural and does not accept an expression")
		return nil
	}
	if !validProviderURI(s, kind) {
		b.errorAt(CodeInvalidProviderURI, v.Pos,
			"%q is not a well-formed %s-provider URI; expected mwl:provider.%s/<namespace>/<name>",
			s, kind, kind)
		return nil
	}
	return &ProviderRef{URI: s, Pos: v.Pos}
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

func (b *binder) bindSuccessArm(v *Value) *SuccessArm {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "onSuccess must be an object, not %s", v.Kind)
		return nil
	}
	a := &SuccessArm{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "value":
			a.Value = m.Value
		case "assign":
			a.Assign = b.bindAssign(m.Value)
		default:
			b.unknownMember(m, "an onSuccess arm")
		}
		b.pop()
	}
	return a
}

func (b *binder) bindFailureArm(v *Value) *FailureArm {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "onFailure must be an object, not %s", v.Kind)
		return nil
	}
	a := &FailureArm{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "assign":
			a.Assign = b.bindAssign(m.Value)
		default:
			b.unknownMember(m, "an onFailure arm")
		}
		b.pop()
	}
	return a
}

// Catch -------------------------------------------------------------------

func (b *binder) bindCatch(v *Value) []CatchClause {
	if v.Kind != KindArray {
		b.errorAt(CodeWrongType, v.Pos, "catch must be an array of clauses, not %s", v.Kind)
		return nil
	}
	var clauses []CatchClause
	for i, e := range v.Array {
		b.pushIdx(i)
		if c, ok := b.bindCatchClause(e); ok {
			clauses = append(clauses, c)
		}
		b.pop()
	}
	return clauses
}

func (b *binder) bindCatchClause(v *Value) (CatchClause, bool) {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a catch clause must be an object, not %s", v.Kind)
		return CatchClause{}, false
	}
	c := CatchClause{Pos: v.Pos}
	var hasMatch, hasNext bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "match":
			hasMatch = true
			c.Match = b.bindMatcher(m.Value)
		case "output":
			c.Output = m.Value
		case "assign":
			c.Assign = b.bindAssign(m.Value)
		case "next":
			hasNext = true
			c.Next = b.bindName(m.Value, "next")
		case "comment":
			c.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a catch clause")
		}
		b.pop()
	}
	if !hasMatch {
		b.missingMember(v.Pos, "a catch clause requires match")
	}
	if !hasNext {
		b.missingMember(v.Pos, "a catch clause requires next")
	}
	return c, true
}

func (b *binder) bindMatcher(v *Value) FailureMatcher {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a failure matcher must be an object, not %s", v.Kind)
		return FailureMatcher{}
	}
	f := FailureMatcher{Pos: v.Pos}
	var known bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "codes":
			known = true
			f.Codes = b.bindMatcherList(m.Value, "codes", false)
		case "types":
			known = true
			f.Types = b.bindMatcherList(m.Value, "types", true)
		case "retryable":
			known = true
			if m.Value.Kind != KindBool {
				b.errorAt(CodeWrongType, m.Value.Pos,
					"matcher retryable must be a literal boolean, not %s", m.Value.Kind)
			} else {
				lit := m.Value.Bool
				f.Retryable = &lit
			}
		default:
			b.unknownMember(m, "a failure matcher")
		}
		b.pop()
	}
	if !known {
		b.errorAt(CodeInvalidValue, v.Pos, "a failure matcher must have at least one member")
	}
	return f
}

func (b *binder) bindMatcherList(v *Value, what string, rejectSuccess bool) []StringLit {
	if v.Kind != KindArray {
		b.errorAt(CodeWrongType, v.Pos, "%s must be an array of strings, not %s", what, v.Kind)
		return nil
	}
	if len(v.Array) == 0 {
		b.errorAt(CodeInvalidValue, v.Pos, "%s must be a non-empty array", what)
		return nil
	}
	var list []StringLit
	for i, e := range v.Array {
		b.pushIdx(i)
		if s, ok := b.requireString(e, what+" entry"); ok {
			if rejectSuccess && s == "success" {
				b.errorAt(CodeInvalidValue, e.Pos,
					`"success" is not a permissible matcher type; a matcher selects among failures`)
			}
			list = append(list, StringLit{Value: s, Pos: e.Pos})
		}
		b.pop()
	}
	return list
}

// Middleware --------------------------------------------------------------

func (b *binder) bindMiddlewareStack(v *Value) []MiddlewareEntry {
	if v.Kind != KindArray {
		b.errorAt(CodeWrongType, v.Pos, "middleware must be an array of entries, not %s", v.Kind)
		return nil
	}
	var entries []MiddlewareEntry
	for i, e := range v.Array {
		b.pushIdx(i)
		if entry, ok := b.bindMiddlewareEntry(e); ok {
			entries = append(entries, entry)
		}
		b.pop()
	}
	return entries
}

func (b *binder) bindMiddlewareEntry(v *Value) (MiddlewareEntry, bool) {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "a middleware entry must be an object, not %s", v.Kind)
		return MiddlewareEntry{}, false
	}
	e := MiddlewareEntry{Pos: v.Pos}
	var hasProvider bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "provider":
			hasProvider = true
			if ref := b.bindProviderRef(m.Value, "middleware"); ref != nil {
				e.Provider = *ref
			}
		case "onEntry":
			e.OnEntry = b.bindPhaseOnEntry(m.Value)
		case "onSuccess":
			e.OnSuccess = b.bindPhaseOnSuccess(m.Value)
		case "onFailure":
			e.OnFailure = b.bindPhaseOnFailure(m.Value)
		case "onAlways":
			e.OnAlways = b.bindPhaseOnAlways(m.Value)
		case "comment":
			e.Comment = b.bindComment(m.Value)
		default:
			b.unknownMember(m, "a middleware entry")
		}
		b.pop()
	}
	if !hasProvider {
		b.missingMember(v.Pos, "a middleware entry requires provider")
	}
	return e, true
}

func (b *binder) bindPhaseOnEntry(v *Value) *PhaseOnEntry {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "onEntry must be an object, not %s", v.Kind)
		return nil
	}
	p := &PhaseOnEntry{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "when":
			p.When = b.bindBoolOrExpr(m.Value, "when")
		case "with":
			p.With = b.bindWith(m.Value)
		case "assign":
			p.Assign = b.bindAssign(m.Value)
		case "output":
			p.Output = m.Value
		default:
			b.unknownMember(m, "an onEntry phase")
		}
		b.pop()
	}
	return p
}

func (b *binder) bindPhaseOnSuccess(v *Value) *PhaseOnSuccess {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "onSuccess must be an object, not %s", v.Kind)
		return nil
	}
	p := &PhaseOnSuccess{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "when":
			p.When = b.bindBoolOrExpr(m.Value, "when")
		case "with":
			p.With = b.bindWith(m.Value)
		case "assign":
			p.Assign = b.bindAssign(m.Value)
		case "value":
			p.Value = m.Value
		default:
			b.unknownMember(m, "an onSuccess phase")
		}
		b.pop()
	}
	return p
}

func (b *binder) bindPhaseOnFailure(v *Value) *PhaseOnFailure {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "onFailure must be an object, not %s", v.Kind)
		return nil
	}
	p := &PhaseOnFailure{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "when":
			p.When = b.bindBoolOrExpr(m.Value, "when")
		case "with":
			p.With = b.bindWith(m.Value)
		case "assign":
			p.Assign = b.bindAssign(m.Value)
		case "type":
			if x, ok := b.bindStringOrExpr(m.Value, "type"); ok {
				if x.Literal != nil && *x.Literal == "success" {
					b.errorAt(CodeInvalidValue, m.Value.Pos, "type must be a non-success Result type")
				}
				p.Type = &x
			}
		case "code":
			if x, ok := b.bindStringOrExpr(m.Value, "code"); ok {
				p.Code = &x
			}
		case "message":
			if x, ok := b.bindStringOrExpr(m.Value, "message"); ok {
				p.Message = &x
			}
		case "details":
			p.Details = m.Value
		case "retryable":
			p.Retryable = b.bindRetryable(m.Value)
		case "previous":
			p.Previous = b.bindPrevious(m.Value)
		default:
			b.unknownMember(m, "an onFailure phase")
		}
		b.pop()
	}
	return p
}

func (b *binder) bindPhaseOnAlways(v *Value) *PhaseOnAlways {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "onAlways must be an object, not %s", v.Kind)
		return nil
	}
	p := &PhaseOnAlways{Pos: v.Pos}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "when":
			p.When = b.bindBoolOrExpr(m.Value, "when")
		case "with":
			p.With = b.bindWith(m.Value)
		case "assign":
			p.Assign = b.bindAssign(m.Value)
		default:
			b.unknownMember(m, "an onAlways phase")
		}
		b.pop()
	}
	return p
}

// Expression-bearing field shapes -----------------------------------------

func (b *binder) bindAssign(v *Value) []Member {
	if v.Kind != KindObject {
		b.errorAt(CodeWrongType, v.Pos, "assign must be an object, not %s", v.Kind)
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
		b.errorAt(CodeWrongType, v.Pos, "with must be an object or an expression")
	default:
		b.errorAt(CodeWrongType, v.Pos, "with must be an object or an expression, not %s", v.Kind)
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
		b.errorAt(CodeInvalidValue, v.Pos, "%s must be an expression", what)
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
		b.errorAt(CodeWrongType, v.Pos, "%s must be a boolean or an expression", what)
	default:
		b.errorAt(CodeWrongType, v.Pos, "%s must be a boolean or an expression, not %s", what, v.Kind)
	}
	return nil
}

func (b *binder) bindIntOrExpr(v *Value, what string) (OrExpr[int64], bool) {
	switch v.Kind {
	case KindNumber:
		n, ok := intFromLexeme(v.Number)
		if !ok {
			b.errorAt(CodeInvalidValue, v.Pos, "%s must be an integer", what)
			return OrExpr[int64]{}, false
		}
		return OrExpr[int64]{Literal: &n}, true
	case KindString:
		if body, ok := ExprBody(v.Str); ok {
			return OrExpr[int64]{Expr: &Expr{Body: body, Pos: v.Pos}}, true
		}
		b.errorAt(CodeWrongType, v.Pos, "%s must be an integer or an expression", what)
	default:
		b.errorAt(CodeWrongType, v.Pos, "%s must be an integer or an expression, not %s", what, v.Kind)
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
		b.errorAt(CodeWrongType, v.Pos, "retryable must be a boolean, null, or an expression")
	default:
		b.errorAt(CodeWrongType, v.Pos, "retryable must be a boolean, null, or an expression, not %s", v.Kind)
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
		b.errorAt(CodeWrongType, v.Pos, "previous must be an object, null, or an expression")
	default:
		b.errorAt(CodeWrongType, v.Pos, "previous must be an object, null, or an expression, not %s", v.Kind)
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
