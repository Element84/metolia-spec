package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// callProviderDoc builds a minimal valid call-provider definition with the
// given failureCatalog value.
func callProviderDoc(catalog string) string {
	return callProviderDocWith(`"failureCatalog": ` + catalog)
}

// middlewareProviderDoc builds a minimal valid middleware-provider
// definition with the given phases value.
func middlewareProviderDoc(phases string) string {
	return middlewareProviderDocWith(`"phases": ` + phases)
}

func TestParseProviderKinds(t *testing.T) {
	prov, diags := ParseProvider([]byte(callProviderDoc(`{"closed": [], "open": []}`)))
	if prov == nil || len(diags) != 0 {
		t.Fatalf("provider=%v diagnostics=%v, want clean parse", prov != nil, diags)
	}
	if prov.Call == nil || prov.Middleware != nil {
		t.Errorf("arms = (%v, %v), want the Call arm alone", prov.Call, prov.Middleware)
	}
	if prov.Call.Parameters == nil {
		t.Error("Parameters not retained")
	}
	if prov.URI.Value != "mwl:provider.call/example/echo/v1" || prov.CodePrefix.Value != "Echo" {
		t.Errorf("shared fields = %q, %q", prov.URI.Value, prov.CodePrefix.Value)
	}

	prov, diags = ParseProvider([]byte(middlewareProviderDoc(
		`{"onEntry": {"action": {"kind": "side-effect", "description": "Records entry."}}}`)))
	if prov == nil || len(diags) != 0 {
		t.Fatalf("provider=%v diagnostics=%v, want clean parse", prov != nil, diags)
	}
	if prov.Middleware == nil || prov.Call != nil {
		t.Fatalf("arms = (%v, %v), want the Middleware arm alone", prov.Call, prov.Middleware)
	}
	if prov.Middleware.Phases.OnEntry == nil {
		t.Fatal("onEntry contract not bound")
	}
	if kind := prov.Middleware.Phases.OnEntry.Action.Kind.Value; kind != ActionKindSideEffect {
		t.Errorf("action kind = %q", kind)
	}
}

func TestParseProviderNilOnError(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"syntax error", `{"uri": }`},
		{"decode error with surviving tree", `{"uri": "mwl:provider.call/example/echo/v1", "uri": "mwl:provider.call/example/echo/v2"}`},
		{"bind error", callProviderDoc(`{"closed": ["Provider.Call.Echo"], "open": []}`)},
		{"unknown kind segment", `{"$schema": "https://mwl.dev/v0.1/provider/schema.json", "uri": "mwl:provider.saga/example/x/v1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov, diags := ParseProvider([]byte(tt.src))
			if prov != nil {
				t.Error("Provider should be nil when errors exist")
			}
			if !diags.HasErrors() {
				t.Error("expected error diagnostics")
			}
		})
	}
}

func TestParseProviderWarningsKeepTheProvider(t *testing.T) {
	src := []byte(callProviderDoc(`{"closed": ["Provider.Call.Other.NotFound"], "open": []}`))
	prov, diags := ParseProvider(src)
	if prov == nil {
		t.Fatal("warnings alone must not nil the Provider")
	}
	if len(diags) != 1 || diags[0].Code != CodeCatalogOutsidePrefix || diags[0].Severity != SeverityWarning {
		t.Errorf("diagnostics = %v, want exactly the catalog warning", diags)
	}
}

// TestParseProviderFixtureCorpus runs the public entry point over the
// provider fixture corpus — the schema fixtures plus the published
// reference definitions — mirroring TestParseFlowFixtureCorpus: the parser
// and the published schema must agree on every verdict.
func TestParseProviderFixtureCorpus(t *testing.T) {
	valid, _ := filepath.Glob("../tests/schemas/valid/provider/*.json")
	reference, _ := filepath.Glob("../content/reference/providers/*.v1.json")
	invalid, _ := filepath.Glob("../tests/schemas/invalid/provider/*.json")
	valid = append(valid, reference...)
	if len(valid) == 0 || len(invalid) == 0 {
		t.Fatal("provider fixture corpus not found")
	}
	for _, path := range valid {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		prov, diags := ParseProvider(src, WithFilename(filepath.Base(path)))
		if prov == nil || len(diags) != 0 {
			t.Errorf("%s: provider=%v diagnostics=%v, want clean parse", path, prov != nil, diags)
		}
	}
	for _, path := range invalid {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		prov, diags := ParseProvider(src, WithFilename(filepath.Base(path)))
		if prov != nil || !diags.HasErrors() {
			t.Errorf("%s: provider=%v diagnostics=%v, want nil provider with errors", path, prov != nil, diags)
		}
	}
}
