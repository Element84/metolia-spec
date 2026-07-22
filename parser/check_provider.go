package parser

import "strings"

// checkProvider is the provider check pass: the findings that span the
// document's fields. Both are spec-authorized warnings — the document is
// legal, but rarely what the author meant. A failure-catalog entry is
// expected to sit under the provider's own kind and codePrefix segments
// (the lone "*" aside), and a structural name is expected to name a key of
// its phase's parameters top-level properties. It expects an AST the bind
// pass produced without errors.
func checkProvider(p *Provider) Diagnostics {
	c := &checker{}

	kind := "Call"
	if p.Middleware != nil {
		kind = "Middleware"
	}
	c.pushKey("failureCatalog")
	c.checkCatalogEntries(p, kind, "closed", p.FailureCatalog.Closed)
	c.checkCatalogEntries(p, kind, "open", p.FailureCatalog.Open)
	c.pop()

	if p.Middleware != nil {
		c.pushKey("phases")
		for _, ph := range []struct {
			name     string
			contract *PhaseContract
		}{
			{"onEntry", p.Middleware.Phases.OnEntry},
			{"onSuccess", p.Middleware.Phases.OnSuccess},
			{"onFailure", p.Middleware.Phases.OnFailure},
			{"onAlways", p.Middleware.Phases.OnAlways},
		} {
			if ph.contract == nil {
				continue
			}
			c.pushKey(ph.name)
			c.checkStructural(ph.contract)
			c.pop()
		}
		c.pop()
	}
	return c.diags
}

// checkCatalogEntries warns where an entry's kind or codePrefix segment
// disagrees with the provider's own. The lone "*" declares the whole code
// space open and is exempt; every other entry reaching this pass carries
// its kind and codePrefix segments, since malformed entries are bind
// errors and errors skip the check pass.
func (c *checker) checkCatalogEntries(p *Provider, kind, member string, entries []StringLit) {
	c.pushKey(member)
	for i, e := range entries {
		if e.Value == "*" {
			continue
		}
		segs := strings.Split(e.Value, ".")
		if segs[1] != kind || segs[2] != p.CodePrefix.Value {
			c.pushIdx(i)
			c.report(CodeCatalogOutsidePrefix, e.Pos, renderPath(c.path, ""),
				"%q is outside the provider's own code space, Provider.%s.%s.*",
				e.Value, kind, p.CodePrefix.Value)
			c.pop()
		}
	}
	c.pop()
}

// checkStructural warns where a structural name matches no key of the
// phase's parameters top-level properties: the declaration marks nothing,
// and the member it was meant to mark evaluates as an ordinary with member.
func (c *checker) checkStructural(ph *PhaseContract) {
	if len(ph.Structural) == 0 {
		return
	}
	keys := make(map[string]bool)
	if props := ph.Parameters.Field("properties"); props != nil && props.Kind == KindObject {
		for _, m := range props.Object {
			keys[m.Name] = true
		}
	}
	c.pushKey("structural")
	for i, s := range ph.Structural {
		if !keys[s.Value] {
			c.pushIdx(i)
			c.report(CodeStructuralUnknownParameter, s.Pos, renderPath(c.path, ""),
				"structural name %q matches no key of the phase's parameters properties", s.Value)
			c.pop()
		}
	}
	c.pop()
}
