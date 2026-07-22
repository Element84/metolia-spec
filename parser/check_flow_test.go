package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// checkSrc runs all three passes over src, failing the test if decode or
// bind report anything: check tests exercise documents that are structurally
// clean.
func checkSrc(t *testing.T, src string) Diagnostics {
	t.Helper()
	v, diags := decode([]byte(src), "test.json")
	if len(diags) != 0 {
		t.Fatalf("decode diagnostics: %v", diags)
	}
	f, diags := bind(v)
	if len(diags) != 0 {
		t.Fatalf("bind diagnostics: %v", diags)
	}
	return check(f)
}

func TestCheckDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
		path string
	}{
		{
			name: "entrypoint unknown",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "strat", "steps": {"start": {"action": "Return"}}}`,
			code: CodeUnresolvedStep,
			path: "entrypoint",
		},
		{
			name: "next unknown on pass step",
			src:  flowDoc(`{"a": {"action": "Pass", "next": "b"}}`),
			code: CodeUnresolvedStep,
			path: "steps.a.next",
		},
		{
			name: "next unknown on catch clause",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}, "catch": [{"match": {"codes": ["*"]}, "next": "handler"}], "next": "done"}, "done": {"action": "Return"}}`),
			code: CodeUnresolvedStep,
			path: "steps.a.catch[0].next",
		},
		{
			name: "next unknown on match clause",
			src:  flowDoc(`{"a": {"action": "Match", "cases": [{"when": "{{ true }}", "next": "missing"}], "default": {"next": "done"}}, "done": {"action": "Return"}}`),
			code: CodeUnresolvedStep,
			path: "steps.a.cases[0].next",
		},
		{
			name: "next unknown on match default",
			src:  flowDoc(`{"a": {"action": "Match", "cases": [], "default": {"next": "missing"}}, "done": {"action": "Return"}}`),
			code: CodeUnresolvedStep,
			path: "steps.a.default.next",
		},
		{
			name: "next cannot cross into subflow steps",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "flows": {"F": {"entrypoint": "inner", "steps": {"inner": {"action": "Return"}}}}, "steps": {"a": {"action": "Pass", "next": "inner"}}}`,
			code: CodeUnresolvedStep,
			path: "steps.a.next",
		},
		{
			name: "flow name unresolved",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"flow": "Missing"}, "next": "done"}, "done": {"action": "Return"}}`),
			code: CodeUnresolvedFlow,
			path: "steps.a.call.flow",
		},
		{
			name: "flow name not visible from sibling scope",
			src: `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
				"flows": {
					"A": {"entrypoint": "s", "flows": {"Inner": {"entrypoint": "t", "steps": {"t": {"action": "Return"}}}}, "steps": {"s": {"action": "Return"}}},
					"B": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "Inner"}, "next": "t"}, "t": {"action": "Return"}}}
				},
				"steps": {"a": {"action": "Return"}}}`,
			code: CodeUnresolvedFlow,
			path: "flows.B.steps.s.call.flow",
		},
		{
			name: "self reference cycles",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "flows": {"A": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "A"}, "next": "t"}, "t": {"action": "Return"}}}}, "steps": {"a": {"action": "Call", "call": {"flow": "A"}, "next": "z"}, "z": {"action": "Return"}}}`,
			code: CodeFlowCycle,
			path: "flows.A.steps.s.call.flow",
		},
		{
			name: "mutual reference cycles",
			src: `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
				"flows": {
					"A": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "B"}, "next": "t"}, "t": {"action": "Return"}}},
					"B": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "A"}, "next": "t"}, "t": {"action": "Return"}}}
				},
				"steps": {"a": {"action": "Call", "call": {"flow": "A"}, "next": "z"}, "z": {"action": "Return"}}}`,
			code: CodeFlowCycle,
		},
		{
			name: "inline flow closes a cycle",
			src: `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
				"flows": {
					"A": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": {"entrypoint": "i", "steps": {"i": {"action": "Call", "call": {"flow": "A"}, "next": "j"}, "j": {"action": "Return"}}}}, "next": "t"}, "t": {"action": "Return"}}}
				},
				"steps": {"a": {"action": "Call", "call": {"flow": "A"}, "next": "z"}, "z": {"action": "Return"}}}`,
			code: CodeFlowCycle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := checkSrc(t, tt.src)
			if !diags.HasErrors() {
				t.Fatalf("no errors; want code %q", tt.code)
			}
			for _, d := range diags {
				if d.Code == tt.code && (tt.path == "" || d.Path == tt.path) {
					return
				}
			}
			t.Errorf("no diagnostic with code %q at path %q; got %v", tt.code, tt.path, diags)
		})
	}
}

func TestCheckCleanDocuments(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "routing loop is not a flow cycle",
			src:  flowDoc(`{"a": {"action": "Pass", "next": "b"}, "b": {"action": "Match", "cases": [{"when": "{{ step.input.again }}", "next": "a"}], "default": {"next": "done"}}, "done": {"action": "Return"}}`),
		},
		{
			name: "diamond of flow references is not a cycle",
			src: `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
				"flows": {
					"D": {"entrypoint": "s", "steps": {"s": {"action": "Return"}}},
					"B": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "D"}, "next": "t"}, "t": {"action": "Return"}}},
					"C": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "D"}, "next": "t"}, "t": {"action": "Return"}}}
				},
				"steps": {"a": {"action": "Call", "call": {"flow": "B"}, "next": "b"}, "b": {"action": "Call", "call": {"flow": "C"}, "next": "z"}, "z": {"action": "Return"}}}`,
		},
		{
			name: "subflow resolves a name declared at the root",
			src: `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
				"flows": {
					"Shared": {"entrypoint": "s", "steps": {"s": {"action": "Return"}}},
					"Outer": {"entrypoint": "s", "steps": {"s": {"action": "Call", "call": {"flow": "Shared"}, "next": "t"}, "t": {"action": "Return"}}}
				},
				"steps": {"a": {"action": "Call", "call": {"flow": "Outer"}, "next": "z"}, "z": {"action": "Return"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diags := checkSrc(t, tt.src); len(diags) != 0 {
				t.Errorf("unexpected diagnostics: %v", diags)
			}
		})
	}
}

func TestCheckShadowWarning(t *testing.T) {
	// The inner flows entry "A" shadows the root's "A". The call inside
	// the outer A resolves to the nearer inner A, so no cycle exists; the
	// only finding is the warning.
	src := `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a",
		"flows": {
			"A": {
				"entrypoint": "s",
				"flows": {"A": {"entrypoint": "t", "steps": {"t": {"action": "Return"}}}},
				"steps": {"s": {"action": "Call", "call": {"flow": "A"}, "next": "u"}, "u": {"action": "Return"}}
			}
		},
		"steps": {"a": {"action": "Call", "call": {"flow": "A"}, "next": "z"}, "z": {"action": "Return"}}}`
	diags := checkSrc(t, src)

	if diags.HasErrors() {
		t.Fatalf("shadowing is legal; got errors: %v", diags)
	}
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %v, want exactly the shadow warning", diags)
	}
	d := diags[0]
	if d.Severity != SeverityWarning || d.Code != CodeShadowedFlow {
		t.Errorf("diagnostic = %+v, want a %s warning", d, CodeShadowedFlow)
	}
	if d.Path != "flows.A.flows.A" {
		t.Errorf("path = %q, want flows.A.flows.A", d.Path)
	}
}

// TestCheckFixtureCorpusValid runs the complete pipeline over the valid
// corpus: a valid fixture must come through all three passes clean.
func TestCheckFixtureCorpusValid(t *testing.T) {
	paths, err := filepath.Glob("../tests/schemas/valid/flow/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no valid fixtures found")
	}
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		v, diags := decode(src, filepath.Base(path))
		if len(diags) != 0 {
			t.Errorf("%s: decode diagnostics: %v", path, diags)
			continue
		}
		f, diags := bind(v)
		if len(diags) != 0 {
			t.Errorf("%s: bind diagnostics: %v", path, diags)
			continue
		}
		if diags := check(f); len(diags) != 0 {
			t.Errorf("%s: check diagnostics: %v", path, diags)
		}
	}
}
