package parser

import (
	"strings"
	"testing"
)

// mustDecode decodes src and fails the test on any diagnostic.
func mustDecode(t *testing.T, src string) *Value {
	t.Helper()
	v, diags := decode([]byte(src), "test.json")
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if v == nil {
		t.Fatal("nil tree without diagnostics")
	}
	return v
}

// offsetOf locates the first occurrence of sub in src, failing the test if
// absent. Tests derive expected positions from it rather than hand-counting.
func offsetOf(t *testing.T, src, sub string) int {
	t.Helper()
	i := strings.Index(src, sub)
	if i < 0 {
		t.Fatalf("%q not found in source", sub)
	}
	return i
}

func TestDecodeScalars(t *testing.T) {
	src := `{"null": null, "true": true, "false": false, "num": 1.50, "str": "hi"}`
	v := mustDecode(t, src)

	if got := v.Field("null"); got.Kind != KindNull {
		t.Errorf("null member: kind %v", got.Kind)
	}
	if got := v.Field("true"); got.Kind != KindBool || !got.Bool {
		t.Errorf("true member: kind %v bool %v", got.Kind, got.Bool)
	}
	if got := v.Field("false"); got.Kind != KindBool || got.Bool {
		t.Errorf("false member: kind %v bool %v", got.Kind, got.Bool)
	}
	if got := v.Field("str"); got.Kind != KindString || got.Str != "hi" {
		t.Errorf("str member: kind %v str %q", got.Kind, got.Str)
	}
}

func TestDecodeNumberLexemePreserved(t *testing.T) {
	src := `{"a": 1.50, "b": 9007199254740993, "c": 1e3}`
	v := mustDecode(t, src)
	for member, want := range map[string]string{"a": "1.50", "b": "9007199254740993", "c": "1e3"} {
		if got := v.Field(member).Number; got != want {
			t.Errorf("member %q: number lexeme %q, want %q", member, got, want)
		}
	}
}

func TestDecodeMemberOrder(t *testing.T) {
	src := `{"zebra": 1, "apple": 2, "mango": 3}`
	v := mustDecode(t, src)
	want := []string{"zebra", "apple", "mango"}
	if len(v.Object) != len(want) {
		t.Fatalf("member count %d, want %d", len(v.Object), len(want))
	}
	for i, name := range want {
		if v.Object[i].Name != name {
			t.Errorf("member %d: %q, want %q", i, v.Object[i].Name, name)
		}
	}
}

func TestDecodePositions(t *testing.T) {
	src := "{\n  \"steps\": {\n    \"greet\": [true, null]\n  }\n}"
	v := mustDecode(t, src)

	if v.Pos.Offset != 0 || v.Pos.Line != 1 || v.Pos.Column != 1 {
		t.Errorf("root position %+v, want offset 0, 1:1", v.Pos)
	}

	steps := v.Object[0]
	if want := offsetOf(t, src, `"steps"`); steps.NamePos.Offset != want {
		t.Errorf("steps name offset %d, want %d", steps.NamePos.Offset, want)
	}
	if steps.NamePos.Line != 2 || steps.NamePos.Column != 3 {
		t.Errorf("steps name at %d:%d, want 2:3", steps.NamePos.Line, steps.NamePos.Column)
	}
	if want := offsetOf(t, src, `{`+"\n    "); steps.Value.Pos.Offset != want {
		t.Errorf("steps value offset %d, want %d", steps.Value.Pos.Offset, want)
	}

	greet := steps.Value.Object[0]
	if greet.NamePos.Line != 3 || greet.NamePos.Column != 5 {
		t.Errorf("greet name at %d:%d, want 3:5", greet.NamePos.Line, greet.NamePos.Column)
	}
	arr := greet.Value
	if want := offsetOf(t, src, `[true`); arr.Pos.Offset != want {
		t.Errorf("array offset %d, want %d", arr.Pos.Offset, want)
	}
	if want := offsetOf(t, src, `true`); arr.Array[0].Pos.Offset != want {
		t.Errorf("array[0] offset %d, want %d", arr.Array[0].Pos.Offset, want)
	}
	if want := offsetOf(t, src, `null`); arr.Array[1].Pos.Offset != want {
		t.Errorf("array[1] offset %d, want %d", arr.Array[1].Pos.Offset, want)
	}

	if got := steps.NamePos.String(); got != "test.json:2:3" {
		t.Errorf("position string %q, want %q", got, "test.json:2:3")
	}
	if got := (Position{Line: 2, Column: 3}).String(); got != "2:3" {
		t.Errorf("position string without filename %q, want %q", got, "2:3")
	}
}

func TestDecodeDuplicateMember(t *testing.T) {
	src := `{"a": 1, "a": 2}`
	v, diags := decode([]byte(src), "test.json")

	if len(diags) != 1 {
		t.Fatalf("diagnostics %v, want exactly one", diags)
	}
	d := diags[0]
	if d.Severity != SeverityError || d.Code != "duplicate-member" {
		t.Errorf("diagnostic %+v, want duplicate-member error", d)
	}
	if d.Path != "a" {
		t.Errorf("path %q, want %q", d.Path, "a")
	}
	if want := offsetOf(t, src, `"a": 2`); d.Pos.Offset != want {
		t.Errorf("diagnostic offset %d, want %d (the second key)", d.Pos.Offset, want)
	}
	if !strings.Contains(d.Message, `"a"`) || !strings.Contains(d.Message, "1:2") {
		t.Errorf("message %q should name the member and the first position", d.Message)
	}

	// The tree survives, keeping the first occurrence.
	if v == nil {
		t.Fatal("tree is nil despite recoverable error")
	}
	if len(v.Object) != 1 || v.Object[0].Value.Number != "1" {
		t.Errorf("tree kept %d members, want first occurrence only", len(v.Object))
	}
}

func TestDecodeDuplicateMemberNestedPath(t *testing.T) {
	src := `{"steps": {"greet": {"x": 1, "x": 2}}, "arr": [{"k": 1, "k": 2}]}`
	_, diags := decode([]byte(src), "")

	if len(diags) != 2 {
		t.Fatalf("diagnostics %v, want two", diags)
	}
	if got, want := diags[0].Path, "steps.greet.x"; got != want {
		t.Errorf("path %q, want %q", got, want)
	}
	if got, want := diags[1].Path, "arr[0].k"; got != want {
		t.Errorf("path %q, want %q", got, want)
	}
}

func TestDecodeSyntaxError(t *testing.T) {
	v, diags := decode([]byte(`{"a": }`), "test.json")
	if v != nil {
		t.Error("tree should be nil on syntax error")
	}
	if len(diags) != 1 || diags[0].Code != "json-syntax" || diags[0].Severity != SeverityError {
		t.Fatalf("diagnostics %v, want one json-syntax error", diags)
	}
}

func TestDecodeUnexpectedEnd(t *testing.T) {
	v, diags := decode([]byte(`{"a": 1`), "test.json")
	if v != nil {
		t.Error("tree should be nil on truncated input")
	}
	if len(diags) != 1 || diags[0].Code != "json-syntax" {
		t.Fatalf("diagnostics %v, want one json-syntax error", diags)
	}
}

func TestDecodeEmptyDocument(t *testing.T) {
	for _, src := range []string{"", "  \n\t"} {
		v, diags := decode([]byte(src), "test.json")
		if v != nil {
			t.Errorf("%q: tree should be nil", src)
		}
		if len(diags) != 1 || diags[0].Code != "empty-document" {
			t.Errorf("%q: diagnostics %v, want one empty-document error", src, diags)
		}
	}
}

func TestDecodeTrailingContent(t *testing.T) {
	src := `{"a": 1} {"b": 2}`
	v, diags := decode([]byte(src), "test.json")
	if v == nil {
		t.Fatal("tree should survive trailing content")
	}
	if len(diags) != 1 || diags[0].Code != "trailing-content" {
		t.Fatalf("diagnostics %v, want one trailing-content error", diags)
	}
	if want := offsetOf(t, src, `{"b"`); diags[0].Pos.Offset != want {
		t.Errorf("diagnostic offset %d, want %d", diags[0].Pos.Offset, want)
	}
}

func TestDecodeNonObjectRoot(t *testing.T) {
	// The decode layer is generic JSON; requiring an object root is the
	// bind pass's job.
	v := mustDecode(t, `[1, "two", false]`)
	if v.Kind != KindArray || len(v.Array) != 3 {
		t.Errorf("root %v with %d elements, want array of 3", v.Kind, len(v.Array))
	}
}

func TestFieldLookup(t *testing.T) {
	v := mustDecode(t, `{"a": 1}`)
	if v.Field("missing") != nil {
		t.Error("missing member should return nil")
	}
	if v.Field("a").Field("nested") != nil {
		t.Error("Field on a non-object should return nil")
	}
	var nilValue *Value
	if nilValue.Field("a") != nil {
		t.Error("Field on a nil Value should return nil")
	}
}

func TestDiagnosticsHasErrors(t *testing.T) {
	if (Diagnostics{}).HasErrors() {
		t.Error("empty diagnostics should have no errors")
	}
	warnOnly := Diagnostics{{Severity: SeverityWarning}}
	if warnOnly.HasErrors() {
		t.Error("warnings alone are not errors")
	}
	mixed := Diagnostics{{Severity: SeverityWarning}, {Severity: SeverityError}}
	if !mixed.HasErrors() {
		t.Error("an error among warnings should be reported")
	}
}

func TestDiagnosticString(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityError,
		Code:     "duplicate-member",
		Pos:      Position{Filename: "flow.json", Line: 3, Column: 7},
		Message:  `duplicate object member "a"`,
	}
	want := `flow.json:3:7: error: duplicate object member "a" [duplicate-member]`
	if got := d.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
