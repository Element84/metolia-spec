package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFlowSuccess(t *testing.T) {
	src := []byte(flowDoc(`{"a": {"action": "Return"}}`))
	flow, diags := ParseFlow(src)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if flow == nil {
		t.Fatal("nil Flow without diagnostics")
	}
	if flow.Entrypoint.Value != "a" {
		t.Errorf("entrypoint = %q", flow.Entrypoint.Value)
	}
}

func TestParseFlowNilOnError(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"syntax error", `{"entrypoint": }`},
		{"decode error with surviving tree", `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "entrypoint": "b", "steps": {"a": {"action": "Return"}}}`},
		{"bind error", flowDoc(`{"a": {"action": "Frobnicate"}}`)},
		{"check error", flowDoc(`{"a": {"action": "Pass", "next": "missing"}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow, diags := ParseFlow([]byte(tt.src))
			if flow != nil {
				t.Error("Flow should be nil when errors exist")
			}
			if !diags.HasErrors() {
				t.Error("expected error diagnostics")
			}
		})
	}
}

func TestParseFlowWarningsKeepTheFlow(t *testing.T) {
	src := []byte(`{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
		"flows": {
			"A": {
				"entrypoint": "s",
				"flows": {"A": {"entrypoint": "t", "steps": {"t": {"action": "Return"}}}},
				"steps": {"s": {"action": "Call", "call": {"flow": "A"}, "next": "u"}, "u": {"action": "Return"}}
			}
		},
		"steps": {"a": {"action": "Return"}}}`)
	flow, diags := ParseFlow(src)
	if flow == nil {
		t.Fatal("warnings alone must not nil the Flow")
	}
	if len(diags) != 1 || diags[0].Code != CodeShadowedFlow || diags[0].Severity != SeverityWarning {
		t.Errorf("diagnostics = %v, want exactly the shadow warning", diags)
	}
}

func TestParseFlowFilenameLabel(t *testing.T) {
	src := []byte(flowDoc(`{"a": {"action": "Frobnicate"}}`))

	_, diags := ParseFlow(src, WithFilename("flow.json"))
	if len(diags) == 0 || diags[0].Pos.Filename != "flow.json" {
		t.Errorf("diagnostic position = %+v, want filename label", diags)
	}

	_, diags = ParseFlow(src)
	if len(diags) == 0 || diags[0].Pos.Filename != "" {
		t.Errorf("diagnostic position = %+v, want empty filename", diags)
	}
}

func TestParseFlowDiagnosticsInDocumentOrder(t *testing.T) {
	// The duplicate member sits late in the document and is found by the
	// decode pass; the unknown member sits early and is found by the bind
	// pass. Emission order is decode first, so only sorting puts the
	// early finding first.
	src := []byte(`{
		"$schema": "https://mwl.dev/v0.1/flow/schema.json",
		"bogus": 1,
		"entrypoint": "a",
		"steps": {"a": {"action": "Return", "comment": "x", "comment": "y"}}
	}`)
	_, diags := ParseFlow(src)
	if len(diags) < 2 {
		t.Fatalf("diagnostics = %v, want at least two", diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i].Pos.Offset < diags[i-1].Pos.Offset {
			t.Fatalf("diagnostics out of document order: %v", diags)
		}
	}
	if diags[0].Code != CodeUnknownMember {
		t.Errorf("first diagnostic = %+v, want the early unknown-member", diags[0])
	}
}

// TestParseFlowFixtureCorpus runs the public entry point over the schema
// fixture corpus: valid fixtures parse into a Flow with no diagnostics,
// invalid fixtures return nil with errors.
func TestParseFlowFixtureCorpus(t *testing.T) {
	valid, _ := filepath.Glob("../tests/schemas/valid/flow/*.json")
	invalid, _ := filepath.Glob("../tests/schemas/invalid/flow/*.json")
	if len(valid) == 0 || len(invalid) == 0 {
		t.Fatal("fixture corpus not found under ../tests/schemas/")
	}
	for _, path := range valid {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		flow, diags := ParseFlow(src, WithFilename(filepath.Base(path)))
		if flow == nil || len(diags) != 0 {
			t.Errorf("%s: flow=%v diagnostics=%v, want clean parse", path, flow != nil, diags)
		}
	}
	for _, path := range invalid {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		flow, diags := ParseFlow(src, WithFilename(filepath.Base(path)))
		if flow != nil || !diags.HasErrors() {
			t.Errorf("%s: flow=%v diagnostics=%v, want nil flow with errors", path, flow != nil, diags)
		}
	}
}
