package parser

import (
	"strings"
	"testing"
)

// bindProviderSrc runs the decode and provider bind passes over src.
func bindProviderSrc(t *testing.T, src string) (*Provider, Diagnostics) {
	t.Helper()
	v, diags := decode([]byte(src), "test.json")
	if diags.HasErrors() {
		t.Fatalf("decode failed: %v", diags)
	}
	p, bindDiags := bindProvider(v)
	return p, append(diags, bindDiags...)
}

// TestBindProviderDiagnostics pins the diagnosis, not just the verdict: each
// rejection rule fires its intended code at its intended document path.
func TestBindProviderDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
		path string
	}{
		{
			name: "root not an object",
			src:  `[]`,
			code: "wrong-type",
			path: "",
		},
		{
			name: "uri missing",
			src:  `{"$schema": "https://mwl.dev/v0.1/provider/schema.json"}`,
			code: "missing-member",
			path: "",
		},
		{
			name: "uri not a string",
			src:  `{"uri": 7}`,
			code: "wrong-type",
			path: "uri",
		},
		{
			name: "uri unknown type segment",
			src:  `{"uri": "mwl:provider.saga/example/x/v1"}`,
			code: "invalid-provider-uri",
			path: "uri",
		},
		{
			name: "uri too few segments",
			src:  callProviderDocWith(`"uri": "mwl:provider.call/example"`),
			code: "invalid-provider-uri",
			path: "uri",
		},
		{
			name: "uri dot segment",
			src:  callProviderDocWith(`"uri": "mwl:provider.call/example/../echo"`),
			code: "invalid-provider-uri",
			path: "uri",
		},
		{
			name: "$schema missing",
			src:  `{"uri": "mwl:provider.call/example/echo/v1", "codePrefix": "Echo", "description": "d", "parameters": {"type": "object"}, "failureCatalog": {"closed": [], "open": []}}`,
			code: "missing-member",
			path: "",
		},
		{
			name: "$schema names the flow schema",
			src:  callProviderDocWith(`"$schema": "https://mwl.dev/v0.1/flow/schema.json"`),
			code: "invalid-value",
			path: "$schema",
		},
		{
			name: "codePrefix not Pascal",
			src:  callProviderDocWith(`"codePrefix": "echo"`),
			code: "invalid-value",
			path: "codePrefix",
		},
		{
			name: "codePrefix not a string",
			src:  callProviderDocWith(`"codePrefix": 7`),
			code: "wrong-type",
			path: "codePrefix",
		},
		{
			name: "description not a string",
			src:  callProviderDocWith(`"description": 7`),
			code: "wrong-type",
			path: "description",
		},
		{
			name: "metadata missing type",
			src:  callProviderDocWith(`"metadata": {}`),
			code: "missing-member",
			path: "metadata",
		},
		{
			name: "metadata type not object",
			src:  callProviderDocWith(`"metadata": {"type": "array"}`),
			code: "invalid-value",
			path: "metadata",
		},
		{
			name: "root unknown member",
			src:  callProviderDocWith(`"bogus": 1`),
			code: "unknown-member",
			path: "bogus",
		},
		{
			name: "call provider missing parameters",
			src:  `{"$schema": "https://mwl.dev/v0.1/provider/schema.json", "uri": "mwl:provider.call/example/echo/v1", "codePrefix": "Echo", "description": "d", "failureCatalog": {"closed": [], "open": []}}`,
			code: "missing-member",
			path: "",
		},
		{
			name: "call provider with attachment",
			src:  callProviderDocWith(`"attachment": ["step"]`),
			code: "unknown-member",
			path: "attachment",
		},
		{
			name: "middleware provider with parameters",
			src:  middlewareProviderDocWith(`"parameters": {"type": "object"}`),
			code: "unknown-member",
			path: "parameters",
		},
		{
			name: "middleware provider missing attachment",
			src:  `{"$schema": "https://mwl.dev/v0.1/provider/schema.json", "uri": "mwl:provider.middleware/example/trace/v1", "codePrefix": "Trace", "description": "d", "phases": {"onEntry": {"action": {"kind": "side-effect", "description": "d"}}}, "failureCatalog": {"closed": [], "open": []}}`,
			code: "missing-member",
			path: "",
		},
		{
			name: "attachment empty",
			src:  middlewareProviderDocWith(`"attachment": []`),
			code: "invalid-value",
			path: "attachment",
		},
		{
			name: "attachment duplicate level",
			src:  middlewareProviderDocWith(`"attachment": ["step", "step"]`),
			code: "invalid-value",
			path: "attachment[1]",
		},
		{
			name: "attachment unknown level",
			src:  middlewareProviderDocWith(`"attachment": ["step", "call"]`),
			code: "invalid-value",
			path: "attachment[1]",
		},
		{
			name: "attachment entry not a string",
			src:  middlewareProviderDocWith(`"attachment": [1]`),
			code: "wrong-type",
			path: "attachment[0]",
		},
		{
			name: "phases empty",
			src:  middlewareProviderDocWith(`"phases": {}`),
			code: "invalid-value",
			path: "phases",
		},
		{
			name: "phases unknown phase",
			src:  middlewareProviderDocWith(`"phases": {"onExit": {"action": {"kind": "side-effect", "description": "d"}}}`),
			code: "unknown-member",
			path: "phases.onExit",
		},
		{
			name: "phase contract missing action",
			src:  middlewareProviderDocWith(`"phases": {"onEntry": {"parameters": {"type": "object"}}}`),
			code: "missing-member",
			path: "phases.onEntry",
		},
		{
			name: "action unknown kind",
			src:  middlewareProviderDocWith(`"phases": {"onEntry": {"action": {"kind": "observer", "description": "d"}}}`),
			code: "invalid-value",
			path: "phases.onEntry.action.kind",
		},
		{
			name: "action missing description",
			src:  middlewareProviderDocWith(`"phases": {"onEntry": {"action": {"kind": "side-effect"}}}`),
			code: "missing-member",
			path: "phases.onEntry.action",
		},
		{
			name: "action unknown member",
			src:  middlewareProviderDocWith(`"phases": {"onEntry": {"action": {"kind": "side-effect", "description": "d", "bogus": 1}}}`),
			code: "unknown-member",
			path: "phases.onEntry.action.bogus",
		},
		{
			name: "phase parameters not an object schema",
			src:  middlewareProviderDocWith(`"phases": {"onEntry": {"action": {"kind": "side-effect", "description": "d"}, "parameters": {"type": "string"}}}`),
			code: "invalid-value",
			path: "phases.onEntry.parameters",
		},
		{
			name: "structural entry not a string",
			src:  middlewareProviderDocWith(`"phases": {"onEntry": {"action": {"kind": "side-effect", "description": "d"}, "structural": ["shape", 3]}}`),
			code: "wrong-type",
			path: "phases.onEntry.structural[1]",
		},
		{
			name: "failure catalog missing open",
			src:  callProviderDocWith(`"failureCatalog": {"closed": []}`),
			code: "missing-member",
			path: "failureCatalog",
		},
		{
			name: "closed code stops at the prefix",
			src:  callProviderDocWith(`"failureCatalog": {"closed": ["Provider.Call.Echo"], "open": []}`),
			code: "invalid-value",
			path: "failureCatalog.closed[0]",
		},
		{
			name: "closed code unknown kind segment",
			src:  callProviderDocWith(`"failureCatalog": {"closed": ["Provider.Saga.Echo.Bad"], "open": []}`),
			code: "invalid-value",
			path: "failureCatalog.closed[0]",
		},
		{
			name: "open entry not a sub-prefix",
			src:  callProviderDocWith(`"failureCatalog": {"closed": [], "open": ["Provider.Call.Echo.Bad"]}`),
			code: "invalid-value",
			path: "failureCatalog.open[0]",
		},
		{
			name: "catalog unknown member",
			src:  callProviderDocWith(`"failureCatalog": {"closed": [], "open": [], "bogus": []}`),
			code: "unknown-member",
			path: "failureCatalog.bogus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, diags := bindProviderSrc(t, tt.src)
			if !diags.HasErrors() {
				t.Fatalf("no errors; want code %q", tt.code)
			}
			for _, d := range diags {
				if d.Code == tt.code && d.Path == tt.path {
					return
				}
			}
			t.Errorf("no diagnostic with code %q at path %q; got %v", tt.code, tt.path, diags)
		})
	}
}

// TestBindProviderCollects verifies one parse surfaces independent findings
// together rather than stopping at the first.
func TestBindProviderCollects(t *testing.T) {
	src := `{
		"$schema": "https://mwl.dev/v0.1/provider/schema.json",
		"uri": "mwl:provider.call/example/echo/v1",
		"codePrefix": "echo",
		"description": "d",
		"bogus": 1,
		"parameters": {"type": "object"},
		"failureCatalog": {"closed": ["Provider.Call.Echo"], "open": []}
	}`
	_, diags := bindProviderSrc(t, src)
	want := map[string]bool{
		"invalid-value:codePrefix":               false,
		"unknown-member:bogus":                   false,
		"invalid-value:failureCatalog.closed[0]": false,
	}
	for _, d := range diags {
		key := d.Code + ":" + d.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("missing diagnostic %s; got %v", key, diags)
		}
	}
	if got := len(diags); got != len(want) {
		t.Errorf("diagnostics = %v, want exactly %d", diags, len(want))
	}
}

// callProviderDocWith builds a call-provider definition with one member
// overridden or added: entries in override replace same-named defaults.
func callProviderDocWith(override string) string {
	return providerDocWith(map[string]string{
		"$schema":        `"https://mwl.dev/v0.1/provider/schema.json"`,
		"uri":            `"mwl:provider.call/example/echo/v1"`,
		"codePrefix":     `"Echo"`,
		"description":    `"d"`,
		"parameters":     `{"type": "object"}`,
		"failureCatalog": `{"closed": [], "open": []}`,
	}, override)
}

// middlewareProviderDocWith builds a middleware-provider definition with one
// member overridden or added.
func middlewareProviderDocWith(override string) string {
	return providerDocWith(map[string]string{
		"$schema":        `"https://mwl.dev/v0.1/provider/schema.json"`,
		"uri":            `"mwl:provider.middleware/example/trace/v1"`,
		"codePrefix":     `"Trace"`,
		"description":    `"d"`,
		"attachment":     `["step"]`,
		"phases":         `{"onEntry": {"action": {"kind": "side-effect", "description": "d"}}}`,
		"failureCatalog": `{"closed": [], "open": []}`,
	}, override)
}

func providerDocWith(defaults map[string]string, override string) string {
	name := override[1:] // strip the opening quote
	name = name[:strings.IndexByte(name, '"')]

	doc := "{" + override
	for _, key := range []string{"$schema", "uri", "codePrefix", "description", "parameters", "attachment", "phases", "failureCatalog"} {
		val, ok := defaults[key]
		if !ok || key == name {
			continue
		}
		doc += `, "` + key + `": ` + val
	}
	return doc + "}"
}
