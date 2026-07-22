package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bindSrc runs the decode and bind passes over src.
func bindSrc(t *testing.T, src string) (*Flow, Diagnostics) {
	t.Helper()
	v, diags := decode([]byte(src), "test.json")
	if diags.HasErrors() {
		t.Fatalf("decode failed: %v", diags)
	}
	f, bindDiags := bind(v)
	return f, append(diags, bindDiags...)
}

// flowDoc wraps a steps object in a minimal valid root document.
func flowDoc(steps string) string {
	return `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "steps": ` + steps + `}`
}

func TestBindMinimalFlow(t *testing.T) {
	src := flowDoc(`{
		"a": {
			"action": "Call",
			"call": {
				"provider": "mwl:provider.call/example/http/v1",
				"with": {"method": "GET", "path": "/hello"}
			},
			"next": "done"
		},
		"done": {"action": "Return"}
	}`)
	f, diags := bindSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if f.Schema != flowSchemaURI {
		t.Errorf("Schema = %q", f.Schema)
	}
	if f.Entrypoint.Value != "a" {
		t.Errorf("Entrypoint = %q", f.Entrypoint.Value)
	}
	if len(f.Steps) != 2 {
		t.Fatalf("bound %d steps, want 2", len(f.Steps))
	}

	ns, ok := f.StepByName("a")
	if !ok {
		t.Fatal("StepByName(a) not found")
	}
	call, ok := ns.Step.(*CallStep)
	if !ok {
		t.Fatalf("step a is %T, want *CallStep", ns.Step)
	}
	if call.Next.Value != "done" {
		t.Errorf("next = %q", call.Next.Value)
	}
	if call.Call.Provider == nil || call.Call.Provider.URI != "mwl:provider.call/example/http/v1" {
		t.Errorf("provider = %+v", call.Call.Provider)
	}
	if call.Call.With == nil || call.Call.With.Literal == nil {
		t.Fatalf("with = %+v, want literal object", call.Call.With)
	}
	if members := *call.Call.With.Literal; len(members) != 2 || members[0].Name != "method" {
		t.Errorf("with members = %+v", members)
	}

	if done, ok := f.StepByName("done"); !ok {
		t.Error("StepByName(done) not found")
	} else if _, ok := done.Step.(*ReturnStep); !ok {
		t.Errorf("step done is %T, want *ReturnStep", done.Step)
	}
	if _, ok := f.StepByName("missing"); ok {
		t.Error("StepByName(missing) should not resolve")
	}
}

func TestBindCoverageFixture(t *testing.T) {
	src, err := os.ReadFile("../tests/schemas/valid/flow/schema-coverage.json")
	if err != nil {
		t.Fatal(err)
	}
	v, diags := decode(src, "schema-coverage.json")
	if len(diags) != 0 {
		t.Fatalf("decode diagnostics: %v", diags)
	}
	f, diags := bind(v)
	if len(diags) != 0 {
		t.Fatalf("bind diagnostics: %v", diags)
	}

	// Spot-check shapes the minimal test does not reach.
	if len(f.Middleware) != 1 || f.Middleware[0].OnFailure == nil {
		t.Fatalf("flow middleware = %+v", f.Middleware)
	}
	onFail := f.Middleware[0].OnFailure
	if onFail.Previous == nil || !onFail.Previous.Null {
		t.Errorf("onFailure previous = %+v, want explicit null", onFail.Previous)
	}
	if onFail.Retryable == nil || onFail.Retryable.Literal == nil || *onFail.Retryable.Literal {
		t.Errorf("onFailure retryable = %+v, want literal false", onFail.Retryable)
	}

	if _, ok := f.FlowByName("Child"); !ok {
		t.Error("FlowByName(Child) not found")
	}

	scatter, _ := f.StepByName("scatter")
	g := scatter.Step.(*GatherStep)
	if len(g.Calls) != 3 {
		t.Errorf("scatter calls = %d, want 3", len(g.Calls))
	}
	if g.Calls[2].Flow == nil || g.Calls[2].Flow.Inline == nil {
		t.Errorf("third scatter call should be an inline flow target")
	}
	if g.Concurrency == nil || g.Concurrency.Value != 2 {
		t.Errorf("concurrency = %+v, want 2", g.Concurrency)
	}
	if g.Completion == nil || g.Completion.Successes.Expr == nil {
		t.Errorf("scatter completion successes should be an expression")
	}
	if g.Completion.Wait == nil || *g.Completion.Wait {
		t.Errorf("scatter completion wait should be literal false")
	}

	iterate, _ := f.StepByName("iterate")
	ig := iterate.Step.(*GatherStep)
	if ig.Over == nil || ig.Over.Expr == nil {
		t.Errorf("iterate over should be an expression")
	}
	if ig.Completion == nil || ig.Completion.Successes.Literal == nil || *ig.Completion.Successes.Literal != 1 {
		t.Errorf("iterate successes = %+v, want literal 1", ig.Completion)
	}

	fetch, _ := f.StepByName("fetch")
	cs := fetch.Step.(*CallStep)
	if len(cs.Catch) != 2 || len(cs.Catch[0].Match.Codes) != 2 {
		t.Errorf("fetch catch = %+v", cs.Catch)
	}
	if cs.Catch[1].Match.Retryable == nil || *cs.Catch[1].Match.Retryable {
		t.Errorf("second clause retryable should be literal false")
	}

	failed, _ := f.StepByName("failed")
	rr := failed.Step.(*RaiseStep).Result
	if rr == nil || rr.Code.Literal == nil || *rr.Code.Literal != "Pipeline.Failed" {
		t.Errorf("raise code = %+v", rr)
	}
	if rr.Retryable == nil || rr.Retryable.Expr == nil {
		t.Errorf("raise retryable should be an expression, got %+v", rr.Retryable)
	}
}

// TestBindFixtureCorpus is the parser half of the differential check: every
// valid fixture binds without diagnostics, every invalid fixture produces at
// least one error.
func TestBindFixtureCorpus(t *testing.T) {
	valid, err := filepath.Glob("../tests/schemas/valid/flow/*.json")
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := filepath.Glob("../tests/schemas/invalid/flow/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) == 0 || len(invalid) == 0 {
		t.Fatal("fixture corpus not found under ../tests/schemas/")
	}

	for _, path := range valid {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		v, diags := decode(src, filepath.Base(path))
		if len(diags) != 0 {
			t.Errorf("%s: decode diagnostics: %v", path, diags)
			continue
		}
		if _, diags := bind(v); len(diags) != 0 {
			t.Errorf("%s: valid fixture produced diagnostics: %v", path, diags)
		}
	}
	for _, path := range invalid {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		v, diags := decode(src, filepath.Base(path))
		if v == nil {
			continue // rejected at decode; still an error
		}
		_, bindDiags := bind(v)
		if !append(diags, bindDiags...).HasErrors() {
			t.Errorf("%s: invalid fixture bound without errors", path)
		}
	}
}

func TestBindDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
		path string
	}{
		{
			name: "root not object",
			src:  `[]`,
			code: "wrong-type",
		},
		{
			name: "root missing $schema",
			src:  `{"entrypoint": "a", "steps": {"a": {"action": "Return"}}}`,
			code: "missing-member",
		},
		{
			name: "root wrong $schema value",
			src:  `{"$schema": "https://example.com/other.json", "entrypoint": "a", "steps": {"a": {"action": "Return"}}}`,
			code: "invalid-value",
			path: "$schema",
		},
		{
			name: "unknown flow member",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "version": 2, "steps": {"a": {"action": "Return"}}}`,
			code: "unknown-member",
			path: "version",
		},
		{
			name: "empty steps",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "steps": {}}`,
			code: "invalid-value",
			path: "steps",
		},
		{
			name: "expression in entrypoint",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "{{ vars.start }}", "steps": {"a": {"action": "Return"}}}`,
			code: "expression-not-permitted",
			path: "entrypoint",
		},
		{
			name: "unknown action",
			src:  flowDoc(`{"a": {"action": "Frobnicate"}}`),
			code: "invalid-value",
			path: "steps.a",
		},
		{
			name: "expression in next",
			src:  flowDoc(`{"a": {"action": "Pass", "next": "{{ vars.n }}"}}`),
			code: "expression-not-permitted",
			path: "steps.a.next",
		},
		{
			name: "call step missing next",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}}}`),
			code: "missing-member",
			path: "steps.a",
		},
		{
			name: "call both targets",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y", "flow": "F"}, "next": "a"}}`),
			code: "exclusive-members",
			path: "steps.a.call",
		},
		{
			name: "call no target",
			src:  flowDoc(`{"a": {"action": "Call", "call": {}, "next": "a"}}`),
			code: "exclusive-members",
			path: "steps.a.call",
		},
		{
			name: "call provider with middleware URI",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.middleware/mwl/retry/v1"}, "next": "a"}}`),
			code: "invalid-provider-uri",
			path: "steps.a.call.provider",
		},
		{
			name: "provider URI single segment",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/only"}, "next": "a"}}`),
			code: "invalid-provider-uri",
		},
		{
			name: "provider URI dot segment",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/ns/../y"}, "next": "a"}}`),
			code: "invalid-provider-uri",
		},
		{
			name: "gather both forms",
			src:  flowDoc(`{"a": {"action": "Gather", "over": "{{ step.input }}", "call": {"provider": "mwl:provider.call/x/y"}, "calls": [{"provider": "mwl:provider.call/x/y"}], "next": "a"}}`),
			code: "exclusive-members",
			path: "steps.a",
		},
		{
			name: "gather neither form",
			src:  flowDoc(`{"a": {"action": "Gather", "next": "a"}}`),
			code: "exclusive-members",
			path: "steps.a",
		},
		{
			name: "gather over without call",
			src:  flowDoc(`{"a": {"action": "Gather", "over": "{{ step.input }}", "next": "a"}}`),
			code: "missing-member",
			path: "steps.a",
		},
		{
			name: "gather empty calls",
			src:  flowDoc(`{"a": {"action": "Gather", "calls": [], "next": "a"}}`),
			code: "invalid-value",
			path: "steps.a.calls",
		},
		{
			name: "gather concurrency zero",
			src:  flowDoc(`{"a": {"action": "Gather", "calls": [{"provider": "mwl:provider.call/x/y"}], "concurrency": 0, "next": "a"}}`),
			code: "invalid-value",
			path: "steps.a.concurrency",
		},
		{
			name: "gather concurrency expression",
			src:  flowDoc(`{"a": {"action": "Gather", "calls": [{"provider": "mwl:provider.call/x/y"}], "concurrency": "{{ vars.n }}", "next": "a"}}`),
			code: "wrong-type",
			path: "steps.a.concurrency",
		},
		{
			name: "gather with middleware",
			src:  flowDoc(`{"a": {"action": "Gather", "calls": [{"provider": "mwl:provider.call/x/y"}], "middleware": [], "next": "a"}}`),
			code: "unknown-member",
			path: "steps.a.middleware",
		},
		{
			name: "completion missing successes",
			src:  flowDoc(`{"a": {"action": "Gather", "calls": [{"provider": "mwl:provider.call/x/y"}], "completion": {"wait": true}, "next": "a"}}`),
			code: "missing-member",
			path: "steps.a.completion",
		},
		{
			name: "completion wait expression",
			src:  flowDoc(`{"a": {"action": "Gather", "calls": [{"provider": "mwl:provider.call/x/y"}], "completion": {"successes": 1, "wait": "{{ true }}"}, "next": "a"}}`),
			code: "wrong-type",
			path: "steps.a.completion.wait",
		},
		{
			name: "match default with when",
			src:  flowDoc(`{"a": {"action": "Match", "cases": [], "default": {"when": "{{ true }}", "next": "a"}}}`),
			code: "unknown-member",
			path: "steps.a.default.when",
		},
		{
			name: "match case literal when",
			src:  flowDoc(`{"a": {"action": "Match", "cases": [{"when": true, "next": "a"}], "default": {"next": "a"}}}`),
			code: "wrong-type",
			path: "steps.a.cases[0].when",
		},
		{
			name: "sleep both for and until",
			src:  flowDoc(`{"a": {"action": "Sleep", "for": "PT1S", "until": "2026-01-01T00:00:00Z", "next": "a"}}`),
			code: "exclusive-members",
			path: "steps.a",
		},
		{
			name: "sleep neither for nor until",
			src:  flowDoc(`{"a": {"action": "Sleep", "next": "a"}}`),
			code: "exclusive-members",
			path: "steps.a",
		},
		{
			name: "raise result missing code",
			src:  flowDoc(`{"a": {"action": "Raise", "result": {"message": "no code"}}}`),
			code: "missing-member",
			path: "steps.a.result",
		},
		{
			name: "raise type success",
			src:  flowDoc(`{"a": {"action": "Raise", "result": {"type": "success", "code": "X.Y"}}}`),
			code: "invalid-value",
			path: "steps.a.result.type",
		},
		{
			name: "catch clause without match",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}, "catch": [{"next": "a"}], "next": "a"}}`),
			code: "missing-member",
			path: "steps.a.catch[0]",
		},
		{
			name: "matcher empty",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}, "catch": [{"match": {}, "next": "a"}], "next": "a"}}`),
			code: "invalid-value",
			path: "steps.a.catch[0].match",
		},
		{
			name: "matcher codes empty",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}, "catch": [{"match": {"codes": []}, "next": "a"}], "next": "a"}}`),
			code: "invalid-value",
			path: "steps.a.catch[0].match.codes",
		},
		{
			name: "matcher types success",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}, "catch": [{"match": {"types": ["success"]}, "next": "a"}], "next": "a"}}`),
			code: "invalid-value",
			path: "steps.a.catch[0].match.types[0]",
		},
		{
			name: "matcher retryable expression",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y"}, "catch": [{"match": {"retryable": "{{ true }}"}, "next": "a"}], "next": "a"}}`),
			code: "wrong-type",
			path: "steps.a.catch[0].match.retryable",
		},
		{
			name: "subflow with $schema",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "flows": {"F": {"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "s", "steps": {"s": {"action": "Return"}}}}, "steps": {"a": {"action": "Return"}}}`,
			code: "unknown-member",
			path: "flows.F.$schema",
		},
		{
			name: "parameters missing type",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "parameters": {"properties": {}}, "steps": {"a": {"action": "Return"}}}`,
			code: "missing-member",
			path: "parameters",
		},
		{
			name: "parameters type not object",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "parameters": {"type": "array"}, "steps": {"a": {"action": "Return"}}}`,
			code: "invalid-value",
			path: "parameters",
		},
		{
			name: "middleware entry missing provider",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "middleware": [{"onEntry": {}}], "steps": {"a": {"action": "Return"}}}`,
			code: "missing-member",
			path: "middleware[0]",
		},
		{
			name: "middleware provider with call URI",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "middleware": [{"provider": "mwl:provider.call/x/y"}], "steps": {"a": {"action": "Return"}}}`,
			code: "invalid-provider-uri",
			path: "middleware[0].provider",
		},
		{
			name: "onFailure type success",
			src:  `{"$schema": "https://mwl.dev/v0.1/flow/schema.json", "entrypoint": "a", "middleware": [{"provider": "mwl:provider.middleware/x/y", "onFailure": {"type": "success"}}], "steps": {"a": {"action": "Return"}}}`,
			code: "invalid-value",
			path: "middleware[0].onFailure.type",
		},
		{
			name: "with plain string",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y", "with": "literal"}, "next": "a"}}`),
			code: "wrong-type",
			path: "steps.a.call.with",
		},
		{
			name: "onSuccess arm unknown member",
			src:  flowDoc(`{"a": {"action": "Call", "call": {"provider": "mwl:provider.call/x/y", "onSuccess": {"output": 1}}, "next": "a"}}`),
			code: "unknown-member",
			path: "steps.a.call.onSuccess.output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, diags := bindSrc(t, tt.src)
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

func TestBindConcurrencyNullBindsAsAbsent(t *testing.T) {
	src := flowDoc(`{"a": {"action": "Gather", "calls": [{"provider": "mwl:provider.call/x/y"}], "concurrency": null, "next": "done"}, "done": {"action": "Return"}}`)
	f, diags := bindSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	ns, _ := f.StepByName("a")
	g := ns.Step.(*GatherStep)
	if g.Concurrency != nil {
		t.Errorf("concurrency = %+v; an explicit null is an empty spelling and binds as absent", g.Concurrency)
	}
}

func TestBindCollectsAcrossSteps(t *testing.T) {
	src := flowDoc(`{
		"a": {"action": "Frobnicate"},
		"b": {"action": "Sleep", "next": "c"},
		"c": {"action": "Return"}
	}`)
	_, diags := bindSrc(t, src)
	if got := len(diags); got != 2 {
		t.Fatalf("diagnostics = %v, want 2 (one per bad step)", diags)
	}
	if !strings.Contains(diags[0].Message, "Frobnicate") {
		t.Errorf("first diagnostic %q should be the unknown action", diags[0].Message)
	}
	if diags[1].Code != "exclusive-members" {
		t.Errorf("second diagnostic %+v should be the Sleep form error", diags[1])
	}
}
