package parser

import "testing"

// checkProviderSrc runs all three passes over src, failing the test if
// decode or bind report anything: check tests exercise documents that are
// structurally clean.
func checkProviderSrc(t *testing.T, src string) Diagnostics {
	t.Helper()
	v, diags := decode([]byte(src), "test.json")
	if len(diags) != 0 {
		t.Fatalf("decode diagnostics: %v", diags)
	}
	p, diags := bindProvider(v)
	if len(diags) != 0 {
		t.Fatalf("bind diagnostics: %v", diags)
	}
	return checkProvider(p)
}

// TestCheckProviderCatalogWarnings pins the catalog-trespass warning: an
// entry whose kind or codePrefix segment disagrees with the provider's own
// warns at the entry's path; the lone "*" and in-slice entries do not.
func TestCheckProviderCatalogWarnings(t *testing.T) {
	tests := []struct {
		name    string
		catalog string
		paths   []string
	}{
		{
			name:    "closed code under another codePrefix",
			catalog: `{"closed": ["Provider.Call.Other.NotFound"], "open": []}`,
			paths:   []string{"failureCatalog.closed[0]"},
		},
		{
			name:    "closed code under the other kind",
			catalog: `{"closed": ["Provider.Middleware.Echo.NotFound"], "open": []}`,
			paths:   []string{"failureCatalog.closed[0]"},
		},
		{
			name:    "trespassing entry at a later index",
			catalog: `{"closed": ["Provider.Call.Echo.NotFound", "Provider.Call.Other.NotFound"], "open": []}`,
			paths:   []string{"failureCatalog.closed[1]"},
		},
		{
			name:    "open sub-prefix outside the slice, past a lone star",
			catalog: `{"closed": [], "open": ["*", "Provider.Call.Other.Backend.*"]}`,
			paths:   []string{"failureCatalog.open[1]"},
		},
		{
			name:    "lone star is exempt",
			catalog: `{"closed": [], "open": ["*"]}`,
			paths:   nil,
		},
		{
			name:    "entries inside the slice",
			catalog: `{"closed": ["Provider.Call.Echo.NotFound"], "open": ["Provider.Call.Echo.Backend.*"]}`,
			paths:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := checkProviderSrc(t, callProviderDoc(tt.catalog))
			assertWarnings(t, diags, CodeCatalogOutsidePrefix, tt.paths)
		})
	}
}

// TestCheckProviderStructuralWarnings pins the structural-name warning: a
// name matching no key of its phase's parameters top-level properties warns
// at the entry's path.
func TestCheckProviderStructuralWarnings(t *testing.T) {
	tests := []struct {
		name   string
		phases string
		paths  []string
	}{
		{
			name: "name matching no properties key, past a matching one",
			phases: `{"onEntry": {
				"action": {"kind": "side-effect", "description": "d"},
				"parameters": {"type": "object", "properties": {"shape": {}}},
				"structural": ["shape", "shpae"]
			}}`,
			paths: []string{"phases.onEntry.structural[1]"},
		},
		{
			name: "name without a parameters schema",
			phases: `{"onEntry": {
				"action": {"kind": "side-effect", "description": "d"},
				"structural": ["shape"]
			}}`,
			paths: []string{"phases.onEntry.structural[0]"},
		},
		{
			name: "warning carries the declaring phase",
			phases: `{
				"onEntry": {"action": {"kind": "side-effect", "description": "d"}},
				"onSuccess": {
					"action": {"kind": "transform", "description": "d"},
					"structural": ["shape"]
				}
			}`,
			paths: []string{"phases.onSuccess.structural[0]"},
		},
		{
			name: "name matching a properties key",
			phases: `{"onEntry": {
				"action": {"kind": "side-effect", "description": "d"},
				"parameters": {"type": "object", "properties": {"shape": {}}},
				"structural": ["shape"]
			}}`,
			paths: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := checkProviderSrc(t, middlewareProviderDoc(tt.phases))
			assertWarnings(t, diags, CodeStructuralUnknownParameter, tt.paths)
		})
	}
}

// assertWarnings requires diags to be exactly warnings of the given code at
// the given paths, in order.
func assertWarnings(t *testing.T, diags Diagnostics, code string, paths []string) {
	t.Helper()
	if len(diags) != len(paths) {
		t.Fatalf("diagnostics = %v, want %d warning(s) at %v", diags, len(paths), paths)
	}
	for i, d := range diags {
		if d.Severity != SeverityWarning || d.Code != code || d.Path != paths[i] {
			t.Errorf("diagnostic %d = %+v, want %s warning at %s", i, d, code, paths[i])
		}
	}
}
