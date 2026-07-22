package parser

import "strings"

// check is the static-semantics pass: the rules that span the document,
// which the schema cannot express. It verifies that entrypoint and every
// next resolve within their own Flow's steps map, that every flow name
// resolves along its lexical scope chain, and that the flow-reference graph
// is acyclic; it warns when a flows entry shadows an enclosing declaration.
// It expects an AST the bind pass produced without errors.
func check(f *Flow) Diagnostics {
	c := &flowChecker{
		names: make(map[*Flow]string),
		edges: make(map[*Flow][]flowEdge),
	}
	c.checkFlow(f, []*Flow{f}, "(root)")
	c.detectCycles()
	return c.diags
}

// flowChecker carries the flow-reference graph beside the shared checker
// machinery: every Flow object is a node, and an edge runs from the Flow
// containing a call to the Flow the call targets. The graph is flow-only
// state, so it lives here rather than on the checker itself.
type flowChecker struct {
	checker

	nodes []*Flow
	names map[*Flow]string
	edges map[*Flow][]flowEdge
}

// flowEdge is one flow reference: the target, and the site it was written
// at for reporting.
type flowEdge struct {
	to   *Flow
	pos  Position
	path string
}

// checkFlow walks one Flow. chain is the lexical scope chain for call sites
// within it: the Flow itself first, then each enclosing Flow outward to the
// root.
func (c *flowChecker) checkFlow(f *Flow, chain []*Flow, name string) {
	c.nodes = append(c.nodes, f)
	c.names[f] = name

	if _, ok := f.StepByName(f.Entrypoint.Value); !ok {
		c.report(CodeUnresolvedStep, f.Entrypoint.Pos,
			renderPath(c.path, "entrypoint"),
			"entrypoint names unknown Step %q", f.Entrypoint.Value)
	}

	c.pushKey("steps")
	for _, ns := range f.Steps {
		c.pushKey(ns.Name)
		c.checkStep(f, ns.Step, chain)
		c.pop()
	}
	c.pop()

	c.pushKey("flows")
	for _, nf := range f.Flows {
		for _, enclosing := range chain[1:] {
			if outer, ok := enclosing.FlowByName(nf.Name); ok {
				c.report(CodeShadowedFlow, nf.NamePos,
					renderPath(c.path, nf.Name),
					"flows entry %q shadows the declaration at %s", nf.Name, outer.NamePos)
				break
			}
		}
		c.pushKey(nf.Name)
		c.checkFlow(nf.Flow, prependFlow(nf.Flow, chain), nf.Name)
		c.pop()
	}
	c.pop()
}

func (c *flowChecker) checkStep(f *Flow, step Step, chain []*Flow) {
	switch s := step.(type) {
	case *CallStep:
		c.checkNext(f, s.Next)
		c.checkCatch(f, s.Catch)
		c.pushKey("call")
		c.checkCall(f, s.Call, chain)
		c.pop()
	case *GatherStep:
		c.checkNext(f, s.Next)
		c.checkCatch(f, s.Catch)
		if s.Call != nil {
			c.pushKey("call")
			c.checkCall(f, s.Call, chain)
			c.pop()
		}
		if len(s.Calls) > 0 {
			c.pushKey("calls")
			for i, call := range s.Calls {
				c.pushIdx(i)
				c.checkCall(f, call, chain)
				c.pop()
			}
			c.pop()
		}
	case *MatchStep:
		c.pushKey("cases")
		for i, cs := range s.Cases {
			c.pushIdx(i)
			c.checkNext(f, cs.Next)
			c.pop()
		}
		c.pop()
		c.pushKey("default")
		c.checkNext(f, s.Default.Next)
		c.pop()
	case *PassStep:
		c.checkNext(f, s.Next)
	case *SleepStep:
		c.checkNext(f, s.Next)
	case *ReturnStep, *RaiseStep:
		// Terminal: no references.
	}
}

// checkNext verifies a Step reference against the referencing Step's own
// steps map; cross-scope transitions are not permitted.
func (c *flowChecker) checkNext(f *Flow, n Name) {
	if _, ok := f.StepByName(n.Value); !ok {
		c.report(CodeUnresolvedStep, n.Pos,
			renderPath(c.path, "next"),
			"next names unknown Step %q", n.Value)
	}
}

func (c *flowChecker) checkCatch(f *Flow, clauses []CatchClause) {
	if len(clauses) == 0 {
		return
	}
	c.pushKey("catch")
	for i, clause := range clauses {
		c.pushIdx(i)
		c.checkNext(f, clause.Next)
		c.pop()
	}
	c.pop()
}

// checkCall resolves a call's flow target, records the flow-reference edge,
// and recurses into an inline target. A provider target needs nothing
// document-local.
func (c *flowChecker) checkCall(f *Flow, call *Call, chain []*Flow) {
	if call.Flow == nil {
		return
	}
	c.pushKey("flow")
	defer c.pop()

	if target := call.Flow.Name; target != nil {
		for _, scope := range chain {
			if nf, ok := scope.FlowByName(target.Value); ok {
				c.edges[f] = append(c.edges[f], flowEdge{
					to:   nf.Flow,
					pos:  target.Pos,
					path: renderPath(c.path, ""),
				})
				return
			}
		}
		c.report(CodeUnresolvedFlow, target.Pos,
			renderPath(c.path, ""),
			"flow name %q resolves to no flows entry in scope", target.Value)
		return
	}

	inline := call.Flow.Inline
	c.edges[f] = append(c.edges[f], flowEdge{
		to:   inline,
		pos:  inline.Pos,
		path: renderPath(c.path, ""),
	})
	c.checkFlow(inline, prependFlow(inline, chain), "(inline)")
}

// detectCycles runs a depth-first search over the flow-reference graph and
// reports every back edge: a reference that closes a cycle.
func (c *flowChecker) detectCycles() {
	const (
		unvisited = iota
		visiting
		done
	)
	state := make(map[*Flow]int)
	var stack []*Flow

	var visit func(f *Flow)
	visit = func(f *Flow) {
		state[f] = visiting
		stack = append(stack, f)
		for _, e := range c.edges[f] {
			switch state[e.to] {
			case visiting:
				c.report(CodeFlowCycle, e.pos, e.path,
					"flow references form a cycle: %s", c.cycleNames(stack, e.to))
			case unvisited:
				visit(e.to)
			}
		}
		stack = stack[:len(stack)-1]
		state[f] = done
	}
	for _, n := range c.nodes {
		if state[n] == unvisited {
			visit(n)
		}
	}
}

// cycleNames renders the cycle a back edge to target closes, from the
// target's position on the visiting stack around to the target again.
func (c *flowChecker) cycleNames(stack []*Flow, target *Flow) string {
	start := 0
	for i, f := range stack {
		if f == target {
			start = i
			break
		}
	}
	var names []string
	for _, f := range stack[start:] {
		names = append(names, c.names[f])
	}
	names = append(names, c.names[target])
	return strings.Join(names, " -> ")
}

func prependFlow(f *Flow, chain []*Flow) []*Flow {
	out := make([]*Flow, 0, len(chain)+1)
	out = append(out, f)
	return append(out, chain...)
}
