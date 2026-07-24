package parser

import (
	"strings"
)

// providerSchemaURI is the $schema value a provider definition must carry.
const providerSchemaURI = "https://mwl.dev/v0.1/provider/schema.json"

// bindProvider is the structural pass for a provider definition: it walks
// the positioned value tree and produces the typed Provider, validating
// shape as it goes. The document's kind — call or middleware — is
// discriminated by the type segment of its uri, so the uri is read first;
// without a usable uri there is no shape to bind against.
func bindProvider(root *Value) (*Provider, Diagnostics) {
	b := &binder{}
	if root.Kind != KindObject {
		b.report(CodeWrongType, root.Pos,
			"the root of a provider definition must be an object, not %s", root.Kind)
		return nil, b.diags
	}
	p := b.bindProviderRoot(root)
	return p, b.diags
}

func (b *binder) bindProviderRoot(v *Value) *Provider {
	uriVal := v.Field("uri")
	if uriVal == nil {
		b.missingMember(v.Pos, "a provider definition requires uri")
		return nil
	}
	b.pushKey("uri")
	uri, ok := b.requireString(uriVal, "uri")
	if !ok {
		b.pop()
		return nil
	}
	kind, known := providerURIKind(uri)
	if !known {
		b.report(CodeInvalidProviderURI, uriVal.Pos,
			"%q is not a well-formed provider URI; expected mwl:provider.call/... or mwl:provider.middleware/...",
			uri)
		b.pop()
		return nil
	}
	if !validProviderURI(uri, kind) {
		b.invalidProviderURI(uriVal.Pos, uri, kind)
	}
	b.pop()

	p := &Provider{Pos: v.Pos, URI: StringLit{Value: uri, Pos: uriVal.Pos}}
	if kind == "call" {
		p.Call = &CallProvider{}
	} else {
		p.Middleware = &MiddlewareProvider{}
	}
	in := "a " + kind + "-provider definition"

	var hasSchema, hasCodePrefix, hasDescription, hasCatalog bool
	var hasParameters, hasAttachment, hasPhases bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "$schema":
			hasSchema = true
			if s, ok := b.requireString(m.Value, "$schema"); ok {
				p.Schema = s
				if s != providerSchemaURI {
					b.report(CodeInvalidValue, m.Value.Pos, "$schema must be %q", providerSchemaURI)
				}
			}
		case "uri":
			// Bound above, before the sweep.
		case "codePrefix":
			hasCodePrefix = true
			if s, ok := b.requireString(m.Value, "codePrefix"); ok {
				if !validCodePrefix(s) {
					b.report(CodeInvalidValue, m.Value.Pos,
						"codePrefix must be an uppercase letter followed by letters and digits")
				}
				p.CodePrefix = StringLit{Value: s, Pos: m.Value.Pos}
			}
		case "description":
			hasDescription = true
			p.Description, _ = b.requireString(m.Value, "description")
		case "metadata":
			p.Metadata = b.bindObjectSchema(m.Value, "metadata")
		case "failureCatalog":
			hasCatalog = true
			p.FailureCatalog = b.bindFailureCatalog(m.Value)
		case "parameters":
			if p.Call == nil {
				b.unknownMember(m, in)
				break
			}
			hasParameters = true
			p.Call.Parameters = b.bindObjectSchema(m.Value, "parameters")
		case "attachment":
			if p.Middleware == nil {
				b.unknownMember(m, in)
				break
			}
			hasAttachment = true
			p.Middleware.Attachment = b.bindAttachment(m.Value)
		case "phases":
			if p.Middleware == nil {
				b.unknownMember(m, in)
				break
			}
			hasPhases = true
			p.Middleware.Phases = b.bindPhaseContracts(m.Value)
		default:
			b.unknownMember(m, in)
		}
		b.pop()
	}

	if !hasSchema {
		b.missingMember(v.Pos, "a provider definition requires $schema")
	}
	if !hasCodePrefix {
		b.missingMember(v.Pos, "a provider definition requires codePrefix")
	}
	if !hasDescription {
		b.missingMember(v.Pos, "a provider definition requires description")
	}
	if !hasCatalog {
		b.missingMember(v.Pos, "a provider definition requires failureCatalog")
	}
	if p.Call != nil && !hasParameters {
		b.missingMember(v.Pos, "a call-provider definition requires parameters")
	}
	if p.Middleware != nil {
		if !hasAttachment {
			b.missingMember(v.Pos, "a middleware-provider definition requires attachment")
		}
		if !hasPhases {
			b.missingMember(v.Pos, "a middleware-provider definition requires phases")
		}
	}
	return p
}

func (b *binder) bindAttachment(v *Value) []StringLit {
	if v.Kind != KindArray {
		b.report(CodeWrongType, v.Pos, "attachment must be an array of levels, not %s", v.Kind)
		return nil
	}
	if len(v.Array) == 0 {
		b.report(CodeInvalidValue, v.Pos, "attachment must name at least one level")
		return nil
	}
	var list []StringLit
	seen := make(map[string]bool)
	for i, e := range v.Array {
		b.pushIdx(i)
		if s, ok := b.requireString(e, "attachment entry"); ok {
			switch {
			case s != AttachmentStep && s != AttachmentFlow:
				b.report(CodeInvalidValue, e.Pos,
					`unknown attachment level %q; expected "step" or "flow"`, s)
			case seen[s]:
				b.report(CodeInvalidValue, e.Pos, "attachment repeats level %q", s)
			}
			seen[s] = true
			list = append(list, StringLit{Value: s, Pos: e.Pos})
		}
		b.pop()
	}
	return list
}

func (b *binder) bindPhaseContracts(v *Value) PhaseContracts {
	if v.Kind != KindObject {
		b.report(CodeWrongType, v.Pos, "phases must be an object, not %s", v.Kind)
		return PhaseContracts{Pos: v.Pos}
	}
	p := PhaseContracts{Pos: v.Pos}
	if len(v.Object) == 0 {
		b.report(CodeInvalidValue, v.Pos, "phases must declare at least one phase")
	}
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "onEntry":
			p.OnEntry = b.bindPhaseContract(m.Value)
		case "onSuccess":
			p.OnSuccess = b.bindPhaseContract(m.Value)
		case "onFailure":
			p.OnFailure = b.bindPhaseContract(m.Value)
		case "onAlways":
			p.OnAlways = b.bindPhaseContract(m.Value)
		default:
			b.report(CodeUnknownMember, m.NamePos,
				"unknown phase %q; expected onEntry, onSuccess, onFailure, or onAlways", m.Name)
		}
		b.pop()
	}
	return p
}

func (b *binder) bindPhaseContract(v *Value) *PhaseContract {
	if v.Kind != KindObject {
		b.report(CodeWrongType, v.Pos, "a phase contract must be an object, not %s", v.Kind)
		return nil
	}
	c := &PhaseContract{Pos: v.Pos}
	var hasAction bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "action":
			hasAction = true
			if a := b.bindPhaseAction(m.Value); a != nil {
				c.Action = *a
			}
		case "parameters":
			c.Parameters = b.bindObjectSchema(m.Value, "parameters")
		case "structural":
			c.Structural = b.bindStructural(m.Value)
		default:
			b.unknownMember(m, "a phase contract")
		}
		b.pop()
	}
	if !hasAction {
		b.missingMember(v.Pos, "a phase contract requires action")
	}
	return c
}

func (b *binder) bindPhaseAction(v *Value) *PhaseAction {
	if v.Kind != KindObject {
		b.report(CodeWrongType, v.Pos, "action must be an object, not %s", v.Kind)
		return nil
	}
	a := &PhaseAction{Pos: v.Pos}
	var hasKind, hasDescription bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "kind":
			hasKind = true
			if s, ok := b.requireString(m.Value, "kind"); ok {
				if s != ActionKindSideEffect && s != ActionKindControl && s != ActionKindTransform {
					b.report(CodeInvalidValue, m.Value.Pos,
						"unknown action kind %q; expected side-effect, control, or transform", s)
				}
				a.Kind = StringLit{Value: s, Pos: m.Value.Pos}
			}
		case "description":
			hasDescription = true
			a.Description, _ = b.requireString(m.Value, "description")
		case "acceptance":
			a.Acceptance, _ = b.requireString(m.Value, "acceptance")
		default:
			b.unknownMember(m, "a phase action")
		}
		b.pop()
	}
	if !hasKind {
		b.missingMember(v.Pos, "a phase action requires kind")
	}
	if !hasDescription {
		b.missingMember(v.Pos, "a phase action requires description")
	}
	return a
}

func (b *binder) bindStructural(v *Value) []StringLit {
	if v.Kind != KindArray {
		b.report(CodeWrongType, v.Pos, "structural must be an array of parameter names, not %s", v.Kind)
		return nil
	}
	var list []StringLit
	for i, e := range v.Array {
		b.pushIdx(i)
		if s, ok := b.requireString(e, "structural entry"); ok {
			list = append(list, StringLit{Value: s, Pos: e.Pos})
		}
		b.pop()
	}
	return list
}

func (b *binder) bindFailureCatalog(v *Value) FailureCatalog {
	if v.Kind != KindObject {
		b.report(CodeWrongType, v.Pos, "failureCatalog must be an object, not %s", v.Kind)
		return FailureCatalog{Pos: v.Pos}
	}
	c := FailureCatalog{Pos: v.Pos}
	var hasClosed, hasOpen bool
	for _, m := range v.Object {
		b.pushKey(m.Name)
		switch m.Name {
		case "closed":
			hasClosed = true
			c.Closed = b.bindCatalogList(m.Value, "closed", validClosedCode,
				"a closed code must be fully qualified: Provider.<Kind>.<codePrefix>.<Code>")
		case "open":
			hasOpen = true
			c.Open = b.bindCatalogList(m.Value, "open", validOpenPrefix,
				`an open entry must be a sub-prefix ending in ".*", or "*" alone`)
		default:
			b.unknownMember(m, "a failure catalog")
		}
		b.pop()
	}
	if !hasClosed {
		b.missingMember(v.Pos, "a failure catalog requires closed")
	}
	if !hasOpen {
		b.missingMember(v.Pos, "a failure catalog requires open")
	}
	return c
}

func (b *binder) bindCatalogList(v *Value, what string, valid func(string) bool, requirement string) []StringLit {
	if v.Kind != KindArray {
		b.report(CodeWrongType, v.Pos, "%s must be an array of strings, not %s", what, v.Kind)
		return nil
	}
	var list []StringLit
	for i, e := range v.Array {
		b.pushIdx(i)
		if s, ok := b.requireString(e, what+" entry"); ok {
			if !valid(s) {
				b.report(CodeInvalidValue, e.Pos, "%s", requirement)
			}
			list = append(list, StringLit{Value: s, Pos: e.Pos})
		}
		b.pop()
	}
	return list
}

// validCodePrefix reports whether s is a codePrefix: an uppercase letter
// followed by letters and digits.
func validCodePrefix(s string) bool {
	if len(s) == 0 || s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isAlnum(s[i]) {
			return false
		}
	}
	return true
}

// validClosedCode reports whether s is a fully qualified failure code:
// Provider, a kind segment, then at least a codePrefix segment and a code
// segment, all alphanumeric.
func validClosedCode(s string) bool {
	segs, ok := providerCodeSegments(s)
	return ok && len(segs) >= 4 && alnumSegments(segs[2:])
}

// validOpenPrefix reports whether s is an open sub-prefix: "*" alone, or
// Provider, a kind segment, at least one alphanumeric segment, and a final
// "*".
func validOpenPrefix(s string) bool {
	if s == "*" {
		return true
	}
	segs, ok := providerCodeSegments(s)
	if !ok || len(segs) < 4 || segs[len(segs)-1] != "*" {
		return false
	}
	return alnumSegments(segs[2 : len(segs)-1])
}

// providerCodeSegments splits a catalog entry on "." and verifies its fixed
// head: "Provider" then "Call" or "Middleware".
func providerCodeSegments(s string) ([]string, bool) {
	segs := strings.Split(s, ".")
	if len(segs) < 2 || segs[0] != "Provider" || (segs[1] != "Call" && segs[1] != "Middleware") {
		return nil, false
	}
	return segs, true
}

func alnumSegments(segs []string) bool {
	for _, seg := range segs {
		if seg == "" {
			return false
		}
		for i := 0; i < len(seg); i++ {
			if !isAlnum(seg[i]) {
				return false
			}
		}
	}
	return true
}

func isAlnum(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}
